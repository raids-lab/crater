package order_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemOrder = "order"

func TestApproveOrderSendsOneAtomicWrite(t *testing.T) {
	for _, tt := range []struct {
		name     string
		lang     string
		args     []string
		lock     *api.JobOrderLockRequest
		approved api.ApprovedJobLock
		want     string
	}{
		{
			name:     "default requested duration",
			lang:     "en",
			args:     []string{"--json"},
			approved: api.ApprovedJobLock{DurationMinutes: 180},
			want:     `"durationMinutes": 180`,
		},
		{
			name:     "whole hours",
			lang:     "en",
			approved: api.ApprovedJobLock{DurationMinutes: 180},
			want:     "Approved lock: 3 h\n",
		},
		{
			name:     "hours and remaining minutes",
			lang:     "zh-CN",
			args:     []string{"--lock", "--hours", "3", "--minutes", "30"},
			lock:     &api.JobOrderLockRequest{Hours: 3, Minutes: 30},
			approved: api.ApprovedJobLock{DurationMinutes: 210},
			want:     "实际批准时长: 3 小时 30 分钟\n",
		},
		{
			name:     "minutes only",
			lang:     "zh-CN",
			args:     []string{"--lock", "--minutes", "30"},
			lock:     &api.JobOrderLockRequest{Minutes: 30},
			approved: api.ApprovedJobLock{DurationMinutes: 30},
			want:     "实际批准时长: 30 分钟\n",
		},
		{
			name:     "permanent",
			lang:     "zh-CN",
			args:     []string{"--lock", "--permanent"},
			lock:     &api.JobOrderLockRequest{IsPermanent: true},
			approved: api.ApprovedJobLock{IsPermanent: true},
			want:     "实际批准时长: 永久锁定\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := make(chan api.ApproveJobOrderRequest, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/approvalorder/17":
					_, _ = io.WriteString(w, `{"code":0,"data":{"id":17,"name":"job-a","type":"job","status":"Pending"}}`)
				case r.Method == http.MethodPut && r.URL.Path == "/api/v1/admin/approvalorder/17/approve-and-lock":
					var req api.ApproveJobOrderRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					writes <- req
					result := api.ApproveJobOrderResult{Message: "approved", LockMessage: "locked", ApprovedLock: tt.approved}
					if err := json.NewEncoder(w).Encode(api.Response[api.ApproveJobOrderResult]{Data: result}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected separate request: %s %s", r.Method, r.URL)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			env := append(snaptest.EnvMinimal(t.TempDir(), tt.lang),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			args := append([]string{"admin", "order", "approve", "17", "--review-notes", "confirmed"}, tt.args...)
			result, err := snaptest.Run(snaptest.CraterExecutable(t), env, args)
			if err != nil {
				t.Fatal(err)
			}
			if result.ExitCode != 0 || !strings.Contains(result.Stdout, tt.want) {
				t.Fatalf("approve result = %+v, want success with %q", result, tt.want)
			}
			if len(writes) != 1 {
				t.Fatalf("approval writes = %d, want 1", len(writes))
			}
			got := <-writes
			if got.ReviewNotes != "confirmed" || (got.Lock == nil) != (tt.lock == nil) ||
				(got.Lock != nil && *got.Lock != *tt.lock) {
				t.Fatalf("request = %+v, want notes and lock %+v", got, tt.lock)
			}
		})
	}
}

func TestOrderReadWithResourceDetails(t *testing.T) {
	const approvedJobOrder = `{"id":12,"name":"job-a","type":"job","status":"Approved","createdAt":"2026-11-08T01:00:00Z","reviewedAt":"2026-11-08T01:10:00Z","content":{"approvalorderExtensionHours":6,"approvedLock":{"durationMinutes":210,"lockedUntil":"2026-11-08T04:40:00Z"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/approvalorder/17":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"code":40301,"msg":"permission denied to view this approval order"}`)
		case "/api/v1/admin/approvalorder/17":
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":17,"name":"job-a","type":"job","status":"Pending","content":{"approvalorderTypeID":1,"approvalorderExtensionHours":8}}}`)
		case "/api/v1/admin/approvalorder/12":
			_, _ = io.WriteString(w, `{"code":0,"data":`+approvedJobOrder+`}`)
		case "/api/v1/admin/approvalorder/18":
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":18,"name":"deleted-job","type":"job","status":"Approved"}}`)
		case "/api/v1/admin/approvalorder/19":
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":19,"name":"my-dataset","type":"dataset","status":"Pending"}}`)
		case "/api/v1/vcjobs/job-a/detail":
			_, _ = io.WriteString(w, `{"code":0,"data":{"jobName":"job-a","status":"Running","queue":"batch","resources":{"cpu":"8","memory":"16Gi","nvidia.com/a100":"2"}}}`)
		case "/api/v1/vcjobs/deleted-job/detail":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"code":40401,"msg":"job not found"}`)
		case "/api/v1/approvalorder/name/job-a":
			if r.URL.Query().Get("type") != "job" {
				t.Errorf("job history query did not specify type=job: %s", r.URL)
			}
			_, _ = io.WriteString(w, `{"code":0,"data":[
				{"id":17,"name":"job-a","type":"job","status":"Pending","createdAt":"2026-11-09T05:00:00Z","content":{"approvalorderExtensionHours":8}},
				`+approvedJobOrder+`,
				{"id":2,"name":"job-a","type":"dataset","status":"Approved","createdAt":"2026-11-07T01:00:00Z"}
			]}`)
		case "/api/v1/approvalorder/name/deleted-job":
			_, _ = io.WriteString(w, `{"code":0,"data":[]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	env := append(snaptest.EnvMinimal(t.TempDir(), "en"),
		"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
	bin := snaptest.CraterExecutable(t)
	for _, tt := range []struct {
		name   string
		args   []string
		failed bool
		want   []string
	}{
		{"administrator hint preserves forbidden code", []string{"order", "get", "17", "--json"}, true,
			[]string{`"code": "ERR_FORBIDDEN_403"`, `"suggested_command": "crater admin order get 17"`}},
		{"readable job resources", []string{"admin", "order", "get", "17"}, false,
			[]string{"Pending", "Job status: Running", "Requested job resources:", "16Gi", "nvidia.com/a100", "request #2", "3 h 30 min", "10m0s"}},
		{"approved job order detail", []string{"admin", "order", "get", "12"}, false,
			[]string{"Approved lock: 3 h 30 min\n"}},
		{"structured job resources", []string{"admin", "order", "get", "17", "--json"}, false,
			[]string{`"order": {`, `"job": {`, `"nvidia.com/a100": "2"`, `"currentAttempt": 2`, `"requestedHours": 6`, `"durationMinutes": 210`, `"reviewSeconds": 600`}},
		{"missing job preserves the order", []string{"admin", "order", "get", "18", "--json"}, false,
			[]string{`"id": 18`, `"job_error": {`, `"code": "ERR_NOT_FOUND_404"`, `"history_error":`}},
		{"dataset order skips the job lookup", []string{"admin", "order", "get", "19", "--json"}, false,
			[]string{`"id": 19`, `"type": "dataset"`}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := snaptest.Run(bin, env, tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if (result.ExitCode != 0) != tt.failed {
				t.Fatalf("exit code = %d, stderr = %s", result.ExitCode, result.Stderr)
			}
			got := result.Stdout
			if tt.failed {
				got = result.Stderr
			}
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("output is missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestOrderSnapshotsEN(t *testing.T) {
	runOrderSnapshots(t, "en")
}

func TestOrderSnapshotsZhCN(t *testing.T) {
	runOrderSnapshots(t, "zh-CN")
}

func runOrderSnapshots(t *testing.T, lang string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "order", goldenStemOrder, lang)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, lang)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-submit-missing-json", Args: []string{"order", "submit", "--json", "--no-interactive"}},
		{ID: "02-submit-invalid-type-json", Args: []string{"order", "submit", "--name", "job-a", "--type", "bad", "--reason", "need more time", "--json", "--no-interactive"}},
		{ID: "03-edit-invalid-status-json", Args: []string{"order", "edit", "1", "--status", "Approved", "--json", "--no-interactive"}},
		{ID: "04-cancel-confirm-required-json", Args: []string{"order", "cancel", "1", "--json", "--no-interactive"}},
		{ID: "05-admin-reject-missing-notes-json", Args: []string{"admin", "order", "reject", "1", "--json", "--no-interactive"}},
		{ID: "06-admin-approve-lock-no-duration-json", Args: []string{"admin", "order", "approve", "1", "--lock", "--json", "--no-interactive"}},
		{ID: "07-admin-approve-lock-negative-json", Args: []string{"admin", "order", "approve", "1", "--lock", "--days", "-1", "--json", "--no-interactive"}},
		{ID: "08-admin-check-confirm-required-json", Args: []string{"admin", "order", "check", "--json", "--no-interactive"}},
		{ID: "09-submit-404-json", Args: []string{"order", "submit", "--name", "job-a", "--type", "job", "--reason", "need more time", "--json", "--no-interactive"}},
		{ID: "10-edit-404-json", Args: []string{"order", "edit", "1", "--reason", "updated reason", "--json", "--no-interactive"}},
		{ID: "11-cancel-404-json", Args: []string{"order", "cancel", "1", "--yes", "--json", "--no-interactive"}},
		{ID: "12-admin-approve-404-json", Args: []string{"admin", "order", "approve", "1", "--json", "--no-interactive"}},
		{ID: "13-admin-reject-404-json", Args: []string{"admin", "order", "reject", "1", "--review-notes", "no", "--json", "--no-interactive"}},
		{ID: "14-admin-check-404-json", Args: []string{"admin", "order", "check", "--yes", "--json", "--no-interactive"}},
		{ID: "15-admin-ls-multiple-list-issues-json", Args: []string{"admin", "order", "ls", "--page", "0", "--page-size", "201", "--status", "Unknown", "--type", "model", "--json", "--no-interactive"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := baseEnv
		switch cases[i].ID {
		case "09-submit-404-json", "10-edit-404-json", "11-cancel-404-json", "12-admin-approve-404-json", "13-admin-reject-404-json", "14-admin-check-404-json":
			env = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=error404")
		}
		r, err := snaptest.Run(bin, env, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		results[i] = r
	}
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, lang, cases, results, update); err != nil {
		t.Fatal(err)
	}
}
