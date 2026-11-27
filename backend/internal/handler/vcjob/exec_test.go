// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package vcjob

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/jobexec"
)

func TestUserExecRejectsAdminScopeBeforeTargetLookup(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		platformRole model.Role
		projectRole  model.Role
		bearer       bool
		wantStatus   int
	}{
		{"regular user", "?scope=admin", model.RoleUser, model.RoleUser, true, http.StatusForbidden},
		{"project administrator", "?scope=admin", model.RoleUser, model.RoleAdmin, true, http.StatusForbidden},
		{"query token administrator", "?scope=admin&token=fixture", model.RoleAdmin, model.RoleUser, false, http.StatusUnauthorized},
		{"empty scope", "?scope=", model.RoleAdmin, model.RoleUser, true, http.StatusBadRequest},
		{"unknown scope", "?scope=user", model.RoleAdmin, model.RoleUser, true, http.StatusBadRequest},
		{"duplicate scope", "?scope=admin&scope=admin", model.RoleAdmin, model.RoleUser, true, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &VolcanojobMgr{}
			router := gin.New()
			router.GET("/vcjobs/:name/exec", func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{RolePlatform: tt.platformRole, RoleAccount: tt.projectRole})
				mgr.UserExec(c)
			})
			req := httptest.NewRequest(http.MethodGet, "/vcjobs/job/exec"+tt.query, http.NoBody)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Protocol", jobexec.Subprotocol)
			if tt.bearer {
				req.Header.Set("Authorization", "Bearer fixture")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus || rec.Header().Get(execScopeHeader) != "" {
				t.Fatalf("status = %d, scope = %q, body = %q; want status %d with no scope acknowledgement",
					rec.Code, rec.Header().Get(execScopeHeader), rec.Body.String(), tt.wantStatus)
			}
		})
	}
}
