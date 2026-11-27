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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestNodeDetailSnapshots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := map[string]any{
			"name": "cpu-node", "role": "worker", "isReady": "Ready",
			"arch": "amd64", "address": "192.0.2.10", "os": "linux", "osVersion": "Ubuntu 22.04",
			"kubeletVersion": "v1.34.0", "containerRuntimeVersion": "containerd://2.0.0",
			"allocatable": map[string]string{"cpu": "16", "memory": "64Gi"},
			"used":        map[string]string{"cpu": "2", "memory": "8Gi"},
		}
		switch r.URL.Path {
		case "/api/v1/nodes/with-model":
			node["cpuModel"] = "Intel Xeon Silver 4210"
		case "/api/v1/nodes/empty-model":
			node["cpuModel"] = ""
		case "/api/v1/nodes/legacy":
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": node}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-model-text", Args: []string{"node", "get", "with-model", "--no-interactive"}},
		{ID: "02-model-json", Args: []string{"node", "get", "with-model", "--json", "--no-interactive"}},
		{ID: "03-legacy-text", Args: []string{"node", "get", "legacy", "--no-interactive"}},
		{ID: "04-legacy-json", Args: []string{"node", "get", "legacy", "--json", "--no-interactive"}},
		{ID: "05-empty-text", Args: []string{"node", "get", "empty-model", "--no-interactive"}},
		{ID: "06-empty-json", Args: []string{"node", "get", "empty-model", "--json", "--no-interactive"}},
	}
	for _, language := range []string{"en", "zh-CN"} {
		t.Run(language, func(t *testing.T) {
			env := append(snaptest.EnvMinimal(t.TempDir(), language),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			results := make([]*snaptest.Result, len(cases))
			for i, test := range cases {
				result, err := snaptest.Run(bin, env, test.Args)
				if err != nil {
					t.Fatalf("%s: %v", test.ID, err)
				}
				if result.ExitCode != 0 || result.Stderr != "" {
					t.Fatalf("%s: exit=%d stderr=%s", test.ID, result.ExitCode, result.Stderr)
				}
				results[i] = result
			}
			path := snaptest.GoldenFileT(t, "read", "node_detail", language)
			update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
			if err := snaptest.MatchOrUpdateGolden(path, language, cases, results, update); err != nil {
				t.Fatal(err)
			}
		})
	}
}
