package job_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemJob = "job"

func runJobCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
	t.Helper()
	out := make([]*snaptest.Result, len(cases))
	for i := range cases {
		r, err := snaptest.Run(bin, baseEnv, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		out[i] = r
	}
	return out
}

func jobCases() []snaptest.Case {
	return []snaptest.Case{
		{ID: "01-unknown-nojson", Args: []string{"job", "wat", "--no-interactive"}},
		{ID: "02-unknown-json", Args: []string{"job", "wat", "--no-interactive", "--json"}},
		{ID: "03-get-missing-name-nojson", Args: []string{"job", "get", "--no-interactive"}},
		{ID: "04-get-missing-name-json", Args: []string{"job", "get", "--no-interactive", "--json"}},
		{ID: "05-ls-invalid-days-nojson", Args: []string{"job", "ls", "--no-interactive", "--days", "-2"}},
		{ID: "06-ls-invalid-status-json", Args: []string{"job", "ls", "--no-interactive", "--json", "--status", "bad"}},
		{ID: "07-create-jupyter-multi-usage-nojson", Args: []string{"job", "create", "jupyter", "--no-interactive", "--name", "demo", "--cpu", "-1", "--memory", "-2Gi", "--gpu", "-1"}},
		{ID: "08-create-jupyter-multi-usage-json", Args: []string{"job", "create", "jupyter", "--no-interactive", "--json", "--name", "demo", "--cpu", "-1", "--memory", "-2Gi", "--gpu", "-1"}},
		{ID: "09-create-custom-missing-working-dir-json", Args: []string{"job", "create", "custom", "--no-interactive", "--json", "--name", "demo", "--image", "example/image:tag", "--memory", "2Gi", "--working-dir", ""}},
		{ID: "10-admin-lock-missing-duration-nojson", Args: []string{"admin", "job", "lock", "job-123", "--no-interactive"}},
		{ID: "11-admin-clean-low-gpu-invalid-json", Args: []string{"admin", "job", "clean", "low-gpu", "--no-interactive", "--json", "--time-range", "0", "--wait-time", "-1"}},
		{ID: "12-ls-network-timeout-json", Args: []string{"job", "ls", "--no-interactive", "--json"}},
		{ID: "13-delete-confirm-required-json", Args: []string{"job", "delete", "job-123", "--no-interactive", "--json"}},
		{ID: "14-admin-clean-long-running-thresholds-json", Args: []string{"admin", "job", "clean", "long-running", "--no-interactive", "--json"}},
		{ID: "15-admin-clean-confirm-required-json", Args: []string{"admin", "job", "clean", "low-gpu", "--time-range", "90", "--wait-time", "30", "--no-interactive", "--json"}},
		{ID: "16-ls-multiple-list-issues-json", Args: []string{"job", "ls", "--page", "0", "--page-size", "201", "--status", "bad", "--json", "--no-interactive"}},
		{ID: "17-pods-multiple-list-issues-json", Args: []string{"job", "pods", "job-123", "--page", "0", "--status", "bad", "--json", "--no-interactive"}},
		{ID: "18-ls-search-page-timeout-json", Args: []string{"job", "ls", "--search", "trainer", "--page", "2", "--page-size", "15", "--sort=-createdAt", "--json", "--no-interactive"}},
		{ID: "19-ls-search-sort-usage-json", Args: []string{"job", "ls", "--search", "这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词这是一个超过一百二十八个字符限制的搜索词", "--sort", "name,-name,bad,createdAt", "--json", "--no-interactive"}},
	}
}

func jobSuccessCases() []snaptest.Case {
	return []snaptest.Case{
		{ID: "20-ls-page-success-nojson", Args: []string{"job", "ls", "--page", "2", "--page-size", "2", "--no-interactive"}},
		{ID: "21-ls-page-success-json", Args: []string{"job", "ls", "--page", "2", "--page-size", "2", "--json", "--no-interactive"}},
	}
}

func jobFilterCases() []snaptest.Case {
	return []snaptest.Case{
		{ID: "22-ls-invalid-multi-filters-json", Args: []string{"job", "ls", "--status", "Running,bad", "--type", "custom,nope", "--schedule", "normal,0", "--no-interactive", "--json"}},
		{ID: "23-ls-valid-multi-filters-timeout-json", Args: []string{"job", "ls", "--search", "demo", "--status", "Running,Pending", "--type", "pytorch,tensorflow", "--schedule", "normal,backfill", "--no-interactive", "--json"}},
	}
}

func jobLogCases() []snaptest.Case {
	return []snaptest.Case{
		{ID: "24-logs-missing-name-nojson", Args: []string{"job", "logs", "--no-interactive"}},
		{ID: "25-logs-negative-tail-json", Args: []string{"job", "logs", "job-123", "--tail", "-1", "--no-interactive", "--json"}},
		{ID: "26-logs-pod-all-pods-conflict-json", Args: []string{"job", "logs", "job-123", "--pod", "pod-1", "--all-pods", "--no-interactive", "--json"}},
		{ID: "27-logs-follow-json-conflict", Args: []string{"job", "logs", "job-123", "--follow", "--no-interactive", "--json"}},
		{ID: "28-logs-follow-previous-conflict-nojson", Args: []string{"job", "logs", "job-123", "--follow", "--previous", "--no-interactive"}},
	}
}

func newJobListSnapshotServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/vcjobs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "2" {
			t.Errorf("unexpected pagination query: %s", r.URL.RawQuery)
			http.Error(w, "unexpected pagination", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "code": 0,
  "data": {
    "items": [
      {
        "name": "training-demo",
        "jobName": "vcjob-alice-training-demo",
        "owner": "alice",
        "userInfo": {"username": "alice", "nickname": "Alice"},
        "jobType": "pytorch",
        "scheduleType": 0,
        "queue": "default",
        "status": "Running",
        "createdAt": "2026-07-25T08:00:00Z",
        "startedAt": "2026-07-25T08:01:00Z",
        "completedAt": "0001-01-01T00:00:00Z",
        "nodes": ["gpu-02"],
        "resources": {
          "cpu": "4",
          "memory": "16Gi",
          "nvidia.com/a100": "2"
        },
        "locked": false,
        "permanentLocked": false,
        "lockedTimestamp": "0001-01-01T00:00:00Z",
        "billedPointsTotal": 12.5
      }
    ],
    "total": 3,
    "page": 2,
    "page_size": 2
  },
  "msg": ""
}`)
	}))
}

func runJobSnapshots(t *testing.T, lang string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "job", goldenStemJob, lang)
	home := t.TempDir()
	bin := snaptest.CraterExecutable(t)

	cases := jobCases()
	timeoutEnv := append(
		snaptest.EnvMinimal(home, lang),
		"CRATER_TEST_SANDBOX_HTTP=timeout",
	)
	results := runJobCases(t, bin, timeoutEnv, cases)

	server := newJobListSnapshotServer(t)
	defer server.Close()
	successCases := jobSuccessCases()
	successEnv := append(
		snaptest.EnvMinimal(home, lang),
		"CRATER_TEST_SANDBOX_HTTP=passthrough",
		"CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL,
	)
	successResults := runJobCases(t, bin, successEnv, successCases)
	cases = append(cases, successCases...)
	results = append(results, successResults...)

	filterCases := jobFilterCases()
	filterResults := runJobCases(t, bin, timeoutEnv, filterCases)
	cases = append(cases, filterCases...)
	results = append(results, filterResults...)

	logCases := jobLogCases()
	logResults := runJobCases(t, bin, timeoutEnv, logCases)
	cases = append(cases, logCases...)
	results = append(results, logResults...)

	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, lang, cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestJobSnapshotsEN(t *testing.T) {
	runJobSnapshots(t, "en")
}

func TestJobSnapshotsZhCN(t *testing.T) {
	runJobSnapshots(t, "zh-CN")
}
