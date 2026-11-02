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

package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

//nolint:gocyclo // Sequential HTTP cases verify grant, revoke, legacy clients and self-service isolation.
func TestAdminUserContainerPermissionAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Name: "target", Space: "target", Nickname: "before", Role: model.RoleUser, Status: model.StatusActive,
		Attributes: datatypes.NewJSONType(model.UserAttribute{UID: ptr.To("1001"), GID: ptr.To("1002")})}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	old := query.User
	query.User = &q.User
	t.Cleanup(func() { query.User = old })
	mgr := handler.NewUserMgr(&handler.RegisterConfig{})
	ctxMgr := handler.NewContextMgr(&handler.RegisterConfig{})
	request := func(role model.Role, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		router := gin.New()
		router.Use(func(c *gin.Context) {
			util.SetJWTContext(c, util.JWTMessage{UserID: user.ID, Username: user.Name, RolePlatform: role})
			c.Next()
		})
		mgr.RegisterAdmin(router.Group("/admin/users"))
		ctxMgr.RegisterProtected(router.Group("/context"))
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	load := func() model.User {
		t.Helper()
		var saved model.User
		if err := db.First(&saved, user.ID).Error; err != nil {
			t.Fatal(err)
		}
		return saved
	}
	for _, tc := range []struct {
		name       string
		role       model.Role
		path, body string
		status     int
		enabled    bool
	}{
		{"ordinary user cannot grant", model.RoleUser, "/admin/users/target/attributes", `{"nickname":"unauthorized","containerSysAdminEnabled":true}`, http.StatusForbidden, false},
		{"admin grant", model.RoleAdmin, "/admin/users/target/attributes", `{"nickname":"granted","uid":"1001","gid":"1002","containerSysAdminEnabled":true}`, http.StatusOK, true},
		{"legacy edit preserves grant", model.RoleAdmin, "/admin/users/target/attributes", `{"nickname":"legacy","uid":"1001","gid":"1002"}`, http.StatusOK, true},
		{"invalid boolean rejected", model.RoleAdmin, "/admin/users/target/attributes", `{"containerSysAdminEnabled":"true"}`, http.StatusBadRequest, true},
		{"missing user", model.RoleAdmin, "/admin/users/missing/attributes", `{"containerSysAdminEnabled":true}`, http.StatusNotFound, true},
		{"ordinary profile edit cannot revoke", model.RoleUser, "/context/attributes", `{"nickname":"self","uid":"9999","containerSysAdminEnabled":false}`, http.StatusOK, true},
		{"admin revoke", model.RoleAdmin, "/admin/users/target/attributes", `{"nickname":"revoked","uid":"1001","gid":"1002","containerSysAdminEnabled":false}`, http.StatusOK, false},
		{"ordinary profile edit cannot grant", model.RoleUser, "/context/attributes", `{"nickname":"self","containerSysAdminEnabled":true}`, http.StatusOK, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := load()
			w := request(tc.role, http.MethodPut, tc.path, tc.body)
			if w.Code != tc.status {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			saved := load()
			if saved.ContainerSysAdminEnabled != tc.enabled || saved.Role != model.RoleUser {
				t.Fatalf("unexpected authorization: %+v", saved)
			}
			if tc.status != http.StatusOK && saved.Nickname != before.Nickname {
				t.Fatal("failed update changed attributes")
			}
			if tc.path == "/context/attributes" && (saved.Attributes.Data().UID == nil || *saved.Attributes.Data().UID != "1001") {
				t.Fatal("profile edit changed UID")
			}
			listed := request(model.RoleAdmin, http.MethodGet, "/admin/users", "")
			var resp resputil.Response[[]handler.UserResp]
			if listed.Code != http.StatusOK {
				t.Fatal(listed.Body.String())
			}
			if err := json.Unmarshal(listed.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if len(resp.Data) != 1 || resp.Data[0].ContainerSysAdminEnabled != tc.enabled {
				t.Fatal("admin list returned stale authorization")
			}
		})
	}
}
