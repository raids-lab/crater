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

package util

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
)

func clientHeader(clientType, userAgent string) http.Header {
	header := http.Header{}
	if clientType != "" {
		header.Set(ClientHeader, clientType)
	}
	if userAgent != "" {
		header.Set("User-Agent", userAgent)
	}
	return header
}

func TestParseClientInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clientType string
		userAgent  string
		wantType   model.ClientType
		wantVer    string
	}{
		{
			name:       "explicit cli with user agent version",
			clientType: "cli",
			userAgent:  "crater-cli/1.4.1",
			wantType:   model.ClientTypeCLI,
			wantVer:    "1.4.1",
		},
		{
			name:       "explicit cli without user agent",
			clientType: "cli",
			wantType:   model.ClientTypeCLI,
		},
		{
			name:       "explicit cli with browser user agent",
			clientType: "cli",
			userAgent:  "Mozilla/5.0 (X11; Linux x86_64) Chrome/131.0.0.0 Safari/537.36",
			wantType:   model.ClientTypeCLI,
		},
		{
			name:       "explicit web ignores cli user agent",
			clientType: "web",
			userAgent:  "crater-cli/1.4.1",
			wantType:   model.ClientTypeWeb,
		},
		{
			name:       "web value is normalized",
			clientType: "  WEB  ",
			wantType:   model.ClientTypeWeb,
		},
		{
			name:      "legacy cli user agent fallback",
			userAgent: "crater-cli/1.4.1",
			wantType:  model.ClientTypeCLI,
			wantVer:   "1.4.1",
		},
		{
			name:      "legacy cli development version",
			userAgent: "crater-cli/1.4.1+dev.23.gabcdef0",
			wantType:  model.ClientTypeCLI,
			wantVer:   "1.4.1+dev.23.gabcdef0",
		},
		{
			name:      "legacy cli user agent is case insensitive",
			userAgent: "Crater-CLI/1.4.1",
			wantType:  model.ClientTypeCLI,
			wantVer:   "1.4.1",
		},
		{
			name:      "legacy cli user agent with trailing token",
			userAgent: "crater-cli/1.4.1 (corporate proxy)",
			wantType:  model.ClientTypeCLI,
			wantVer:   "1.4.1",
		},
		{
			name:      "legacy cli without version",
			userAgent: "crater-cli/",
			wantType:  model.ClientTypeCLI,
		},
		{
			name:      "browser user agent alone is unknown",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
			wantType:  model.ClientTypeUnknown,
		},
		{
			name:     "missing all identity is unknown",
			wantType: model.ClientTypeUnknown,
		},
		{
			name:       "invalid explicit value is unknown and does not fall back",
			clientType: "frontend",
			userAgent:  "crater-cli/1.4.1",
			wantType:   model.ClientTypeUnknown,
		},
		{
			name:       "invalid user agent version keeps cli type",
			clientType: "cli",
			userAgent:  "crater-cli/../etc/passwd",
			wantType:   model.ClientTypeCLI,
		},
		{
			name:       "empty explicit value falls back to user agent",
			clientType: "",
			userAgent:  "crater-cli/1.4.1",
			wantType:   model.ClientTypeCLI,
			wantVer:    "1.4.1",
		},
		{
			name:      "overlong version is dropped",
			userAgent: "crater-cli/" + strings.Repeat("1", 65),
			wantType:  model.ClientTypeCLI,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := ParseClientInfo(clientHeader(test.clientType, test.userAgent))
			if got.Type != test.wantType || got.Version != test.wantVer {
				t.Fatalf("ParseClientInfo() = %+v, want type=%q version=%q", got, test.wantType, test.wantVer)
			}
		})
	}
}

func TestClientInfoGinContext(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetClientInfo(c); got.Type != model.ClientTypeUnknown || got.Version != "" {
		t.Fatalf("GetClientInfo() without middleware = %+v, want unknown", got)
	}

	SetClientInfo(c, ClientInfo{Type: model.ClientTypeCLI, Version: "1.4.1"})
	if got := GetClientInfo(c); got.Type != model.ClientTypeCLI || got.Version != "1.4.1" {
		t.Fatalf("GetClientInfo() = %+v, want cli/1.4.1", got)
	}

	SetClientInfo(c, ClientInfo{Type: model.ClientTypeWeb, Version: "should-be-dropped"})
	if got := GetClientInfo(c); got.Type != model.ClientTypeWeb || got.Version != "" {
		t.Fatalf("GetClientInfo() = %+v, want web without version", got)
	}
}
