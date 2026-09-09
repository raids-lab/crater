package tensorboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	interutil "github.com/raids-lab/crater/internal/util"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "valid", header: "Bearer signed-token", want: "signed-token", ok: true},
		{name: "extra spaces", header: "  Bearer   signed-token ", want: "signed-token", ok: true},
		{name: "wrong scheme", header: "Basic signed-token"},
		{name: "missing token", header: "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bearerToken(tt.header)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("bearerToken(%q) = (%q, %v), want (%q, %v)", tt.header, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestIsOwnedTensorboardURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		username string
		want     bool
	}{
		{
			name:     "panel root",
			rawURL:   "https://crater.example/ingress/alice-12ab34cd/",
			username: "alice",
			want:     true,
		},
		{
			name:     "panel asset",
			rawURL:   "https://crater.example/ingress/alice-12ab34cd/data/plugin/scalars/scalars?run=demo",
			username: "alice",
			want:     true,
		},
		{
			name:     "hyphenated username",
			rawURL:   "/ingress/alice-lab-deadbeef/static/index.js",
			username: "alice-lab",
			want:     true,
		},
		{
			name:     "different owner",
			rawURL:   "/ingress/bob-12ab34cd/",
			username: "alice",
		},
		{
			name:     "invalid panel id",
			rawURL:   "/ingress/alice-not-an-id/",
			username: "alice",
		},
		{
			name:     "prefix confusion",
			rawURL:   "/ingress/alice2-12ab34cd/",
			username: "alice",
		},
		{
			name:     "encoded traversal",
			rawURL:   "/ingress/alice-12ab34cd%2f..%2fbob-deadbeef/",
			username: "alice",
		},
		{name: "missing original URL", username: "alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOwnedTensorboardURL(tt.rawURL, tt.username); got != tt.want {
				t.Fatalf("isOwnedTensorboardURL(%q, %q) = %v, want %v", tt.rawURL, tt.username, got, tt.want)
			}
		})
	}
}

func TestAuthorizeIngress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	accessToken, _, err := interutil.GetTokenMgr().CreateTokens(&interutil.JWTMessage{
		Username: "alice",
	})
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}

	tests := []struct {
		name        string
		originalURL string
		cookie      string
		wantStatus  int
	}{
		{
			name:        "owned panel",
			originalURL: "https://crater.example/ingress/alice-12ab34cd/",
			cookie:      accessToken,
			wantStatus:  http.StatusNoContent,
		},
		{
			name:        "different owner",
			originalURL: "https://crater.example/ingress/bob-12ab34cd/",
			cookie:      accessToken,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "missing cookie",
			originalURL: "https://crater.example/ingress/alice-12ab34cd/",
			wantStatus:  http.StatusUnauthorized,
		},
	}

	mgr := &TensorboardMgr{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			request := httptest.NewRequest(http.MethodGet, "/api/tensorboard/auth", http.NoBody)
			request.Header.Set("X-Original-URL", tt.originalURL)
			if tt.cookie != "" {
				request.AddCookie(&http.Cookie{Name: tensorboardAccessCookie, Value: tt.cookie})
			}
			context.Request = request

			mgr.AuthorizeIngress(context)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}
