package api

import (
	"net/url"
	"strconv"
)

// DefaultCLIPageSize keeps interactive list output compact unless callers opt in to more rows.
const DefaultCLIPageSize = 15

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
		options.PageSize = DefaultCLIPageSize
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

// fetchAllListPages adapts paginated list endpoints to complete-list consumers.
func fetchAllListPages[T any](client *Client, path, sort string) ([]T, error) {
	return fetchAllListPagesWithParams[T](client, path, map[string]string{"sort": sort})
}

func fetchAllListPagesWithParams[T any](client *Client, path string, params map[string]string) ([]T, error) {
	return FetchAllPages(ListOptions{PageSize: 200}, func(options ListOptions) (Page[T], error) {
		var result Response[Page[T]]
		values := make(map[string]string, len(params)+2)
		for key, value := range params {
			values[key] = value
		}
		values["page"] = strconv.Itoa(options.Page)
		values["page_size"] = strconv.Itoa(options.PageSize)
		response, err := client.rawRequest(values).SetSuccessResult(&result).SetErrorResult(&result).Get(path)
		if err != nil {
			return Page[T]{}, &NetworkError{Cause: err}
		}
		if err := errorFromResponse(response, result.Code, result.Message); err != nil {
			return Page[T]{}, err
		}
		return result.Data, nil
	})
}
