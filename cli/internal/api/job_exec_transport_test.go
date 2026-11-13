// Copyright 2026 The Crater Project Team, RAIDS-Lab

package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gorilla/websocket"
)

// failingExecConn models an input-side transport failure after the server has
// queued its final output. Reads still carry the authoritative exit frame.
type failingExecConn struct {
	net.Conn
	failWrites chan struct{}
	writeError chan struct{}
	closed     chan struct{}
	writeOnce  sync.Once
	closeOnce  sync.Once
}

func (c *failingExecConn) Write(p []byte) (int, error) {
	select {
	case <-c.failWrites:
		c.writeOnce.Do(func() { close(c.writeError) })
		return 0, io.ErrClosedPipe
	default:
		return c.Conn.Write(p)
	}
}

func (c *failingExecConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type execWriterFunc func([]byte) (int, error)

func (f execWriterFunc) Write(p []byte) (int, error) { return f(p) }

type execReaderFunc func([]byte) (int, error)

func (f execReaderFunc) Read(p []byte) (int, error) { return f(p) }

func TestExecDrainsOutputAfterInputTransportFailure(t *testing.T) {
	for _, tt := range []struct {
		name          string
		resize        bool
		eof           bool
		missingExit   bool
		outputFailure bool
		exitCode      int
	}{
		{name: "stdin with successful exit"},
		{name: "stdin with nonzero exit", exitCode: 7},
		{name: "stdin EOF", eof: true},
		{name: "terminal resize", resize: true},
		{name: "missing exit remains a failure", missingExit: true},
		{name: "local output failure is preserved", outputFailure: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			queued := make(chan struct{})
			payload := bytes.Repeat([]byte{0, 0xff, 'a', '\n'}, 16<<10)
			server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
				var msg execFrame
				if err := ws.ReadJSON(&msg); err != nil {
					t.Error(err)
					return
				}
				frames := []execFrame{
					execReady(), {Type: "stdout", Data: []byte("prefix")},
					{Type: "stdout", Data: payload}, {Type: "stderr", Data: []byte("diagnostic")},
				}
				if !tt.missingExit {
					frames = append(frames, execFrame{Type: "exit", ExitCode: &tt.exitCode})
				}
				for _, frame := range frames {
					if err := ws.WriteJSON(frame); err != nil {
						t.Error(err)
						return
					}
				}
				close(queued)
				if tt.missingExit {
					return
				}
				// Keep the read half open until the client finishes or cancels.
				for ws.ReadJSON(&msg) == nil {
				}
			})
			transport := &failingExecConn{failWrites: queued, writeError: make(chan struct{}), closed: make(chan struct{})}
			dialer := websocket.Dialer{
				Subprotocols: []string{execSubprotocol},
				NetDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
					if err != nil {
						return nil, err
					}
					transport.Conn = conn
					return transport, nil
				},
			}
			ws, resp, err := dialer.DialContext(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				if resp != nil {
					_ = resp.Body.Close()
				}
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = ws.Close() })
			options := JobExecOptions{Command: []string{"head", "-c", "65536"}}
			if tt.resize {
				input, writer := io.Pipe()
				t.Cleanup(func() { _ = writer.Close() })
				options.TTY, options.Input = true, input
				sizes := make(chan TerminalSize)
				options.Resize = sizes
				go func() {
					defer close(sizes)
					select {
					case <-queued:
					case <-ctx.Done():
						return
					}
					select {
					case sizes <- TerminalSize{Cols: 100, Rows: 30}:
					case <-ctx.Done():
					}
				}()
			} else {
				options.Input = io.NopCloser(execReaderFunc(func(p []byte) (int, error) {
					select {
					case <-queued:
						if tt.eof {
							return 0, io.EOF
						}
						return copy(p, "unused input"), nil
					case <-ctx.Done():
						return 0, ctx.Err()
					}
				}))
			}
			var stdout, stderr bytes.Buffer
			var firstWrite sync.Once
			var closedEarly atomic.Bool
			outputError := errors.New("local stdout failed")
			blockedOutput := execWriterFunc(func(p []byte) (int, error) {
				firstWrite.Do(func() {
					select {
					case <-transport.writeError:
					case <-ctx.Done():
						return
					}
					// Let cancellation close the socket while stdout is backpressured.
					// A healthy receive half remains open throughout this interval.
					select {
					case <-transport.closed:
						closedEarly.Store(true)
					case <-time.After(100 * time.Millisecond):
					case <-ctx.Done():
					}
				})
				if tt.outputFailure {
					return 0, outputError
				}
				return stdout.Write(p)
			})
			result, err := streamJobExec(ctx, ws, options, blockedOutput, &stderr)
			if closedEarly.Load() {
				t.Fatal("connection closed before receiving final output")
			}
			if tt.outputFailure {
				var streamErr *ExecStreamError
				if !errors.As(err, &streamErr) || !streamErr.Local || !errors.Is(err, outputError) {
					t.Fatalf("error=%v, want local stdout failure", err)
				}
				return
			}
			if tt.missingExit {
				var networkErr *NetworkError
				if !errors.As(err, &networkErr) {
					t.Fatalf("error=%v, want input transport failure without an exit frame", err)
				}
			} else if err != nil || result.ExitCode != tt.exitCode {
				t.Fatalf("closed before draining=%v, result=%+v, error=%v; want exit %d", closedEarly.Load(), result, err, tt.exitCode)
			}
			want := append([]byte("prefix"), payload...)
			if !bytes.Equal(stdout.Bytes(), want) || stderr.String() != "diagnostic" {
				t.Fatalf("stdout length=%d, stderr=%q; want %d bytes and diagnostic", stdout.Len(), stderr.String(), len(want))
			}
		})
	}
}

func TestExecLocalInputFailureCancelsSession(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cause error
	}{
		{"read failure", errors.New("local stdin failed")},
		{"read failure wrapping a network error", &NetworkError{Cause: io.ErrUnexpectedEOF}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
				var msg execFrame
				for ws.ReadJSON(&msg) == nil {
				}
			})
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			_, err := NewClient(server.URL).StreamJobExec(ctx, "job", JobExecOptions{
				Command: []string{"cat"}, Input: io.NopCloser(iotest.ErrReader(tt.cause)),
			}, io.Discard, io.Discard)
			var streamErr *ExecStreamError
			if ctx.Err() != nil || !errors.As(err, &streamErr) || !streamErr.Local || !errors.Is(err, tt.cause) {
				t.Fatalf("error=%v, context error=%v; want local stdin failure before timeout", err, ctx.Err())
			}
		})
	}
}
