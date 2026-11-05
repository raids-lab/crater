package api

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestExecCompletionClosesUnusedInput(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
	server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
		var start execFrame
		if err := ws.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		code := 0
		for _, msg := range []execFrame{execReady(), {Type: "exit", ExitCode: &code}} {
			if err := ws.WriteJSON(msg); err != nil {
				t.Error(err)
				return
			}
		}
	})
	input, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	done := make(chan error, 1)
	go func() {
		_, err := NewClient(server.URL).StreamJobExec(t.Context(), "job", JobExecOptions{Command: []string{"true"}, Input: input}, io.Discard, io.Discard)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("completion left unused stdin blocked")
	}
}
