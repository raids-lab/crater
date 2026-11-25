package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/spf13/cobra"
)

type fakeDownloadLister struct {
	requests []api.ModelDownloadListOptions
	pages    map[int]api.ModelDownloadPage
}

func (fake *fakeDownloadLister) ListDownloadPage(
	options api.ModelDownloadListOptions,
) (api.ModelDownloadPage, error) {
	fake.requests = append(fake.requests, options)
	return fake.pages[options.Page], nil
}

func TestReadDownloadListOptionsDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	addDownloadListFlags(cmd)

	options, err := readDownloadListOptions(cmd)
	if err != nil {
		t.Fatalf("readDownloadListOptions returned error: %v", err)
	}
	if options.Page != 1 || options.PageSize != api.DefaultCLIPageSize || options.AllPages {
		t.Fatalf("unexpected defaults: %#v", options.ListOptions)
	}
}

func TestReadDownloadListOptions(t *testing.T) {
	cmd := &cobra.Command{}
	addDownloadListFlags(cmd)
	for name, value := range map[string]string{
		"page":      "2",
		"page-size": "25",
		"category":  " model ",
		"status":    " Downloading ",
		"search":    " Qwen ",
	} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}

	options, err := readDownloadListOptions(cmd)
	if err != nil {
		t.Fatalf("readDownloadListOptions returned error: %v", err)
	}
	if options.Page != 2 || options.PageSize != 25 || options.AllPages {
		t.Fatalf("unexpected pagination: %#v", options.ListOptions)
	}
	if options.Category != "model" || options.Status != "Downloading" || options.Search != "Qwen" {
		t.Fatalf("unexpected filters: %#v", options)
	}
}

func TestReadDownloadListOptionsUsesEndpointMaxBatchForAllPages(t *testing.T) {
	cmd := &cobra.Command{}
	addDownloadListFlags(cmd)
	if err := cmd.Flags().Set("all-pages", "true"); err != nil {
		t.Fatal(err)
	}

	options, err := readDownloadListOptions(cmd)
	if err != nil {
		t.Fatalf("readDownloadListOptions returned error: %v", err)
	}
	if !options.AllPages || options.PageSize != maxDownloadPageSize {
		t.Fatalf("unexpected all-pages pagination: %#v", options.ListOptions)
	}
}

func TestReadDownloadListOptionsUsesEndpointPageSizeLimit(t *testing.T) {
	cmd := &cobra.Command{}
	addDownloadListFlags(cmd)
	if err := cmd.Flags().Set("page-size", "101"); err != nil {
		t.Fatal(err)
	}

	_, err := readDownloadListOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "page-size must be between 1 and 100") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadDownloadListOptionsRejectsUnknownStatus(t *testing.T) {
	cmd := &cobra.Command{}
	addDownloadListFlags(cmd)
	if err := cmd.Flags().Set("status", "Unknown"); err != nil {
		t.Fatal(err)
	}

	_, err := readDownloadListOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "invalid download status: Unknown") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchDownloadListPreservesServerPage(t *testing.T) {
	fake := &fakeDownloadLister{
		pages: map[int]api.ModelDownloadPage{
			2: {
				Page: api.Page[api.ModelDownloadResp]{
					Items:    []api.ModelDownloadResp{{ID: 2}},
					Total:    30,
					Page:     2,
					PageSize: 15,
				},
				Summary: map[string]int64{"Ready": 30},
			},
		},
	}
	options := api.ModelDownloadListOptions{
		ListOptions: api.ListOptions{Page: 2, PageSize: 15},
		Category:    "model",
	}

	page, err := fetchDownloadList(fake, options)
	if err != nil {
		t.Fatalf("fetchDownloadList returned error: %v", err)
	}
	if len(fake.requests) != 1 || fake.requests[0].Page != 2 {
		t.Fatalf("unexpected requests: %#v", fake.requests)
	}
	if page.Total != 30 || page.Page.Page != 2 || len(page.Items) != 1 || page.Items[0].ID != 2 {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestFetchDownloadListAllPagesSequentially(t *testing.T) {
	summary := map[string]int64{"Downloading": 1, "Ready": 2}
	fake := &fakeDownloadLister{
		pages: map[int]api.ModelDownloadPage{
			1: {
				Page: api.Page[api.ModelDownloadResp]{
					Items: []api.ModelDownloadResp{{ID: 3}, {ID: 2}},
					Total: 3,
					Page:  1, PageSize: 2,
				},
				Summary: summary,
			},
			2: {
				Page: api.Page[api.ModelDownloadResp]{
					Items: []api.ModelDownloadResp{{ID: 1}},
					Total: 3,
					Page:  2, PageSize: 2,
				},
			},
		},
	}
	options := api.ModelDownloadListOptions{
		ListOptions: api.ListOptions{Page: 9, PageSize: 2, AllPages: true},
		Status:      "Ready",
		Search:      "qwen",
	}

	page, err := fetchDownloadList(fake, options)
	if err != nil {
		t.Fatalf("fetchDownloadList returned error: %v", err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("unexpected request count: %#v", fake.requests)
	}
	gotPages := []int{fake.requests[0].Page, fake.requests[1].Page}
	if !reflect.DeepEqual(gotPages, []int{1, 2}) {
		t.Fatalf("requests are not sequential: %v", gotPages)
	}
	for _, request := range fake.requests {
		if request.PageSize != 2 || request.Status != "Ready" || request.Search != "qwen" {
			t.Fatalf("filters changed between pages: %#v", request)
		}
	}
	gotIDs := []uint{page.Items[0].ID, page.Items[1].ID, page.Items[2].ID}
	if !reflect.DeepEqual(gotIDs, []uint{3, 2, 1}) {
		t.Fatalf("unexpected item order: %v", gotIDs)
	}
	if page.Total != 3 || !reflect.DeepEqual(page.Summary, summary) {
		t.Fatalf("unexpected aggregate page: %#v", page)
	}
}

func TestDownloadListPayloadUsesCommonPagination(t *testing.T) {
	page := api.ModelDownloadPage{
		Page: api.Page[api.ModelDownloadResp]{
			Items:    []api.ModelDownloadResp{{ID: 7}},
			Total:    22,
			Page:     2,
			PageSize: 15,
		},
		Summary: map[string]int64{"Ready": 22},
	}

	payload := downloadListPayload(page, false)
	pagination, ok := payload["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("pagination missing from payload: %#v", payload)
	}
	if pagination["page"] != 2 || pagination["page_size"] != 15 || pagination["total"] != int64(22) {
		t.Fatalf("unexpected pagination: %#v", pagination)
	}
	if !reflect.DeepEqual(payload["summary"], page.Summary) {
		t.Fatalf("unexpected summary: %#v", payload["summary"])
	}

	allPagesPayload := downloadListPayload(page, true)
	if _, exists := allPagesPayload["pagination"]; exists {
		t.Fatalf("all-pages payload contains pagination: %#v", allPagesPayload)
	}
}

func TestFilterAdminModelDownloadsBeforePagination(t *testing.T) {
	downloads := []api.ModelDownloadResp{
		{ID: 1, Name: "Qwen/Qwen3", Category: "model", Status: "Ready"},
		{ID: 2, Name: "Qwen/Dataset", Category: "dataset", Status: "Ready"},
		{ID: 3, Name: "Other/Model", Category: "model", Status: "Failed"},
	}
	filtered := filterAdminModelDownloads(downloads, api.ModelDownloadListOptions{
		Category: "model",
		Status:   "Ready",
		Search:   "qwen",
	})
	page := paginateList(filtered, api.ListOptions{Page: 1, PageSize: 1})
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != 1 {
		t.Fatalf("unexpected filtered page: %#v", page)
	}
}
