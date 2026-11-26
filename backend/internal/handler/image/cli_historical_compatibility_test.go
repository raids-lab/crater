//go:build cli_compatibility

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

package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

// TestHistoricalCLIMinimumCompatibility uses binaries built from unmodified
// repository revisions that declare API V3 and V2, respectively.
func TestHistoricalCLIMinimumCompatibility(t *testing.T) {
	fixture := newReleasedCLIFixture(t)
	defer fixture.server.Close()
	run := func(t *testing.T, binary releasedCompatibilityCLI, args ...string) (int, []byte, []byte, []string) {
		t.Helper()
		if binary.path == "" {
			t.Fatal("V3_CLI_BIN and V2_CLI_BIN must point to binaries from historical revisions")
		}
		home := t.TempDir()
		command := binary.command(t, append(args, "--json", "--no-interactive")...)
		command.Env = []string{
			"PATH=" + os.Getenv("PATH"), "HOME=" + home, "XDG_CONFIG_HOME=" + home,
			"CRATER_LANG=en", "CRATER_TEST_SANDBOX=1", "CRATER_TEST_SANDBOX_HTTP=passthrough",
			"CRATER_TEST_SANDBOX_PLATFORM_URL=" + fixture.server.URL,
		}
		fixture.traceMu.Lock()
		fixture.trace = nil
		fixture.traceMu.Unlock()
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		exit := 0
		if err := command.Run(); err != nil {
			var failure *exec.ExitError
			if !errors.As(err, &failure) {
				t.Fatal(err)
			}
			exit = failure.ExitCode()
		}
		fixture.traceMu.Lock()
		trace := slices.Clone(fixture.trace)
		fixture.traceMu.Unlock()
		return exit, stdout.Bytes(), stderr.Bytes(), trace
	}

	for _, tc := range []struct {
		binary  releasedCompatibilityCLI
		version int
	}{
		{binary: releasedCompatibilityCLI{"v3", os.Getenv("V3_CLI_BIN")}, version: 3},
		{binary: releasedCompatibilityCLI{"v2", os.Getenv("V2_CLI_BIN")}, version: 2},
	} {
		t.Run(tc.binary.name+"/diagnostic", func(t *testing.T) {
			exit, stdout, stderr, trace := run(t, tc.binary, "version")
			var build struct {
				Data struct {
					Version struct {
						API int `json:"api_version"`
					} `json:"version"`
				} `json:"data"`
			}
			if err := json.Unmarshal(stdout, &build); err != nil || exit != 0 ||
				len(stderr) != 0 || len(trace) != 0 || build.Data.Version.API != tc.version {
				t.Fatalf("binary did not report API V%d: exit=%d out=%s err=%s", tc.version, exit, stdout, stderr)
			}

			exit, stdout, stderr, trace = run(t, tc.binary, "compatibility", "--platform", fixture.server.URL)
			if !slices.Equal(trace, []string{"/api/cli/compatibility"}) {
				t.Fatalf("unexpected diagnostic requests: %v", trace)
			}
			if tc.version == 3 {
				var success struct {
					Data struct {
						Compatibility struct {
							Status  string `json:"status"`
							Backend struct {
								API        int `json:"api_version"`
								MinimumCLI int `json:"min_supported_cli_api_version"`
							} `json:"backend"`
						} `json:"compatibility"`
					} `json:"data"`
				}
				if err := json.Unmarshal(stdout, &success); err != nil || exit != 0 || len(stderr) != 0 ||
					success.Data.Compatibility.Status != "compatible" ||
					success.Data.Compatibility.Backend.API != 8 || success.Data.Compatibility.Backend.MinimumCLI != 3 {
					t.Fatalf("V3 diagnostic: exit=%d out=%s err=%s", exit, stdout, stderr)
				}
				return
			}
			var failure struct {
				Code    string `json:"code"`
				Context struct {
					Status     string `json:"status"`
					CLI        int    `json:"cli_api_version"`
					Backend    int    `json:"backend_api_version"`
					MinimumCLI int    `json:"backend_min_supported_cli_api_version"`
				} `json:"context"`
			}
			if err := json.Unmarshal(stderr, &failure); err != nil || exit != 4 || len(stdout) != 0 ||
				failure.Code != "ERR_API_VERSION_MISMATCH" || failure.Context.Status != "cli_maybe_too_old" ||
				failure.Context.CLI != 2 || failure.Context.Backend != 8 || failure.Context.MinimumCLI != 3 {
				t.Fatalf("V2 diagnostic: exit=%d out=%s err=%s", exit, stdout, stderr)
			}
		})
	}

	v3 := releasedCompatibilityCLI{"v3", os.Getenv("V3_CLI_BIN")}
	for _, tc := range []struct {
		name string
		want int
	}{{"legacy template array", 201}, {"legacy empty template null", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want == 0 {
				if err := fixture.db.Where("1 = 1").Delete(&model.Jobtemplate{}).Error; err != nil {
					t.Fatal(err)
				}
			}
			exit, stdout, stderr, trace := run(t, v3, "template", "ls")
			if exit != 0 || len(stderr) != 0 || !slices.Equal(trace, []string{"/api/v1/jobtemplate/list"}) {
				t.Fatalf("legacy template command: exit=%d trace=%v out=%s err=%s", exit, trace, stdout, stderr)
			}
			assertReleasedCLIJSON(t, stdout, "templates", tc.want, tc.want == 0)
		})
	}
}
