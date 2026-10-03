package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/imroc/req/v3"
)

const maxFileErrorBody = 32 << 10

// FileDownloadClient exposes the streaming remote-file download API.
type FileDownloadClient interface {
	DownloadFile(ctx context.Context, remotePath string, destination io.Writer) (int64, error)
}

// NewFileDownloadClient creates a typed remote-file download client.
func NewFileDownloadClient(baseURL, token string) FileDownloadClient {
	return NewClient(baseURL).SetToken(token)
}

// DestinationWriteError identifies a failure writing the downloaded stream to
// the caller-owned destination. The command layer maps this to a local system
// error rather than an API error.
type DestinationWriteError struct {
	Cause error
}

func (e *DestinationWriteError) Error() string {
	return "write download destination: " + e.Cause.Error()
}

func (e *DestinationWriteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// DownloadFile streams one remote file into destination without buffering the
// full response in memory.
func (c *Client) DownloadFile(ctx context.Context, remotePath string, destination io.Writer) (written int64, resultErr error) {
	requestPath := FileDownloadPath + "/" + escapeRemotePath(remotePath)
	resp, err := c.httpClient.R().
		SetContext(ctx).
		DisableAutoReadResponse().
		Get(requestPath)
	if err != nil {
		return 0, &NetworkError{Cause: err}
	}
	if resp.Body != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil && resultErr == nil {
				resultErr = &NetworkError{Cause: err}
			}
		}()
	}
	// This command does not request ranges or support partial downloads.
	if resp.GetStatusCode() != http.StatusOK || resp.Header.Get("Content-Range") != "" {
		if resp.IsSuccessState() {
			return 0, &RequestError{
				HTTPStatus: resp.GetStatusCode(),
				Msg:        "expected a complete 200 OK file response without Content-Range",
			}
		}
		if resp.Body == nil {
			return 0, &RequestError{
				HTTPStatus: resp.GetStatusCode(),
				Msg:        http.StatusText(resp.GetStatusCode()),
			}
		}
		return 0, rawFileRequestError(resp)
	}
	if resp.Body == nil {
		return 0, &NetworkError{Cause: io.ErrUnexpectedEOF}
	}

	tracked := &downloadDestinationWriter{destination: destination}
	written, copyErr := io.Copy(tracked, resp.Body)
	if copyErr != nil {
		if tracked.writeErr != nil {
			return written, &DestinationWriteError{Cause: tracked.writeErr}
		}
		return written, &NetworkError{Cause: copyErr}
	}
	return written, nil
}

type downloadDestinationWriter struct {
	destination io.Writer
	writeErr    error
}

func (writer *downloadDestinationWriter) Write(data []byte) (int, error) {
	written, err := writer.destination.Write(data)
	if err != nil {
		writer.writeErr = err
	} else if written != len(data) {
		writer.writeErr = io.ErrShortWrite
	}
	return written, err
}

func rawFileRequestError(resp *req.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxFileErrorBody+1))
	if readErr != nil {
		message := http.StatusText(resp.GetStatusCode())
		if message == "" {
			message = "failed to read error response"
		}
		return &RequestError{
			HTTPStatus: resp.GetStatusCode(),
			Msg:        message + ": " + readErr.Error(),
		}
	}
	if len(body) > maxFileErrorBody {
		body = body[:maxFileErrorBody]
	}

	var envelope Response[json.RawMessage]
	_ = json.Unmarshal(body, &envelope)
	message := strings.TrimSpace(envelope.Message)
	if message == "" {
		message = strings.TrimSpace(strings.ToValidUTF8(string(body), "\uFFFD"))
	}
	if message == "" {
		message = http.StatusText(resp.GetStatusCode())
	}
	return &RequestError{
		HTTPStatus: resp.GetStatusCode(),
		CraterCode: envelope.Code,
		Msg:        message,
	}
}
