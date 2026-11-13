package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestJobExecUserShellAndEnvironmentModeFrames(t *testing.T) {
	for _, mode := range []string{"user", "container"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			server := newExecServer(t, func(ws *websocket.Conn, _ *http.Request) {
				var start execFrame
				if err := ws.ReadJSON(&start); err != nil {
					t.Error(err)
					return
				}
				if len(start.Command) != 0 || !start.TTY || !start.Stdin || start.EnvMode != mode {
					t.Errorf("start=%+v, want empty command, stdin/tty and mode %q", start, mode)
				}
				code := 0
				for _, frame := range []execFrame{execReady(), {Type: "exit", ExitCode: &code}} {
					if err := ws.WriteJSON(frame); err != nil {
						t.Error(err)
						return
					}
				}
			})
			_, err := NewClient(server.URL).StreamJobExec(t.Context(), "job", JobExecOptions{
				EnvMode: mode, TTY: true, Input: io.NopCloser(strings.NewReader("")),
			}, io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
