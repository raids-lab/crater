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

package version

import "testing"

func TestEvaluateCompatibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cli        int
		minBackend int
		backend    int
		minCLI     int
		want       CompatibilityStatus
	}{
		{name: "compatible", cli: 2, minBackend: 1, backend: 2, minCLI: 1, want: CompatibilityCompatible},
		{name: "cli too old", cli: 1, minBackend: 1, backend: 2, minCLI: 2, want: CompatibilityCLITooOld},
		{name: "backend too old", cli: 2, minBackend: 2, backend: 1, minCLI: 1, want: CompatibilityBackendTooOld},
		{name: "explicit zero backend is pre-contract", cli: 1, minBackend: 1, backend: 0, minCLI: 0, want: CompatibilityBackendTooOld},
		{name: "both too old", cli: 1, minBackend: 2, backend: 1, minCLI: 2, want: CompatibilityBothTooOld},
		{name: "negative response is invalid", cli: 1, minBackend: 1, backend: -1, minCLI: 1, want: CompatibilityUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := evaluateCompatibility(test.cli, test.minBackend, test.backend, test.minCLI); got != test.want {
				t.Fatalf("evaluateCompatibility(%d, %d, %d, %d) = %q, want %q", test.cli, test.minBackend, test.backend, test.minCLI, got, test.want)
			}
		})
	}
}

func TestUserAgentUsesProductVersion(t *testing.T) {
	original := ProductVersion
	ProductVersion = "v0.4.0"
	t.Cleanup(func() { ProductVersion = original })

	if got := UserAgent(); got != "crater-cli/0.4.0" {
		t.Fatalf("UserAgent() = %q, want crater-cli/0.4.0", got)
	}
}

func TestCurrentBuildInfoUsesInjectedFields(t *testing.T) {
	originalProductVersion := ProductVersion
	originalCommitSHA := CommitSHA
	originalBuildType := BuildType
	originalBuildTime := BuildTime
	ProductVersion = "v1.2.3"
	CommitSHA = "0123456789abcdef"
	BuildType = "release"
	BuildTime = "2026-09-07T08:30:00Z"
	t.Cleanup(func() {
		ProductVersion = originalProductVersion
		CommitSHA = originalCommitSHA
		BuildType = originalBuildType
		BuildTime = originalBuildTime
	})

	info := CurrentBuildInfo()
	if info.ProductVersion != "1.2.3" {
		t.Fatalf("ProductVersion = %q, want 1.2.3", info.ProductVersion)
	}
	if info.CommitSHA != "0123456789abcdef" {
		t.Fatalf("CommitSHA = %q, want 0123456789abcdef", info.CommitSHA)
	}
	if info.BuildType != "release" {
		t.Fatalf("BuildType = %q, want release", info.BuildType)
	}
	if info.BuildTime != "2026-09-07T08:30:00Z" {
		t.Fatalf("BuildTime = %q, want 2026-09-07T08:30:00Z", info.BuildTime)
	}
	if info.APIVersion != APIVersion {
		t.Fatalf("APIVersion = %d, want %d", info.APIVersion, APIVersion)
	}
	if info.MinSupportedBackendAPIVersion != MinSupportedBackendAPIVersion {
		t.Fatalf(
			"MinSupportedBackendAPIVersion = %d, want %d",
			info.MinSupportedBackendAPIVersion,
			MinSupportedBackendAPIVersion,
		)
	}
	if info.GoVersion == "" || info.OS == "" || info.Arch == "" {
		t.Fatalf("runtime build fields must not be empty: %#v", info)
	}
}

func TestEffectiveBuildFieldsUseStableDefaults(t *testing.T) {
	originalCommitSHA := CommitSHA
	originalBuildType := BuildType
	originalBuildTime := BuildTime
	CommitSHA = unknownBuildValue
	BuildType = ""
	BuildTime = unknownBuildValue
	t.Cleanup(func() {
		CommitSHA = originalCommitSHA
		BuildType = originalBuildType
		BuildTime = originalBuildTime
	})

	if got := EffectiveBuildType(); got != defaultDevelopmentBuildType {
		t.Fatalf("EffectiveBuildType() = %q, want %q", got, defaultDevelopmentBuildType)
	}
	if got := EffectiveCommitSHA(); got == "" {
		t.Fatal("EffectiveCommitSHA() returned an empty value")
	}
	if got := EffectiveBuildTime(); got != unknownBuildValue {
		t.Fatalf("EffectiveBuildTime() = %q, want %q", got, unknownBuildValue)
	}
}

func TestShortCommitSHA(t *testing.T) {
	tests := []struct {
		name      string
		commitSHA string
		want      string
	}{
		{name: "full SHA", commitSHA: "0123456789abcdef", want: "0123456"},
		{name: "already short", commitSHA: "abc1234", want: "abc1234"},
		{name: "unknown", commitSHA: unknownBuildValue, want: unknownBuildValue},
		{name: "empty", commitSHA: "  ", want: unknownBuildValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortCommitSHA(tt.commitSHA); got != tt.want {
				t.Fatalf("ShortCommitSHA(%q) = %q, want %q", tt.commitSHA, got, tt.want)
			}
		})
	}
}
