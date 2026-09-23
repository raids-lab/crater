package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/imroc/req/v3"
)

const maxUploadErrorBody = 32 << 10

// FileUploadClient exposes the ordinary-user APIs needed to upload one file.
type FileUploadClient interface {
	UploadFile(ctx context.Context, remotePath string, source io.Reader, overwrite bool) (FileUploadResult, error)
}

// NewFileUploadClient creates a typed remote-file upload client.
func NewFileUploadClient(baseURL, token string) FileUploadClient {
	return NewClient(baseURL).SetToken(token)
}

// FileUploadResult is the server-confirmed metadata for an atomic upload.
type FileUploadResult struct {
	RemotePath  string `json:"remote_path"`
	Bytes       int64  `json:"bytes"`
	Overwritten bool   `json:"overwritten"`
}

// SourceReadError identifies a failure reading the caller-owned upload source.
type SourceReadError struct {
	Cause error
}

func (e *SourceReadError) Error() string {
	return "read upload source: " + e.Cause.Error()
}

func (e *SourceReadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// UploadFile streams one source to the storage service without buffering it in
// memory. The dedicated endpoint stages the body and atomically publishes it.
func (c *Client) UploadFile(
	ctx context.Context,
	remotePath string,
	source io.Reader,
	overwrite bool,
) (FileUploadResult, error) {
	requestPath := FileUploadPath + "/" + escapeRemotePath(remotePath)
	tracked := &uploadSourceReader{source: source}
	request := c.httpClient.R().
		SetContext(ctx).
		SetContentType("application/octet-stream").
		SetQueryParam("overwrite", strconv.FormatBool(overwrite)).
		SetBody(tracked).
		DisableAutoReadResponse()

	resp, err := request.Post(requestPath)
	if err != nil {
		if tracked.readErr != nil {
			return FileUploadResult{}, &SourceReadError{Cause: tracked.readErr}
		}
		if resp != nil && resp.Response != nil {
			return FileUploadResult{}, uploadProtocolError(resp, "failed to process upload response: "+err.Error())
		}
		return FileUploadResult{}, &NetworkError{Cause: err}
	}
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if tracked.readErr != nil {
		return FileUploadResult{}, &SourceReadError{Cause: tracked.readErr}
	}
	if !resp.IsSuccessState() {
		if resp.Body == nil {
			return FileUploadResult{}, &RequestError{
				HTTPStatus: resp.GetStatusCode(),
				Msg:        http.StatusText(resp.GetStatusCode()),
			}
		}
		return FileUploadResult{}, rawUploadRequestError(resp)
	}
	if resp.GetStatusCode() != http.StatusOK && resp.GetStatusCode() != http.StatusCreated {
		return FileUploadResult{}, uploadProtocolError(resp, "unexpected upload success status")
	}
	if resp.Body == nil {
		return FileUploadResult{}, uploadProtocolError(resp, "upload response body is empty")
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxUploadErrorBody+1))
	if readErr != nil {
		return FileUploadResult{}, uploadProtocolError(resp, "failed to read upload response: "+readErr.Error())
	}
	if len(body) > maxUploadErrorBody {
		return FileUploadResult{}, uploadProtocolError(resp, "upload response exceeds size limit")
	}
	var result Response[FileUploadResult]
	if err := json.Unmarshal(body, &result); err != nil {
		return FileUploadResult{}, uploadProtocolError(resp, "invalid upload response: "+err.Error())
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return FileUploadResult{}, err
	}
	if result.Data.RemotePath != remotePath {
		return FileUploadResult{}, uploadProtocolError(resp, "upload response remote_path does not match the request")
	}
	if result.Data.Bytes < 0 || result.Data.Bytes != tracked.read {
		return FileUploadResult{}, uploadProtocolError(resp, "upload response byte count does not match the streamed source")
	}
	if result.Data.Overwritten && !overwrite {
		return FileUploadResult{}, uploadProtocolError(resp, "server reported an overwrite without client authorization")
	}
	return result.Data, nil
}

func uploadProtocolError(resp *req.Response, message string) *RequestError {
	status := 0
	if resp != nil && resp.Response != nil {
		status = resp.GetStatusCode()
	}
	return &RequestError{
		HTTPStatus: status,
		Msg:        message,
	}
}

type uploadSourceReader struct {
	source  io.Reader
	read    int64
	readErr error
}

func (reader *uploadSourceReader) Read(data []byte) (int, error) {
	read, err := reader.source.Read(data)
	reader.read += int64(read)
	if err != nil && !errors.Is(err, io.EOF) {
		reader.readErr = err
	}
	return read, err
}

func rawUploadRequestError(resp *req.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxUploadErrorBody+1))
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
	if len(body) > maxUploadErrorBody {
		body = body[:maxUploadErrorBody]
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
