package image

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
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
