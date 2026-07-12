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

package reconciler

import (
	"strings"
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

func TestClassifyDownloadFailure(t *testing.T) {
	tests := []struct {
		name string
		logs string
		want string
	}{
		{name: "gated", logs: "Access to model is restricted", want: "gated"},
		{name: "authentication", logs: "HTTP 401 Unauthorized", want: "access denied"},
		{name: "missing revision", logs: "revision not found (404)", want: "repository or revision not found"},
		{name: "validated missing revision", logs: "[ERROR] revision_not_found: 'main'", want: "requested revision does not exist"},
		{name: "storage", logs: "write failed: no space left on device", want: "no space left"},
		{name: "network", logs: "connection reset by peer", want: "network error"},
		{name: "fallback", logs: "trace\ncustom downloader error\n", want: "custom downloader error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDownloadFailure(tt.logs); !strings.Contains(got, tt.want) {
				t.Fatalf("classifyDownloadFailure() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestStoredLogTailAndDescription(t *testing.T) {
	logs := strings.Repeat("old line\n", 20) + "[DESC] Useful repository summary.\n"
	truncated := truncateLogTail(logs, 64)

	if len(truncated) > 64 {
		t.Fatalf("stored log tail has %d bytes, want at most 64", len(truncated))
	}
	if got := parseDescriptionFromLogs(truncated); got != "Useful repository summary." {
		t.Fatalf("parseDescriptionFromLogs() = %q", got)
	}
}

func TestParseRepositoryMetadata(t *testing.T) {
	metadata := parseRepositoryMetadata(`noise
[META] {"downloads":4235273,"likes":333,"updated_at":"2025-07-26T16:12:41Z","tags":["text-generation"]}
`)

	if metadata.Downloads != 4235273 || metadata.Likes != 333 || metadata.UpdatedAt != "2025-07-26T16:12:41Z" {
		t.Fatalf("unexpected repository metadata: %#v", metadata)
	}
	if strings.Join(metadata.Tags, ",") != "text-generation" {
		t.Fatalf("unexpected repository tags: %#v", metadata.Tags)
	}
}

func TestDatasetExtraForDownloadPreservesTags(t *testing.T) {
	url := "https://modelscope.cn/models/Qwen/Qwen3-32B"
	extra := datasetExtraForDownload(model.ExtraContent{Tags: []string{"llm"}}, &model.ModelDownload{
		Source: model.ModelSourceModelScope,
	}, url, []string{"text-generation"})

	if strings.Join(extra.Tags, ",") != "llm,modelscope,text-generation" {
		t.Fatalf("unexpected tags: %#v", extra.Tags)
	}
	if extra.WebURL == nil || *extra.WebURL != url || extra.Editable {
		t.Fatalf("unexpected dataset extra: %#v", extra)
	}
}
