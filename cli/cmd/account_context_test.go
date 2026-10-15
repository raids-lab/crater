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

package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/internal/testutil"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func setupAccountSession(t *testing.T, platform string) session.State {
	t.Helper()
	testutil.IsolateUserConfigDir(t)
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "")
	previousJSON := outputJSON
	outputJSON = true
	t.Cleanup(func() { outputJSON = previousJSON })
	active := session.ActiveContext{PlatformURL: platform, Username: "alice", Method: "normal"}
	st := session.State{
		ActiveContext: active,
		Language:      "zh-CN",
		AuthInfos: []session.AuthInfo{
			{PlatformURL: platform, Username: "alice", Method: "normal", UserID: 1, Nickname: "Alice", Role: "user", Token: "original-access"},
			{PlatformURL: platform, Username: "bob", Method: "ldap", UserID: 2, Nickname: "Bob", Role: "admin", Token: "other-access"},
		},
	}
	if err := session.SaveState(st); err != nil {
		t.Fatal(err)
	}
	return st
}

func captureAccountOutput(t *testing.T, run func() error) (string, error) {
	t.Helper()
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdout = previous
		reader.Close()
		writer.Close()
	})
	os.Stdout = writer
	runErr := run()
	os.Stdout = previous
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data), runErr
}

func TestAccountUsePersistsCredentialsForSubsequentRequests(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case api.AuthSwitchPath:
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer original-access" {
				t.Error("switch must POST with the original credentials")
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(body, map[string]string{"queue": "research"}) {
				t.Errorf("switch body = %v, want only queue=research", body)
			}
			io.WriteString(w, `{"code":0,"data":{"accessToken":"scoped-access","refreshToken":"private-refresh","context":{"queue":"research","roleQueue":2}}}`)
		case api.AuthCheckPath:
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer scoped-access" {
				t.Error("current must GET with the persisted project credentials")
			}
			io.WriteString(w, `{"code":0,"data":{"context":{"queue":"research","roleQueue":2}}}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	want := setupAccountSession(t, server.URL)
	stdout, err := captureAccountOutput(t, func() error { return runAccountUse(accountUseCmd, []string{" research "}) })
	if err != nil {
		t.Fatalf("switch account: %v", err)
	}
	var result struct {
		Status string `json:"status"`
		Data   struct {
			Username string             `json:"username"`
			Context  api.AccountContext `json:"context"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode switch output: %v", err)
	}
	if result.Status != "OK" || result.Data.Username != "alice" || result.Data.Context.Queue != "research" {
		t.Fatalf("unexpected switch result: %s", stdout)
	}
	for _, secret := range []string{"scoped-access", "private-refresh", "accessToken", "refreshToken"} {
		if strings.Contains(stdout, secret) {
			t.Fatalf("output leaked credentials: %s", secret)
		}
	}
	want.AuthInfos[0].Token = "scoped-access"
	got, err := session.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("switch must change only the original identity's access token")
	}
	current, err := captureAccountOutput(t, func() error { return runAccountCurrent(accountCurrentCmd, nil) })
	if err != nil || !strings.Contains(current, `"queue": "research"`) {
		t.Fatalf("subsequent current: error=%v output=%s", err, current)
	}
	afterRead, err := session.LoadState()
	if err != nil || !reflect.DeepEqual(afterRead, want) {
		t.Fatal("reading the current project must preserve local state")
	}
	if !reflect.DeepEqual(methods, []string{"POST " + api.AuthSwitchPath, "GET " + api.AuthCheckPath}) {
		t.Errorf("requests = %v, want switch followed by check without extra requests", methods)
	}
}

func TestAccountUseFailurePreservesCredentials(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", 403, `{"code":1001,"msg":"not a member"}`},
		{"business failure", 200, `{"code":1001,"msg":"Queue not found"}`},
		{"null response", 200, `{"code":0,"data":null}`},
		{"missing token", 200, `{"code":0,"data":{"context":{"queue":"research"}}}`},
		{"blank token", 200, `{"code":0,"data":{"accessToken":" ","context":{"queue":"research"}}}`},
		{"wrong queue", 200, `{"code":0,"data":{"accessToken":"unexpected-access","context":{"queue":"other"}}}`},
		{"missing context", 200, `{"code":0,"data":{"accessToken":"unexpected-access"}}`},
		{"malformed response", 200, `{`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				io.WriteString(w, test.body)
			}))
			t.Cleanup(server.Close)
			want := setupAccountSession(t, server.URL)
			stdout, err := captureAccountOutput(t, func() error { return runAccountUse(accountUseCmd, []string{"research"}) })
			var cliErr *clierror.Error
			if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryAPI || stdout != "" {
				t.Fatalf("failed switch: error=%v stdout=%q, want API error and empty stdout", err, stdout)
			}
			got, err := session.LoadState()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("failed switch changed saved credentials or local state")
			}
		})
	}
}

func TestAccountUsePreservesAnIdentityChangedDuringRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st, err := session.LoadState()
		if err != nil {
			t.Error(err)
			return
		}
		st.ActiveContext.Username = "bob"
		st.ActiveContext.Method = "ldap"
		if err := session.SaveState(st); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":0,"data":{"accessToken":"scoped-access","context":{"queue":"research"}}}`)
	}))
	t.Cleanup(server.Close)
	want := setupAccountSession(t, server.URL)
	_, err := captureAccountOutput(t, func() error { return runAccountUse(accountUseCmd, []string{"research"}) })
	if err != nil {
		t.Fatal(err)
	}
	want.ActiveContext.Username = "bob"
	want.ActiveContext.Method = "ldap"
	want.AuthInfos[0].Token = "scoped-access"
	got, err := session.LoadState()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("switch replaced the wrong identity or reset the newer active identity")
	}
}

func TestAccountUseRejectsIdentityRemovedDuringRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st, err := session.LoadState()
		if err != nil {
			t.Error(err)
			return
		}
		st.AuthInfos = nil
		if err := session.SaveState(st); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":0,"data":{"accessToken":"scoped-access","context":{"queue":"research"}}}`)
	}))
	t.Cleanup(server.Close)
	setupAccountSession(t, server.URL)
	stdout, err := captureAccountOutput(t, func() error { return runAccountUse(accountUseCmd, []string{"research"}) })
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Code != errorcodes.ErrConfigWriteFailed || stdout != "" {
		t.Fatalf("persistence failure: error=%v stdout=%q, want config error and no success output", err, stdout)
	}
}
