// Package jobexec bridges a versioned WebSocket session to Kubernetes exec.
package jobexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/klog/v2"
)

const (
	// Subprotocol identifies the argv, byte streams, resize, and exit protocol.
	Subprotocol  = "crater.exec.v1"
	frameLimit   = 1 << 20
	writeTimeout = 10 * time.Second
)

// Message is a protocol frame. Data is Base64-encoded by JSON to preserve bytes.
type Message struct {
	Type      string   `json:"type"`
	Command   []string `json:"command,omitempty"`
	EnvMode   string   `json:"env_mode,omitempty"`
	Stdin     bool     `json:"stdin,omitempty"`
	TTY       bool     `json:"tty,omitempty"`
	Data      []byte   `json:"data,omitempty"`
	Cols      uint16   `json:"cols,omitempty"`
	Rows      uint16   `json:"rows,omitempty"`
	ExitCode  *int     `json:"exit_code,omitempty"`
	Message   string   `json:"message,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Pod       string   `json:"pod,omitempty"`
	Container string   `json:"container,omitempty"`
}

// ReadStart reads a bounded initial command without exposing argv in the URL.
func ReadStart(ws *websocket.Conn) (Message, error) {
	ws.SetReadLimit(frameLimit)
	if err := ws.SetReadDeadline(time.Now().Add(writeTimeout)); err != nil {
		return Message{}, err
	}
	var start Message
	if err := ws.ReadJSON(&start); err != nil {
		return start, err
	}
	if err := validateStart(&start); err != nil {
		return start, err
	}
	return start, ws.SetReadDeadline(time.Time{})
}

func validateStart(start *Message) error {
	if start.Type != "start" || (len(start.Command) == 0 && (!start.TTY || !start.Stdin)) ||
		(len(start.Command) > 0 && start.Command[0] == "") || (start.TTY && !start.Stdin) {
		return fmt.Errorf("start requires command argv or an interactive terminal; TTY requires stdin")
	}
	if start.EnvMode != "" && start.EnvMode != "user" && start.EnvMode != "container" {
		return fmt.Errorf("env_mode must be user or container")
	}
	for _, arg := range start.Command {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("command arguments cannot contain NUL")
		}
	}
	return nil
}

type session struct {
	ws     *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	sizes  chan remotecommand.TerminalSize
}

func (s *session) send(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return s.ws.WriteJSON(msg)
}

func (s *session) Next() *remotecommand.TerminalSize {
	select {
	case size := <-s.sizes:
		return &size
	case <-s.ctx.Done():
		return nil
	}
}

type streamWriter struct {
	s    *session
	kind string
}

func (w streamWriter) Write(p []byte) (int, error) {
	const chunkSize = 32 << 10
	written := 0
	for len(p) > 0 {
		size := min(len(p), chunkSize)
		if err := w.s.send(&Message{Type: w.kind, Data: p[:size]}); err != nil {
			w.s.cancel()
			return written, err
		}
		written += size
		p = p[size:]
	}
	return written, nil
}

func (s *session) readInput(input *io.PipeWriter, start *Message) error {
	eof := !start.Stdin
	for {
		var msg Message
		if err := s.ws.ReadJSON(&msg); err != nil {
			return err
		}
		switch msg.Type {
		case "stdin":
			if eof {
				return fmt.Errorf("stdin is disabled or already closed")
			}
			if _, err := input.Write(msg.Data); err != nil {
				return err
			}
		case "eof":
			eof = true
			if err := input.Close(); err != nil {
				return err
			}
		case "resize":
			if !start.TTY || msg.Cols == 0 || msg.Rows == 0 {
				return fmt.Errorf("resize requires TTY and positive dimensions")
			}
			select {
			case s.sizes <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
			case <-s.ctx.Done():
				return s.ctx.Err()
			}
		default:
			return fmt.Errorf("unsupported exec input frame")
		}
	}
}

// Run forwards streams and always sends an explicit exit or error frame before
// closing. Closing the client cancels Kubernetes exec and unblocks stdin/resize.
func Run(ctx context.Context, ws *websocket.Conn, executor remotecommand.Executor, start, target *Message, inputPrefix []byte) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s := &session{ws: ws, ctx: ctx, cancel: cancel, sizes: make(chan remotecommand.TerminalSize, 1)}
	if err := s.send(target); err != nil {
		return
	}
	if start.TTY && start.Cols > 0 && start.Rows > 0 {
		s.sizes <- remotecommand.TerminalSize{Width: start.Cols, Height: start.Rows}
	}
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := s.readInput(writer, start); err != nil {
			cancel()
		}
	}()
	watchDone := make(chan struct{})
	go s.watch(watchDone)
	options := remotecommand.StreamOptions{Stdout: streamWriter{s, "stdout"}, Tty: start.TTY}
	if start.Stdin {
		options.Stdin = input
	}
	if len(inputPrefix) > 0 {
		if options.Stdin == nil {
			options.Stdin = bytes.NewReader(inputPrefix)
		} else {
			options.Stdin = io.MultiReader(bytes.NewReader(inputPrefix), options.Stdin)
		}
	}
	if start.TTY {
		options.TerminalSizeQueue = s
	} else {
		options.Stderr = streamWriter{s, "stderr"}
	}
	s.finish(executor.StreamWithContext(ctx, options))
	cancel()
	_ = input.Close()
	_ = writer.Close()
	<-watchDone
	<-done
}

func (s *session) watch(done chan<- struct{}) {
	defer close(done)
	const heartbeatInterval = 20 * time.Second
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			_ = s.ws.Close()
			return
		case <-ticker.C:
			if err := s.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
				s.cancel()
			}
		}
	}
}

func (s *session) finish(err error) {
	const diagnosticLogLevel = 4
	code := 0
	var exitErr utilexec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitStatus()
	default:
		klog.ErrorS(err, "Container exec stream failed")
		message := &Message{Type: "error", Message: "container execution failed; check command, container state, and platform connection"}
		if sendErr := s.send(message); sendErr != nil {
			klog.V(diagnosticLogLevel).InfoS("Exec error frame could not be delivered")
		}
	}
	if err == nil || exitErr != nil {
		if sendErr := s.send(&Message{Type: "exit", ExitCode: &code}); sendErr != nil {
			klog.V(diagnosticLogLevel).InfoS("Exec exit frame could not be delivered")
		}
	}
}
