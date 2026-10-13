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

func TestDownloadFileRequiresCompleteResponse(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		contentRange string
		wantError    bool
	}{
		{name: "empty file", status: http.StatusOK},
		{name: "complete file", status: http.StatusOK, body: "complete"},
		{name: "no content", status: http.StatusNoContent, wantError: true},
		{name: "created", status: http.StatusCreated, body: "not a file response", wantError: true},
		{name: "partial content", status: http.StatusPartialContent, body: "part", contentRange: "bytes 0-3/10", wantError: true},
		{name: "partial without range header", status: http.StatusPartialContent, body: "part", wantError: true},
		{name: "range with status OK", status: http.StatusOK, body: "part", contentRange: "bytes 0-3/10", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Range") != "" {
					t.Error("unexpected Range request")
				}
				if test.contentRange != "" {
					w.Header().Set("Content-Range", test.contentRange)
				}
				w.WriteHeader(test.status)
				if test.body != "" {
					_, _ = io.WriteString(w, test.body)
				}
			}))
			t.Cleanup(server.Close)
			var destination bytes.Buffer
			n, err := NewClient(server.URL).DownloadFile(context.Background(), "user/result.bin", &destination)
			if test.wantError {
				var requestErr *RequestError
				if !errors.As(err, &requestErr) || requestErr.HTTPStatus != test.status {
					t.Fatalf("error = %v, want RequestError with HTTP %d", err, test.status)
				}
				if n != 0 || destination.Len() != 0 {
					t.Fatalf("rejected response wrote %d bytes: %q", n, destination.String())
				}
				return
			}
			if err != nil || n != int64(len(test.body)) || destination.String() != test.body {
				t.Fatalf("download = (%d, %v, %q), want %q", n, err, destination.String(), test.body)
			}
		})
	}
}

type cancelingDownloadWriter struct {
	cancel context.CancelFunc
}

func (w cancelingDownloadWriter) Write(data []byte) (int, error) {
	w.cancel()
	return len(data), nil
}

func TestDownloadFileCancellationStopsHTTPStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "first chunk")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			t.Error("request was not canceled")
		}
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := NewClient(server.URL).DownloadFile(ctx, "user/result.bin", cancelingDownloadWriter{cancel: cancel})
	var networkErr *NetworkError
	if !errors.As(err, &networkErr) || !errors.Is(networkErr.Cause, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

func TestDownloadFileRejectsTruncatedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "partial")
	}))
	t.Cleanup(server.Close)
	_, err := NewClient(server.URL).DownloadFile(context.Background(), "user/result.bin", io.Discard)
	var networkErr *NetworkError
	if !errors.As(err, &networkErr) || !errors.Is(networkErr.Cause, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want unexpected EOF network error", err)
	}
}

type closeErrorBody struct {
	io.Reader
	closed bool
	err    error
}

func (body *closeErrorBody) Close() error {
	body.closed = true
	return body.err
}

func TestDownloadFileClosesBodyAndPreservesPrimaryError(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusPartialContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			closeErr := errors.New("close failed")
			body := &closeErrorBody{Reader: strings.NewReader("data"), err: closeErr}
			client := NewClient("https://example.invalid")
			client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
				return func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: body, Request: r}, nil
				}
			})
			_, err := client.DownloadFile(context.Background(), "user/result.bin", io.Discard)
			if !body.closed {
				t.Fatal("response body was not closed")
			}
			if status == http.StatusOK {
				var networkErr *NetworkError
				if !errors.As(err, &networkErr) || !errors.Is(networkErr.Cause, closeErr) {
					t.Fatalf("error = %v, want close failure", err)
				}
			} else {
				var requestErr *RequestError
				if !errors.As(err, &requestErr) || requestErr.HTTPStatus != status {
					t.Fatalf("error = %v, want primary HTTP %d error", err, status)
				}
			}
		})
	}
}
