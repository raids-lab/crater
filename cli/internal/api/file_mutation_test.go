package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
)

func TestCreateDirectoryUsesMKCOLAndEncodesPathSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "MKCOL" {
			t.Errorf("method = %q, want MKCOL", request.Method)
		}
		if request.URL.EscapedPath() != "/api/ss/user/%E5%AE%9E%E9%AA%8C%20%231" {
			t.Errorf("escaped path = %q", request.URL.EscapedPath())
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		writer.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := NewClient(server.URL).SetToken("secret").CreateDirectory(
		context.Background(),
		"user/实验 #1",
	); err != nil {
		t.Fatalf("CreateDirectory: %v", err)
	}
}

func TestCreateDirectoryRequiresHTTP201(t *testing.T) {
	for _, test := range []struct {
		name        string
		status      int
		body        string
		wantCode    int
		wantMessage string
	}{
		{
			name: "unexpected 200", status: http.StatusOK,
			wantMessage: "OK",
		},
		{
			name: "Crater conflict", status: http.StatusConflict,
			body:     `{"code":40901,"data":null,"msg":"directory already exists"}`,
			wantCode: 40901, wantMessage: "directory already exists",
		},
		{
			name: "WebDAV conflict", status: http.StatusConflict,
			body:        "parent directory does not exist",
			wantMessage: "parent directory does not exist",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			err := NewClient(server.URL).CreateDirectory(context.Background(), "user/results")
			var requestErr *RequestError
			if !errors.As(err, &requestErr) {
				t.Fatalf("error = %T %v, want *RequestError", err, err)
			}
			if requestErr.HTTPStatus != test.status ||
				requestErr.CraterCode != test.wantCode ||
				requestErr.Msg != test.wantMessage {
				t.Fatalf("request error = %#v", requestErr)
			}
		})
	}
}

func TestMoveFileUsesExactDestinationAndEncodesSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		if request.URL.EscapedPath() != "/api/ss/move/user/%E6%BA%90%20%231.txt" {
			t.Errorf("escaped path = %q", request.URL.EscapedPath())
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		var body struct {
			Destination string `json:"dst"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Destination != "account/目标/result.txt" {
			t.Errorf("destination = %q", body.Destination)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"code":0,"data":"move files successfully","msg":""}`)
	}))
	defer server.Close()

	if err := NewClient(server.URL).SetToken("secret").MoveFile(
		context.Background(),
		"user/源 #1.txt",
		"account/目标/result.txt",
	); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
}

func TestMoveFilePreservesServerErrors(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		wantCode int
		wantMsg  string
	}{
		{
			name: "forbidden", status: http.StatusForbidden,
			body:     `{"code":40301,"data":null,"msg":"write permission is required"}`,
			wantCode: 40301, wantMsg: "write permission is required",
		},
		{
			name: "destination exists", status: http.StatusConflict,
			body:     `{"code":40901,"data":null,"msg":"destination path already exists"}`,
			wantCode: 40901, wantMsg: "destination path already exists",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			err := NewClient(server.URL).MoveFile(
				context.Background(),
				"user/source",
				"user/destination",
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

func TestMoveFileRejectsInvalidSuccessResponseWithHTTPStatus(t *testing.T) {
	for _, body := range []string{
		"",
		`{"code":`,
		`{"code":40901,"data":null,"msg":"unexpected conflict"}`,
		strings.Repeat("x", maxUploadErrorBody+1),
	} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, body)
		}))

		err := NewClient(server.URL).MoveFile(context.Background(), "user/source", "user/destination")
		server.Close()
		var requestErr *RequestError
		if !errors.As(err, &requestErr) {
			t.Fatalf("body %q: error = %T %v, want *RequestError", body, err, err)
		}
		if requestErr.HTTPStatus != http.StatusOK {
			t.Fatalf("body %q: status = %d, want 200", body, requestErr.HTTPStatus)
		}
	}
}

func TestFileMutationNetworkErrorsHaveNoHTTPStatus(t *testing.T) {
	sentinel := errors.New("network unavailable")
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(*http.Request) (*http.Response, error) {
			return nil, sentinel
		}
	})

	for _, call := range []func() error{
		func() error {
			return client.CreateDirectory(context.Background(), "user/results")
		},
		func() error {
			return client.MoveFile(context.Background(), "user/source", "user/destination")
		},
	} {
		err := call()
		var networkErr *NetworkError
		if !errors.As(err, &networkErr) {
			t.Fatalf("error = %T %v, want *NetworkError", err, err)
		}
		if !errors.Is(networkErr, sentinel) {
			t.Fatalf("error = %v, want sentinel", networkErr)
		}
	}
}

func TestMoveFileRejectsUncommittedSuccess(t *testing.T) {
	for _, status := range []int{http.StatusAccepted, http.StatusPartialContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"code":0,"msg":""}`)
			}))
			defer server.Close()
			if err := NewClient(server.URL).MoveFile(t.Context(), "user/a", "user/b"); err == nil {
				t.Fatalf("accepted unexpected status %d", status)
			}
		})
	}
}
