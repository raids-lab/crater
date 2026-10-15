/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tensorboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	tensorboardservice "github.com/raids-lab/crater/internal/service/tensorboard"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
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

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add Kubernetes scheme: %v", err)
	}
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatalf("add Volcano scheme: %v", err)
	}
	panel := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tb-12ab34cd",
			Namespace: config.GetConfig().Namespaces.Job,
			Labels: map[string]string{
				crclient.LabelKeyTaskUser: "alice",
			},
		},
		Status: batch.JobStatus{State: batch.JobState{Phase: batch.Pending}},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(panel).Build()
	mgr := &TensorboardMgr{service: tensorboardservice.NewTensorboardService(client, nil)}
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
