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

package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	internalversion "github.com/raids-lab/crater/cli/internal/version"
)

func TestGetCompatibilitySendsCLIHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != CompatibilityPath {
			t.Errorf("path = %q, want %q", r.URL.Path, CompatibilityPath)
		}
		if got := r.Header.Get("User-Agent"); got != internalversion.UserAgent() {
			t.Errorf("User-Agent = %q, want %q", got, internalversion.UserAgent())
		}
		if got := r.Header.Get(internalversion.APIVersionHeader); got != strconv.Itoa(internalversion.APIVersion) {
			t.Errorf("%s = %q, want %d", internalversion.APIVersionHeader, got, internalversion.APIVersion)
		}
		if got := r.Header.Get(internalversion.ClientHeader); got != internalversion.ClientTypeCLI {
			t.Errorf("%s = %q, want %q", internalversion.ClientHeader, got, internalversion.ClientTypeCLI)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"apiVersion":1,"minSupportedCliApiVersion":1,`+
			`"appVersion":"v1.1.1","shortCommitSHA":"f42b0c2","buildType":"release",`+
			`"buildTime":"2026-07-26T08:30:00Z"},"msg":""}`)
	}))
	defer server.Close()

	info, err := NewClient(server.URL).GetCompatibility()
	if err != nil {
		t.Fatal(err)
	}
	if info.APIVersion != 1 || info.MinSupportedCLIAPIVersion != 1 {
		t.Fatalf("compatibility info = %+v", info)
	}
	if !info.VersionsReported {
		t.Fatal("VersionsReported = false, want true")
	}
	if info.AppVersion != "v1.1.1" ||
		info.ShortCommitSHA != "f42b0c2" ||
		info.BuildType != "release" ||
		info.BuildTime != "2026-07-26T08:30:00Z" {
		t.Fatalf("backend build info = %+v", info)
	}
}

func TestGetCompatibilityDistinguishesExplicitZeroFromMissingFields(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantReported bool
	}{
		{
			name:         "explicit zero",
			body:         `{"code":0,"data":{"apiVersion":0,"minSupportedCliApiVersion":0},"msg":""}`,
			wantReported: true,
		},
		{
			name:         "missing fields",
			body:         `{"code":0,"data":{},"msg":""}`,
			wantReported: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			info, err := NewClient(server.URL).GetCompatibility()
			if err != nil {
				t.Fatal(err)
			}
			if info.VersionsReported != tt.wantReported {
				t.Fatalf("VersionsReported = %v, want %v", info.VersionsReported, tt.wantReported)
			}
		})
	}
}

func TestGetCompatibilityAllowsMissingBuildInformation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"apiVersion":1,"minSupportedCliApiVersion":1},"msg":""}`)
	}))
	defer server.Close()

	info, err := NewClient(server.URL).GetCompatibility()
	if err != nil {
		t.Fatal(err)
	}
	if !info.VersionsReported {
		t.Fatal("VersionsReported = false, want true")
	}
	if info.AppVersion != "" || info.ShortCommitSHA != "" || info.BuildType != "" || info.BuildTime != "" {
		t.Fatalf("backend build info = %+v, want empty optional fields", info)
	}
}

func TestGetCompatibilityPreservesPlainText404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "404 page not found")
	}))
	defer server.Close()

	_, err := NewClient(server.URL).GetCompatibility()
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error = %T %v, want *RequestError", err, err)
	}
	if requestErr.HTTPStatus != http.StatusNotFound {
		t.Fatalf("HTTP status = %d, want %d", requestErr.HTTPStatus, http.StatusNotFound)
	}
	if requestErr.CraterCode != 0 {
		t.Fatalf("Crater code = %d, want 0", requestErr.CraterCode)
	}
	if requestErr.Msg != "404 page not found" {
		t.Fatalf("message = %q, want %q", requestErr.Msg, "404 page not found")
	}
}
