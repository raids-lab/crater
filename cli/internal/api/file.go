package api

import (
	"net/url"
	"strings"
	"time"
)

// FileClient exposes the ordinary-user remote file APIs used by the CLI.
type FileClient interface {
	ListFiles(remotePath string) ([]FileInfo, error)
}

// NewFileClient creates a typed remote-file client.
func NewFileClient(baseURL, token string) FileClient {
	return NewClient(baseURL).SetToken(token)
}

// FileInfo is the stable subset of the storage service file-list response.
type FileInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"isdir"`
	ModifyTime time.Time `json:"modifytime"`
}

// ListFiles lists a logical user-visible storage path. remotePath must already
// be validated and normalized by the command layer.
func (c *Client) ListFiles(remotePath string) ([]FileInfo, error) {
	requestPath := FileListPath
	if remotePath != "" {
		requestPath += "/" + escapeRemotePath(remotePath)
	}

	var result Response[[]FileInfo]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(requestPath)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	if result.Data == nil {
		return []FileInfo{}, nil
	}
	return result.Data, nil
}

func escapeRemotePath(remotePath string) string {
	segments := strings.Split(remotePath, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return strings.Join(segments, "/")
}
