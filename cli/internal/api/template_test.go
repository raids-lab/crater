package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestListJobTemplatesReadsCompleteList(t *testing.T) {
	for _, mode := range []string{"all pages", "empty", "second page error"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX", "1")
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				if r.Method != http.MethodGet || r.URL.Path != JobTemplatePrefix+"/list" ||
					page != requests || r.URL.Query().Get("page_size") != "200" || r.URL.Query().Get("sort") != "createdAt" {
					t.Errorf("unexpected template request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				if mode == "second page error" && page == 2 {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(Response[any]{Code: 50002, Message: "list job templates failed"})
					return
				}
				items := make([]map[string]int, 0)
				total := 201
				if mode == "empty" {
					total = 0
				}
				for id := (page-1)*200 + 1; id <= page*200 && id <= total; id++ {
					items = append(items, map[string]int{"id": id})
				}
				if err := json.NewEncoder(w).Encode(Response[Page[map[string]int]]{
					Data: Page[map[string]int]{Items: items, Total: int64(total), Page: page, PageSize: 200},
				}); err != nil {
					t.Errorf("encode template fixture: %v", err)
				}
			}))
			t.Cleanup(server.Close)
			items, err := NewClient(server.URL).ListJobTemplates()
			if mode == "second page error" {
				var apiErr *RequestError
				if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 500 || apiErr.CraterCode != 50002 ||
					apiErr.Msg != "list job templates failed" || items != nil || requests != 2 {
					t.Fatalf("partial failure lost API facts: items=%v requests=%d error=%v", items, requests, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "empty" {
				if items == nil || len(items) != 0 || requests != 1 {
					t.Fatalf("unexpected empty list: items=%v requests=%d", items, requests)
				}
			} else if len(items) != 201 || requests != 2 || items[200].(map[string]interface{})["id"] != float64(201) {
				t.Fatalf("incomplete template list: length=%d requests=%d", len(items), requests)
			}
		})
	}
}
