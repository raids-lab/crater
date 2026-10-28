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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

const emptyJSONArray = "[]"

func TestLegacyTemplateListAndExplicitPageShareRoute(t *testing.T) {
	router, _ := newJobTemplateRoutesTestRouter(t)
	for id := uint(4); id <= 14; id++ {
		if err := query.Jobtemplate.WithContext(t.Context()).Create(&model.Jobtemplate{
			Model: gorm.Model{ID: id}, Name: fmt.Sprintf("legacy-template-%d", id), Template: "{}", UserID: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	legacy := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list")
	var list resputil.Response[[]JobTemplateresp]
	if err := json.Unmarshal(legacy.Body.Bytes(), &list); err != nil || len(list.Data) != 14 {
		t.Fatalf("legacy complete array was lost: %s; %v", legacy.Body.String(), err)
	}
	pageResponse := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list?page=2&page_size=5")
	var page resputil.Response[resputil.Page[JobTemplateresp]]
	if err := json.Unmarshal(pageResponse.Body.Bytes(), &page); err != nil || len(page.Data.Items) != 5 || page.Data.Total != 14 {
		t.Fatalf("explicit page was lost: %s; %v", pageResponse.Body.String(), err)
	}
	if _, err := query.Jobtemplate.WithContext(t.Context()).Where(query.Jobtemplate.ID.IsNotNull()).Delete(); err != nil {
		t.Fatal(err)
	}
	empty := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list")
	var result map[string]json.RawMessage
	if err := json.Unmarshal(empty.Body.Bytes(), &result); err != nil || string(result["data"]) != emptyJSONArray {
		t.Fatalf("empty legacy data must be []: %s", empty.Body.String())
	}
}

func TestLegacyNodePodsRemainCompleteAndRejectPartialResults(t *testing.T) {
	for _, test := range []struct {
		name   string
		count  int
		next   string
		failed bool
	}{
		{name: "more than one snapshot page", count: 55},
		{name: "empty", count: 0},
		{name: "unexpected continuation", count: 5, next: "not-complete"},
		{name: "Kubernetes failure", failed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()
			requests := 0
			kubeClient.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				requests++
				options := action.(clienttesting.ListActionImpl).GetListOptions()
				if options.Limit != 0 || options.Continue != "" || options.FieldSelector != "spec.nodeName=node-a" {
					t.Fatalf("legacy query changed scope or added truncation: %+v", options)
				}
				if test.failed {
					return true, nil, bizerr.Internal.K8sServiceError.New("test list failure")
				}
				items := make([]corev1.Pod, test.count)
				for i := range items {
					items[i].ObjectMeta = metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", i), Namespace: "test"}
				}
				return true, &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: test.next}, Items: items}, nil
			})
			handler := &NodeMgr{nodeClient: &crclient.NodeClient{KubeClient: kubeClient}}
			ctx, recorder := snapshotContext(t, "/v1/nodes/node-a/pods", snapshotTestToken())
			ctx.Params = gin.Params{{Key: "name", Value: "node-a"}}
			handler.GetPodsForNode(ctx)
			if requests != 1 {
				t.Fatalf("legacy read duplicated Kubernetes queries: %d", requests)
			}
			if test.failed || test.next != "" {
				if recorder.Code != http.StatusInternalServerError {
					t.Fatalf("partial or failed read must fail: %s", recorder.Body.String())
				}
				return
			}
			var response resputil.Response[[]crclient.Pod]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || len(response.Data) != test.count || response.Data == nil {
				t.Fatalf("legacy Pod array changed: %s; %v", recorder.Body.String(), err)
			}
		})
	}
}

func newLegacyDatabaseListsRouter(t *testing.T) *gin.Engine {
	t.Helper()
	router := newDatasetPageTestRouter(t, []uint{1})
	db := compatibilityFixtureDB(t)
	if err := db.AutoMigrate(&model.Account{}, &model.UserAccount{}, &model.Resource{}, &model.ApprovalOrder{}, &model.GpuAnalysis{}, &model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}
	// SQLite index names are database-wide; use only the joined job projection
	// to avoid the existing Job/ModelDownload "status" index name collision.
	if err := db.Table("jobs").AutoMigrate(&struct {
		JobName, Name, JobType, Resources, Nodes, Status string
		LockedTimestamp                                  *time.Time
		DeletedAt                                        gorm.DeletedAt
	}{}); err != nil {
		t.Fatal(err)
	}
	for id := uint(1); id <= 14; id++ {
		if id > 2 {
			if err := db.Create(&model.User{Model: gorm.Model{ID: id}, Name: fmt.Sprintf("user-%02d", id), Space: fmt.Sprintf("/tmp/compat-user-%d", id), Role: model.RoleUser, Status: model.StatusActive}).Error; err != nil {
				t.Fatal(err)
			}
		}
		for _, row := range []any{
			&model.Account{Model: gorm.Model{ID: id}, Name: fmt.Sprintf("account-%d", id), Nickname: "compat", Space: fmt.Sprintf("/tmp/compat-account-%d", id)},
			&model.UserAccount{UserID: id, AccountID: 1, Role: model.RoleUser, AccessMode: model.AccessMode(2)},
			&model.Resource{Model: gorm.Model{ID: id}, ResourceName: fmt.Sprintf("resource-%d", id), ResourceType: "cpu", Format: "count"},
			&model.ApprovalOrder{Model: gorm.Model{ID: id}, Name: fmt.Sprintf("order-%d", id), CreatorID: 1},
			&model.GpuAnalysis{ID: id, UserID: 1, JobName: fmt.Sprintf("job-%d", id)},
		} {
			if err := db.Create(row).Error; err != nil {
				t.Fatal(err)
			}
		}
		if err := db.Create(&model.Dataset{Model: gorm.Model{ID: id + 10}, Name: fmt.Sprintf("dataset-%d", id), Type: model.DataTypeDataset, UserID: 1}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.UserDataset{UserID: 1, DatasetID: id + 10}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.ApprovalOrder{Name: "foreign-order", CreatorID: 2}).Error; err != nil {
		t.Fatal(err)
	}
	account := &AccountMgr{}
	router.GET("/compat/accounts", account.AdminListAccounts)
	router.GET("/compat/users", (&UserMgr{}).AdminListUsers)
	router.GET("/compat/members/:aid", account.UserListAccountMembers)
	router.GET("/compat/admin-members/:aid", account.AdminListAccountMembers)
	router.GET("/compat/resources", func(c *gin.Context) { (&ResourceMgr{}).listResourcePageWithDB(c, db) })
	router.GET("/compat/orders", (&ApprovalOrderMgr{}).GetMyApprovalOrders)
	router.GET("/compat/admin-orders", (&ApprovalOrderMgr{}).ListAllApprovalOrders)
	router.GET("/compat/analyses", (&GpuAnalysisMgr{}).ListAnalyses)
	return router
}

func TestLegacyDatabaseListsSharePermissionScopedPages(t *testing.T) {
	router := newLegacyDatabaseListsRouter(t)
	for _, tc := range []struct {
		path  string
		total int
	}{
		{"/compat/accounts", 14}, {"/compat/users", 14},
		{"/compat/members/1", 14}, {"/compat/admin-members/1", 14},
		{"/compat/resources", 14}, {"/compat/orders", 14},
		{"/compat/admin-orders", 15}, {"/compat/analyses", 14},
		{"/v1/dataset/mydataset", 15}, {"/v1/admin/dataset/alldataset", 21},
	} {
		t.Run(tc.path, func(t *testing.T) {
			assertLegacyAndExplicitListPage(t, router, tc.path, tc.total)
		})
	}
}

func TestLegacyDatabaseListsRespectScopes(t *testing.T) {
	router := newLegacyDatabaseListsRouter(t)
	for _, suffix := range []string{"", "?page=1&page_size=3"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/compat/members/2"+suffix, http.NoBody))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("account membership denial changed: HTTP %d; %s", recorder.Code, recorder.Body.String())
		}
	}
	// This user has no visible datasets; neither mode may expose user 1's rows.
	for _, suffix := range []string{"", "?page=1&page_size=3"} {
		ctx, recorder := snapshotContext(t, "/v1/dataset/mydataset"+suffix, util.JWTMessage{UserID: 14, AccountID: 14, RolePlatform: model.RoleUser})
		(&DatasetMgr{}).GetDatasets(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("empty dataset read failed: %s", recorder.Body.String())
		}
		var response resputil.Response[json.RawMessage]
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if suffix == "" && string(response.Data) != emptyJSONArray {
			t.Fatalf("empty legacy array changed: %s", response.Data)
		}
		if suffix != "" {
			var page resputil.Page[json.RawMessage]
			if err := json.Unmarshal(response.Data, &page); err != nil || page.Total != 0 || len(page.Items) != 0 || page.Items == nil {
				t.Fatalf("empty page leaked inaccessible datasets: %s; %v", response.Data, err)
			}
		}
	}
}

func TestLegacyDownloadsPreserveArrayAndHistoricalPage(t *testing.T) {
	router := newDatasetPageTestRouter(t, nil)
	if err := compatibilityFixtureDB(t).AutoMigrate(&model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}
	for id := uint(3); id <= 27; id++ {
		if err := query.ModelDownload.WithContext(t.Context()).Create(&model.ModelDownload{
			Model: gorm.Model{ID: id}, Name: fmt.Sprintf("download-%d", id), CreatorID: 1,
			Category: model.DownloadCategoryModel, Source: model.ModelSourceHuggingFace, Status: model.ModelDownloadStatusReady,
		}); err != nil {
			t.Fatal(err)
		}
	}
	router.GET("/compat/downloads", (&ModelDownloadMgr{}).ListDownloads)
	for _, tc := range []struct {
		query string
		array bool
		size  int
		meta  bool
	}{
		{"", true, 27, false},
		{"?page=2&pageSize=5", false, 5, false},
		{"?page=2&page_size=5", false, 5, true},
		{"?page=1&pageSize=500", false, 27, false},
	} {
		t.Run(tc.query, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/compat/downloads"+tc.query, http.NoBody))
			var response resputil.Response[json.RawMessage]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || recorder.Code != http.StatusOK {
				t.Fatalf("download compatibility read failed: %s; %v", recorder.Body.String(), err)
			}
			var items []ModelDownloadResp
			if tc.array {
				if err := json.Unmarshal(response.Data, &items); err != nil {
					t.Fatal(err)
				}
			} else {
				items = assertDownloadCompatibilityPage(t, response.Data, tc.meta)
			}
			if len(items) != tc.size {
				t.Fatalf("download list truncated: got %d, want %d", len(items), tc.size)
			}
			for _, item := range items {
				if !item.CanViewLogs || item.UserInfo.Username != "alice" {
					t.Fatalf("download creator permissions changed: %+v", item)
				}
			}
		})
	}
}

func compatibilityFixtureDB(t *testing.T) *gorm.DB {
	t.Helper()
	connection, err := query.User.WithContext(t.Context()).UnderlyingDB().DB()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: connection}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	return db
}

func assertLegacyAndExplicitListPage(t *testing.T, router *gin.Engine, path string, total int) {
	t.Helper()
	legacy := httptest.NewRecorder()
	legacyRequest := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	legacyRequest.Header.Set("X-Crater-API-Version", "5")
	router.ServeHTTP(legacy, legacyRequest)
	var response resputil.Response[[]json.RawMessage]
	if err := json.Unmarshal(legacy.Body.Bytes(), &response); err != nil || legacy.Code != http.StatusOK || len(response.Data) != total {
		t.Fatalf("legacy list incomplete: %s; %v; want %d", legacy.Body.String(), err, total)
	}
	page := httptest.NewRecorder()
	pageRequest := httptest.NewRequest(http.MethodGet, path+"?page=2&page_size=3", http.NoBody)
	pageRequest.Header.Set("X-Crater-API-Version", "4")
	router.ServeHTTP(page, pageRequest)
	var paged resputil.Response[resputil.Page[json.RawMessage]]
	if err := json.Unmarshal(page.Body.Bytes(), &paged); err != nil || page.Code != http.StatusOK || paged.Data.Total != int64(total) || len(paged.Data.Items) != 3 || paged.Data.Page != 2 || paged.Data.PageSize != 3 {
		t.Fatalf("explicit page changed: %s; %v", page.Body.String(), err)
	}
}

func assertDownloadCompatibilityPage(t *testing.T, data json.RawMessage, metadata bool) []ModelDownloadResp {
	t.Helper()
	var page ModelDownloadListResp
	if err := json.Unmarshal(data, &page); err != nil || page.Total != 27 || page.Summary[string(model.ModelDownloadStatusReady)] != 27 {
		t.Fatalf("historical totals or summary changed: %s; %v", data, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if (fields["page"] != nil) != metadata || (fields["page_size"] != nil) != metadata {
		t.Fatalf("page metadata changed historical response: %s", data)
	}
	return page.Items
}
