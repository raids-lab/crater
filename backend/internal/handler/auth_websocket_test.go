// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func TestAuthenticationWebSocketClientsUseCurrentPermissions(t *testing.T) {
	for _, client := range []struct {
		name, route, path string
		websocket, query  bool
	}{
		{"browser terminal", "/api/v1/websocket/namespaces/:namespace/pods/:name/containers/:container/terminal", "/api/v1/websocket/namespaces/tasks/pods/job/containers/main/terminal", true, true},
		{"CLI exec", "/api/v1/vcjobs/:name/exec", "/api/v1/vcjobs/job/exec", true, false},
		{"ordinary read", "/api/v1/vcjobs/:name/detail", "/api/v1/vcjobs/job/detail", false, false},
	} {
		t.Run(client.name, func(t *testing.T) {
			for _, state := range []string{"active", "disabled", "project revoked", "role changed"} {
				t.Run(state, func(t *testing.T) {
					mgr, db := newFirstLoginHarness(t, false, false)
					session := firstSession(t, mgr)
					status, code := http.StatusOK, bizerr.BizCode(0)
					var err error
					switch state {
					case "disabled":
						err = db.Model(&model.User{}).Where("id = ?", session.User.ID).Update("status", model.StatusInactive).Error
						status, code = http.StatusForbidden, bizerr.Forbidden.AccountDisabled
					case "project revoked":
						err = db.Where("user_id = ?", session.User.ID).Delete(&model.UserAccount{}).Error
						status, code = http.StatusForbidden, bizerr.Forbidden.NoProjectAccess
					case "role changed":
						err = db.Model(&model.User{}).Where("id = ?", session.User.ID).Update("role", model.RoleGuest).Error
					}
					if err != nil {
						t.Fatal(err)
					}

					// Probe the real middleware before a WebSocket handler upgrades the connection.
					router := gin.New()
					router.GET(client.route, middleware.AuthProtected(mgr.userBanService), func(c *gin.Context) {
						resputil.Success(c, util.GetToken(c))
					})
					path := client.path
					if client.query {
						path += "?token=" + url.QueryEscape(session.AccessToken)
					}
					request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
					if client.websocket {
						request.Header.Set("Upgrade", "websocket")
					}
					if !client.query {
						request.Header.Set("Authorization", "Bearer "+session.AccessToken)
					}
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					requireAuthResponse(t, response, status, code)
					if state == "role changed" {
						var result resputil.Response[util.JWTMessage]
						if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
							t.Fatal(err)
						}
						if result.Data.RolePlatform != model.RoleGuest {
							t.Fatal("request retained the role from the old token")
						}
					}
				})
			}
		})
	}
}
