package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
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

	request, err := c.storageRequest(context.Background())
	if err != nil {
		return nil, err
	}
	var result Response[[]FileInfo]
	resp, err := request.
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

func (c *Client) storageRequest(ctx context.Context) (*req.Request, error) {
	tokens := c.currentTokens()
	if tokens.RefreshToken != "" && accessTokenExpired(tokens.AccessToken, time.Now()) {
		if _, err := c.renewAccessToken(ctx, tokens.AccessToken); err != nil {
			return nil, err
		}
	}
	return c.httpClient.R().SetContext(ctx), nil
}

// The JWT expiry claim only schedules renewal. Storage still authenticates
// every request, including those with a missing or unreadable expiry claim.
func accessTokenExpired(token string, now time.Time) bool {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		ExpiresAt *float64 `json:"exp"`
	}
	return json.Unmarshal(payload, &claims) == nil && claims.ExpiresAt != nil &&
		*claims.ExpiresAt <= float64(now.UnixNano())/float64(time.Second)
}

func escapeRemotePath(remotePath string) string {
	segments := strings.Split(remotePath, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return strings.Join(segments, "/")
}
