package payload

import "testing"

func TestListPageQueryNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         ListPageQuery
		wantOffset int
		wantLimit  int
		wantOrder  Order
	}{
		{
			name:       "defaults when zero",
			in:         ListPageQuery{},
			wantOffset: 0,
			wantLimit:  20,
			wantOrder:  Desc,
		},
		{
			name:       "page two with explicit size",
			in:         ListPageQuery{Page: 2, PageSize: 50, Order: Asc},
			wantOffset: 50,
			wantLimit:  50,
			wantOrder:  Asc,
		},
		{
			name:       "page size capped at MaxListPageSize",
			in:         ListPageQuery{Page: 1, PageSize: 9999},
			wantOffset: 0,
			wantLimit:  MaxListPageSize,
			wantOrder:  Desc,
		},
		{
			name:       "negative page treated as 1",
			in:         ListPageQuery{Page: -3, PageSize: 10},
			wantOffset: 0,
			wantLimit:  10,
			wantOrder:  Desc,
		},
		{
			name:       "unknown order falls back to desc",
			in:         ListPageQuery{Page: 1, PageSize: 10, Order: Order("garbage")},
			wantOffset: 0,
			wantLimit:  10,
			wantOrder:  Desc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			off, lim, ord := tt.in.Normalize()
			if off != tt.wantOffset {
				t.Fatalf("offset: got %d want %d", off, tt.wantOffset)
			}
			if lim != tt.wantLimit {
				t.Fatalf("limit: got %d want %d", lim, tt.wantLimit)
			}
			if ord != tt.wantOrder {
				t.Fatalf("order: got %q want %q", ord, tt.wantOrder)
			}
		})
	}
}
