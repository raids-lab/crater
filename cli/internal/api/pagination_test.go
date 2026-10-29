package api

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestListOptionsNormalize(t *testing.T) {
	options := (ListOptions{}).Normalize()
	if options.Page != 1 || options.PageSize != DefaultCLIPageSize {
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

func TestFetchAllPagesAcceptsEmptyCollection(t *testing.T) {
	requests := 0
	fetch := func(options ListOptions) (Page[int], error) {
		requests++
		return Page[int]{Items: []int{}, Total: 0, Page: options.Page, PageSize: 2}, nil
	}

	items, err := FetchAllPages(ListOptions{AllPages: true}, fetch)
	if err != nil {
		t.Fatalf("FetchAllPages returned error: %v", err)
	}
	if requests != 1 || len(items) != 0 {
		t.Fatalf("unexpected empty page handling: requests=%d items=%v", requests, items)
	}
}

func TestFetchAllPagesRejectsUnexpectedEmptyPage(t *testing.T) {
	requests := 0
	fetch := func(options ListOptions) (Page[int], error) {
		requests++
		if options.Page == 1 {
			return Page[int]{Items: []int{1, 2}, Total: 5, Page: 1, PageSize: 2}, nil
		}
		return Page[int]{Items: []int{}, Total: 5, Page: options.Page, PageSize: 2}, nil
	}

	items, err := FetchAllPages(ListOptions{PageSize: 2}, fetch)
	if err == nil {
		t.Fatalf("expected an incomplete-pagination error, got items %v", items)
	}
	if items != nil {
		t.Fatalf("expected no partial result on error, got %v", items)
	}
	if requests != 2 {
		t.Fatalf("expected to stop at the first inconsistent empty page, got %d requests", requests)
	}
}

func TestFetchAllPagesHandlesChangingTotalAndShortLastPage(t *testing.T) {
	fetch := func(options ListOptions) (Page[int], error) {
		switch options.Page {
		case 1:
			return Page[int]{Items: []int{1, 2}, Total: 4, Page: 1, PageSize: 2}, nil
		case 2:
			return Page[int]{Items: []int{3, 4}, Total: 5, Page: 2, PageSize: 2}, nil
		default:
			return Page[int]{Items: []int{5}, Total: 5, Page: 3, PageSize: 2}, nil
		}
	}

	items, err := FetchAllPages(ListOptions{PageSize: 2}, fetch)
	if err != nil {
		t.Fatalf("FetchAllPages returned error: %v", err)
	}
	if !reflect.DeepEqual(items, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("unexpected items: %v", items)
	}
}

func TestFetchAllPagesStopsAfterMaximumPageRequests(t *testing.T) {
	requests := 0
	fetch := func(options ListOptions) (Page[int], error) {
		requests++
		return Page[int]{Items: []int{options.Page}, Total: int64(options.Page + 1), Page: options.Page, PageSize: 1}, nil
	}

	items, err := FetchAllPages(ListOptions{PageSize: 1}, fetch)
	if err == nil {
		t.Fatalf("expected a maximum-request error, got items length %d", len(items))
	}
	if requests != MaxPageRequests {
		t.Fatalf("expected %d requests, got %d", MaxPageRequests, requests)
	}
	if items != nil {
		t.Fatalf("expected no partial result, got %d items", len(items))
	}
}

func TestFetchAllPagesRejectsInvalidPages(t *testing.T) {
	first := Page[int]{Items: []int{1, 2}, Total: 4, Page: 1, PageSize: 2}
	for _, tc := range []struct {
		name  string
		pages []Page[int]
		want  string
	}{
		{"missing page", []Page[int]{{Total: 0, PageSize: 2}}, "requested page 1, received page 0"},
		{"negative page", []Page[int]{{Page: -1, PageSize: 2}}, "received page -1"},
		{"skipped first page", []Page[int]{{Page: 2, PageSize: 2}}, "received page 2"},
		{"repeated page", []Page[int]{first, first}, "requested page 2, received page 1"},
		{"skipped page", []Page[int]{first, {Page: 3, PageSize: 2}}, "requested page 2, received page 3"},
		{"missing size", []Page[int]{{Page: 1}}, "page_size"},
		{"negative size", []Page[int]{{Page: 1, PageSize: -1}}, "page_size"},
		{"changing size", []Page[int]{first, {Items: []int{3}, Total: 3, Page: 2, PageSize: 1}}, "page_size"},
		{"negative total", []Page[int]{{Total: -1, Page: 1, PageSize: 2}}, "invalid total"},
		{"oversized page", []Page[int]{{Items: []int{1, 2, 3}, Total: 3, Page: 1, PageSize: 2}}, "item count"},
		{"nonempty zero total", []Page[int]{{Items: []int{1}, Total: 0, Page: 1, PageSize: 2}}, "exceeding reported total"},
		{"short intermediate page", []Page[int]{{Items: []int{1}, Total: 3, Page: 1, PageSize: 2}}, "short page 1"},
		{"total below collected count", []Page[int]{first, {Items: []int{3, 4}, Total: 3, Page: 2, PageSize: 2}}, "exceeding reported total"},
		{"empty page total below collected count", []Page[int]{first, {Total: 1, Page: 2, PageSize: 2}}, "exceeding reported total"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			items, err := FetchAllPages(ListOptions{PageSize: 2}, func(_ ListOptions) (Page[int], error) {
				if requests >= len(tc.pages) {
					t.Fatal("requested a page after invalid response")
				}
				page := tc.pages[requests]
				requests++
				return page, nil
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) || items != nil || requests != len(tc.pages) {
				t.Fatalf("items=%v err=%v requests=%d; want nil, %q, %d", items, err, requests, tc.want, len(tc.pages))
			}
		})
	}
}

func TestFetchAllPagesAcceptsConsistentDecreasingTotal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second Page[int]
		want   []int
	}{
		{"short final page", Page[int]{Items: []int{3}, Total: 3, Page: 2, PageSize: 2}, []int{1, 2, 3}},
		{"empty final page", Page[int]{Items: []int{}, Total: 2, Page: 2, PageSize: 2}, []int{1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, err := FetchAllPages(ListOptions{PageSize: 2}, func(options ListOptions) (Page[int], error) {
				if options.Page == 1 {
					return Page[int]{Items: []int{1, 2}, Total: 5, Page: 1, PageSize: 2}, nil
				}
				return tc.second, nil
			})
			if err != nil || !reflect.DeepEqual(items, tc.want) {
				t.Fatalf("items=%v err=%v; want %v", items, err, tc.want)
			}
		})
	}
}

func TestFetchAllPagesAcceptsStableServerPageSizeAndEqualValues(t *testing.T) {
	// Page size may be capped by the server; generic values need not be unique.
	items, err := FetchAllPages(ListOptions{PageSize: 200}, func(options ListOptions) (Page[int], error) {
		return Page[int]{Items: []int{1}, Total: 2, Page: options.Page, PageSize: 1}, nil
	})
	if err != nil || !reflect.DeepEqual(items, []int{1, 1}) {
		t.Fatalf("items=%v err=%v; want [1 1]", items, err)
	}
}

func TestFetchAllPagesPreservesRequestFailure(t *testing.T) {
	failure := errors.New("network failure")
	items, err := FetchAllPages(ListOptions{PageSize: 1}, func(options ListOptions) (Page[int], error) {
		if options.Page == 1 {
			return Page[int]{Items: []int{1}, Total: 2, Page: 1, PageSize: 1}, nil
		}
		return Page[int]{}, failure
	})
	if items != nil || !errors.Is(err, failure) {
		t.Fatalf("items=%v err=%v; want nil and original failure", items, err)
	}
}

func TestFetchAllPagesCanCompleteOnLastAllowedRequest(t *testing.T) {
	items, err := FetchAllPages(ListOptions{PageSize: 1}, func(options ListOptions) (Page[int], error) {
		return Page[int]{Items: []int{options.Page}, Total: MaxPageRequests, Page: options.Page, PageSize: 1}, nil
	})
	if err != nil || len(items) != MaxPageRequests {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
}
