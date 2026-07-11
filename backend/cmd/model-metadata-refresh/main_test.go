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

package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateTextPreservesUTF8(t *testing.T) {
	text := strings.Repeat("模型介绍", 100)
	truncated := truncateText(text, 101)
	if !utf8.ValidString(truncated) {
		t.Fatalf("truncated text is not valid UTF-8: %q", truncated)
	}
	if len(truncated) > 101 {
		t.Fatalf("truncated text exceeds byte limit: %d", len(truncated))
	}
}

func TestSourceDescriptionStripsMarkupAndPreservesUTF8(t *testing.T) {
	readme := "<h1>模型介绍</h1> " + strings.Repeat("这是一个中文模型。", 100)
	description := sourceDescription("", readme)
	if !utf8.ValidString(description) || strings.Contains(description, "<h1>") {
		t.Fatalf("unexpected description: %q", description)
	}
}

func TestCleanReadmeRemovesRawHTMLAndFrontMatter(t *testing.T) {
	readme := "---\nlicense: test\n---\n<div><strong>模型介绍</strong></div><script>alert(1)</script>\n## 用法"
	cleaned := cleanReadme(readme)
	if strings.Contains(cleaned, "license: test") || strings.Contains(cleaned, "<div>") ||
		strings.Contains(cleaned, "alert(1)") {
		t.Fatalf("unsafe or presentation-only markup remains: %q", cleaned)
	}
	if !strings.Contains(cleaned, "模型介绍") || !strings.Contains(cleaned, "## 用法") {
		t.Fatalf("meaningful README content was removed: %q", cleaned)
	}
}
