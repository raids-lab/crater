package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/raids-lab/crater/cli/internal/testenv"
	"github.com/raids-lab/crater/cli/internal/version"
)

const execSubprotocol = "crater.exec.v1"

// TerminalSize is independent of the node's CPU or accelerator type.
type TerminalSize struct {
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// JobExecOptions describes exact argv and the streams of one container command.
// StreamJobExec closes Input on completion to unblock pending input reads.
type JobExecOptions struct {
	Pod, Container string
	Command        []string
	EnvMode        string
	TTY            bool
	Input          io.ReadCloser
	Size           TerminalSize
	Resize         <-chan TerminalSize
}

// JobExecResult contains the server-confirmed target and process exit status.
type JobExecResult struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	ExitCode  int    `json:"exit_code"`
}

// ExecStreamError distinguishes protocol and local I/O failures from HTTP errors.
type ExecStreamError struct {
	Local bool
	Cause error
}

func (e *ExecStreamError) Error() string { return e.Cause.Error() }
func (e *ExecStreamError) Unwrap() error { return e.Cause }

type execFrame struct {
	Type    string   `json:"type"`
	Command []string `json:"command,omitempty"`
	EnvMode string   `json:"env_mode,omitempty"`
	Stdin   bool     `json:"stdin,omitempty"`
	TTY     bool     `json:"tty,omitempty"`
	Data    []byte   `json:"data,omitempty"`
	TerminalSize
	ExitCode  *int   `json:"exit_code,omitempty"`
	Message   string `json:"message,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

type execSocket struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *execSocket) send(frame execFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return s.conn.WriteJSON(frame)
}

func (c *Client) dialJobExec(ctx context.Context, name string, options JobExecOptions) (*websocket.Conn, error) {
	path := VCJobsPrefix + "/" + url.PathEscape(name) + "/exec"
	mode := testenv.SandboxHTTPMode()
	// Keep transport simulations on the shared HTTP client; no socket may escape
	// the test sandbox when HTTP is configured to fail deterministically.
	if mode != "" && mode != "passthrough" {
		var result Response[any]
		resp, err := c.httpClient.R().SetContext(ctx).SetSuccessResult(&result).SetErrorResult(&result).Get(path)
		if err != nil {
			return nil, &NetworkError{Cause: err}
		}
		if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
			return nil, err
		}
		return nil, &ExecStreamError{Cause: errors.New("expected WebSocket upgrade")}
	}
	u, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + path)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, &ExecStreamError{Cause: errors.New("platform URL must use HTTP or HTTPS without user information")}
	}
	if mode == "passthrough" {
		ip := net.ParseIP(u.Hostname())
		if !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, &NetworkError{Cause: errors.New("test sandbox rejected non-loopback exec host")}
		}
	}
	query := u.Query()
	if options.Pod != "" {
		query.Set("pod", options.Pod)
	}
	if options.Container != "" {
		query.Set("container", options.Container)
	}
	u.RawQuery = query.Encode()
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	headers := http.Header{
		"Authorization":          {"Bearer " + c.token},
		"User-Agent":             {version.UserAgent()},
		version.APIVersionHeader: {strconv.Itoa(version.APIVersion)},
	}
	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 10 * time.Second, Subprotocols: []string{execSubprotocol}}
	conn, resp, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			var result Response[any]
			_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result)
			if result.Message == "" {
				result.Message = http.StatusText(resp.StatusCode)
			}
			return nil, &RequestError{HTTPStatus: resp.StatusCode, CraterCode: result.Code, Msg: result.Message}
		}
		return nil, &NetworkError{Cause: err}
	}
	if conn.Subprotocol() != execSubprotocol {
		_ = conn.Close()
		return nil, &ExecStreamError{Cause: errors.New("platform did not negotiate crater.exec.v1")}
	}
	conn.SetReadLimit(1 << 20)
	return conn, nil
}

// StreamJobExec forwards raw bytes and requires a final exit frame. Input must
// support Close interrupting Read; callers retain ownership of stdout/stderr.
func (c *Client) StreamJobExec(ctx context.Context, name string, options JobExecOptions, stdout, stderr io.Writer) (JobExecResult, error) {
	conn, err := c.dialJobExec(ctx, name, options)
	if err != nil {
		if options.Input != nil {
			_ = options.Input.Close()
		}
		return JobExecResult{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	socket := &execSocket{conn: conn}
	var workers sync.WaitGroup
	defer func() {
		cancel()
		_ = conn.Close()
		if options.Input != nil {
			_ = options.Input.Close()
		}
		workers.Wait()
	}()
	if err := socket.send(execFrame{Type: "start", Command: options.Command, EnvMode: options.EnvMode, Stdin: options.Input != nil, TTY: options.TTY, TerminalSize: options.Size}); err != nil {
		return JobExecResult{}, &NetworkError{Cause: err}
	}
	workers.Go(func() { <-ctx.Done(); _ = conn.Close() })
	inputErrors := make(chan error, 1)
	if options.Input != nil {
		workers.Go(func() {
			if err := forwardExecInput(socket, options.Input); err != nil {
				inputErrors <- err
				cancel()
			}
		})
	}
	if options.Resize != nil {
		workers.Go(func() {
			for {
				select {
				case size, ok := <-options.Resize:
					if !ok {
						return
					}
					if err := socket.send(execFrame{Type: "resize", TerminalSize: size}); err != nil {
						cancel()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		})
	}
	result, err := receiveExec(conn, stdout, stderr)
	// A command can finish without consuming stdin. Its explicit exit frame
	// remains authoritative even if an in-flight input write then sees closure.
	if err == nil {
		return result, nil
	}
	select {
	case inputErr := <-inputErrors:
		return result, inputErr
	default:
		return result, err
	}
}

func forwardExecInput(socket *execSocket, input io.Reader) error {
	buf := make([]byte, 32<<10)
	for {
		n, err := input.Read(buf)
		if n > 0 {
			if sendErr := socket.send(execFrame{Type: "stdin", Data: buf[:n]}); sendErr != nil {
				return &NetworkError{Cause: sendErr}
			}
		}
		if errors.Is(err, io.EOF) {
			if sendErr := socket.send(execFrame{Type: "eof"}); sendErr != nil {
				return &NetworkError{Cause: sendErr}
			}
			return nil
		}
		if err != nil {
			return &ExecStreamError{Local: true, Cause: err}
		}
	}
}

func receiveExec(conn *websocket.Conn, stdout, stderr io.Writer) (JobExecResult, error) {
	result := JobExecResult{}
	ready := false
	for {
		var msg execFrame
		if err := conn.ReadJSON(&msg); err != nil {
			return result, &ExecStreamError{Cause: fmt.Errorf("exec closed without a final exit status: %w", err)}
		}
		switch msg.Type {
		case "ready":
			if ready || msg.Namespace == "" || msg.Pod == "" || msg.Container == "" {
				return result, &ExecStreamError{Cause: errors.New("invalid exec target frame")}
			}
			result.Namespace, result.Pod, result.Container = msg.Namespace, msg.Pod, msg.Container
			ready = true
		case "stdout", "stderr":
			if !ready {
				return result, &ExecStreamError{Cause: errors.New("exec output arrived before target confirmation")}
			}
			dst := stdout
			if msg.Type == "stderr" {
				dst = stderr
			}
			n, err := dst.Write(msg.Data)
			if err == nil && n != len(msg.Data) {
				err = io.ErrShortWrite
			}
			if err != nil {
				return result, &ExecStreamError{Local: true, Cause: err}
			}
		case "exit":
			if !ready || msg.ExitCode == nil || *msg.ExitCode < 0 || *msg.ExitCode > 255 {
				return result, &ExecStreamError{Cause: errors.New("invalid exec exit frame")}
			}
			result.ExitCode = *msg.ExitCode
			return result, nil
		case "error":
			return result, &ExecStreamError{Cause: errors.New(msg.Message)}
		default:
			return result, &ExecStreamError{Cause: errors.New("unknown exec frame")}
		}
	}
}
