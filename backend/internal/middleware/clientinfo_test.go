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

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestClientIdentityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		clientType string
		userAgent  string
		wantType   model.ClientType
		wantVer    string
	}{
		{
			name:       "explicit web",
			clientType: "web",
			wantType:   model.ClientTypeWeb,
		},
		{
			name:      "legacy cli user agent",
			userAgent: "crater-cli/1.4.1",
			wantType:  model.ClientTypeCLI,
			wantVer:   "1.4.1",
		},
		{
			name:       "invalid header does not abort and is unknown",
			clientType: "unknown-client",
			userAgent:  "crater-cli/1.4.1",
			wantType:   model.ClientTypeUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ClientIdentity())
			router.GET("/client", func(c *gin.Context) {
				info := util.GetClientInfo(c)
				c.JSON(http.StatusOK, gin.H{"type": info.Type, "version": info.Version})
			})

			request := httptest.NewRequest(http.MethodGet, "/client", http.NoBody)
			if test.clientType != "" {
				request.Header.Set(util.ClientHeader, test.clientType)
			}
			if test.userAgent != "" {
				request.Header.Set("User-Agent", test.userAgent)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}

			var got struct {
				Type    model.ClientType `json:"type"`
				Version string           `json:"version"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Type != test.wantType || got.Version != test.wantVer {
				t.Fatalf("client info = %+v, want type=%q version=%q", got, test.wantType, test.wantVer)
			}
		})
	}
}
