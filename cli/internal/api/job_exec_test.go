package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newExecServer(t *testing.T, serve func(*websocket.Conn, *http.Request)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{Subprotocols: []string{execSubprotocol}}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer ws.Close()
		serve(ws, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func execReady() execFrame {
	return execFrame{Type: "ready", Namespace: "actual-ns", Pod: "cpu-worker", Container: "main"}
}

func TestJobExecExactArgvAndBytes(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
	argv := []string{"printf", "%s", "a b", "$(ignored)", "--json", ""}
	server := newExecServer(t, func(ws *websocket.Conn, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-token" || r.URL.Query().Has("token") || r.URL.Path != VCJobsPrefix+"/cpu-job/exec" || r.URL.Query().Get("pod") != "cpu-worker" {
			t.Errorf("unexpected exec request path or authentication")
		}
		var start execFrame
		if err := ws.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		if start.Type != "start" || !reflect.DeepEqual(start.Command, argv) {
			t.Errorf("start = %+v, want argv %q", start, argv)
		}
		code := 7
		for _, msg := range []execFrame{execReady(), {Type: "stdout", Data: []byte{0xff, 0, 'a'}}, {Type: "stderr", Data: []byte("diagnostic")}, {Type: "exit", ExitCode: &code}} {
			if err := ws.WriteJSON(msg); err != nil {
				t.Error(err)
				return
			}
		}
	})
	var stdout, stderr bytes.Buffer
	result, err := NewClient(server.URL).SetToken("fixture-token").StreamJobExec(t.Context(), "cpu-job", JobExecOptions{Command: argv, Pod: "cpu-worker"}, &stdout, &stderr)
	if err != nil || result.ExitCode != 7 || result.Namespace != "actual-ns" || !bytes.Equal(stdout.Bytes(), []byte{0xff, 0, 'a'}) || stderr.String() != "diagnostic" {
		t.Fatalf("result=%+v, stdout=%q, stderr=%q, error=%v", result, stdout.Bytes(), stderr.String(), err)
	}
}

func TestJobExecInputEOF(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
	server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
		var start execFrame
		if err := ws.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		if !start.Stdin {
			t.Error("stdin not enabled")
		}
		var input []byte
		for {
			var msg execFrame
			if err := ws.ReadJSON(&msg); err != nil {
				t.Error(err)
				return
			}
			if msg.Type == "eof" {
				break
			}
			input = append(input, msg.Data...)
		}
		code := 0
		for _, msg := range []execFrame{execReady(), {Type: "stdout", Data: input}, {Type: "exit", ExitCode: &code}} {
			if err := ws.WriteJSON(msg); err != nil {
				t.Error(err)
				return
			}
		}
	})
	var output bytes.Buffer
	_, err := NewClient(server.URL).StreamJobExec(t.Context(), "job", JobExecOptions{Command: []string{"cat"}, Input: io.NopCloser(strings.NewReader("输入 after EOF"))}, &output, io.Discard)
	if err != nil || output.String() != "输入 after EOF" {
		t.Fatalf("output=%q, error=%v", output.String(), err)
	}
}

func TestJobExecRejectsIncompleteProtocol(t *testing.T) {
	for _, tt := range []struct {
		name   string
		frames []execFrame
	}{
		{"closed without exit", []execFrame{execReady()}},
		{"exit lacks status", []execFrame{execReady(), {Type: "exit"}}},
		{"output before target", []execFrame{{Type: "stdout", Data: []byte("invalid")}}},
		{"server error", []execFrame{{Type: "error", Message: "container unavailable"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
				var start execFrame
				if err := ws.ReadJSON(&start); err != nil {
					t.Error(err)
					return
				}
				for _, msg := range tt.frames {
					if err := ws.WriteJSON(msg); err != nil {
						t.Error(err)
						return
					}
				}
			})
			_, err := NewClient(server.URL).StreamJobExec(t.Context(), "job", JobExecOptions{Command: []string{"true"}}, io.Discard, io.Discard)
			var streamErr *ExecStreamError
			if !errors.As(err, &streamErr) {
				t.Fatalf("error=%v, want protocol error", err)
			}
		})
	}
}

func TestJobExecCancellationClosesBlockedInput(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
	ready := make(chan struct{})
	server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
		var start execFrame
		if err := ws.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		close(ready)
		for {
			if err := ws.ReadJSON(&start); err != nil {
				return
			}
		}
	})
	input, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := NewClient(server.URL).StreamJobExec(ctx, "job", JobExecOptions{Command: []string{"cat"}, Input: input}, io.Discard, io.Discard)
		done <- err
	}()
	<-ready
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled exec returned success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled input goroutine leaked")
	}
}

func TestJobExecSandboxAndHTTPFailures(t *testing.T) {
	t.Run("simulated HTTP failure", func(t *testing.T) {
		t.Setenv("CRATER_TEST_SANDBOX_HTTP", "error404")
		_, err := NewClient("http://external.invalid").StreamJobExec(t.Context(), "job", JobExecOptions{}, io.Discard, io.Discard)
		var requestErr *RequestError
		if !errors.As(err, &requestErr) || requestErr.HTTPStatus != 404 {
			t.Fatalf("error=%v, want HTTP 404", err)
		}
	})
	t.Run("loopback guard", func(t *testing.T) {
		t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
		_, err := NewClient("http://external.invalid").StreamJobExec(t.Context(), "job", JobExecOptions{}, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "non-loopback") {
			t.Fatalf("error=%v, want sandbox rejection", err)
		}
	})
	t.Run("real authorization response", func(t *testing.T) {
		t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = io.WriteString(w, `{"code":40301,"msg":"job belongs to another project"}`)
		}))
		t.Cleanup(server.Close)
		_, err := NewClient(server.URL).StreamJobExec(t.Context(), "job", JobExecOptions{}, io.Discard, io.Discard)
		var requestErr *RequestError
		if !errors.As(err, &requestErr) || requestErr.HTTPStatus != 403 || requestErr.CraterCode != 40301 {
			t.Fatalf("error=%v, want structured forbidden", err)
		}
	})
}
