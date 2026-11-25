package api

import (
	"encoding/json"
	"fmt"
	"math"
)

type OperationLogs struct {
	Items []json.RawMessage `json:"items"`
	Total int64             `json:"total"`
}

// GetOperationLogs reads one logical CLI page using bounded server pages.
func (c *Client) GetOperationLogs(page, limit int, params map[string]string) (OperationLogs, error) {
	if page < 1 || limit < 1 || page > math.MaxInt/limit {
		return OperationLogs{}, fmt.Errorf("invalid operation log page or limit")
	}
	const maximumPageSize = 200
	size := min(limit, maximumPageSize)
	start := (page - 1) * limit
	end := start + limit
	options := ListOptions{Page: start/size + 1, PageSize: size}
	result := OperationLogs{Items: []json.RawMessage{}}
	for request := 0; request < MaxPageRequests; request++ {
		if options.Page > math.MaxInt/size {
			return OperationLogs{}, fmt.Errorf("operation log page exceeds server pagination bounds")
		}
		part, err := fetchListPage[json.RawMessage](c, AdminOperationLogs, params, options)
		if err != nil {
			return OperationLogs{}, err
		}
		if part.Page != options.Page || part.PageSize != size || part.Total < 0 {
			return OperationLogs{}, fmt.Errorf("invalid pagination metadata for operation logs page %d", options.Page)
		}
		if request != 0 && result.Total != part.Total {
			return OperationLogs{}, fmt.Errorf("operation log total changed while reading page %d; retry the command", page)
		}
		result.Total = part.Total
		offset := (options.Page - 1) * size
		expected := int(min(int64(size), max(0, part.Total-int64(offset))))
		if len(part.Items) != expected {
			return OperationLogs{}, fmt.Errorf("incomplete operation logs page %d: received %d items, expected %d", options.Page, len(part.Items), expected)
		}
		first := max(start-offset, 0)
		last := min(end-offset, len(part.Items))
		if first < last {
			result.Items = append(result.Items, part.Items[first:last]...)
		}
		if offset+size >= end || int64(offset+size) >= result.Total {
			return result, nil
		}
		options.Page++
	}
	return OperationLogs{}, fmt.Errorf("operation log page exceeded maximum of %d requests", MaxPageRequests)
}
