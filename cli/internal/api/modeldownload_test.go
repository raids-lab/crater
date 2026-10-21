package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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
				"total":   31,
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
			Data: map[string]interface{}{"total": 0},
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
			Data: Page[ModelDownloadResp]{Items: []ModelDownloadResp{{ID: 9}}, Total: 1},
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
			Data: Page[ModelDownloadResp]{Items: []ModelDownloadResp{{ID: 11, Name: "Qwen/Qwen3"}}, Total: 1},
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
