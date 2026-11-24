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

package read_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestUserFieldsSnapshots(t *testing.T) {
	for _, language := range []string{"en", "zh-CN"} {
		t.Run(language, func(t *testing.T) { runUserFieldsSnapshots(t, language) })
	}
}

func runUserFieldsSnapshots(t *testing.T, language string) {
	t.Helper()
	cases := []struct {
		snapshot       snaptest.Case
		status         int
		empty          bool
		noSession      bool
		configRequests int32
		exit           int
	}{
		{snapshot: snaptest.Case{ID: "01-help", Args: []string{"user", "fields", "--help"}}},
		{snapshot: snaptest.Case{ID: "02-fields-json", Args: []string{"user", "fields", "--json"}}, configRequests: 1},
		{snapshot: snaptest.Case{ID: "03-fields-text", Args: []string{"user", "fields", "--no-interactive"}}, configRequests: 1},
		{snapshot: snaptest.Case{ID: "04-disabled-fields", Args: []string{"user", "fields", "--json"}}, empty: true, configRequests: 1},
		{snapshot: snaptest.Case{ID: "05-no-session", Args: []string{"user", "fields", "--json"}}, noSession: true, exit: 2},
		{snapshot: snaptest.Case{ID: "06-old-backend", Args: []string{"user", "fields", "--json"}}, status: 404, configRequests: 1, exit: 4},
		{snapshot: snaptest.Case{ID: "07-admin-table", Args: []string{"admin", "user", "ls", "--no-interactive"}}, configRequests: 1},
		{snapshot: snaptest.Case{ID: "08-admin-json", Args: []string{"admin", "user", "ls", "--json"}}},
		{snapshot: snaptest.Case{ID: "09-admin-fallback", Args: []string{"admin", "user", "ls", "--no-interactive"}}, status: 404, configRequests: 1},
		{snapshot: snaptest.Case{ID: "10-admin-config-error", Args: []string{"admin", "user", "ls", "--json=false", "--no-interactive"}}, status: 500, configRequests: 1, exit: 4},
		{snapshot: snaptest.Case{ID: "11-extra-argument", Args: []string{"user", "fields", "extra", "--json"}}, exit: 2},
	}
	bin := snaptest.CraterExecutable(t)
	baseEnv := snaptest.EnvMinimal(t.TempDir(), language)
	snapshots := make([]snaptest.Case, len(cases))
	results := make([]*snaptest.Result, len(cases))
	for index, test := range cases {
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method %s", r.Method)
			}
			switch r.URL.Path {
			case "/api/v1/system-config/user-fields":
				requests.Add(1)
				if test.status != 0 {
					w.WriteHeader(test.status)
					fmt.Fprint(w, `{"code":1,"msg":"field configuration unavailable"}`)
				} else if test.empty {
					fmt.Fprint(w, `{"code":0,"data":{}}`)
				} else {
					fmt.Fprint(w, `{"code":0,"data":{"group":{"labels":{"en":"Graduation year","zh":"毕业年份"},"showToAll":false},"phone":{"labels":{},"showToAll":false}}}`)
				}
			case "/api/v1/admin/users":
				fmt.Fprint(w, `{"code":0,"data":[{"id":1,"name":"alice","nickname":"Alice","role":2,"status":2,"newField":{"kept":true},"attributes":{"group":"2029","phone":null}}]}`)
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		env := append(append([]string{}, baseEnv...), "CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
		if test.noSession {
			env = append(env, "CRATER_TEST_SANDBOX_SESSION_EMPTY=1")
		}
		result, err := snaptest.Run(bin, env, test.snapshot.Args)
		server.Close()
		if err != nil {
			t.Fatalf("%s: %v", test.snapshot.ID, err)
		}
		if result.ExitCode != test.exit {
			t.Fatalf("%s: exit %d, want %d: %s", test.snapshot.ID, result.ExitCode, test.exit, result.Stderr)
		}
		if got := requests.Load(); got != test.configRequests {
			t.Fatalf("%s: configuration requests %d, want %d", test.snapshot.ID, got, test.configRequests)
		}
		if test.snapshot.ID == "08-admin-json" {
			if !json.Valid([]byte(result.Stdout)) || !strings.Contains(result.Stdout, `"kept": true`) {
				t.Fatalf("JSON lost server fields: %s", result.Stdout)
			}
		}
		result.Stdout = strings.ReplaceAll(result.Stdout, server.URL, "https://example.invalid")
		result.Stderr = strings.ReplaceAll(result.Stderr, server.URL, "https://example.invalid")
		snapshots[index], results[index] = test.snapshot, result
	}
	path := snaptest.GoldenFileT(t, "read", "user_fields", language)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, language, snapshots, results, update); err != nil {
		t.Fatal(err)
	}
}
