package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/imroc/req/v3"
)

// FileMutationClient exposes the ordinary-user APIs for creating directories
// and moving one remote storage entry.
type FileMutationClient interface {
	CreateDirectory(ctx context.Context, remotePath string) error
	MoveFile(ctx context.Context, sourcePath, destinationPath string) error
}

// NewFileMutationClient creates a typed remote-storage mutation client.
func NewFileMutationClient(baseURL, token string) FileMutationClient {
	return NewClient(baseURL).SetToken(token)
}

type moveFileRequest struct {
	Destination string `json:"dst"`
}

// CreateDirectory creates exactly one directory. Parent directories are never
// created implicitly, and HTTP 201 is the only accepted success status.
func (c *Client) CreateDirectory(ctx context.Context, remotePath string) error {
	requestPath := StoragePrefix + "/" + escapeRemotePath(remotePath)
	resp, err := c.httpClient.R().
		SetContext(ctx).
		DisableAutoReadResponse().
		Send("MKCOL", requestPath)
	if resp != nil && resp.Response != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return fileRequestTransportError(resp, err)
	}
	if resp.GetStatusCode() != http.StatusCreated {
		return rawUploadRequestError(resp)
	}
	return nil
}

// MoveFile moves one file or directory to the exact destination path. The
// storage service must reject an existing destination.
func (c *Client) MoveFile(ctx context.Context, sourcePath, destinationPath string) error {
	requestPath := FileMovePath + "/" + escapeRemotePath(sourcePath)
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetContentType("application/json").
		SetBody(moveFileRequest{Destination: destinationPath}).
		DisableAutoReadResponse().
		Post(requestPath)
	if resp != nil && resp.Response != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return fileRequestTransportError(resp, err)
	}
	if resp.GetStatusCode() != http.StatusOK {
		return rawUploadRequestError(resp)
	}
	if resp.Body == nil {
		return fileProtocolError(resp, "move response body is empty")
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxUploadErrorBody+1))
	if readErr != nil {
		return fileProtocolError(resp, "failed to read move response: "+readErr.Error())
	}
	if len(body) > maxUploadErrorBody {
		return fileProtocolError(resp, "move response exceeds size limit")
	}
	var envelope Response[json.RawMessage]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fileProtocolError(resp, "invalid move response: "+err.Error())
	}
	return errorFromResponse(resp, envelope.Code, envelope.Message)
}

func fileRequestTransportError(resp *req.Response, err error) error {
	if resp != nil && resp.Response != nil {
		return rawUploadRequestError(resp)
	}
	return &NetworkError{Cause: err}
}

func fileProtocolError(resp *req.Response, message string) *RequestError {
	status := 0
	if resp != nil && resp.Response != nil {
		status = resp.GetStatusCode()
	}
	return &RequestError{
		HTTPStatus: status,
		Msg:        message,
	}
}
