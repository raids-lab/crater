package job_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func runExecSnapshots(t *testing.T, lang string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/vcjobs/cpu-job/ssh" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"ip":"cpu.example.test","port":"30222"}}`)
			return
		}
		upgrader := websocket.Upgrader{Subprotocols: []string{"crater.exec.v1"}}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer ws.Close()
		var start struct {
			Command []string `json:"command"`
		}
		if err := ws.ReadJSON(&start); err != nil {
			t.Error(err)
			return
		}
		code := 0
		if len(start.Command) > 0 && start.Command[0] == "false" {
			code = 7
		}
		for _, frame := range []any{
			map[string]any{"type": "ready", "namespace": "real-ns", "pod": "cpu-worker", "container": "main"},
			map[string]any{"type": "stdout", "data": []byte("hello CPU/ARM\n")},
			map[string]any{"type": "stderr", "data": []byte("diagnostic\n")},
			map[string]any{"type": "exit", "exit_code": code},
		} {
			if err := ws.WriteJSON(frame); err != nil {
				t.Error(err)
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	cases := []snaptest.Case{
		{ID: "01-missing-command", Args: []string{"job", "exec", "cpu-job", "--no-interactive", "--json"}},
		{ID: "02-missing-separator", Args: []string{"job", "exec", "cpu-job", "uname", "--no-interactive", "--json"}},
		{ID: "03-tty-invalid", Args: []string{"job", "exec", "cpu-job", "-t", "--pod=", "--json", "--", "sh"}},
		{ID: "04-ssh-connect-json", Args: []string{"job", "ssh", "cpu-job", "--connect", "--json"}},
		{ID: "05-ssh-options-without-connect", Args: []string{"job", "ssh", "cpu-job", "--user", "alice", "--identity-file", "key", "--json"}},
		{ID: "06-exec-text", Args: []string{"job", "exec", "cpu-job", "--no-interactive", "--", "uname", "-m"}},
		{ID: "07-exec-json", Args: []string{"job", "exec", "cpu-job", "--json", "--", "uname", "-m"}},
		{ID: "08-exec-remote-failure-json", Args: []string{"job", "exec", "cpu-job", "--json", "--", "false"}},
		{ID: "09-ssh-address", Args: []string{"job", "ssh", "cpu-job", "--no-interactive"}},
		{ID: "10-ssh-address-json", Args: []string{"job", "ssh", "cpu-job", "--json"}},
		{ID: "11-exec-help", Args: []string{"job", "exec", "--help"}},
		{ID: "12-ssh-help", Args: []string{"job", "ssh", "--help"}},
		{ID: "15-invalid-environment-mode", Args: []string{"job", "exec", "cpu-job", "--env-mode=invalid", "--json", "--", "env"}},
		{ID: "16-container-environment", Args: []string{"job", "exec", "cpu-job", "--env-mode=container", "--json", "--", "env"}},
	}
	bin := snaptest.CraterExecutable(t)
	baseEnv := append(snaptest.EnvMinimal(t.TempDir(), lang), "CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
	results := runJobCases(t, bin, baseEnv, cases)
	failureCases := []snaptest.Case{
		{ID: "13-exec-timeout-json", Args: []string{"job", "exec", "cpu-job", "--json", "--", "uname"}},
		{ID: "14-exec-timeout-text", Args: []string{"job", "exec", "cpu-job", "--no-interactive", "--", "uname"}},
	}
	failureResults := runJobCases(t, bin, append(snaptest.EnvMinimal(t.TempDir(), lang), "CRATER_TEST_SANDBOX_HTTP=timeout"), failureCases)
	cases = append(cases, failureCases...)
	results = append(results, failureResults...)
	path := snaptest.GoldenFileT(t, "job", "exec", lang)
	if err := snaptest.MatchOrUpdateGolden(path, lang, cases, results, os.Getenv("UPDATE_SNAPSHOTS") == "1"); err != nil {
		t.Fatal(err)
	}
}

func TestExecSnapshotsEN(t *testing.T)   { runExecSnapshots(t, "en") }
func TestExecSnapshotsZhCN(t *testing.T) { runExecSnapshots(t, "zh-CN") }
