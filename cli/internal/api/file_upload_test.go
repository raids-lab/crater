package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/imroc/req/v3"
)

func TestUploadFileStreamsAndEncodesPathSegments(t *testing.T) {
	requestStarted := make(chan struct{})
	firstChunkRead := make(chan struct{})
	handlerDone := make(chan error, 1)
	firstChunk := []byte{0x00, 0x01, 0xff, 'A'}
	secondChunk := []byte("第二块")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(requestStarted)
		if request.Method != http.MethodPost {
			handlerDone <- errors.New("unexpected method: " + request.Method)
			return
		}
		if request.URL.EscapedPath() != "/api/ss/upload/user/%E5%AE%9E%E9%AA%8C%20%231/100%25.bin" {
			handlerDone <- errors.New("unexpected escaped path: " + request.URL.EscapedPath())
			return
		}
		if request.URL.Query().Get("overwrite") != "false" {
			handlerDone <- errors.New("unexpected overwrite query")
			return
		}
		if request.Header.Get("Content-Type") != "application/octet-stream" {
			handlerDone <- errors.New("unexpected content type: " + request.Header.Get("Content-Type"))
			return
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			handlerDone <- errors.New("unexpected authorization header")
			return
		}
		gotFirst := make([]byte, len(firstChunk))
		if _, err := io.ReadFull(request.Body, gotFirst); err != nil {
			handlerDone <- err
			return
		}
		if !bytes.Equal(gotFirst, firstChunk) {
			handlerDone <- errors.New("unexpected first chunk")
			return
		}
		close(firstChunkRead)
		gotRest, err := io.ReadAll(request.Body)
		if err != nil {
			handlerDone <- err
			return
		}
		if !bytes.Equal(gotRest, secondChunk) {
			handlerDone <- errors.New("unexpected second chunk")
			return
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, `{"code":0,"data":{"remote_path":"user/实验 #1/100%.bin","bytes":13,"overwritten":false},"msg":""}`)
		handlerDone <- nil
	}))
	defer server.Close()

	reader, writer := io.Pipe()
	type uploadResult struct {
		upload FileUploadResult
		err    error
	}
	result := make(chan uploadResult, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		upload, err := NewClient(server.URL).SetToken("secret").UploadFile(
			ctx,
			"user/实验 #1/100%.bin",
			reader,
			false,
		)
		result <- uploadResult{upload: upload, err: err}
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not start before the complete source was available")
	}
	if _, err := writer.Write(firstChunk); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstChunkRead:
	case <-time.After(time.Second):
		t.Fatal("server did not receive the first chunk incrementally")
	}
	if _, err := writer.Write(secondChunk); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	got := <-result
	if got.err != nil {
		t.Fatalf("UploadFile: %v", got.err)
	}
	if handlerErr := <-handlerDone; handlerErr != nil {
		t.Fatal(handlerErr)
	}
	wantBytes := int64(len(firstChunk) + len(secondChunk))
	if got.upload.Bytes != wantBytes || got.upload.RemotePath != "user/实验 #1/100%.bin" || got.upload.Overwritten {
		t.Fatalf("upload = %#v, want %d bytes", got.upload, wantBytes)
	}
}

func TestUploadFileOverwriteSetsExplicitQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("overwrite") != "true" {
			t.Errorf("overwrite query = %q", request.URL.Query().Get("overwrite"))
		}
		_, _ = io.Copy(io.Discard, request.Body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"code":0,"data":{"remote_path":"user/result.bin","bytes":11,"overwritten":true},"msg":""}`)
	}))
	defer server.Close()

	upload, err := NewClient(server.URL).UploadFile(
		context.Background(),
		"user/result.bin",
		bytes.NewBufferString("replacement"),
		true,
	)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if upload.Bytes != int64(len("replacement")) || !upload.Overwritten {
		t.Fatalf("upload = %#v", upload)
	}
}

func TestUploadFileAcceptsEmptySource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("body = %v, want empty", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, `{"code":0,"data":{"remote_path":"user/empty.bin","bytes":0,"overwritten":false},"msg":""}`)
	}))
	defer server.Close()

	upload, err := NewClient(server.URL).UploadFile(
		context.Background(),
		"user/empty.bin",
		bytes.NewReader(nil),
		false,
	)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if upload.Bytes != 0 || upload.Overwritten {
		t.Fatalf("upload = %#v", upload)
	}
}

func TestUploadFileDecodesJSONAndPlainTextErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		content  string
		wantCode int
		wantMsg  string
	}{
		{
			name:     "Crater envelope",
			status:   http.StatusConflict,
			body:     `{"code":40901,"data":null,"msg":"target file already exists"}`,
			content:  "application/json",
			wantCode: 40901,
			wantMsg:  "target file already exists",
		},
		{
			name:    "plain text",
			status:  http.StatusConflict,
			body:    "parent directory does not exist",
			content: "text/plain",
			wantMsg: "parent directory does not exist",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				_, _ = io.Copy(io.Discard, request.Body)
				writer.Header().Set("Content-Type", test.content)
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			_, err := NewClient(server.URL).UploadFile(
				context.Background(),
				"user/result.bin",
				bytes.NewBufferString("data"),
				false,
			)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != test.status ||
				requestErr.CraterCode != test.wantCode ||
				requestErr.Msg != test.wantMsg {
				t.Fatalf("request error = %#v", requestErr)
			}
		})
	}
}

func TestUploadFileRejectsInvalidSuccessResponsesWithHTTPStatus(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		overwrite bool
	}{
		{name: "empty body"},
		{name: "malformed JSON", body: `{"code":`},
		{name: "null metadata", body: `{"code":0,"data":null,"msg":""}`},
		{name: "wrong path", body: `{"code":0,"data":{"remote_path":"user/other.bin","bytes":4,"overwritten":false},"msg":""}`},
		{name: "wrong byte count", body: `{"code":0,"data":{"remote_path":"user/result.bin","bytes":3,"overwritten":false},"msg":""}`},
		{name: "negative byte count", body: `{"code":0,"data":{"remote_path":"user/result.bin","bytes":-1,"overwritten":false},"msg":""}`},
		{name: "unauthorized overwrite", body: `{"code":0,"data":{"remote_path":"user/result.bin","bytes":4,"overwritten":true},"msg":""}`},
		{name: "oversized body", body: strings.Repeat("x", maxUploadErrorBody+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				_, _ = io.Copy(io.Discard, request.Body)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			_, err := NewClient(server.URL).UploadFile(
				context.Background(),
				"user/result.bin",
				bytes.NewBufferString("data"),
				test.overwrite,
			)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != http.StatusCreated {
				t.Fatalf("HTTP status = %d, want %d", requestErr.HTTPStatus, http.StatusCreated)
			}
		})
	}
}

type failingUploadReader struct {
	err error
}

func (reader failingUploadReader) Read([]byte) (int, error) {
	return 0, reader.err
}

func TestUploadFileIdentifiesSourceReadFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sentinel := errors.New("local disk failed")
	_, err := NewClient(server.URL).UploadFile(
		context.Background(),
		"user/result.bin",
		failingUploadReader{err: sentinel},
		false,
	)
	var sourceErr *SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %T %v, want *SourceReadError", err, err)
	}
	if !errors.Is(sourceErr, sentinel) {
		t.Fatalf("error = %v, want sentinel", sourceErr)
	}
}

type failingFileReadCloser struct {
	err error
}

func (reader failingFileReadCloser) Read([]byte) (int, error) {
	return 0, reader.err
}

func (failingFileReadCloser) Close() error {
	return nil
}

func TestUploadFileKeepsHTTPStatusWhenErrorBodyReadFails(t *testing.T) {
	sentinel := errors.New("broken error body")
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			_, _ = io.Copy(io.Discard, request.Body)
			return &http.Response{
				StatusCode: http.StatusConflict,
				Status:     "409 Conflict",
				Header:     make(http.Header),
				Body:       failingFileReadCloser{err: sentinel},
				Request:    request,
			}, nil
		}
	})

	_, err := client.UploadFile(
		context.Background(),
		"user/result.bin",
		bytes.NewBufferString("data"),
		false,
	)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusConflict ||
		requestErr.Msg != "Conflict: broken error body" {
		t.Fatalf("request error = %#v", requestErr)
	}
}

func TestUploadFileKeepsHTTPStatusWhenSuccessBodyReadFails(t *testing.T) {
	sentinel := errors.New("broken success body")
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			_, _ = io.Copy(io.Discard, request.Body)
			return &http.Response{
				StatusCode: http.StatusCreated,
				Status:     "201 Created",
				Header:     make(http.Header),
				Body:       failingFileReadCloser{err: sentinel},
				Request:    request,
			}, nil
		}
	})

	_, err := client.UploadFile(
		context.Background(),
		"user/result.bin",
		bytes.NewBufferString("data"),
		false,
	)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusCreated ||
		requestErr.Msg != "failed to read upload response: broken success body" {
		t.Fatalf("request error = %#v", requestErr)
	}
}

func TestUploadFileRejectsUnexpectedSuccessStatus(t *testing.T) {
	for _, status := range []int{http.StatusAccepted, http.StatusNoContent, http.StatusPartialContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"code":0,"data":{"remote_path":"user/result.bin","bytes":4,"overwritten":false}}`)
			}))
			defer server.Close()
			_, err := NewClient(server.URL).UploadFile(t.Context(), "user/result.bin", strings.NewReader("data"), false)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) || requestErr.HTTPStatus != status {
				t.Fatalf("status %d: expected protocol error, got %v", status, err)
			}
		})
	}
}
