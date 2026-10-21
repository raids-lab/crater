package read_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestTemplateSnapshotsEN(t *testing.T) {
	runTemplateSnapshots(t, "en")
}

func TestTemplateSnapshotsZhCN(t *testing.T) {
	runTemplateSnapshots(t, "zh-CN")
}

func runTemplateSnapshots(t *testing.T, lang string) {
	t.Helper()
	cases := []snaptest.Case{
		{ID: "01-template-list-json", Args: []string{"template", "ls", "--json"}},
		{ID: "02-template-empty-json", Args: []string{"template", "ls", "--json"}},
		{ID: "03-template-later-page-failure-json", Args: []string{"template", "ls", "--json"}},
		{ID: "04-template-list-table", Args: []string{"template", "ls", "--no-interactive"}},
	}
	bin := snaptest.CraterExecutable(t)
	home := t.TempDir()
	results := make([]*snaptest.Result, 0, len(cases))
	for index, testCase := range cases {
		server := templateSnapshotServer(t, index)
		env := append(snaptest.EnvMinimal(home, lang),
			"CRATER_TEST_SANDBOX_HTTP=passthrough",
			"CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
		result, err := snaptest.Run(bin, env, testCase.Args)
		if err != nil {
			t.Fatal(err)
		}
		if index == 2 {
			if result.ExitCode != 4 || result.Stdout != "" {
				t.Fatalf("partial list was reported as success: %+v", result)
			}
		} else if result.ExitCode != 0 || result.Stderr != "" {
			t.Fatalf("template read failed: %+v", result)
		}
		results = append(results, result)
	}
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(snaptest.GoldenFileT(t, "read", "template", lang), lang, cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func templateSnapshotServer(t *testing.T, mode int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != api.JobTemplatePrefix+"/list" {
			t.Errorf("unexpected template request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if mode == 2 && r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(api.Response[any]{Code: 50002, Message: "list job templates failed"})
			return
		}
		items := []map[string]interface{}{
			{"id": 1, "name": "training", "createdAt": "2026-01-01T00:00:00Z", "userInfo": map[string]string{"nickname": "owner"}},
			{"id": 2, "name": "evaluation", "createdAt": "2026-01-02T00:00:00Z", "userInfo": map[string]string{"nickname": "owner"}},
		}
		total := len(items)
		if mode == 1 {
			items = []map[string]interface{}{}
			total = 0
		} else if mode == 2 {
			items = make([]map[string]interface{}, 200)
			for i := range items {
				items[i] = map[string]interface{}{"id": i + 1}
			}
			total = 201
		}
		if err := json.NewEncoder(w).Encode(api.Response[api.Page[map[string]interface{}]]{
			Data: api.Page[map[string]interface{}]{Items: items, Total: int64(total), Page: 1, PageSize: 200},
		}); err != nil {
			t.Errorf("encode template snapshot fixture: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
