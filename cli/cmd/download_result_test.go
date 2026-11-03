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
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

type downloadResultTestClient struct {
	api.ModelDownloadClient
	downloads []*api.ModelDownloadResp
	calls     int
}

func (client *downloadResultTestClient) GetDownload(_ uint) (*api.ModelDownloadResp, error) {
	index := min(client.calls, len(client.downloads)-1)
	client.calls++
	return client.downloads[index], nil
}

func TestDownloadReuseWarning(t *testing.T) {
	for _, language := range []string{"en", "zh-CN"} {
		t.Run(language, func(t *testing.T) {
			previous := i18n.GetCurrentLanguage()
			i18n.SetLanguage(language)
			t.Cleanup(func() { i18n.SetLanguage(previous) })
			for _, testCase := range []struct {
				name, source, requested, actual string
				wantWarning, wantRevision       bool
			}{
				{"matched", "modelscope", "v1", "v1", false, false},
				{"revision only", "modelscope", "v2", "v1", true, true},
				{"source only", "huggingface", "v1", "v1", true, false},
				{"both", "huggingface", "v2", "v1", true, true},
				{"source alias", "ms", "v1", "v1", false, false},
				{"requested default", "modelscope", "", "v1", true, true},
				{"actual unspecified", "modelscope", "v2", "", true, true},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					download := &api.ModelDownloadResp{
						Source: "modelscope", Revision: testCase.actual, Status: "Ready", Path: "public/Models/owner/model",
					}
					result := downloadCreateResult{source: testCase.source, revision: testCase.requested}
					warning := result.reuseWarning(download)
					if (warning != "") != testCase.wantWarning {
						t.Fatalf("warning = %q, want warning %v", warning, testCase.wantWarning)
					}
					if !testCase.wantWarning {
						return
					}
					for _, value := range []string{download.Source, download.Status, download.Path,
						downloadRevisionLabel(testCase.requested), downloadRevisionLabel(testCase.actual)} {
						if !strings.Contains(warning, value) {
							t.Fatalf("warning lacks %q: %s", value, warning)
						}
					}
					if strings.Contains(warning, i18n.T("download_requested_revision_not_downloaded")) != testCase.wantRevision {
						t.Fatalf("incorrect revision warning: %s", warning)
					}
				})
			}
		})
	}
}

func TestDownloadCreateAndWaitOutputRetainsRequestedIdentity(t *testing.T) {
	previousLanguage, previousJSON := i18n.GetCurrentLanguage(), outputJSON
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage); outputJSON = previousJSON })
	for _, language := range []string{"en", "zh-CN"} {
		for _, jsonOutput := range []bool{false, true} {
			for _, wait := range []bool{false, true} {
				for _, status := range []string{"Ready", "Failed", "Paused"} {
					t.Run(fmt.Sprintf("%s/json=%v/wait=%v/%s", language, jsonOutput, wait, status), func(t *testing.T) {
						i18n.SetLanguage(language)
						outputJSON = jsonOutput
						actual := &api.ModelDownloadResp{
							ID: 123, Source: "modelscope", Revision: "v1", Status: status, Path: "public/Models/owner/model",
						}
						result := downloadCreateResult{source: "huggingface", revision: "v2", message: "backend reuse message"}
						output := captureDownloadResultStdout(t, func() error {
							if !wait {
								return result.write(actual)
							}
							command := &cobra.Command{}
							addDownloadCreateFlags(command, false)
							if err := command.Flags().Set("poll-interval", "1ms"); err != nil {
								t.Fatal(err)
							}
							pending := *actual
							pending.Status = "Downloading"
							client := &downloadResultTestClient{downloads: []*api.ModelDownloadResp{&pending, actual}}
							return waitForDownload(command, client, actual.ID, result)
						})
						for _, value := range []string{"huggingface", "modelscope", "v2", "v1", status, actual.Path} {
							if !strings.Contains(output, value) {
								t.Fatalf("output lacks %q: %s", value, output)
							}
						}
						if !jsonOutput {
							if !strings.Contains(output, i18n.T("download_requested_revision_not_downloaded")) {
								t.Fatalf("missing unmet revision warning: %s", output)
							}
							return
						}
						var envelope struct {
							Message string `json:"message"`
							Data    struct {
								Download api.ModelDownloadResp `json:"download"`
							} `json:"data"`
						}
						if err := json.Unmarshal([]byte(output), &envelope); err != nil {
							t.Fatalf("stdout is not pure JSON: %v\n%s", err, output)
						}
						if envelope.Data.Download.Source != "modelscope" || envelope.Data.Download.Revision != "v1" {
							t.Fatalf("actual identity was overwritten: %#v", envelope.Data.Download)
						}
						if !strings.Contains(envelope.Message, result.message) ||
							!strings.Contains(envelope.Message, i18n.T("download_requested_revision_not_downloaded")) {
							t.Fatalf("missing JSON messages: %s", envelope.Message)
						}
					})
				}
			}
		}
	}
}

func captureDownloadResultStdout(t *testing.T, run func() error) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "download-output")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	previous := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = previous }()
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
