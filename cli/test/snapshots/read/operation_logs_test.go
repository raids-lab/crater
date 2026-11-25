// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package read_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/snaptest"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestOperationLogsProcessPagination(t *testing.T) {
	bin := snaptest.CraterExecutable(t)
	for _, tc := range []struct {
		name, code, detail string
		failAt             int
	}{
		{name: "exactly 500 records"},
		{name: "second page HTTP failure", failAt: 2, code: errorcodes.ErrServerInternal5XX, detail: "operation log fixture failed"},
		{name: "third page HTTP failure", failAt: 3, code: errorcodes.ErrServerInternal5XX, detail: "operation log fixture failed"},
		{name: "third page invalid metadata", failAt: 3, code: errorcodes.ErrAPIOther, detail: "invalid pagination metadata"},
		{name: "third page connection failure", failAt: 3, code: errorcodes.ErrNetworkFailure, detail: "EOF"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const total, pageSize = 500, 200
			var mu sync.Mutex
			var requested []int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				page, err := strconv.Atoi(q.Get("page"))
				if err != nil || page < 1 || page > 3 || q.Get("page_size") != "200" ||
					q.Has("limit") || q.Get("operator") != "fixture-operator" ||
					r.Method != http.MethodGet || r.URL.Path != api.AdminOperationLogs {
					t.Errorf("unexpected list request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mu.Lock()
				requested = append(requested, page)
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				if page == tc.failAt {
					switch tc.code {
					case errorcodes.ErrServerInternal5XX:
						w.WriteHeader(http.StatusInternalServerError)
						if err := json.NewEncoder(w).Encode(api.Response[any]{Code: 50002, Message: tc.detail}); err != nil {
							t.Error(err)
						}
						return
					case errorcodes.ErrNetworkFailure:
						conn, _, err := http.NewResponseController(w).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
						return
					}
				}

				start, end := (page-1)*pageSize, min(page*pageSize, total)
				items := make([]map[string]int, end-start)
				for index := range items {
					items[index] = map[string]int{"id": start + index + 1}
				}
				reply := api.Page[map[string]int]{Items: items, Total: total, Page: page, PageSize: pageSize}
				if page == tc.failAt && tc.code == errorcodes.ErrAPIOther {
					reply.Page = page - 1
				}
				if err := json.NewEncoder(w).Encode(api.Response[api.Page[map[string]int]]{Data: reply}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			// A reused connection can retry an idempotent GET after EOF.
			server.Config.SetKeepAlivesEnabled(false)

			env := append(snaptest.EnvMinimal(t.TempDir(), "en"),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			result, err := snaptest.Run(bin, env, []string{
				"admin", "operation-logs", "--page", "1", "--limit", "500",
				"--operator", "fixture-operator", "--json", "--no-interactive",
			})
			if err != nil {
				t.Fatal(err)
			}

			wantPages := 3
			if tc.failAt > 0 {
				wantPages = tc.failAt
			}
			mu.Lock()
			pages := append([]int(nil), requested...)
			mu.Unlock()
			if len(pages) != wantPages {
				t.Fatalf("requested pages %v, want %d requests", pages, wantPages)
			}
			for i, page := range pages {
				if page != i+1 {
					t.Fatalf("requests out of order: %v", pages)
				}
			}

			if tc.failAt > 0 {
				var failure struct {
					Category string         `json:"category"`
					Code     string         `json:"code"`
					Message  string         `json:"message"`
					Context  map[string]any `json:"context"`
				}
				if err := json.Unmarshal([]byte(result.Stderr), &failure); err != nil {
					t.Fatalf("stderr is not one structured error: %q: %v", result.Stderr, err)
				}
				if result.ExitCode != 4 || result.Stdout != "" || failure.Category != errorcodes.CategoryAPI ||
					failure.Code != tc.code || !strings.Contains(failure.Message, tc.detail) {
					t.Fatalf("unexpected failure or partial success: %+v", result)
				}
				if tc.code == errorcodes.ErrServerInternal5XX &&
					(failure.Context["http_status"] != float64(500) || failure.Context["crater_code"] != float64(50002)) {
					t.Fatalf("server error context not preserved: %+v", failure.Context)
				}
				return
			}

			if result.ExitCode != 0 || result.Stderr != "" {
				t.Fatalf("operation log query failed: %+v", result)
			}
			var success struct {
				Data struct {
					Logs api.OperationLogs `json:"operation_logs"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(result.Stdout), &success); err != nil {
				t.Fatal(err)
			}
			logs := success.Data.Logs
			if logs.Total != total || len(logs.Items) != total {
				t.Fatalf("total=%d, items=%d, want %d of each", logs.Total, len(logs.Items), total)
			}
			for i, item := range logs.Items {
				var record struct {
					ID int `json:"id"`
				}
				if err := json.Unmarshal(item, &record); err != nil || record.ID != i+1 {
					t.Fatalf("item %d was missing, duplicated or out of order: %s", i+1, item)
				}
			}
		})
	}
}
