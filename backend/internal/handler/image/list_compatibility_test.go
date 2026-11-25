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

package image

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
)

func TestLegacyKanikoWrapperIsCompleteAndPermissionScoped(t *testing.T) {
	db := setupKanikoListTestDB(t)
	createKanikoListTestUsers(t, db, "alice", "bob")
	for id := uint(1); id <= 14; id++ {
		owner := uint(1)
		if id == 14 {
			owner = 2
		}
		createKanikoListTestRecords(t, db, model.Kaniko{Model: gorm.Model{ID: id}, UserID: owner})
	}
	for _, test := range []struct {
		name  string
		owner *uint
		count int
	}{
		{name: "ordinary user", owner: uintPointer(1), count: 13},
		{name: "administrator", count: 14},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/kaniko", http.NoBody)
			(&ImagePackMgr{}).listKanikoPageWithDB(ctx, db, test.owner)
			var response resputil.Response[ListKanikoResponse]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || len(response.Data.KanikoInfoList) != test.count {
				t.Fatalf("legacy wrapper truncated or changed: %s; %v", recorder.Body.String(), err)
			}
			if test.owner != nil {
				for _, build := range response.Data.KanikoInfoList {
					if build.UserInfo.Username != "alice" {
						t.Fatalf("another user's build leaked: %+v", build)
					}
				}
			}
		})
	}
	page := requestKanikoListPage(t, db, uintPointer(1), "page=2&page_size=5")
	if page.Total != 13 || len(page.Items) != 5 {
		t.Fatalf("explicit pagination changed: %+v", page)
	}
}

func uintPointer(value uint) *uint { return &value }
