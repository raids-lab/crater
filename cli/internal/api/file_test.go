package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/imroc/req/v3"
)

func fileTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			return recorder.Result(), nil
		}
	})
	return client
}

func writeFileTestResponse(t *testing.T, writer http.ResponseWriter, data interface{}) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]interface{}{
		"code": 0,
		"data": data,
		"msg":  "",
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestListFilesRoutesAndDecodes(t *testing.T) {
	tests := []struct {
		name        string
		remotePath  string
		escapedPath string
	}{
		{name: "visible root", remotePath: "", escapedPath: "/api/ss/files"},
		{name: "nested ASCII", remotePath: "user/projects", escapedPath: "/api/ss/files/user/projects"},
		{name: "spaces unicode and reserved bytes", remotePath: "user/实验 #1/100%", escapedPath: "/api/ss/files/user/%E5%AE%9E%E9%AA%8C%20%231/100%25"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fileTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", request.Method)
				}
				if request.URL.EscapedPath() != test.escapedPath {
					t.Errorf("escaped path = %q, want %q", request.URL.EscapedPath(), test.escapedPath)
				}
				writeFileTestResponse(t, writer, []map[string]interface{}{{
					"name":       "checkpoint.bin",
					"size":       42,
					"isdir":      false,
					"modifytime": "2026-07-26T08:09:10Z",
					"sys":        map[string]string{"ignored": "value"},
				}})
			})

			files, err := client.ListFiles(test.remotePath)
			if err != nil {
				t.Fatalf("ListFiles: %v", err)
			}
			if len(files) != 1 || files[0].Name != "checkpoint.bin" || files[0].Size != 42 || files[0].IsDir {
				t.Fatalf("files = %#v", files)
			}
			if got := files[0].ModifyTime.UTC().Format("2006-01-02T15:04:05Z"); got != "2026-07-26T08:09:10Z" {
				t.Fatalf("modify time = %q", got)
			}
		})
	}
}

func TestListFilesNormalizesNullToEmptySlice(t *testing.T) {
	client := fileTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
		writeFileTestResponse(t, writer, nil)
	})

	files, err := client.ListFiles("user/empty")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if files == nil || !reflect.DeepEqual(files, []FileInfo{}) {
		t.Fatalf("files = %#v, want non-nil empty slice", files)
	}
}

func TestListFilesPreservesStorageError(t *testing.T) {
	client := fileTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{
			"code": 7,
			"data": nil,
			"msg":  "permission denied",
		})
	})

	_, err := client.ListFiles("public/private")
	requestErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusUnauthorized || requestErr.CraterCode != 7 || requestErr.Msg != "permission denied" {
		t.Fatalf("request error = %#v", requestErr)
	}
}
