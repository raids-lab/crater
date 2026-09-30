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

import (
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	APIVersion                       = 3
	MinSupportedBackendAPIVersion    = 3
	APIVersionHeader                 = "X-Crater-API-Version"
	defaultDevelopmentProductVersion = "dev"
	defaultDevelopmentBuildType      = "development"
	unknownBuildValue                = "unknown"
)

// Build fields may be overridden at build time with -ldflags.
var (
	ProductVersion = defaultDevelopmentProductVersion
	CommitSHA      = unknownBuildValue
	BuildType      = defaultDevelopmentBuildType
	BuildTime      = unknownBuildValue
)

// BuildInfo describes the local CLI binary and its API compatibility contract.
type BuildInfo struct {
	ProductVersion                string `json:"product_version"`
	CommitSHA                     string `json:"commit_sha"`
	BuildType                     string `json:"build_type"`
	BuildTime                     string `json:"build_time"`
	GoVersion                     string `json:"go_version"`
	OS                            string `json:"os"`
	Arch                          string `json:"arch"`
	APIVersion                    int    `json:"api_version"`
	MinSupportedBackendAPIVersion int    `json:"min_supported_backend_api_version"`
}

type CompatibilityStatus string

const (
	CompatibilityUnknown       CompatibilityStatus = "unknown"
	CompatibilityCompatible    CompatibilityStatus = "compatible"
	CompatibilityCLITooOld     CompatibilityStatus = "cli_maybe_too_old"
	CompatibilityBackendTooOld CompatibilityStatus = "backend_maybe_too_old"
	CompatibilityBothTooOld    CompatibilityStatus = "both_maybe_too_old"
)

func EffectiveProductVersion() string {
	if version := strings.TrimSpace(ProductVersion); version != "" && version != defaultDevelopmentProductVersion {
		return strings.TrimPrefix(version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		version := strings.TrimSpace(info.Main.Version)
		if version != "" && version != "(devel)" {
			return strings.TrimPrefix(version, "v")
		}
	}
	return defaultDevelopmentProductVersion
}

func EffectiveCommitSHA() string {
	if commitSHA := strings.TrimSpace(CommitSHA); commitSHA != "" && commitSHA != unknownBuildValue {
		return commitSHA
	}
	if revision := buildSetting("vcs.revision"); revision != "" {
		return revision
	}
	return unknownBuildValue
}

func EffectiveBuildType() string {
	if buildType := strings.TrimSpace(BuildType); buildType != "" {
		return buildType
	}
	return defaultDevelopmentBuildType
}

func EffectiveBuildTime() string {
	if buildTime := strings.TrimSpace(BuildTime); buildTime != "" && buildTime != unknownBuildValue {
		return buildTime
	}
	return unknownBuildValue
}

// ShortCommitSHA returns the seven-character commit used in concise version output.
func ShortCommitSHA(commitSHA string) string {
	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA == "" {
		return unknownBuildValue
	}
	if len(commitSHA) <= 7 {
		return commitSHA
	}
	return commitSHA[:7]
}

func CurrentBuildInfo() BuildInfo {
	return BuildInfo{
		ProductVersion:                EffectiveProductVersion(),
		CommitSHA:                     EffectiveCommitSHA(),
		BuildType:                     EffectiveBuildType(),
		BuildTime:                     EffectiveBuildTime(),
		GoVersion:                     runtime.Version(),
		OS:                            runtime.GOOS,
		Arch:                          runtime.GOARCH,
		APIVersion:                    APIVersion,
		MinSupportedBackendAPIVersion: MinSupportedBackendAPIVersion,
	}
}

func buildSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == key {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

func UserAgent() string {
	return "crater-cli/" + EffectiveProductVersion()
}

func EvaluateCompatibility(backendAPIVersion, minSupportedCLIAPIVersion int) CompatibilityStatus {
	return evaluateCompatibility(
		APIVersion,
		MinSupportedBackendAPIVersion,
		backendAPIVersion,
		minSupportedCLIAPIVersion,
	)
}

func evaluateCompatibility(
	cliAPIVersion,
	minSupportedBackendAPIVersion,
	backendAPIVersion,
	minSupportedCLIAPIVersion int,
) CompatibilityStatus {
	if cliAPIVersion <= 0 || minSupportedBackendAPIVersion <= 0 ||
		backendAPIVersion < 0 || minSupportedCLIAPIVersion < 0 {
		return CompatibilityUnknown
	}
	cliTooOld := cliAPIVersion < minSupportedCLIAPIVersion
	backendTooOld := backendAPIVersion < minSupportedBackendAPIVersion
	switch {
	case cliTooOld && backendTooOld:
		return CompatibilityBothTooOld
	case cliTooOld:
		return CompatibilityCLITooOld
	case backendTooOld:
		return CompatibilityBackendTooOld
	default:
		return CompatibilityCompatible
	}
}
