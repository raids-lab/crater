// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCurrentAccountValidatesBackendContext(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"current queue", `{"code":0,"data":{"context":{"queue":"research","roleQueue":2,"accessQueue":3,"space":"alice"}}}`, false},
		{"null data", `{"code":0,"data":null}`, true},
		{"missing context", `{"code":0,"data":{}}`, true},
		{"null context", `{"code":0,"data":{"context":null}}`, true},
		{"empty queue", `{"code":0,"data":{"context":{"queue":" "}}}`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != AuthCheckPath || r.Header.Get("Authorization") != "Bearer test-access" {
					t.Errorf("unexpected current account request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, test.body)
			}))
			t.Cleanup(server.Close)
			got, err := NewClient(server.URL).SetToken("test-access").CurrentAccount()
			if test.wantErr {
				var requestErr *RequestError
				if !errors.As(err, &requestErr) || requestErr.HTTPStatus != 200 || got != nil {
					t.Fatalf("CurrentAccount = %v, %v, want invalid-response error", got, err)
				}
				return
			}
			if err != nil || got.Queue != "research" || got.RoleQueue != 2 || got.AccessQueue != 3 || got.Space != "alice" {
				t.Fatalf("CurrentAccount = %+v, %v, want backend research context", got, err)
			}
		})
	}
}

func TestAccountContextMalformedJSONPreservesHTTPStatus(t *testing.T) {
	for _, status := range []int{200, 401, 403, 500} {
		for _, operation := range []string{"current", "use"} {
			t.Run(http.StatusText(status)+"/"+operation, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					io.WriteString(w, `{"data":{"accessToken":"private-response-token"},`)
				}))
				t.Cleanup(server.Close)
				client := NewClient(server.URL)
				var err error
				if operation == "current" {
					_, err = client.CurrentAccount()
				} else {
					_, err = client.SwitchAccount("research")
				}
				var requestErr *RequestError
				if !errors.As(err, &requestErr) || requestErr.HTTPStatus != status {
					t.Fatalf("error = %v, want HTTP response error with status %d", err, status)
				}
				if requestErr.Msg == "" || strings.Contains(err.Error(), "private-response-token") {
					t.Fatal("decode failure must have a safe message without raw response credentials")
				}
			})
		}
	}
}

func TestAccountContextTransportFailureIsNetworkError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	for _, operation := range []string{"current", "use"} {
		t.Run(operation, func(t *testing.T) {
			client := NewClient(server.URL)
			var err error
			if operation == "current" {
				_, err = client.CurrentAccount()
			} else {
				_, err = client.SwitchAccount("research")
			}
			var networkErr *NetworkError
			if !errors.As(err, &networkErr) || networkErr.Cause == nil {
				t.Fatalf("error = %v, want network error retaining its cause", err)
			}
		})
	}
}

func TestAccountContextPreservesBackendErrors(t *testing.T) {
	for _, status := range []int{200, 401, 403, 404, 500} {
		for _, operation := range []string{"current", "use"} {
			t.Run(http.StatusText(status)+"/"+operation, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					io.WriteString(w, `{"code":1001,"msg":"Queue not found"}`)
				}))
				t.Cleanup(server.Close)
				client := NewClient(server.URL)
				var err error
				if operation == "current" {
					_, err = client.CurrentAccount()
				} else {
					_, err = client.SwitchAccount("research")
				}
				var requestErr *RequestError
				if !errors.As(err, &requestErr) || requestErr.HTTPStatus != status || requestErr.CraterCode != 1001 || requestErr.Msg != "Queue not found" {
					t.Fatalf("error = %v, want preserved status=%d, code=1001 and message", err, status)
				}
			})
		}
	}
}
