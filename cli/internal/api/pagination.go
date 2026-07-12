package api

import (
	"net/url"
	"strconv"
)

const defaultCLIPageSize = 50

type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type ListOptions struct {
	Page     int
	PageSize int
	Sort     string
	AllPages bool
}

func (options ListOptions) Normalize() ListOptions {
	if options.Page < 1 {
		options.Page = 1
	}
	if options.PageSize < 1 {
		options.PageSize = defaultCLIPageSize
	}
	return options
}

func (options ListOptions) Values() url.Values {
	options = options.Normalize()
	values := url.Values{
		"page":      {strconv.Itoa(options.Page)},
		"page_size": {strconv.Itoa(options.PageSize)},
	}
	if options.Sort != "" {
		values.Set("sort", options.Sort)
	}
	return values
}

func FetchAllPages[T any](
	options ListOptions,
	fetch func(ListOptions) (Page[T], error),
) ([]T, error) {
	options = options.Normalize()
	options.Page = 1
	items := make([]T, 0)
	for {
		page, err := fetch(options)
		if err != nil {
			return nil, err
		}
		if len(page.Items) == 0 {
			return items, nil
		}
		items = append(items, page.Items...)
		if int64(len(items)) >= page.Total {
			return items, nil
		}
		options.Page++
	}
}
