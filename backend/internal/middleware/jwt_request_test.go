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
		{"query token requires websocket", "", "", "browser-fixture", "", false},
		{"invalid header is rejected", "websocket", "Basic invalid", "browser-fixture", "", false},
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

func TestFreshPermissionCheckDoesNotChangeBrowserTerminalReads(t *testing.T) {
	terminal := "/api/v1/websocket/namespaces/:namespace/pods/:name/containers/:container/terminal"
	for _, tt := range []struct {
		name, method, route, path string
		fresh                     bool
	}{
		{"browser terminal", http.MethodGet, terminal, "/api/v1/websocket/namespaces/tasks/pods/job/containers/main/terminal", false},
		{"CLI exec", http.MethodGet, "/api/v1/vcjobs/:name/exec", "/api/v1/vcjobs/job/exec", true},
		{"job read", http.MethodGet, "/api/v1/vcjobs/:name/detail", "/api/v1/vcjobs/job/detail", false},
		{"SSH write", http.MethodPost, "/api/v1/vcjobs/:name/ssh", "/api/v1/vcjobs/job/ssh", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Handle(tt.method, tt.route, func(c *gin.Context) {
				if got := requiresFreshPermissions(c); got != tt.fresh {
					t.Errorf("fresh=%v, want %v for %s %s", got, tt.fresh, tt.method, tt.path)
				}
				c.Status(http.StatusNoContent)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, http.NoBody))
			if response.Code != http.StatusNoContent {
				t.Fatalf("route returned %d, want 204", response.Code)
			}
		})
	}
}
