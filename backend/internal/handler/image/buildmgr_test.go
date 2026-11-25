package image

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

func TestBindGetKanikoRequestSupportsIDAndLegacyName(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantID   uint
		wantName string
	}{
		{name: "stable id", query: "id=42", wantID: 42},
		{name: "legacy name", query: "name=example-build", wantName: "example-build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+tt.query, http.NoBody)

			var request GetKanikoRequest
			if err := ctx.ShouldBindQuery(&request); err != nil {
				t.Fatalf("ShouldBindQuery() error = %v", err)
			}
			if request.ID != tt.wantID || request.ImagePackName != tt.wantName {
				t.Fatalf("request = %+v, want id=%d name=%q", request, tt.wantID, tt.wantName)
			}
		})
	}
}

func TestGetPodNameByImagePackNameReturnsEmptyWhenNameMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	mgr := &ImagePackMgr{}
	name, namespace, node := mgr.getPodNameByImagePackName(ctx, "")
	if name != "" || namespace != UserNameSpace || node != "" {
		t.Fatalf("result = (%q, %q, %q), want empty pod fields and namespace %q", name, namespace, node, UserNameSpace)
	}
}

func TestGetPodNameByImagePackNameReturnsEmptyWhenPodMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	mgr := &ImagePackMgr{imagepackClient: &crclient.ImagePackController{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
	}}
	name, namespace, node := mgr.getPodNameByImagePackName(ctx, "missing-image-pack")
	if name != "" || namespace != UserNameSpace || node != "" {
		t.Fatalf("result = (%q, %q, %q), want empty pod fields and namespace %q", name, namespace, node, UserNameSpace)
	}
}

func TestGetPodNameByImagePackNameReturnsPodMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	pod := &corev1.Pod{}
	pod.Name = "image-pack-pod"
	pod.Namespace = UserNameSpace
	pod.Labels = map[string]string{"job-name": "image-pack"}
	pod.Spec.NodeName = "node-a"
	mgr := &ImagePackMgr{imagepackClient: &crclient.ImagePackController{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build(),
	}}
	name, namespace, node := mgr.getPodNameByImagePackName(ctx, "image-pack")
	if name != pod.Name || namespace != UserNameSpace || node != pod.Spec.NodeName {
		t.Fatalf("result = (%q, %q, %q), want (%q, %q, %q)", name, namespace, node, pod.Name, UserNameSpace, pod.Spec.NodeName)
	}
}

func TestBuildKanikoDetailResponseUsesLoadedRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	description := "loaded before pod lookup"
	dockerfile := "FROM alpine"
	kaniko := &model.Kaniko{
		ImagePackName: "",
		ImageLink:     "registry.example/image:latest",
		Status:        model.BuildJobFinished,
		BuildSource:   model.Dockerfile,
		Description:   &description,
		Dockerfile:    &dockerfile,
	}

	// A nil client is intentional: an empty image-pack name must not trigger a
	// second database lookup or a Kubernetes call after the record was loaded.
	response := (&ImagePackMgr{}).buildKanikoDetailResponse(ctx, kaniko)
	if response.ImageLink != kaniko.ImageLink || response.Status != kaniko.Status || response.Description != description || response.Dockerfile != dockerfile {
		t.Fatalf("response did not preserve loaded record: %+v", response)
	}
	if response.PodName != "" || response.PodNameSpace != UserNameSpace || response.NodeName != "" {
		t.Fatalf("unexpected pod metadata: %+v", response)
	}
}

func TestImageBuildLegacyAndIDReadErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		failQuery  bool
		wantStatus int
		wantCode   bizerr.BizCode
		wantMsg    string
	}{
		{name: "legacy missing build", path: "/v1/images/getbyname?name=missing", wantStatus: http.StatusBadRequest, wantCode: 40001, wantMsg: "fetch kaniko by name failed, err record not found"},
		{name: "legacy inaccessible build", path: "/v1/images/getbyname?name=other-build", wantStatus: http.StatusBadRequest, wantCode: 40001, wantMsg: "fetch kaniko by name failed, err record not found"},
		{name: "legacy missing query", path: "/v1/images/getbyname", wantStatus: http.StatusBadRequest, wantCode: 40001, wantMsg: "either name or id is required"},
		{name: "legacy database failure", path: "/v1/images/getbyname?name=missing", failQuery: true, wantStatus: http.StatusBadRequest, wantCode: 40001, wantMsg: "fetch kaniko by name failed"},
		{name: "new missing build", path: "/v1/images/getbyid?id=99", wantStatus: http.StatusNotFound, wantCode: bizerr.NotFound.DataBaseNotFound, wantMsg: "image build not found"},
		{name: "new inaccessible build", path: "/v1/images/getbyid?id=2", wantStatus: http.StatusNotFound, wantCode: bizerr.NotFound.DataBaseNotFound, wantMsg: "image build not found"},
		{name: "admin missing build", path: "/v1/admin/images/getbyid?id=99", wantStatus: http.StatusNotFound, wantCode: bizerr.NotFound.DataBaseNotFound, wantMsg: "image build not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupKanikoListTestDB(t)
			createKanikoListTestUsers(t, db, "alice", "bob")
			createKanikoListTestRecords(t, db, model.Kaniko{Model: gorm.Model{ID: 2}, UserID: 2, ImagePackName: "other-build"})
			const cause = "isolated image build query failure"
			if tt.failQuery {
				if err := db.Callback().Query().Before("gorm:query").Register("test:image-build-read-failure", func(tx *gorm.DB) {
					_ = tx.AddError(bizerr.Internal.DatabaseError.New(cause))
				}); err != nil {
					t.Fatalf("register failure callback: %v", err)
				}
			}
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{UserID: 1, Username: "alice"})
				c.Next()
			})
			mgr := &ImagePackMgr{}
			mgr.RegisterProtected(router.Group("/v1/images"))
			mgr.RegisterAdmin(router.Group("/v1/admin/images"))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, http.NoBody))
			var response resputil.Response[any]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode failure response: %v", err)
			}
			if recorder.Code != tt.wantStatus || response.Code != tt.wantCode || response.Message != tt.wantMsg || response.Data != nil {
				t.Fatalf("HTTP %d response=%+v, want HTTP %d code=%d msg=%q and null data", recorder.Code, response, tt.wantStatus, tt.wantCode, tt.wantMsg)
			}
			if strings.Contains(recorder.Body.String(), cause) {
				t.Fatal("failure response exposed the database cause")
			}
		})
	}
}
