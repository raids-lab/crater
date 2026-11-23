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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/packer"
)

//nolint:gocyclo // Sequential HTTP cases verify identity boundaries, field isolation and database failures.
func TestImageBuildDetailHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Kaniko{}, &model.Image{}, &model.ImageAccount{}, &model.ImageUser{}); err != nil {
		t.Fatal(err)
	}
	old := query.Q
	query.Q = query.Use(db)
	t.Cleanup(func() { query.Q = old })
	build := model.Kaniko{UserID: 1, ImagePackName: "build", ImageLink: "image:tag", Status: model.BuildJobFinished,
		Description: ptr.To("description"), Dockerfile: ptr.To("FROM image:base")}
	if err := db.Create(&build).Error; err != nil {
		t.Fatal(err)
	}
	image := model.Image{UserID: 1, ImagePackName: ptr.To("build"), ImageLink: "image:tag", IsPublic: true}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "build-pod", Namespace: UserNameSpace, Labels: map[string]string{"job-name": "build"}}, Spec: corev1.PodSpec{NodeName: "build-node"}}
	kube := fake.NewClientBuilder().WithObjects(pod).Build()
	request := func(userID uint, role model.Role, path string, withKube bool) *httptest.ResponseRecorder {
		t.Helper()
		mgr := &ImagePackMgr{}
		if withKube {
			mgr.imagepackClient = &crclient.ImagePackController{Client: kube}
		}
		router := gin.New()
		router.Use(func(c *gin.Context) {
			util.SetJWTContext(c, util.JWTMessage{UserID: userID, AccountID: 20, RolePlatform: role})
			c.Next()
		})
		mgr.RegisterProtected(router.Group("/api/v1/images"))
		mgr.RegisterAdmin(router.Group("/api/v1/admin/images", middleware.AuthAdmin()))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		return w
	}
	for _, tt := range []struct {
		name   string
		user   uint
		role   model.Role
		path   string
		manage bool
	}{
		{"shared reader", 2, model.RoleUser, "/api/v1/images/getbyname?name=build", false},
		{"owner", 1, model.RoleUser, "/api/v1/images/getbyname?name=build", true},
		{"admin", 2, model.RoleAdmin, "/api/v1/admin/images/getbyname?name=build", true},
		{"admin using user route", 2, model.RoleAdmin, "/api/v1/images/getbyname?name=build", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := request(tt.user, tt.role, tt.path, tt.manage)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			var response resputil.Response[GetKanikoResponse]
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			d := response.Data
			if d.Dockerfile != "FROM image:base" || d.Description != "description" || d.CanManage != tt.manage || d.CanViewBuildLogs != tt.manage {
				t.Fatalf("unexpected details: %+v", d)
			}
			if tt.manage {
				if d.PodName != "build-pod" || d.NodeName != "build-node" {
					t.Fatalf("missing pod: %+v", d)
				}
			} else if d.PodName != "" || d.PodNameSpace != "" || d.NodeName != "" {
				t.Fatalf("leaked pod: %+v", d)
			}
		})
	}
	for _, tt := range []struct {
		path   string
		status int
	}{
		{"/api/v1/admin/images/getbyname?name=build", http.StatusForbidden},
		{"/api/v1/images/getbyname", http.StatusBadRequest},
		{"/api/v1/images/getbyname?name=missing", http.StatusNotFound},
	} {
		w := request(2, model.RoleUser, tt.path, false)
		if w.Code != tt.status {
			t.Fatalf("%s: status=%d body=%s", tt.path, w.Code, w.Body)
		}
	}
	if err := db.Model(&image).Update("is_public", false).Error; err != nil {
		t.Fatal(err)
	}
	w := request(2, model.RoleUser, "/api/v1/images/getbyname?name=build", false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("private: %d %s", w.Code, w.Body)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	w = request(2, model.RoleUser, "/api/v1/images/getbyname?name=build", false)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "database is closed") {
		t.Fatalf("database: %d %s", w.Code, w.Body)
	}
}

func TestImageBuildRemovalAction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Kaniko{}, &model.Image{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	oldKaniko, oldImage := query.Kaniko, query.Image
	query.Kaniko, query.Image = &q.Kaniko, &q.Image
	t.Cleanup(func() { query.Kaniko, query.Image = oldKaniko, oldImage })

	for _, tc := range []struct {
		name       string
		action     string
		status     model.BuildStatus
		admin      bool
		httpStatus int
		jobCalls   int
		imageCalls int
	}{
		{"admin cannot cancel finished build", "cancel", model.BuildJobFinished, true, http.StatusConflict, 0, 0},
		{"owner cannot cancel finished build", "cancel", model.BuildJobFinished, false, http.StatusConflict, 0, 0},
		{"cancel running build", "cancel", model.BuildJobRunning, true, http.StatusOK, 1, 0},
		{"delete rejects running build", "delete", model.BuildJobRunning, true, http.StatusConflict, 0, 0},
		{"delete finished build", "delete", model.BuildJobFinished, true, http.StatusOK, 0, 1},
		{"legacy remove deletes finished build", "", model.BuildJobFinished, true, http.StatusOK, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build := model.Kaniko{UserID: 1, ImagePackName: tc.name, ImageLink: "image:tag", Status: tc.status}
			if err := db.Create(&build).Error; err != nil {
				t.Fatal(err)
			}
			image := model.Image{UserID: build.UserID, ImagePackName: &build.ImagePackName, ImageLink: build.ImageLink}
			if err := db.Create(&image).Error; err != nil {
				t.Fatal(err)
			}
			job, registry := &imageBuildRemovalPacker{}, &imageBuildRemovalRegistry{}
			mgr := &ImagePackMgr{imagePacker: job, imageRegistry: registry}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{UserID: build.UserID})
				c.Next()
			})
			route := mgr.UserRemoveKanikoByID
			if tc.admin {
				route = mgr.AdminRemoveKanikoByIDList
			}
			router.POST("/remove", route)
			payload := fmt.Sprintf(`{"idList":[%d]}`, build.ID)
			if tc.action != "" {
				payload = fmt.Sprintf(`{"idList":[%d],"action":%q}`, build.ID, tc.action)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/remove", strings.NewReader(payload)))
			if w.Code != tc.httpStatus || job.calls != tc.jobCalls || registry.calls != tc.imageCalls {
				t.Fatalf("HTTP %d, Kubernetes calls %d, registry calls %d: %s", w.Code, job.calls, registry.calls, w.Body)
			}
			assertImageBuildRemovalRecords(t, db, build.ID, image.ID, int64(1-tc.imageCalls))
		})
	}
}

func assertImageBuildRemovalRecords(t *testing.T, db *gorm.DB, buildID, imageID uint, want int64) {
	t.Helper()
	for _, row := range []struct {
		model any
		id    uint
	}{{&model.Kaniko{}, buildID}, {&model.Image{}, imageID}} {
		var count int64
		if err := db.Model(row.model).Where("id = ?", row.id).Count(&count).Error; err != nil || count != want {
			t.Fatalf("model %T has %d rows, want %d (error: %v)", row.model, count, want, err)
		}
	}
}

type imageBuildRemovalPacker struct {
	packer.ImagePackerInterface
	calls int
}

func (p *imageBuildRemovalPacker) DeleteJob(context.Context, string, string) error {
	p.calls++
	return nil
}

type imageBuildRemovalRegistry struct {
	imageregistry.ImageRegistryInterface
	calls int
}

func (r *imageBuildRemovalRegistry) DeleteImageFromProject(context.Context, string) error {
	r.calls++
	return nil
}
