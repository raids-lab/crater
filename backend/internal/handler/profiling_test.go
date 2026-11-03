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
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/aijob"
	"github.com/raids-lab/crater/internal/handler/spjob"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	recommendapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/util"
)

const profilingRequest = `{
 "name": "profiling-test", "workingDir": "/tmp",
 "image": {"imageLink": "example.invalid/profiling:test"},
 "resource": {"cpu": "1"}, "replicas": 1
}`

type profilingTaskController struct {
	aitaskctl.TaskControllerInterface
}

func (*profilingTaskController) TaskUpdated(_ util.TaskUpdateChan) {}

func profilingJobRequest(t *testing.T, manager handler.Manager, endpoint, body string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		interutil.SetJWTContext(c, interutil.JWTMessage{UserID: 42, Username: "profiling-user", RolePlatform: model.RoleUser})
		c.Next()
	})
	manager.RegisterProtected(router.Group("/jobs"))
	request := httptest.NewRequest(http.MethodPost, "/jobs/"+endpoint, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func createProfilingJob(t *testing.T, manager handler.Manager, endpoint string, sysAdminEnabled ...bool) {
	t.Helper()
	body := profilingRequest
	if len(sysAdminEnabled) > 0 && sysAdminEnabled[0] {
		body = strings.Replace(body, `"name":`, `"sysAdminEnabled": true, "name":`, 1)
	}
	response := profilingJobRequest(t, manager, endpoint, body)
	if response.Code != http.StatusOK {
		t.Fatalf("create job: %d %s", response.Code, response.Body.String())
	}
}

func assertOptionalJobProfiling(t *testing.T, pod *corev1.PodSpec) {
	t.Helper()
	if len(pod.Containers) != 1 {
		t.Fatalf("unexpected containers: %+v", pod.Containers)
	}
	sc := pod.Containers[0].SecurityContext
	if sc == nil || sc.Capabilities == nil {
		t.Fatal("missing profiling capabilities")
	}
	for _, cap := range []corev1.Capability{"SYS_PTRACE"} {
		if !slices.Contains(sc.Capabilities.Add, cap) {
			t.Errorf("missing capability %s", cap)
		}
	}
	if slices.Contains(sc.Capabilities.Add, "PERFMON") {
		t.Fatal("unexpected default PERFMON")
	}
	if pod.HostPID || slices.Contains(sc.Capabilities.Add, "SYS_ADMIN") || (sc.Privileged != nil && *sc.Privileged) {
		t.Fatal("profiling enabled host access or administrative privileges")
	}
}

func setupOptionalJobPermissionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.AITask{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Model: gorm.Model{ID: 42}, Name: "profiling-user", Space: "profiling-user", Role: model.RoleUser, Status: model.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	originalUser, originalTask := query.User, query.AITask
	q := query.Use(db)
	query.User, query.AITask = &q.User, &q.AITask
	t.Cleanup(func() { query.User, query.AITask = originalUser, originalTask })
	return db
}

func TestEMIASJupyterPersistsProfilingCapabilities(t *testing.T) {
	db := setupOptionalJobPermissionDB(t)
	manager := aijob.NewAITaskMgr(&handler.RegisterConfig{AITaskCtrl: &profilingTaskController{}})
	createProfilingJob(t, manager, "jupyter")
	var record model.AITask
	if err := db.First(&record).Error; err != nil {
		t.Fatal(err)
	}
	pod := record.PodTemplate.Data()
	assertOptionalJobProfiling(t, &pod)
	sc := pod.Containers[0].SecurityContext
	if sc.RunAsUser == nil || *sc.RunAsUser != 0 {
		t.Fatal("changed the existing EMIAS execution user")
	}
}

func TestSEACSCreatesProfilingTemplate(t *testing.T) {
	setupOptionalJobPermissionDB(t)
	scheme := runtime.NewScheme()
	if err := recommendapi.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	manager := spjob.NewSparseJobMgr(&handler.RegisterConfig{Client: client})
	createProfilingJob(t, manager, "training")
	var jobs recommendapi.RecommendDLJobList
	if err := client.List(context.Background(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("created %d jobs, want 1", len(jobs.Items))
	}
	pod := &jobs.Items[0].Spec.Template.Spec
	assertOptionalJobProfiling(t, pod)
	if pod.Containers[0].SecurityContext.RunAsUser != nil {
		t.Fatal("profiling overrode the SEACS image user")
	}
}

func TestOptionalBackendsInjectGrantedSysAdmin(t *testing.T) {
	for _, backend := range []string{"emias-jupyter", "emias-custom", "seacs"} {
		t.Run(backend, func(t *testing.T) {
			db := setupOptionalJobPermissionDB(t)
			if err := db.Model(&model.User{}).Where("id = ?", 42).Update("container_sys_admin_enabled", true).Error; err != nil {
				t.Fatal(err)
			}
			var pod corev1.PodSpec
			if backend == "seacs" {
				scheme := runtime.NewScheme()
				if err := recommendapi.AddToScheme(scheme); err != nil {
					t.Fatal(err)
				}
				client := fake.NewClientBuilder().WithScheme(scheme).Build()
				manager := spjob.NewSparseJobMgr(&handler.RegisterConfig{Client: client})
				createProfilingJob(t, manager, "training", true)
				var jobs recommendapi.RecommendDLJobList
				if err := client.List(t.Context(), &jobs); err != nil {
					t.Fatal(err)
				}
				if len(jobs.Items) != 1 {
					t.Fatalf("created %d jobs", len(jobs.Items))
				}
				pod = jobs.Items[0].Spec.Template.Spec
			} else {
				manager := aijob.NewAITaskMgr(&handler.RegisterConfig{AITaskCtrl: &profilingTaskController{}})
				endpoint := "jupyter"
				if backend == "emias-custom" {
					endpoint = "training"
				}
				createProfilingJob(t, manager, endpoint, true)
				var record model.AITask
				if err := db.First(&record).Error; err != nil {
					t.Fatal(err)
				}
				pod = record.PodTemplate.Data()
			}
			if len(pod.Containers) != 1 || !slices.Contains(pod.Containers[0].SecurityContext.Capabilities.Add, "SYS_ADMIN") || slices.Contains(pod.Containers[0].SecurityContext.Capabilities.Add, "PERFMON") {
				t.Fatalf("grant missing from %s: %+v", backend, pod)
			}
		})
	}
}

func TestOptionalBackendsRejectUnauthorizedSysAdmin(t *testing.T) {
	for _, backend := range []string{"emias-jupyter", "emias-custom", "seacs"} {
		t.Run(backend, func(t *testing.T) {
			db := setupOptionalJobPermissionDB(t)
			var manager handler.Manager
			endpoint := "training"
			scheme := runtime.NewScheme()
			if err := recommendapi.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			if backend == "seacs" {
				manager = spjob.NewSparseJobMgr(&handler.RegisterConfig{Client: client})
			} else {
				manager = aijob.NewAITaskMgr(&handler.RegisterConfig{AITaskCtrl: &profilingTaskController{}})
				if backend == "emias-jupyter" {
					endpoint = "jupyter"
				}
			}
			body := strings.Replace(profilingRequest, `"name":`, `"sysAdminEnabled": true, "name":`, 1)
			response := profilingJobRequest(t, manager, endpoint, body)
			if response.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
			}
			var count int64
			if err := db.Model(&model.AITask{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("unauthorized task was persisted")
			}
			var jobs recommendapi.RecommendDLJobList
			if err := client.List(t.Context(), &jobs); err != nil {
				t.Fatal(err)
			}
			if len(jobs.Items) != 0 {
				t.Fatal("unauthorized task was submitted")
			}
		})
	}
}
