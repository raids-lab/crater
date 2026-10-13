package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imroc/req/v3"
)

type signalingWriter struct {
	bytes.Buffer
	firstWrite chan struct{}
	signaled   bool
}

func (writer *signalingWriter) Write(data []byte) (int, error) {
	written, err := writer.Buffer.Write(data)
	if !writer.signaled {
		writer.signaled = true
		close(writer.firstWrite)
	}
	return written, err
}

func TestDownloadFileStreamsAndEncodesPathSegments(t *testing.T) {
	firstDestinationWrite := make(chan struct{})
	handlerDone := make(chan error, 1)
	firstChunk := []byte{0x00, 0x01, 0xff, 'A'}
	secondChunk := []byte("第二块")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			handlerDone <- errors.New("unexpected method: " + request.Method)
			return
		}
		wantPath := "/api/ss/download/user/%E5%AE%9E%E9%AA%8C%20%231/100%25.bin"
		if request.URL.EscapedPath() != wantPath {
			handlerDone <- errors.New("unexpected escaped path: " + request.URL.EscapedPath())
			return
		}
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(firstChunk)
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-firstDestinationWrite:
		case <-time.After(2 * time.Second):
			handlerDone <- errors.New("client buffered the response before writing")
			return
		}
		_, _ = writer.Write(secondChunk)
		handlerDone <- nil
	}))
	defer server.Close()

	destination := &signalingWriter{firstWrite: firstDestinationWrite}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	written, err := NewClient(server.URL).DownloadFile(ctx, "user/实验 #1/100%.bin", destination)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if handlerErr := <-handlerDone; handlerErr != nil {
		t.Fatal(handlerErr)
	}
	want := append(append([]byte{}, firstChunk...), secondChunk...)
	if !bytes.Equal(destination.Bytes(), want) {
		t.Fatalf("downloaded bytes = %v, want %v", destination.Bytes(), want)
	}
	if written != int64(len(want)) {
		t.Fatalf("written = %d, want %d", written, len(want))
	}
}

func TestDownloadFileDecodesJSONAndPlainTextErrors(t *testing.T) {
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
			status:   http.StatusNotFound,
			body:     `{"code":40401,"data":null,"msg":"remote file not found"}`,
			content:  "application/json",
			wantCode: 40401,
			wantMsg:  "remote file not found",
		},
		{
			name:    "plain text",
			status:  http.StatusConflict,
			body:    "download conflict",
			content: "text/plain",
			wantMsg: "download conflict",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", test.content)
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			_, err := NewClient(server.URL).DownloadFile(context.Background(), "user/result.bin", io.Discard)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != test.status || requestErr.CraterCode != test.wantCode || requestErr.Msg != test.wantMsg {
				t.Fatalf("request error = %#v", requestErr)
			}
		})
	}
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestDownloadFileIdentifiesDestinationWriteFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("binary"))
	}))
	defer server.Close()

	sentinel := errors.New("disk full")
	_, err := NewClient(server.URL).DownloadFile(context.Background(), "user/result.bin", failingWriter{err: sentinel})
	var writeErr *DestinationWriteError
	if !errors.As(err, &writeErr) {
		t.Fatalf("error = %T %v, want *DestinationWriteError", err, err)
	}
	if !errors.Is(writeErr, sentinel) {
		t.Fatalf("error = %v, want sentinel", writeErr)
	}
}

func TestRawFileRequestErrorUsesStatusTextForEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := NewClient(server.URL).DownloadFile(context.Background(), "user/result.bin", io.Discard)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.Msg != http.StatusText(http.StatusBadGateway) {
		t.Fatalf("message = %q", requestErr.Msg)
	}
}

type failingReadCloser struct {
	err error
}

func (reader failingReadCloser) Read([]byte) (int, error) {
	return 0, reader.err
}

func (failingReadCloser) Close() error {
	return nil
}

func TestDownloadFileKeepsHTTPStatusWhenErrorBodyReadFails(t *testing.T) {
	sentinel := errors.New("broken error body")
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Header:     make(http.Header),
				Body:       failingReadCloser{err: sentinel},
				Request:    request,
			}, nil
		}
	})

	_, err := client.DownloadFile(context.Background(), "user/missing.bin", io.Discard)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusNotFound || requestErr.Msg != "Not Found: broken error body" {
		t.Fatalf("request error = %#v", requestErr)
	}
}
