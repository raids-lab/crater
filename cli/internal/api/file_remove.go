package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/imroc/req/v3"
)

// FileRemoveClient exposes the ordinary-user API for removing one remote
// storage entry.
type FileRemoveClient interface {
	RemoveFile(ctx context.Context, remotePath string, recursive bool) (FileRemoveResult, error)
}

// FileRemoveResult is the server-confirmed metadata for one removal.
type FileRemoveResult struct {
	RemotePath string `json:"remote_path"`
	Recursive  bool   `json:"recursive"`
}

// NewFileRemoveClient creates a typed remote-file removal client.
func NewFileRemoveClient(baseURL, token string) FileRemoveClient {
	return NewClient(baseURL).SetToken(token)
}

type removeFileResponseData struct {
	RemotePath *string `json:"remote_path"`
	Recursive  *bool   `json:"recursive"`
}

type removeFileResponse struct {
	Code    *int                   `json:"code"`
	Data    removeFileResponseData `json:"data"`
	Message string                 `json:"msg"`
}

// RemoveFile removes exactly one remote entry. Directories are accepted only
// when recursive is true. This method intentionally has no fallback to the
// legacy, unconditional /delete endpoint.
func (c *Client) RemoveFile(
	ctx context.Context,
	remotePath string,
	recursive bool,
) (FileRemoveResult, error) {
	requestPath := FileRemovePath + "/" + escapeRemotePath(remotePath)
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("recursive", strconv.FormatBool(recursive)).
		DisableAutoReadResponse().
		Delete(requestPath)
	if resp != nil && resp.Response != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return FileRemoveResult{}, fileRequestTransportError(resp, err)
	}
	if resp.GetStatusCode() != http.StatusOK {
		return FileRemoveResult{}, rawUploadRequestError(resp)
	}
	if resp.Body == nil {
		return FileRemoveResult{}, fileRemoveProtocolError(resp, "remove response body is empty")
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxUploadErrorBody+1))
	if readErr != nil {
		return FileRemoveResult{}, fileRemoveProtocolError(
			resp,
			"failed to read remove response: "+readErr.Error(),
		)
	}
	if len(body) > maxUploadErrorBody {
		return FileRemoveResult{}, fileRemoveProtocolError(resp, "remove response exceeds size limit")
	}

	var envelope removeFileResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return FileRemoveResult{}, fileRemoveProtocolError(
			resp,
			"invalid remove response: "+err.Error(),
		)
	}
	if envelope.Code == nil {
		return FileRemoveResult{}, fileRemoveProtocolError(resp, "remove response code is missing")
	}
	if err := errorFromResponse(resp, *envelope.Code, envelope.Message); err != nil {
		return FileRemoveResult{}, err
	}
	if envelope.Data.RemotePath == nil || *envelope.Data.RemotePath != remotePath {
		return FileRemoveResult{}, fileRemoveProtocolError(
			resp,
			"remove response remote_path does not match the request",
		)
	}
	if envelope.Data.Recursive == nil || *envelope.Data.Recursive != recursive {
		return FileRemoveResult{}, fileRemoveProtocolError(
			resp,
			"remove response recursive value does not match the request",
		)
	}
	return FileRemoveResult{
		RemotePath: *envelope.Data.RemotePath,
		Recursive:  *envelope.Data.Recursive,
	}, nil
}

func fileRemoveProtocolError(resp *req.Response, message string) *RequestError {
	status := 0
	if resp != nil && resp.Response != nil {
		status = resp.GetStatusCode()
	}
	return &RequestError{
		HTTPStatus: status,
		Msg:        message,
	}
}
