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

package account_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestAccountContextSnapshots(t *testing.T) {
	for _, language := range []string{"en", "zh-CN"} {
		t.Run(language, func(t *testing.T) { runAccountContextSnapshots(t, language) })
	}
}

func runAccountContextSnapshots(t *testing.T, language string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/check":
			io.WriteString(w, `{"code":0,"data":{"context":{"queue":"public","roleQueue":2,"rolePlatform":2,"accessQueue":3,"accessPublic":3,"space":"alice"}}}`)
		case "/api/v1/auth/switch":
			var body struct {
				Queue string `json:"queue"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.Queue == "forbidden" {
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, `{"code":1001,"msg":"not a member"}`)
				return
			}
			if body.Queue == "invalid" {
				io.WriteString(w, `{"code":0,"data":{"context":{"queue":"invalid"}}}`)
				return
			}
			io.WriteString(w, `{"code":0,"data":{"accessToken":"fixture-access","refreshToken":"fixture-refresh","context":{"queue":"research","roleQueue":2,"rolePlatform":2,"accessQueue":3,"accessPublic":3,"space":"alice"}}}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	malformedServers := make(map[int]*httptest.Server)
	for _, status := range []int{401, 403, 500} {
		malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			io.WriteString(w, `{"accessToken":"fixture-access",`)
		}))
		t.Cleanup(malformed.Close)
		malformedServers[status] = malformed
	}
	malformedStatus := map[string]int{
		"21-current-malformed-401": 401,
		"22-use-malformed-401":     401,
		"23-current-malformed-403": 403,
		"24-use-malformed-403":     403,
		"25-current-malformed-500": 500,
		"26-use-malformed-500":     500,
		"27-use-malformed-text":    403,
	}
	baseEnv := snaptest.EnvMinimal(t.TempDir(), language)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-account-help", Args: []string{"account", "--help"}},
		{ID: "02-current-help", Args: []string{"account", "current", "--help"}},
		{ID: "03-use-help", Args: []string{"account", "use", "--help"}},
		{ID: "04-use-missing-json", Args: []string{"account", "use", "--json"}},
		{ID: "05-use-blank-json", Args: []string{"account", "use", " ", "--json"}},
		{ID: "06-use-extra-json", Args: []string{"account", "use", "research", "extra", "--json"}},
		{ID: "07-current-extra-json", Args: []string{"account", "current", "extra", "--json"}},
		{ID: "08-typo-json", Args: []string{"account", "invalid-command", "--json"}},
		{ID: "09-typo-text", Args: []string{"account", "invalid-command", "--no-interactive"}},
		{ID: "10-current-no-active", Args: []string{"account", "current", "--json"}},
		{ID: "11-use-no-active", Args: []string{"account", "use", "research", "--json"}},
		{ID: "12-current-timeout", Args: []string{"account", "current", "--json"}},
		{ID: "13-use-timeout", Args: []string{"account", "use", "research", "--no-interactive"}},
		{ID: "14-current-404", Args: []string{"account", "current", "--json"}},
		{ID: "15-use-forbidden", Args: []string{"account", "use", "forbidden", "--json"}},
		{ID: "16-current-text", Args: []string{"account", "current", "--no-interactive"}},
		{ID: "17-current-json", Args: []string{"account", "current", "--json"}},
		{ID: "18-use-text", Args: []string{"account", "use", "research", "--no-interactive"}},
		{ID: "19-use-json", Args: []string{"account", "use", "research", "--json"}},
		{ID: "20-use-invalid-response", Args: []string{"account", "use", "invalid", "--json"}},
		{ID: "21-current-malformed-401", Args: []string{"account", "current", "--json"}},
		{ID: "22-use-malformed-401", Args: []string{"account", "use", "research", "--json"}},
		{ID: "23-current-malformed-403", Args: []string{"account", "current", "--json"}},
		{ID: "24-use-malformed-403", Args: []string{"account", "use", "research", "--json"}},
		{ID: "25-current-malformed-500", Args: []string{"account", "current", "--json"}},
		{ID: "26-use-malformed-500", Args: []string{"account", "use", "research", "--json"}},
		{ID: "27-use-malformed-text", Args: []string{"account", "use", "research", "--no-interactive"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for index, test := range cases {
		env := append([]string{}, baseEnv...)
		platform := server.URL
		if status := malformedStatus[test.ID]; status != 0 {
			platform = malformedServers[status].URL
		}
		switch test.ID {
		case "10-current-no-active", "11-use-no-active":
			env = append(env, "CRATER_TEST_SANDBOX_SESSION_EMPTY=1", "CRATER_TEST_SANDBOX_HTTP=error404")
		case "12-current-timeout", "13-use-timeout":
			env = append(env, "CRATER_TEST_SANDBOX_HTTP=timeout")
		case "14-current-404":
			env = append(env, "CRATER_TEST_SANDBOX_HTTP=error404")
		default:
			env = append(env, "CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+platform)
		}
		result, err := snaptest.Run(bin, env, test.Args)
		if err != nil {
			t.Fatalf("%s: %v", test.ID, err)
		}
		result.Stdout = strings.ReplaceAll(result.Stdout, platform, "https://example.invalid")
		result.Stderr = strings.ReplaceAll(result.Stderr, platform, "https://example.invalid")
		if strings.Contains(result.Stdout+result.Stderr, "fixture-access") || strings.Contains(result.Stdout+result.Stderr, "fixture-refresh") {
			t.Fatalf("%s: credentials leaked to output", test.ID)
		}
		results[index] = result
	}
	path := snaptest.GoldenFileT(t, "account", "account_context", language)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, language, cases, results, update); err != nil {
		t.Fatal(err)
	}
}
