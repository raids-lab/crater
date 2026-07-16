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
	allowedHosts := []string{"source.example"}
	data, contentType, err := fetchLogo(client, "https://source.example/logo.png", allowedHosts, 32)
	if err != nil {
		t.Fatalf("fetchLogo() error = %v", err)
	}
	if string(data) != "small-logo" || contentType != "image/png" {
		t.Fatalf("data = %q, contentType = %q", data, contentType)
	}
	if _, _, err := fetchLogo(client, "https://source.example/logo.png", allowedHosts, 4); err == nil {
		t.Fatal("fetchLogo() accepted an oversized image")
	}
}

func TestFetchLogoSniffsImageFromGenericContentType(t *testing.T) {
	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	client := testHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return testResponse(http.StatusOK, "application/octet-stream", pngHeader), nil
	})
	data, contentType, err := fetchLogo(
		client, "https://resouces.modelscope.cn/avatar/logo.webp",
		[]string{"resouces.modelscope.cn"}, 64,
	)
	if err != nil {
		t.Fatalf("fetchLogo() error = %v", err)
	}
	if !bytes.Equal(data, pngHeader) || contentType != "image/png" {
		t.Fatalf("data = %q, contentType = %q", data, contentType)
	}
}

func TestFetchModelScopeAvatarFromRepositoryPage(t *testing.T) {
	client := testHTTPClient(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/models/Krea/Krea-2-Turbo" {
			t.Fatalf("unexpected path %q", request.URL.Path)
		}
		body := `<script>{"avatar":"https://resources.modelscope.cn/avatar/krea.webp"}</script>`
		return testResponse(http.StatusOK, "text/html", []byte(body)), nil
	})
	download := &model.ModelDownload{
		Name: "Krea/Krea-2-Turbo", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel,
	}
	avatarURL, err := fetchModelScopeAvatar(client, "https://modelscope.cn", download)
	if err != nil {
		t.Fatalf("fetchModelScopeAvatar() error = %v", err)
	}
	if avatarURL != "https://resources.modelscope.cn/avatar/krea.webp?x-oss-process=image%2Fresize%2Cm_lfit%2Cw_128%2Ch_128" {
		t.Fatalf("avatar URL = %q", avatarURL)
	}
}

func TestFetchLogoFollowsOnlyAllowedRedirects(t *testing.T) {
	var requestedHosts []string
	client := testHTTPClient(func(request *http.Request) (*http.Response, error) {
		requestedHosts = append(requestedHosts, request.URL.Hostname())
		if request.URL.Hostname() == "source.example" {
			response := testResponse(http.StatusFound, "", nil)
			response.Header.Set("Location", "https://cdn.example/logo.png")
			return response, nil
		}
		return testResponse(http.StatusOK, "image/png", []byte("redirected-logo")), nil
	})

	data, _, err := fetchLogo(
		client, "https://source.example/logo.png", []string{"source.example", "cdn.example"}, 32,
	)
	if err != nil || string(data) != "redirected-logo" {
		t.Fatalf("allowed redirect result: data=%q err=%v", data, err)
	}
	if strings.Join(requestedHosts, ",") != "source.example,cdn.example" {
		t.Fatalf("unexpected logo requests: %v", requestedHosts)
	}

	requestedHosts = nil
	_, _, err = fetchLogo(client, "https://source.example/logo.png", []string{"source.example"}, 32)
	if err == nil || strings.Join(requestedHosts, ",") != "source.example" {
		t.Fatalf("disallowed redirect was followed: hosts=%v err=%v", requestedHosts, err)
	}
}

func TestFetchLogoRejectsUnsafeInitialURLBeforeRequest(t *testing.T) {
	requested := false
	client := testHTTPClient(func(_ *http.Request) (*http.Response, error) {
		requested = true
		return testResponse(http.StatusOK, "image/png", []byte("logo")), nil
	})

	for _, endpoint := range []string{
		"http://cdn-avatars.huggingface.co/logo.png",
		"https://169.254.169.254/latest/meta-data",
		"https://user:password@cdn-avatars.huggingface.co/logo.png",
	} {
		if _, _, err := fetchLogo(
			client, endpoint, []string{"cdn-avatars.huggingface.co"}, 32,
		); err == nil {
			t.Fatalf("fetchLogo() accepted unsafe URL %q", endpoint)
		}
	}
	if requested {
		t.Fatal("unsafe logo URL reached the HTTP transport")
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
