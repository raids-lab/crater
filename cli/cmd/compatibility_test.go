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
	"fmt"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	internalversion "github.com/raids-lab/crater/cli/internal/version"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestCompatibilityMismatchError(t *testing.T) {
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage("en") })

	tests := []struct {
		name           string
		status         internalversion.CompatibilityStatus
		wantConditions []string
		wantAction     string
	}{
		{
			name:           "CLI too old",
			status:         internalversion.CompatibilityCLITooOld,
			wantConditions: []string{"The CLI API version (1) is lower than the backend minimum supported CLI API version (2)."},
			wantAction:     "try upgrading the CLI",
		},
		{
			name:           "backend too old",
			status:         internalversion.CompatibilityBackendTooOld,
			wantConditions: []string{"The backend API version (1) is lower than the CLI minimum supported backend API version (2)."},
			wantAction:     "try downgrading the CLI",
		},
		{
			name:   "both too old",
			status: internalversion.CompatibilityBothTooOld,
			wantConditions: []string{
				"The CLI API version (1) is lower than the backend minimum supported CLI API version (2).",
				"The backend API version (1) is lower than the CLI minimum supported backend API version (2).",
			},
			wantAction: "Because both limits are unmet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compatibilityResult{
				PlatformURL: "https://example.invalid",
				Status:      tt.status,
				CLI: compatibilityCLI{
					ProductVersion:                "dev",
					APIVersion:                    1,
					MinSupportedBackendAPIVersion: 2,
				},
				Backend: compatibilityBackend{
					ProductVersion:            "v1.1.1",
					ShortCommitSHA:            "f42b0c2",
					BuildType:                 "release",
					BuildTime:                 "2026-07-26T08:30:00Z",
					APIVersion:                1,
					MinSupportedCLIAPIVersion: 2,
					VersionsReported:          true,
				},
			}
			err := compatibilityMismatchError(result)
			if err.Code != errorcodes.ErrAPIVersionMismatch {
				t.Fatalf("code = %q, want %q", err.Code, errorcodes.ErrAPIVersionMismatch)
			}
			if err.Category != errorcodes.CategoryAPI {
				t.Fatalf("category = %q, want %q", err.Category, errorcodes.CategoryAPI)
			}
			if got := err.Context["status"]; got != tt.status {
				t.Fatalf("status = %q, want %q", got, tt.status)
			}
			for _, want := range []string{
				tt.wantAction,
				"CLI and platform API versions are not compatible.",
				"Platform: https://example.invalid",
				"Situation:",
				"Versions:",
				"CLI product version: dev",
				"CLI API version: 1",
				"CLI minimum backend API version: 2",
				"Backend product version: v1.1.1",
				"Backend short commit SHA: f42b0c2",
				"Backend build type: release",
				"Backend build time: 2026-07-26T08:30:00Z",
				"Backend API version: 1",
				"Backend minimum CLI API version: 2",
				"Guidance:",
				"You can continue using the CLI",
				"first investigate its reported error and more common causes",
				"especially a development build, may also contain a bug",
				"Only after those causes have been ruled out",
				"platform administrator",
			} {
				if !strings.Contains(err.Message, want) {
					t.Errorf("message %q does not contain %q", err.Message, want)
				}
			}
			for _, want := range tt.wantConditions {
				if !strings.Contains(err.Message, "  - "+want) {
					t.Errorf("message %q does not contain condition %q", err.Message, want)
				}
			}
		})
	}
}

func TestCompatibilityAPIErrorMapsNotFoundToVersionMismatch(t *testing.T) {
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage("en") })

	err := compatibilityAPIError("https://example.invalid", &api.RequestError{
		HTTPStatus: 404,
		Msg:        "404 page not found",
	})
	cliErr, ok := err.(*clierror.Error)
	if !ok {
		t.Fatalf("error = %T, want *clierror.Error", err)
	}
	if cliErr.Code != errorcodes.ErrAPIVersionMismatch {
		t.Fatalf("code = %q, want %q", cliErr.Code, errorcodes.ErrAPIVersionMismatch)
	}
	if got := cliErr.Context["http_status"]; got != 404 {
		t.Fatalf("http_status = %#v, want 404", got)
	}
	if got := cliErr.Context["reason"]; got != "compatibility_endpoint_not_found" {
		t.Fatalf("reason = %#v, want compatibility_endpoint_not_found", got)
	}
	if got := cliErr.Context["backend_api_version"]; got != 0 {
		t.Fatalf("backend_api_version = %#v, want 0", got)
	}
	for _, key := range []string{
		"backend_product_version",
		"backend_short_commit_sha",
		"backend_build_type",
		"backend_build_time",
	} {
		if got := cliErr.Context[key]; got != "" {
			t.Fatalf("%s = %#v, want empty string", key, got)
		}
	}
	for _, want := range []string{
		"CLI and platform API versions are not compatible.",
		"Platform: https://example.invalid",
		"Situation:",
		"The platform returned HTTP 404 for the API compatibility endpoint.",
		"Versions:",
		fmt.Sprintf("CLI API version: %d", internalversion.APIVersion),
		fmt.Sprintf("CLI minimum backend API version: %d", internalversion.MinSupportedBackendAPIVersion),
		"Backend product version: unknown",
		"Backend short commit SHA: unknown",
		"Backend build type: unknown",
		"Backend build time: unknown",
		"Backend API version: unknown",
		"Backend minimum CLI API version: unknown",
		"Guidance:",
		"You can continue using the CLI",
		"first investigate its reported error and more common causes",
		"especially a development build, may also contain a bug",
		"Only after those causes have been ruled out",
		"try downgrading the CLI",
		"contact the platform administrator for support",
	} {
		if !strings.Contains(cliErr.Message, want) {
			t.Errorf("message %q does not contain %q", cliErr.Message, want)
		}
	}
	if strings.Contains(cliErr.Message, "upgrad") {
		t.Errorf("404 message should recommend downgrading, got %q", cliErr.Message)
	}
}

func TestNewCompatibilityResultRejectsInvalidVersionData(t *testing.T) {
	result := newCompatibilityResult("https://example.invalid", &api.CompatibilityInfo{
		AppVersion:     "v1.1.1",
		ShortCommitSHA: "f42b0c2",
		BuildType:      "release",
		BuildTime:      "2026-07-26T08:30:00Z",
	})
	if result.Status != internalversion.CompatibilityUnknown {
		t.Fatalf("status = %q, want %q", result.Status, internalversion.CompatibilityUnknown)
	}
	if result.Backend.ProductVersion != "v1.1.1" {
		t.Fatalf("backend product version = %q, want %q", result.Backend.ProductVersion, "v1.1.1")
	}
}

func TestNewCompatibilityResultTreatsReportedZeroAsBackendTooOld(t *testing.T) {
	result := newCompatibilityResult("https://example.invalid", &api.CompatibilityInfo{
		APIVersion:                0,
		MinSupportedCLIAPIVersion: 0,
		AppVersion:                "v0.0.0",
		ShortCommitSHA:            "0000000",
		BuildType:                 "development",
		BuildTime:                 "2026-07-26T08:30:00Z",
		VersionsReported:          true,
	})
	if result.Status != internalversion.CompatibilityBackendTooOld {
		t.Fatalf("status = %q, want %q", result.Status, internalversion.CompatibilityBackendTooOld)
	}
	err := compatibilityMismatchError(result)
	for _, want := range []string{
		fmt.Sprintf("The backend API version (0) is lower than the CLI minimum supported backend API version (%d).", internalversion.MinSupportedBackendAPIVersion),
		"Backend product version: v0.0.0",
		"Backend short commit SHA: 0000000",
		"Backend build type: development",
		"Backend build time: 2026-07-26T08:30:00Z",
		"Backend API version: 0",
		"Backend minimum CLI API version: 0",
		"try downgrading the CLI",
	} {
		if !strings.Contains(err.Message, want) {
			t.Errorf("message %q does not contain %q", err.Message, want)
		}
	}
}

func TestCompatibilityBuildTextTreatsMissingAndUnknownAsUnknown(t *testing.T) {
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage("en") })

	for _, value := range []string{"", " ", "unknown", "UNKNOWN"} {
		if got := compatibilityBuildText(value); got != "unknown" {
			t.Errorf("compatibilityBuildText(%q) = %q, want unknown", value, got)
		}
	}
	if got := compatibilityBuildText("v1.1.1"); got != "v1.1.1" {
		t.Errorf("compatibilityBuildText(v1.1.1) = %q, want v1.1.1", got)
	}
}
