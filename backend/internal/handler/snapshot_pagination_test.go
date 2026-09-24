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
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

func snapshotTestToken(userID, accountID uint) util.JWTMessage {
	return util.JWTMessage{
		UserID:            userID,
		AccountID:         accountID,
		RoleAccount:       model.RoleUser,
		RolePlatform:      model.RoleUser,
		AccountAccessMode: model.AccessModeRW,
		PublicAccessMode:  model.AccessModeRO,
	}
}

func setupSnapshotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open snapshot test database: %v", err)
	}
	// Create only the tables needed by the handler. User and Job both use the
	// historical SQLite index name "status", so create them sequentially and
	// remove the user index before creating the job table.
	if err := db.Migrator().CreateTable(&model.User{}, &model.Account{}); err != nil {
		t.Fatalf("create snapshot identity tables: %v", err)
	}
	if err := db.Exec("DROP INDEX IF EXISTS status").Error; err != nil {
		t.Fatalf("drop duplicate snapshot test index: %v", err)
	}
	if err := db.Migrator().CreateTable(&model.Job{}); err != nil {
		t.Fatalf("create snapshot job table: %v", err)
	}
	query.SetDefault(db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get snapshot test database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func createSnapshotJob(t *testing.T, db *gorm.DB, userID, accountID uint) {
	t.Helper()
	if err := db.Create(&model.User{
		Model:  gorm.Model{ID: userID},
		Name:   "snapshot-user",
		Role:   model.RoleUser,
		Status: model.StatusActive,
		Space:  "/tmp/snapshot-user",
	}).Error; err != nil {
		t.Fatalf("create snapshot user: %v", err)
	}
	if err := db.Create(&model.Account{
		Model:    gorm.Model{ID: accountID},
		Name:     "snapshot-account",
		Nickname: "Snapshot Account",
		Space:    "/tmp/snapshot-account",
	}).Error; err != nil {
		t.Fatalf("create snapshot account: %v", err)
	}
	attributes := datatypes.NewJSONType(&batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "snapshot-jobs",
			Labels: map[string]string{
				crclient.LabelKeyBaseURL: "snapshot-job-id",
			},
		},
	})
	if err := db.Create(&model.Job{
		Model:      gorm.Model{ID: 1},
		Name:       "snapshot-job",
		JobName:    "snapshot-job",
		UserID:     userID,
		AccountID:  accountID,
		JobType:    model.JobTypeCustom,
		Attributes: attributes,
	}).Error; err != nil {
		t.Fatalf("create snapshot job: %v", err)
	}
}

func snapshotContext(t *testing.T, method, path string, token util.JWTMessage) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, http.NoBody)
	util.SetJWTContext(ctx, token)
	return ctx, recorder
}

func TestListNodeSnapshotRejectsInvalidLimit(t *testing.T) {
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{KubeClient: fake.NewSimpleClientset()}}
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/resources/snapshot?limit=201", snapshotTestToken(1, 1))
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid node snapshot limit returned HTTP %d, want 400", recorder.Code)
	}
}

func TestListNodeSnapshotReturnsSnapshotPage(t *testing.T) {
	kubeClient := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-b"}},
	)
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{KubeClient: kubeClient}}
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/resources/snapshot?limit=1", snapshotTestToken(1, 1))
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("node snapshot returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var response resputil.Response[NodeSnapshotPage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode node snapshot response: %v", err)
	}
	if len(response.Data.Items) != 2 || response.Data.SnapshotID == "" {
		t.Fatalf("unexpected node snapshot page: %+v", response.Data)
	}
}

func TestListNodeSnapshotMapsExpiredCursor(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "nodes", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewResourceExpired("expired cursor")
	})
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{KubeClient: kubeClient}}
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/resources/snapshot?cursor=expired", snapshotTestToken(1, 1))
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expired node cursor returned HTTP %d, want 409", recorder.Code)
	}
	var response resputil.Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode expired node cursor response: %v", err)
	}
	if response.Code != 40905 {
		t.Fatalf("expired node cursor code = %d, want 40905", response.Code)
	}
}

func TestListJobPodsEnforcesOwnership(t *testing.T) {
	db := setupSnapshotTestDB(t)
	createSnapshotJob(t, db, 2, 1)
	handler := NewPodSnapshotHandler(&RegisterConfig{KubeClient: fake.NewSimpleClientset()})
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/vcjobs/snapshot-job/pods/snapshot", snapshotTestToken(1, 1))
	ctx.Params = gin.Params{{Key: "job", Value: "snapshot-job"}}
	handler.ListJobPods(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-user pod snapshot returned HTTP %d, want 403", recorder.Code)
	}
}

func TestListJobPodsReturnsSnapshotPage(t *testing.T) {
	db := setupSnapshotTestDB(t)
	createSnapshotJob(t, db, 1, 1)
	kubeClient := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-pod-a",
				Namespace: "snapshot-jobs",
				Labels:    map[string]string{crclient.LabelKeyBaseURL: "snapshot-job-id"},
			},
			Spec:   corev1.PodSpec{NodeName: "node-a"},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-pod",
				Namespace: "snapshot-jobs",
				Labels:    map[string]string{crclient.LabelKeyBaseURL: "other-job-id"},
			},
		},
	)
	handler := NewPodSnapshotHandler(&RegisterConfig{KubeClient: kubeClient})
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/vcjobs/snapshot-job/pods/snapshot", snapshotTestToken(1, 1))
	ctx.Params = gin.Params{{Key: "job", Value: "snapshot-job"}}
	handler.ListJobPods(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("pod snapshot returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var response resputil.Response[PodSnapshotPage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode pod snapshot response: %v", err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].Name != "snapshot-pod-a" || response.Data.SnapshotID == "" {
		t.Fatalf("unexpected pod snapshot page: %+v", response.Data)
	}
}

func TestListJobPodsMapsExpiredCursor(t *testing.T) {
	db := setupSnapshotTestDB(t)
	createSnapshotJob(t, db, 1, 1)
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "pods", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewResourceExpired("expired cursor")
	})
	handler := NewPodSnapshotHandler(&RegisterConfig{KubeClient: kubeClient})
	ctx, recorder := snapshotContext(t, http.MethodGet, "/v1/vcjobs/snapshot-job/pods/snapshot?cursor=expired", snapshotTestToken(1, 1))
	ctx.Params = gin.Params{{Key: "job", Value: "snapshot-job"}}
	handler.ListJobPods(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expired pod cursor returned HTTP %d, want 409", recorder.Code)
	}
	var response resputil.Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode expired pod cursor response: %v", err)
	}
	if response.Code != 40905 {
		t.Fatalf("expired pod cursor code = %d, want 40905", response.Code)
	}
}
