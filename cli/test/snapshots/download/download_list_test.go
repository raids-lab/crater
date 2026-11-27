package download_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/snaptest"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestDownloadListProcessPagination(t *testing.T) {
	bin := snaptest.CraterExecutable(t)
	for _, tc := range []struct {
		name    string
		items   []uint
		total   int64
		legacy  bool
		page    int
		code    string
		message string
	}{
		{name: "V8 full page", page: 2, items: []uint{6, 7, 8, 9, 10}, total: 12},
		{name: "V8 short last page", page: 2, items: []uint{6, 7}, total: 7},
		{name: "V8 empty result", page: 2, items: []uint{}, total: 0},
		{
			name: "V7 ignored page_size", legacy: true, total: 60,
			items: []uint{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40},
			code:  errorcodes.ErrAPIVersionMismatch, message: "upgrade the backend to API V9",
		},
		{name: "wrong page", page: 1, items: []uint{}, code: errorcodes.ErrAPIOther, message: "requested page 2, received page 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != api.ModelDownloadListPath ||
					r.URL.RawQuery != "page=2&page_size=5" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				items := make([]api.ModelDownloadResp, len(tc.items))
				for i, id := range tc.items {
					items[i].ID = id
				}
				data := map[string]any{"items": items, "total": tc.total, "summary": map[string]int64{}}
				if !tc.legacy {
					data["page"], data["page_size"] = tc.page, 5
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(api.Response[map[string]any]{Data: data}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			env := append(snaptest.EnvMinimal(t.TempDir(), "en"),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			result, err := snaptest.Run(bin, env, []string{
				"download", "ls", "--page", "2", "--page-size", "5", "--json", "--no-interactive",
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.code != "" {
				var failure struct {
					Category string         `json:"category"`
					Code     string         `json:"code"`
					Message  string         `json:"message"`
					Context  map[string]any `json:"context"`
				}
				if err := json.Unmarshal([]byte(result.Stderr), &failure); err != nil || result.ExitCode != 4 ||
					result.Stdout != "" || failure.Category != errorcodes.CategoryAPI || failure.Code != tc.code ||
					!strings.Contains(failure.Message, tc.message) {
					t.Fatalf("unexpected failure or partial output: %+v", result)
				}
				if tc.legacy && (failure.Context["reason"] != "legacy_download_pagination" ||
					failure.Context["cli_min_supported_backend_api_version"] != float64(9) ||
					failure.Context["backend_api_version"] != nil ||
					!strings.Contains(failure.Message, "crater compatibility")) {
					t.Fatalf("unexpected compatibility diagnostic: %+v", failure)
				}
				return
			}

			var success struct {
				Data struct {
					Downloads  []api.ModelDownloadResp `json:"downloads"`
					Pagination struct {
						Page     int   `json:"page"`
						PageSize int   `json:"page_size"`
						Total    int64 `json:"total"`
					} `json:"pagination"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(result.Stdout), &success); err != nil || result.ExitCode != 0 || result.Stderr != "" {
				t.Fatalf("unexpected result: %+v", result)
			}
			data := success.Data
			ids := make([]uint, len(data.Downloads))
			for i, item := range data.Downloads {
				ids[i] = item.ID
			}
			if data.Downloads == nil || !reflect.DeepEqual(ids, tc.items) || data.Pagination.Page != 2 ||
				data.Pagination.PageSize != 5 || data.Pagination.Total != tc.total {
				t.Fatalf("incorrect downloads or pagination: %+v", data)
			}
		})
	}
}

func TestAdminDownloadListProcessFailures(t *testing.T) {
	bin := snaptest.CraterExecutable(t)
	for _, tc := range []struct {
		name     string
		kind     string
		wantCode string
		wantText string
	}{
		{name: "V7 legacy response", kind: "legacy", wantCode: errorcodes.ErrAPIOther, wantText: "requested page 1, received page 0"},
		{name: "second page HTTP failure", kind: "http", wantCode: errorcodes.ErrServerInternal5XX, wantText: "admin download fixture failed"},
		{name: "second page invalid metadata", kind: "metadata", wantCode: errorcodes.ErrAPIOther, wantText: "received page 1"},
		{name: "second page connection failure", kind: "network", wantCode: errorcodes.ErrNetworkFailure, wantText: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				if r.Method != http.MethodGet || r.URL.Path != api.AdminModelDLPfx+"/models/downloads" ||
					query.Get("page_size") != "200" || query.Get("sort") != "-updatedAt" {
					t.Errorf("unexpected admin download request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				page := 1
				if raw := query.Get("page"); raw != "" {
					page, _ = strconv.Atoi(raw)
				}
				if tc.kind == "legacy" {
					data := map[string]any{
						"items":   []api.ModelDownloadResp{{ID: 1}},
						"total":   1,
						"summary": map[string]int64{"Ready": 1},
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(api.Response[map[string]any]{Data: data})
					return
				}
				if page == 2 {
					switch tc.kind {
					case "http":
						w.WriteHeader(http.StatusInternalServerError)
						_ = json.NewEncoder(w).Encode(api.Response[any]{Code: 50002, Message: tc.wantText})
						return
					case "network":
						conn, _, err := http.NewResponseController(w).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
						return
					}
				}
				responsePage := page
				if tc.kind == "metadata" && page == 2 {
					responsePage = 1
				}
				start, end := (page-1)*200, min(page*200, 201)
				items := make([]api.ModelDownloadResp, end-start)
				for index := range items {
					items[index].ID = uint(start + index + 1)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(api.Response[api.Page[api.ModelDownloadResp]]{
					Data: api.Page[api.ModelDownloadResp]{Items: items, Total: 201, Page: responsePage, PageSize: 200},
				})
			}))
			defer server.Close()
			server.Config.SetKeepAlivesEnabled(false)

			env := append(snaptest.EnvMinimal(t.TempDir(), "en"),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			result, err := snaptest.Run(bin, env, []string{
				"admin", "model-download", "ls", "--json", "--no-interactive",
			})
			if err != nil {
				t.Fatal(err)
			}
			var failure struct {
				Category string `json:"category"`
				Code     string `json:"code"`
				Message  string `json:"message"`
			}
			if err := json.Unmarshal([]byte(result.Stderr), &failure); err != nil {
				t.Fatalf("stderr is not one structured error: %q: %v", result.Stderr, err)
			}
			if result.ExitCode != 4 || result.Stdout != "" || failure.Category != errorcodes.CategoryAPI ||
				failure.Code != tc.wantCode || (tc.wantText != "" && !strings.Contains(failure.Message, tc.wantText)) {
				t.Fatalf("unexpected failure or partial output: %+v", result)
			}
		})
	}
}
