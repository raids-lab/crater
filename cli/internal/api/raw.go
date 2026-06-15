package api

import (
	"fmt"
	"strconv"

	"github.com/imroc/req/v3"
)

type RawClient interface {
	GetRaw(path string, params map[string]string) (interface{}, error)
	GetRawString(path string, params map[string]string) (string, error)
}

func (c *Client) rawRequest(params map[string]string) *req.Request {
	req := c.httpClient.R()
	for key, value := range params {
		if value != "" {
			req.SetQueryParam(key, value)
		}
	}
	return req
}

func (c *Client) GetRaw(path string, params map[string]string) (interface{}, error) {
	var result Response[interface{}]
	resp, err := c.rawRequest(params).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetRawString(path string, params map[string]string) (string, error) {
	var result Response[string]
	resp, err := c.rawRequest(params).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(path)
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func UintPath(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

func NamedPath(prefix, name, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("%s/%s", prefix, name)
	}
	return fmt.Sprintf("%s/%s/%s", prefix, name, suffix)
}
