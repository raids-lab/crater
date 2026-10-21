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
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/indexer"
)

const jobSnapshotTestCursor = "next-job-pod"

func snapshotTestToken() util.JWTMessage {
	return util.JWTMessage{
		UserID:            1,
		AccountID:         1,
		RoleAccount:       model.RoleUser,
		RolePlatform:      model.RoleUser,
		AccountAccessMode: model.AccessModeRW,
		PublicAccessMode:  model.AccessModeRO,
	}
}

func snapshotContext(t *testing.T, path string, token util.JWTMessage) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, path, http.NoBody)
	util.SetJWTContext(ctx, token)
	return ctx, recorder
}

func TestListNodeSnapshotRejectsInvalidLimit(t *testing.T) {
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{KubeClient: fake.NewSimpleClientset()}}
	ctx, recorder := snapshotContext(t, "/v1/resources/snapshot?limit=201", snapshotTestToken())
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid node snapshot limit returned HTTP %d, want 400", recorder.Code)
	}
}

func TestListNodeSnapshotReturnsSnapshotPage(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
		options := action.(clienttesting.ListActionImpl).GetListOptions()
		if options.Limit != 1 || options.Continue != "" {
			t.Fatalf("unexpected Kubernetes list options: %+v", options)
		}
		return true, &corev1.NodeList{
			ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: "next-node"},
			Items:    []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}}},
		}, nil
	})
	cache := clientfake.NewClientBuilder().WithIndex(&corev1.Pod{}, indexer.PodNodeNameIndex,
		func(obj client.Object) []string { return []string{obj.(*corev1.Pod).Spec.NodeName} }).Build()
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{Client: cache, KubeClient: kubeClient}}
	ctx, recorder := snapshotContext(t, "/v1/resources/snapshot?limit=1", snapshotTestToken())
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("node snapshot returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var response resputil.Response[NodeSnapshotPage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode node snapshot response: %v", err)
	}
	if len(response.Data.Items) != 1 || response.Data.SnapshotID == "" || !response.Data.HasMore ||
		response.Data.NextCursor == nil || *response.Data.NextCursor != "next-node" {
		t.Fatalf("unexpected node snapshot page: %+v", response.Data)
	}
}

func TestNodePodsSnapshotForwardsCursorAndKeepsSnapshotIdentity(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		options := action.(clienttesting.ListActionImpl).GetListOptions()
		if options.Limit != 1 || options.FieldSelector != "spec.nodeName=node-a" {
			t.Fatalf("unexpected Kubernetes list options: %+v", options)
		}
		name, next := "pod-a", "next-pod"
		if options.Continue == "next-pod" {
			name, next = "node-pod-b", ""
		} else if options.Continue != "" {
			t.Fatalf("unexpected cursor %q", options.Continue)
		}
		return true, &corev1.PodList{
			ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: next},
			Items:    []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"}}},
		}, nil
	})
	handler := &NodeMgr{nodeClient: &crclient.NodeClient{KubeClient: kubeClient}}
	var snapshotID string
	for _, cursor := range []string{"", "next-pod"} {
		ctx, recorder := snapshotContext(t, "/v1/nodes/node-a/pods?limit=1&cursor="+cursor, snapshotTestToken())
		ctx.Params = gin.Params{{Key: "name", Value: "node-a"}}
		handler.GetPodsForNode(ctx)
		var response resputil.Response[PodSnapshotPage]
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		page := response.Data
		if recorder.Code != http.StatusOK || len(page.Items) != 1 || page.ResourceVersion != "100" ||
			page.SnapshotID == "" || page.HasMore != (cursor == "") {
			t.Fatalf("unexpected snapshot response: %s", recorder.Body.String())
		}
		if snapshotID != "" && snapshotID != page.SnapshotID {
			t.Fatal("snapshot identity changed between batches")
		}
		snapshotID = page.SnapshotID
	}
}

func TestListNodeSnapshotMapsExpiredCursor(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "nodes", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewResourceExpired("expired cursor")
	})
	handler := &ResourceMgr{nodeClient: &crclient.NodeClient{KubeClient: kubeClient}}
	ctx, recorder := snapshotContext(t, "/v1/resources/snapshot?cursor=expired", snapshotTestToken())
	handler.ListNodeSnapshot(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expired node cursor returned HTTP %d, want 409", recorder.Code)
	}
	var response resputil.Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode expired node cursor response: %v", err)
	}
	if response.Code != 40906 {
		t.Fatalf("expired node cursor code = %d, want 40906", response.Code)
	}
}

func newJobPodSnapshotTestHandler(t *testing.T) (*PodSnapshotHandler, *fake.Clientset, *gorm.DB) {
	t.Helper()
	db := newBillingHandlerTestDB(t, t.Name())
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	// Use only the read fixture's columns; SQLite index names are database-wide.
	if err := db.Exec(`CREATE TABLE jobs (
		id INTEGER PRIMARY KEY, job_name TEXT, user_id INTEGER, account_id INTEGER,
		attributes JSON, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	job := model.Job{
		JobName: "snapshot-job", UserID: 1, AccountID: 1,
		Attributes: datatypes.NewJSONType(&batch.Job{ObjectMeta: metav1.ObjectMeta{
			Namespace: "test", Labels: map[string]string{crclient.LabelKeyBaseURL: "snapshot-job"},
		}}),
	}
	if err := db.Exec("INSERT INTO jobs (job_name, user_id, account_id, attributes) VALUES (?, ?, ?, ?)",
		job.JobName, job.UserID, job.AccountID, job.Attributes).Error; err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewSimpleClientset()
	handler := &PodSnapshotHandler{kubeClient: kubeClient}
	kubeClient.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		options := action.(clienttesting.ListActionImpl).GetListOptions()
		if action.GetNamespace() != "test" || options.Limit != 1 ||
			options.LabelSelector != crclient.LabelKeyBaseURL+"=snapshot-job" {
			t.Fatalf("unexpected job Pod list scope: %+v", options)
		}
		name, next := "pod-a", jobSnapshotTestCursor
		if options.Continue == jobSnapshotTestCursor {
			name, next = "pod-b", ""
		}
		return true, &corev1.PodList{
			ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: next},
			Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "test", Labels: map[string]string{crclient.LabelKeyBaseURL: "snapshot-job"},
			}}},
		}, nil
	})
	return handler, kubeClient, db
}

func readJobPodSnapshotTestPage(
	t *testing.T, handler *PodSnapshotHandler, name, query string, token util.JWTMessage,
) (int, PodSnapshotPage) {
	t.Helper()
	ctx, recorder := snapshotContext(t, "/v1/vcjobs/"+name+"/pods/snapshot?limit=1&"+query, token)
	ctx.Params = gin.Params{{Key: "name", Value: name}}
	handler.ListJobPods(ctx)
	var response resputil.Response[PodSnapshotPage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return recorder.Code, response.Data
}

func TestJobPodSnapshotEnforcesOwnership(t *testing.T) {
	handler, kubeClient, _ := newJobPodSnapshotTestHandler(t)
	for _, tc := range []struct {
		name          string
		user, account uint
		role          model.Role
		want          int
	}{
		{"owner", 1, 1, model.RoleUser, http.StatusOK},
		{"another user", 2, 1, model.RoleUser, http.StatusForbidden},
		{"another account", 1, 2, model.RoleUser, http.StatusForbidden},
		{"administrator", 2, 2, model.RoleAdmin, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kubeClient.ClearActions()
			token := snapshotTestToken()
			token.UserID, token.AccountID, token.RolePlatform = tc.user, tc.account, tc.role
			code, _ := readJobPodSnapshotTestPage(t, handler, "snapshot-job", "", token)
			if code != tc.want {
				t.Fatalf("HTTP %d, want %d", code, tc.want)
			}
			if tc.want == http.StatusForbidden && len(kubeClient.Actions()) != 0 {
				t.Fatal("forbidden request reached Kubernetes")
			}
		})
	}
}

func TestJobPodSnapshotContinuesWithinStableScope(t *testing.T) {
	handler, _, _ := newJobPodSnapshotTestHandler(t)
	code, first := readJobPodSnapshotTestPage(t, handler, "snapshot-job", "", snapshotTestToken())
	if code != http.StatusOK || len(first.Items) != 1 || !first.HasMore || first.NextCursor == nil ||
		*first.NextCursor != jobSnapshotTestCursor {
		t.Fatalf("unexpected first page: HTTP %d %+v", code, first)
	}
	code, second := readJobPodSnapshotTestPage(t, handler, "snapshot-job", "cursor="+*first.NextCursor, snapshotTestToken())
	if code != http.StatusOK || len(second.Items) != 1 || second.Items[0].Name != "pod-b" || second.HasMore || second.SnapshotID != first.SnapshotID {
		t.Fatalf("unexpected continuation page: HTTP %d %+v", code, second)
	}
}

func TestJobPodSnapshotClassifiesReadErrors(t *testing.T) {
	handler, _, db := newJobPodSnapshotTestHandler(t)
	code, _ := readJobPodSnapshotTestPage(t, handler, "missing", "", snapshotTestToken())
	if code != http.StatusNotFound {
		t.Fatalf("missing job returned HTTP %d", code)
	}
	if err := db.Migrator().DropTable(&model.Job{}); err != nil {
		t.Fatal(err)
	}
	code, _ = readJobPodSnapshotTestPage(t, handler, "snapshot-job", "", snapshotTestToken())
	if code != http.StatusInternalServerError {
		t.Fatalf("database failure returned HTTP %d, want 500", code)
	}
}
