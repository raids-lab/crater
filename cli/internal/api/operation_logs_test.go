package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestOperationLogsLogicalPage(t *testing.T) {
	tests := []struct {
		name               string
		page, limit, total int
		requests           []int
		failAt             int
		changeTotal        bool
		errText            string
	}{
		{name: "under limit", page: 2, limit: 19, total: 100, requests: []int{2}},
		{name: "exact server limit", page: 1, limit: 200, total: 300, requests: []int{1}},
		{name: "above server limit", page: 1, limit: 201, total: 650, requests: []int{1, 2}},
		{name: "logical page of 500", page: 1, limit: 500, total: 650, requests: []int{1, 2, 3}},
		{name: "unaligned logical page", page: 2, limit: 500, total: 1100, requests: []int{3, 4, 5}},
		{name: "short last page", page: 1, limit: 500, total: 415, requests: []int{1, 2, 3}},
		{name: "total equals limit", page: 1, limit: 500, total: 500, requests: []int{1, 2, 3}},
		{name: "empty", page: 1, limit: 500, requests: []int{1}},
		{name: "beyond total", page: 2, limit: 500, total: 200, requests: []int{3}},
		{name: "request failure", page: 1, limit: 500, total: 600, requests: []int{1, 2}, failAt: 2, errText: "log page failed"},
		{name: "changed dataset", page: 1, limit: 500, total: 600, requests: []int{1, 2}, changeTotal: true, errText: "total changed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX", "1")
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			requests := make(chan int, 10)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				values := r.URL.Query()
				page, _ := strconv.Atoi(values.Get("page"))
				size, _ := strconv.Atoi(values.Get("page_size"))
				select {
				case requests <- page:
				default:
					t.Error("too many requests")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if r.Method != http.MethodGet || r.URL.Path != AdminOperationLogs || size != min(tc.limit, 200) ||
					values.Get("operator") != "one" || values.Has("limit") {
					t.Errorf("unexpected operation log request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				if page == tc.failAt {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(Response[any]{Code: 50002, Message: tc.errText})
					return
				}
				total := tc.total
				if tc.changeTotal && page > 1 {
					total++
				}
				items := make([]map[string]int, 0)
				for id := (page-1)*size + 1; id <= min(page*size, total); id++ {
					items = append(items, map[string]int{"id": id})
				}
				_ = json.NewEncoder(w).Encode(Response[Page[map[string]int]]{
					Data: Page[map[string]int]{Items: items, Total: int64(total), Page: page, PageSize: size},
				})
			}))
			t.Cleanup(server.Close)
			result, err := NewClient(server.URL).GetOperationLogs(tc.page, tc.limit, map[string]string{"operator": "one"})
			server.Close()
			close(requests)
			gotPages := make([]int, 0)
			for page := range requests {
				gotPages = append(gotPages, page)
			}
			if !slices.Equal(gotPages, tc.requests) {
				t.Fatalf("requested %v, want %v", gotPages, tc.requests)
			}
			if tc.errText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errText) || result.Items != nil {
					t.Fatalf("result=%+v err=%v, want error and no partial output", result, err)
				}
				if tc.failAt > 0 {
					var apiErr *RequestError
					if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 500 || apiErr.CraterCode != 50002 {
						t.Fatalf("structured server error was lost: %v", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			// The older command exposes only items and total, including [] for an empty result.
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			start := (tc.page - 1) * tc.limit
			want := max(0, min(tc.limit, tc.total-start))
			if result.Total != int64(tc.total) || len(result.Items) != want || len(fields) != 2 ||
				fields["items"] == nil || fields["total"] == nil || string(fields["items"]) == "null" {
				t.Fatalf("incompatible output: %s", encoded)
			}
			for i, item := range result.Items {
				if string(item) != `{"id":`+strconv.Itoa(start+i+1)+"}" {
					t.Fatalf("item %d = %s, expected ID %d", i, item, start+i+1)
				}
			}
		})
	}
}
