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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func testResponse(status int, contentType string, body []byte) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

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

func TestGetJSONFromEndpointsFallsBack(t *testing.T) {
	payloadBytes, err := json.Marshal(map[string]string{"name": "model"})
	if err != nil {
		t.Fatal(err)
	}
	client := testHTTPClient(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "unavailable.example" {
			return testResponse(http.StatusNotFound, "text/plain", []byte("unavailable")), nil
		}
		return testResponse(http.StatusOK, "application/json", payloadBytes), nil
	})
	var payload struct {
		Name string `json:"name"`
	}
	selected, err := getJSONFromEndpoints(
		client,
		[]string{"https://unavailable.example", "https://working.example"},
		"/api/model", &payload,
	)
	if err != nil {
		t.Fatalf("getJSONFromEndpoints() error = %v", err)
	}
	if selected != "https://working.example" || payload.Name != "model" {
		t.Fatalf("selected = %q, payload = %#v", selected, payload)
	}
}

func TestFetchLogoCachesOnlyBoundedImages(t *testing.T) {
	client := testHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return testResponse(http.StatusOK, "image/png", []byte("small-logo")), nil
	})
	data, contentType, err := fetchLogo(client, "https://source.example/logo.png", 32)
	if err != nil {
		t.Fatalf("fetchLogo() error = %v", err)
	}
	if string(data) != "small-logo" || contentType != "image/png" {
		t.Fatalf("data = %q, contentType = %q", data, contentType)
	}
	if _, _, err := fetchLogo(client, "https://source.example/logo.png", 4); err == nil {
		t.Fatal("fetchLogo() accepted an oversized image")
	}
}

func TestFindDatasetForDownloadSupportsOnePathlessHistoricalRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:metadata_pathless?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDatasetSource{}, &model.Dataset{}); err != nil {
		t.Fatal(err)
	}
	dataset := model.Dataset{Name: "owner/model", URL: "", Type: model.DataTypeModel}
	if err := db.Create(&dataset).Error; err != nil {
		t.Fatal(err)
	}
	download := &model.ModelDownload{Name: dataset.Name, Category: model.DownloadCategoryModel}
	found, err := findDatasetForDownload(db, download, "shared/Models/owner/model", true)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != dataset.ID {
		t.Fatalf("findDatasetForDownload() = %#v", found)
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

func TestCleanReadmeConvertsHTMLTableToMarkdown(t *testing.T) {
	readme := `<table><thead><tr><th>Benchmark</th><th>Score</th></tr></thead>` +
		`<tbody><tr><td>CountBench</td><td>89.4</td></tr>` +
		`<tr><td colspan="2">Thinking Mode</td></tr></tbody></table>`
	cleaned := cleanReadme(readme)
	for _, expected := range []string{
		"| Benchmark | Score |",
		"| --- | --- |",
		"| CountBench | 89.4 |",
		"| Thinking Mode |  |",
	} {
		if !strings.Contains(cleaned, expected) {
			t.Fatalf("converted table does not contain %q: %s", expected, cleaned)
		}
	}
	if strings.Contains(cleaned, "<table") {
		t.Fatalf("raw table markup remains: %s", cleaned)
	}
}
