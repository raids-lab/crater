package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
)

func TestRemoveFileUsesDedicatedEndpointAndEncodesEachPathSegment(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		t.Run("recursive="+strconv.FormatBool(recursive), func(t *testing.T) {
			recursiveValue := strconv.FormatBool(recursive)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(
				func(writer http.ResponseWriter, request *http.Request) {
					requests++
					if request.Method != http.MethodDelete {
						t.Errorf("method = %q, want DELETE", request.Method)
					}
					if request.URL.EscapedPath() !=
						"/api/ss/files/user/%E5%AE%9E%E9%AA%8C%20%231/result%3F.txt" {
						t.Errorf("escaped path = %q", request.URL.EscapedPath())
					}
					if request.URL.RawQuery != "recursive="+recursiveValue {
						t.Errorf("raw query = %q", request.URL.RawQuery)
					}
					if request.Header.Get("Authorization") != "Bearer secret" {
						t.Errorf("authorization = %q", request.Header.Get("Authorization"))
					}
					writer.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(
						writer,
						`{"code":0,"data":{"remote_path":"user/实验 #1/result?.txt","recursive":`+
							recursiveValue+
							`},"msg":""}`,
					)
				},
			))
			defer server.Close()

			result, err := NewClient(server.URL).SetToken("secret").RemoveFile(
				context.Background(),
				"user/实验 #1/result?.txt",
				recursive,
			)
			if err != nil {
				t.Fatalf("RemoveFile: %v", err)
			}
			if result.RemotePath != "user/实验 #1/result?.txt" ||
				result.Recursive != recursive {
				t.Fatalf("result = %#v", result)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want exactly one", requests)
			}
		})
	}
}

func TestRemoveFileAcceptsOnlyHTTP200AndCodeZero(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		wantCode int
	}{
		{
			name:   "created is not success",
			status: http.StatusCreated,
			body:   `{"code":0,"data":{"remote_path":"user/file","recursive":false},"msg":""}`,
		},
		{
			name:   "no content is not success",
			status: http.StatusNoContent,
		},
		{
			name:     "nonzero Crater code",
			status:   http.StatusOK,
			body:     `{"code":40902,"data":null,"msg":"recursive removal required"}`,
			wantCode: 40902,
		},
		{
			name:     "server conflict",
			status:   http.StatusConflict,
			body:     `{"code":40902,"data":null,"msg":"recursive removal required"}`,
			wantCode: 40902,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(writer http.ResponseWriter, _ *http.Request) {
					writer.Header().Set("Content-Type", "application/json")
					writer.WriteHeader(test.status)
					_, _ = io.WriteString(writer, test.body)
				},
			))
			defer server.Close()

			_, err := NewClient(server.URL).RemoveFile(
				context.Background(),
				"user/file",
				false,
			)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != test.status || requestErr.CraterCode != test.wantCode {
				t.Fatalf("request error = %#v", requestErr)
			}
		})
	}
}

func TestRemoveFileRejectsInvalidSuccessMetadata(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		recursive bool
	}{
		{name: "empty body", body: ""},
		{name: "malformed JSON", body: `{"code":`},
		{
			name: "missing code",
			body: `{"data":{"remote_path":"user/file","recursive":false},"msg":""}`,
		},
		{name: "null data", body: `{"code":0,"data":null,"msg":""}`},
		{
			name: "missing remote path",
			body: `{"code":0,"data":{"recursive":false},"msg":""}`,
		},
		{
			name: "mismatched remote path",
			body: `{"code":0,"data":{"remote_path":"user/other","recursive":false},"msg":""}`,
		},
		{
			name: "missing recursive",
			body: `{"code":0,"data":{"remote_path":"user/file"},"msg":""}`,
		},
		{
			name:      "mismatched recursive",
			body:      `{"code":0,"data":{"remote_path":"user/file","recursive":false},"msg":""}`,
			recursive: true,
		},
		{name: "oversized body", body: strings.Repeat("x", maxUploadErrorBody+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(writer http.ResponseWriter, _ *http.Request) {
					writer.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(writer, test.body)
				},
			))
			defer server.Close()

			_, err := NewClient(server.URL).RemoveFile(
				context.Background(),
				"user/file",
				test.recursive,
			)
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != http.StatusOK {
				t.Fatalf("status = %d, want 200", requestErr.HTTPStatus)
			}
		})
	}
}

func TestRemoveFilePreservesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(
				writer,
				`{"code":40301,"data":null,"msg":"write permission is required"}`,
			)
		},
	))
	defer server.Close()

	_, err := NewClient(server.URL).RemoveFile(context.Background(), "public/file", false)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusForbidden ||
		requestErr.CraterCode != 40301 ||
		requestErr.Msg != "write permission is required" {
		t.Fatalf("request error = %#v", requestErr)
	}
}

func TestRemoveFileNetworkErrorHasNoHTTPStatusAndNoFallback(t *testing.T) {
	sentinel := errors.New("network unavailable")
	requests := 0
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			requests++
			if request.URL.Path != "/api/ss/files/user/file" {
				t.Errorf("request path = %q", request.URL.Path)
			}
			return nil, sentinel
		}
	})

	_, err := client.RemoveFile(context.Background(), "user/file", false)
	var networkErr *NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("error = %T %v, want *NetworkError", err, err)
	}
	if !errors.Is(networkErr, sentinel) {
		t.Fatalf("error = %v, want sentinel", networkErr)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want exactly one and no fallback", requests)
	}
}
