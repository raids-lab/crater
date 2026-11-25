// Copyright 2026 The Crater Project Team, RAIDS-Lab

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProtectedRequestTokenPreservesBrowserAndNativeClients(t *testing.T) {
	for _, tt := range []struct {
		name, upgrade, header, query, want string
		valid                              bool
	}{
		{"browser terminal", "websocket", "", "browser-fixture", "browser-fixture", true},
		{"case-insensitive upgrade", "WebSocket", "", "browser-fixture", "browser-fixture", true},
		{"native exec", "websocket", "Bearer native-fixture", "", "native-fixture", true},
		{"header takes precedence", "websocket", "Bearer native-fixture", "ignored", "native-fixture", true},
		{"ordinary HTTP", "", "Bearer http-fixture", "", "http-fixture", true},
		{"case-insensitive bearer", "websocket", "bearer native-fixture", "", "native-fixture", true},
		{"whitespace in header", "websocket", "  Bearer\t native-fixture  ", "", "native-fixture", true},
		{"query token requires websocket", "", "", "browser-fixture", "", false},
		{"invalid header is rejected", "websocket", "Basic invalid", "browser-fixture", "", false},
		{"missing bearer token", "websocket", "Bearer ", "browser-fixture", "", false},
		{"extra header fields", "websocket", "Bearer native-fixture extra", "browser-fixture", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/terminal?token="+tt.query, http.NoBody)
			c.Request.Header.Set("Upgrade", tt.upgrade)
			c.Request.Header.Set("Authorization", tt.header)
			got, err := protectedRequestToken(c)
			if (err == nil) != tt.valid || got != tt.want {
				t.Fatalf("token=%q error=%v, want %q valid=%v", got, err, tt.want, tt.valid)
			}
		})
	}
}
