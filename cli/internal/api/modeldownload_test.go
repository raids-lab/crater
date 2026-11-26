package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
)

func modelDownloadTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			return recorder.Result(), nil
		}
	})
	return client
}

func TestListDownloadPageSendsPagingAndServerFilters(t *testing.T) {
	client := modelDownloadTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != ModelDownloadListPath {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		want := map[string][]string{
			"page":      {"2"},
			"page_size": {"25"},
			"category":  {"model"},
			"status":    {"Downloading"},
			"search":    {"Qwen"},
		}
		if !reflect.DeepEqual(map[string][]string(query), want) {
			t.Fatalf("unexpected query: %#v", query)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[map[string]interface{}]{
			Data: map[string]interface{}{
				"items": []ModelDownloadResp{{
					ID: 7, Name: "Qwen/Qwen3", RequesterCount: 2,
					Requesters: []UserInfo{{Username: "alice", Nickname: "Alice"}},
					Relation:   "submitted", CanViewLogs: true,
					SourceURL: "https://huggingface.co/Qwen/Qwen3",
				}},
				"total": 31,
				"page":  2, "page_size": 25,
				"summary": map[string]int64{"Downloading": 4, "Ready": 27},
			},
		})
	})

	page, err := client.ListDownloadPage(ModelDownloadListOptions{
		ListOptions: ListOptions{Page: 2, PageSize: 25},
		Category:    "model",
		Status:      "Downloading",
		Search:      "Qwen",
	})
	if err != nil {
		t.Fatalf("ListDownloadPage returned error: %v", err)
	}
	if page.Page.Page != 2 || page.Page.PageSize != 25 || page.Total != 31 {
		t.Fatalf("unexpected pagination: %#v", page.Page)
	}
	if len(page.Items) != 1 || page.Items[0].ID != 7 {
		t.Fatalf("unexpected items: %#v", page.Items)
	}
	if page.Items[0].RequesterCount != 2 ||
		len(page.Items[0].Requesters) != 1 ||
		page.Items[0].Relation != "submitted" ||
		!page.Items[0].CanViewLogs ||
		page.Items[0].SourceURL != "https://huggingface.co/Qwen/Qwen3" {
		t.Fatalf("latest download fields were not preserved: %#v", page.Items[0])
	}
	if !reflect.DeepEqual(page.Summary, map[string]int64{"Downloading": 4, "Ready": 27}) {
		t.Fatalf("unexpected summary: %#v", page.Summary)
	}
}

func TestListDownloadPageNormalizesEmptyCollections(t *testing.T) {
	client := modelDownloadTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[map[string]interface{}]{
			Data: map[string]interface{}{"items": []ModelDownloadResp{}, "total": 0, "page": 1, "page_size": DefaultCLIPageSize},
		})
	})

	page, err := client.ListDownloadPage(ModelDownloadListOptions{})
	if err != nil {
		t.Fatalf("ListDownloadPage returned error: %v", err)
	}
	if page.Items == nil || page.Summary == nil {
		t.Fatalf("empty collections must be non-nil: %#v", page)
	}
	if page.Page.Page != 1 || page.Page.PageSize != DefaultCLIPageSize {
		t.Fatalf("unexpected normalized pagination: %#v", page.Page)
	}
}

func TestListDownloadPageValidatesResponse(t *testing.T) {
	for _, tc := range []struct {
		name, data, message string
		legacy              bool
	}{
		{name: "legacy response", data: `{"items":[{"id":21}],"total":60,"summary":{}}`, legacy: true},
		{name: "malformed legacy-shaped response", data: `{"items":[null],"total":60,"summary":{}}`, message: "item has no valid id"},
		{name: "missing page", data: `{"items":[],"total":0,"page_size":5}`, message: "positive integers"},
		{name: "missing both metadata fields without legacy summary", data: `{"items":[],"total":0}`, message: "positive integers"},
		{name: "null metadata", data: `{"items":[],"total":0,"page":null,"page_size":null,"summary":{}}`, message: "positive integers"},
		{name: "zero metadata", data: `{"items":[],"total":0,"page":0,"page_size":0,"summary":{}}`, message: "positive integers"},
		{name: "wrong page", data: `{"items":[],"total":0,"page":1,"page_size":5}`, message: "requested page 2, received page 1"},
		{name: "wrong page size", data: `{"items":[],"total":0,"page":2,"page_size":20}`, message: "requested page_size 5, received page_size 20"},
		{name: "items are not an array", data: `{"items":{},"total":0,"page":2,"page_size":5}`, message: "invalid download list response"},
		{name: "missing items", data: `{"total":0,"page":2,"page_size":5}`, message: "missing items"},
		{name: "null items", data: `{"items":null,"total":0,"page":2,"page_size":5}`, message: "missing items"},
		{name: "null item", data: `{"items":[null],"total":6,"page":2,"page_size":5}`, message: "item has no valid id"},
		{name: "missing total", data: `{"items":[],"page":2,"page_size":5}`, message: "invalid total"},
		{name: "negative total", data: `{"items":[],"total":-1,"page":2,"page_size":5}`, message: "invalid total"},
		{name: "total smaller than items", data: `{"items":[{"id":6}],"total":0,"page":2,"page_size":5}`, message: "invalid total"},
		{name: "too many items", data: `{"items":[{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11}],"total":11,"page":2,"page_size":5}`, message: "item count exceeds page_size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := modelDownloadTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				writeJobTestResponse(t, w, json.RawMessage(tc.data))
			})
			page, err := client.ListDownloadPage(ModelDownloadListOptions{ListOptions: ListOptions{Page: 2, PageSize: 5}})
			if err == nil || page.Items != nil || errors.Is(err, ErrLegacyDownloadPagination) != tc.legacy ||
				!strings.Contains(err.Error(), tc.message) {
				t.Fatalf("unexpected page or error: page=%+v, error=%v", page, err)
			}
		})
	}
}

func TestDownloadAggregationRejectsRepeatedServerPage(t *testing.T) {
	requests := 0
	client := modelDownloadTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests > 2 {
			t.Fatal("requested a page after repeated server page")
		}
		writeJobTestResponse(t, w, ModelDownloadPage{
			Page: Page[ModelDownloadResp]{Items: []ModelDownloadResp{{ID: uint(requests)}}, Total: 2, Page: 1, PageSize: 1},
		})
	})
	items, err := FetchAllPages(ListOptions{PageSize: 1}, func(options ListOptions) (Page[ModelDownloadResp], error) {
		page, err := client.ListDownloadPage(ModelDownloadListOptions{ListOptions: options})
		return page.Page, err
	})
	if items != nil || err == nil || !strings.Contains(err.Error(), "requested page 2, received page 1") || requests != 2 {
		t.Fatalf("items=%v err=%v requests=%d", items, err, requests)
	}
}

func TestListDownloadsAggregatesSharedPages(t *testing.T) {
	client := modelDownloadTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.Query().Get("page"); got != "1" {
			t.Fatalf("first page = %q, want 1", got)
		}
		if got := request.URL.Query().Get("category"); got != "dataset" {
			t.Fatalf("category = %q, want dataset", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[Page[ModelDownloadResp]]{
			Data: Page[ModelDownloadResp]{Items: []ModelDownloadResp{{ID: 9}}, Total: 1, Page: 1, PageSize: 200},
		})
	})

	items, err := client.ListDownloads("dataset")
	if err != nil {
		t.Fatalf("ListDownloads returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 9 {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestListAdminDownloadsUsesAdminEndpoint(t *testing.T) {
	client := modelDownloadTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		wantPath := AdminModelDLPfx + "/models/downloads"
		if request.Method != http.MethodGet || request.URL.Path != wantPath {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.URL.Query().Get("page_size") != "200" {
			t.Fatalf("admin list missing page size: %q", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[Page[ModelDownloadResp]]{
			Data: Page[ModelDownloadResp]{Items: []ModelDownloadResp{{ID: 11, Name: "Qwen/Qwen3"}}, Total: 1, Page: 1, PageSize: 200},
		})
	})

	items, err := client.ListAdminDownloads()
	if err != nil {
		t.Fatalf("ListAdminDownloads returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 11 {
		t.Fatalf("unexpected items: %#v", items)
	}
}
