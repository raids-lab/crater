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

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

func TestModelDownloadLimitConfigRoutesHideWhitelistFromProtectedUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:model_download_limit_handlers?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}); err != nil {
		t.Fatal(err)
	}
	configService := service.NewConfigService(query.Use(db))
	if err := configService.UpdateModelDownloadLimitConfig(t.Context(), service.ModelDownloadLimitConfig{
		Enabled: true, MaxConcurrent: 1, WindowHours: 2, MaxSuccessfulDownloads: 5,
		WhitelistUserIDs: []uint{7, 9},
	}); err != nil {
		t.Fatal(err)
	}

	mgr := &SystemConfigMgr{service: configService}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{UserID: 7, RolePlatform: model.RoleAdmin})
		c.Next()
	})
	mgr.RegisterProtected(router.Group("/v1/system-config"))
	mgr.RegisterAdmin(router.Group("/v1/admin/system-config"))

	protectedData := requestModelDownloadLimitConfig(t, router, "/v1/system-config/model-download-limit")
	if _, exposed := protectedData["whitelistUserIds"]; exposed {
		t.Fatal("protected model download config must not expose the full whitelist")
	}
	var exempt bool
	if err := json.Unmarshal(protectedData["exempt"], &exempt); err != nil {
		t.Fatal(err)
	}
	if !exempt {
		t.Fatal("whitelisted current user should be marked exempt")
	}

	adminData := requestModelDownloadLimitConfig(t, router, "/v1/admin/system-config/model-download-limit")
	if _, exposed := adminData["exempt"]; exposed {
		t.Fatal("admin model download config should return the whitelist instead of a current-user exemption")
	}
	var whitelistUserIDs []uint
	if err := json.Unmarshal(adminData["whitelistUserIds"], &whitelistUserIDs); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(whitelistUserIDs, []uint{7, 9}) {
		t.Fatalf("admin whitelist = %v, want [7 9]", whitelistUserIDs)
	}
}

func requestModelDownloadLimitConfig(
	t *testing.T, router http.Handler, path string,
) map[string]json.RawMessage {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s returned HTTP %d: %s", path, recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data
}
