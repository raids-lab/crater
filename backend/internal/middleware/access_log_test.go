// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccessLogRedactsURLTokens(t *testing.T) {
	for _, tt := range []struct {
		name, query, wantPath, wantCredential string
	}{
		{"ordinary query", "?scope=admin&pod=main", "/exec?scope=admin&pod=main", ""},
		{
			"repeated and encoded token parameters",
			"?token=fixture-secret&%74oken=encoded-secret&ToKeN=uppercase-secret&scope=admin",
			"/exec?ToKeN=REDACTED&scope=admin&token=REDACTED",
			"fixture-secret",
		},
		{"malformed query", "?token=fixture-secret&pod=%ZZ", "/exec", "fixture-secret"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			router := gin.New()
			router.Use(gin.LoggerWithConfig(gin.LoggerConfig{Output: &output, Formatter: FormatAccessLog}))
			router.GET("/exec", func(c *gin.Context) {
				if token := c.Query("token"); token != tt.wantCredential {
					t.Errorf("authorization input = %q, want %q", token, tt.wantCredential)
				}
				c.Status(http.StatusUnauthorized)
			})
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/exec"+tt.query, http.NoBody))
			got := output.String()
			if strings.Contains(got, "secret") || !strings.Contains(got, `"`+tt.wantPath+`"`) {
				t.Fatalf("access log = %q, want %q with no credential", got, tt.wantPath)
			}
		})
	}
}
