package jobexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
)

type testExecutor struct {
	run func(context.Context, remotecommand.StreamOptions) error
}

func (e testExecutor) Stream(options remotecommand.StreamOptions) error {
	return e.run(context.Background(), options)
}
func (e testExecutor) StreamWithContext(ctx context.Context, options remotecommand.StreamOptions) error {
	return e.run(ctx, options)
}

func testSession(t *testing.T, executor testExecutor, prefix ...[]byte) (connection *websocket.Conn, finished <-chan struct{}) {
	t.Helper()
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		upgrader := websocket.Upgrader{Subprotocols: []string{Subprotocol}}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer ws.Close()
		start, err := ReadStart(ws)
		if err != nil {
			t.Error(err)
			return
		}
		var initial []byte
		if len(prefix) > 0 {
			initial = prefix[0]
		}
		Run(r.Context(), ws, executor, &start, &Message{Type: "ready", Namespace: "ns", Pod: "cpu-worker", Container: "main"}, initial)
	}))
	t.Cleanup(server.Close)
	dialer := websocket.Dialer{Subprotocols: []string{Subprotocol}}
	ws, resp, err := dialer.DialContext(t.Context(), "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ws.Close() })
	if err := ws.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return ws, done
}

func TestPrivateEnvironmentInputWithoutUserStdin(t *testing.T) {
	prefix := []byte("private environment\x00")
	ws, _ := testSession(t, testExecutor{run: func(_ context.Context, options remotecommand.StreamOptions) error {
		data, err := io.ReadAll(options.Stdin)
		if err != nil {
			return err
		}
		if !bytes.Equal(data, prefix) {
			t.Errorf("input=%q, want private prefix %q", data, prefix)
		}
		return nil
	}}, prefix)
	if err := ws.WriteJSON(Message{Type: "start", Command: []string{"true"}}); err != nil {
		t.Fatal(err)
	}
	for {
		var msg Message
		if err := ws.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type == "exit" {
			if msg.ExitCode == nil || *msg.ExitCode != 0 {
				t.Fatalf("exit=%v, want 0", msg.ExitCode)
			}
			return
		}
	}
}

func TestStartUserShellAndEnvironmentValidation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start Message
		valid bool
	}{
		{"user shell", Message{Type: "start", Stdin: true, TTY: true}, true},
		{"explicit container environment", Message{Type: "start", Command: []string{"env"}, EnvMode: "container"}, true},
		{"shell without terminal", Message{Type: "start"}, false},
		{"tty without stdin", Message{Type: "start", TTY: true}, false},
		{"empty executable", Message{Type: "start", Command: []string{""}}, false},
		{"invalid environment", Message{Type: "start", Command: []string{"env"}, EnvMode: "invalid"}, false},
		{"NUL", Message{Type: "start", Command: []string{"a\x00b"}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateStart(&tt.start); (err == nil) != tt.valid {
				t.Fatalf("error=%v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestExecStreamsInputEOFAndExit(t *testing.T) {
	ws, done := testSession(t, testExecutor{run: func(_ context.Context, options remotecommand.StreamOptions) error {
		data, err := io.ReadAll(options.Stdin)
		if err != nil {
			return err
		}
		if _, err := options.Stdout.Write(append([]byte{0xff, 0}, data...)); err != nil {
			return err
		}
		if _, err := io.WriteString(options.Stderr, "diagnostic"); err != nil {
			return err
		}
		return utilexec.CodeExitError{Err: errors.New("exit"), Code: 7}
	}})
	for _, msg := range []Message{{Type: "start", Command: []string{"cat"}, Stdin: true}, {Type: "stdin", Data: []byte("input")}, {Type: "eof"}} {
		if err := ws.WriteJSON(msg); err != nil {
			t.Fatal(err)
		}
	}
	var output, stderr []byte
	for {
		var msg Message
		if err := ws.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		switch msg.Type {
		case "stdout":
			output = append(output, msg.Data...)
		case "stderr":
			stderr = append(stderr, msg.Data...)
		case "exit":
			assertExecOutput(t, msg.ExitCode, output, stderr)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("server did not clean up")
			}
			return
		}
	}
}

func assertExecOutput(t *testing.T, code *int, output, stderr []byte) {
	t.Helper()
	if code == nil || *code != 7 {
		t.Fatalf("exit status=%v, want 7", code)
	}
	if !bytes.Equal(output, append([]byte{0xff, 0}, []byte("input")...)) || string(stderr) != "diagnostic" {
		t.Fatalf("stdout=%q stderr=%q", output, stderr)
	}
}

func TestExecTTYAndDisconnect(t *testing.T) {
	seen := make(chan remotecommand.TerminalSize, 1)
	ws, done := testSession(t, testExecutor{run: func(ctx context.Context, options remotecommand.StreamOptions) error {
		if !options.Tty || options.Stderr != nil {
			return errors.New("invalid TTY streams")
		}
		seen <- *options.TerminalSizeQueue.Next()
		<-ctx.Done()
		return ctx.Err()
	}})
	if err := ws.WriteJSON(Message{Type: "start", Stdin: true, TTY: true, EnvMode: "user"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteJSON(Message{Type: "resize", Cols: 120, Rows: 40}); err != nil {
		t.Fatal(err)
	}
	select {
	case size := <-seen:
		if size.Width != 120 || size.Height != 40 {
			t.Fatalf("size = %+v", size)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("resize was not forwarded")
	}
	if err := ws.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("disconnect did not cancel exec")
	}
}
