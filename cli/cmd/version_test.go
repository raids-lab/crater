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
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	internalversion "github.com/raids-lab/crater/cli/internal/version"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

func TestWriteVersionResult(t *testing.T) {
	info := internalversion.BuildInfo{
		ProductVersion:                "1.2.3",
		CommitSHA:                     "0123456789abcdef",
		BuildType:                     "release",
		BuildTime:                     "2026-09-07T08:30:00Z",
		GoVersion:                     "go1.25.4",
		OS:                            "linux",
		Arch:                          "amd64",
		APIVersion:                    1,
		MinSupportedBackendAPIVersion: 1,
	}

	t.Run("human readable English", func(t *testing.T) {
		previousLanguage := i18n.GetCurrentLanguage()
		i18n.SetLanguage("en")
		t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

		var stdout bytes.Buffer
		if err := writeVersionResult(&stdout, false, info); err != nil {
			t.Fatal(err)
		}
		want := strings.Join([]string{
			"Crater CLI:",
			" Version:                     1.2.3",
			" API version:                 1",
			" Minimum backend API version: 1",
			" Go version:                  go1.25.4",
			" Git commit:                  0123456789abcdef",
			" Built:                       2026-09-07T08:30:00Z",
			" OS/Arch:                     linux/amd64",
			" Build type:                  release",
			"",
		}, "\n")
		if stdout.String() != want {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	})

	t.Run("human readable Chinese", func(t *testing.T) {
		previousLanguage := i18n.GetCurrentLanguage()
		i18n.SetLanguage("zh-CN")
		t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

		var stdout bytes.Buffer
		if err := writeVersionResult(&stdout, false, info); err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{
			"Crater CLI:",
			"版本:",
			"Git 提交:",
			"操作系统/架构:",
			"API 版本:",
		} {
			if !strings.Contains(stdout.String(), fragment) {
				t.Fatalf("stdout %q does not contain %q", stdout.String(), fragment)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		var stdout bytes.Buffer
		if err := writeVersionResult(&stdout, true, info); err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Status string `json:"status"`
			Data   struct {
				Version internalversion.BuildInfo `json:"version"`
			} `json:"data"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if envelope.Status != "OK" {
			t.Fatalf("status = %q, want OK", envelope.Status)
		}
		if envelope.Data.Version != info {
			t.Fatalf("version = %#v, want %#v", envelope.Data.Version, info)
		}
	})
}

func TestWriteShortVersionResult(t *testing.T) {
	info := internalversion.BuildInfo{
		ProductVersion: "1.2.3+dev.5.g0123456",
		CommitSHA:      "0123456789abcdef",
	}

	tests := []struct {
		name     string
		language string
		want     string
	}{
		{
			name:     "English",
			language: "en",
			want:     "Crater CLI version 1.2.3+dev.5.g0123456, build 0123456\n",
		},
		{
			name:     "Chinese",
			language: "zh-CN",
			want:     "Crater CLI 版本 1.2.3+dev.5.g0123456，构建 0123456\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousLanguage := i18n.GetCurrentLanguage()
			i18n.SetLanguage(tt.language)
			t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

			var stdout bytes.Buffer
			if err := writeShortVersionResult(&stdout, info); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestValidateRootArgsRejectsVersionJSONCombination(t *testing.T) {
	previousRootVersion := rootVersion
	previousOutputJSON := outputJSON
	rootVersion = true
	outputJSON = true
	t.Cleanup(func() {
		rootVersion = previousRootVersion
		outputJSON = previousOutputJSON
	})

	err := validateRootArgs(&cobra.Command{Use: "crater"}, nil)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("validateRootArgs() error = %T, want *clierror.Error", err)
	}
	if cliErr.Category != errorcodes.CategoryUsage || cliErr.Code != errorcodes.ErrInvalidFlagValue {
		t.Fatalf("validateRootArgs() error = %#v, want usage invalid flag", cliErr)
	}
	if !strings.Contains(cliErr.Message, "crater version --json") {
		t.Fatalf("validateRootArgs() message = %q, want structured version hint", cliErr.Message)
	}
}

func TestValidateRootArgsRejectsVersionArguments(t *testing.T) {
	previousRootVersion := rootVersion
	previousOutputJSON := outputJSON
	rootVersion = true
	outputJSON = false
	t.Cleanup(func() {
		rootVersion = previousRootVersion
		outputJSON = previousOutputJSON
	})

	err := validateRootArgs(&cobra.Command{Use: "crater"}, []string{"extra"})
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("validateRootArgs() error = %T, want *clierror.Error", err)
	}
	if cliErr.Category != errorcodes.CategoryUsage || cliErr.Code != errorcodes.ErrInvalidFlagValue {
		t.Fatalf("validateRootArgs() error = %#v, want usage invalid flag", cliErr)
	}
	if !strings.Contains(cliErr.Message, "too many arguments") {
		t.Fatalf("validateRootArgs() message = %q, want argument count error", cliErr.Message)
	}
}
