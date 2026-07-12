package api

import (
	"reflect"
	"testing"
)

func TestListOptionsNormalize(t *testing.T) {
	options := (ListOptions{}).Normalize()
	if options.Page != 1 || options.PageSize != 50 {
		t.Fatalf("unexpected defaults: %#v", options)
	}
}

func TestListOptionsValues(t *testing.T) {
	values := (ListOptions{Page: 2, PageSize: 100, Sort: "-createdAt"}).Values()
	if values.Get("page") != "2" || values.Get("page_size") != "100" || values.Get("sort") != "-createdAt" {
		t.Fatalf("unexpected query values: %v", values)
	}
}

func TestFetchAllPagesSequentially(t *testing.T) {
	requested := make([]int, 0, 3)
	fetch := func(options ListOptions) (Page[int], error) {
		requested = append(requested, options.Page)
		items := map[int][]int{1: {1, 2}, 2: {3, 4}, 3: {5}}[options.Page]
		return Page[int]{Items: items, Total: 5, Page: options.Page, PageSize: 2}, nil
	}

	items, err := FetchAllPages(ListOptions{PageSize: 2, AllPages: true}, fetch)
	if err != nil {
		t.Fatalf("FetchAllPages returned error: %v", err)
	}
	if !reflect.DeepEqual(requested, []int{1, 2, 3}) {
		t.Fatalf("unexpected request order: %v", requested)
	}
	if !reflect.DeepEqual(items, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("unexpected items: %v", items)
	}
}

func TestFetchAllPagesStopsOnEmptyPage(t *testing.T) {
	requests := 0
	fetch := func(options ListOptions) (Page[int], error) {
		requests++
		return Page[int]{Items: []int{}, Total: 10, Page: options.Page, PageSize: 2}, nil
	}

	items, err := FetchAllPages(ListOptions{AllPages: true}, fetch)
	if err != nil {
		t.Fatalf("FetchAllPages returned error: %v", err)
	}
	if requests != 1 || len(items) != 0 {
		t.Fatalf("unexpected empty page handling: requests=%d items=%v", requests, items)
	}
}
