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
	"testing"

	"github.com/gin-gonic/gin"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

func TestTensorboardCreateIngressAuthentication(t *testing.T) {
	setupTensorboardCreationTest(t)
	originalMode := gin.Mode()
	t.Cleanup(func() { gin.SetMode(originalMode) })
	tests := []struct {
		name       string
		mode       string
		override   string
		configured *bool
		wantAuth   bool
	}{
		{name: "release default", mode: gin.ReleaseMode, wantAuth: true},
		{name: "release explicit disable", mode: gin.ReleaseMode, configured: ptr.To(false)},
		{name: "release ignores development override", mode: gin.ReleaseMode, override: "false", wantAuth: true},
		{name: "test mode preserves configured authentication", mode: gin.TestMode, wantAuth: true},
		{name: "debug default", mode: gin.DebugMode, wantAuth: true},
		{name: "debug default remains authenticated despite legacy local YAML", mode: gin.DebugMode, configured: ptr.To(false), wantAuth: true},
		{name: "debug explicit authentication", mode: gin.DebugMode, override: "true", configured: ptr.To(false), wantAuth: true},
		{name: "debug explicit disable", mode: gin.DebugMode, override: "false", configured: ptr.To(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(tt.mode)
			t.Setenv("CRATER_DEV_TENSORBOARD_INGRESS_AUTH", tt.override)
			cfg := config.GetConfig()
			cfg.Tensorboard.IngressAuthEnabled = tt.configured
			crClient := newTensorboardTestClient(t)
			if err := networkingv1.AddToScheme(crClient.Scheme()); err != nil {
				t.Fatal(err)
			}
			svc := NewTensorboardService(crClient, crclient.NewServiceManager(crClient, nil))
			result, err := svc.Create(t.Context(), interutil.JWTMessage{
				Username: "alice", UserID: 42, AccountID: model.DefaultAccountID,
			}, &payload.CreateTensorboardReq{Name: "Development panel", LogDir: "/home/alice/logs"})
			if err != nil {
				t.Fatalf("create test panel: %v", err)
			}
			assertTensorboardIngressAuthentication(t, svc, result, tt.wantAuth)
		})
	}
}

func assertTensorboardIngressAuthentication(
	t *testing.T,
	svc *TensorboardService,
	panel *payload.CreateTensorboardResp,
	wantAuth bool,
) {
	t.Helper()
	var ingresses networkingv1.IngressList
	if err := svc.crClient.List(t.Context(), &ingresses); err != nil || len(ingresses.Items) != 1 {
		t.Fatalf("expected one Ingress, got %d, error: %v", len(ingresses.Items), err)
	}
	annotations := ingresses.Items[0].Annotations
	hasAuth := annotations["nginx.ingress.kubernetes.io/auth-url"] != ""
	if wantAuth && annotations["nginx.ingress.kubernetes.io/auth-url"] != "https://$host/api/tensorboard/auth" {
		t.Fatal("Ingress must use the TensorBoard ownership authentication endpoint")
	}
	if hasAuth != wantAuth {
		t.Fatalf("Ingress authentication = %t, want %t", hasAuth, wantAuth)
	}
	if wantAuth && annotations["nginx.ingress.kubernetes.io/auth-method"] != "GET" {
		t.Fatal("Ingress authentication method was not preserved")
	}
	_, err := svc.GetAccessPath(t.Context(), "bob", panel.TensorboardID)
	assertTensorboardErrorCode(t, err, bizerr.Forbidden.PermissionDenied)
	err = svc.Delete(t.Context(), "bob", panel.TensorboardID)
	assertTensorboardErrorCode(t, err, bizerr.Forbidden.PermissionDenied)
	if _, err := svc.GetAccessPath(t.Context(), "alice", panel.TensorboardID); err != nil {
		t.Fatalf("owner lost panel access: %v", err)
	}
}

func TestTensorboardIngressAuthenticationRejectsInvalidDebugOption(t *testing.T) {
	originalMode := gin.Mode()
	t.Cleanup(func() { gin.SetMode(originalMode) })
	t.Setenv("CRATER_DEV_TENSORBOARD_INGRESS_AUTH", "invalid-option")
	gin.SetMode(gin.DebugMode)
	_, err := resolveTensorboardIngressAuth(&config.TensorboardConfig{})
	assertTensorboardErrorCode(t, err, bizerr.Internal.K8sServiceError)

	gin.SetMode(gin.ReleaseMode)
	enabled, err := resolveTensorboardIngressAuth(&config.TensorboardConfig{})
	if err != nil || !enabled {
		t.Fatalf("release mode must ignore the debug option: enabled=%t, err=%v", enabled, err)
	}
}
