package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

const (
	kthenaInferenceTestServiceName = "qwen-demo"
	kthenaInferenceTestModelURI    = "hf://Qwen/Qwen2.5-0.5B-Instruct"
	kthenaInferenceTestImage       = "example.com/vllm:latest"
)

func TestValidateCreateKthenaReqV1Defaults(t *testing.T) {
	t.Parallel()

	req := &CreateKthenaReq{
		Name:        kthenaInferenceTestServiceName,
		ModelSource: inferenceModelSourceExternal,
		ModelURI:    kthenaInferenceTestModelURI,
		BackendType: kthenaBackendVLLM,
		Worker: KthenaWorkerReq{
			Image: "public.ecr.aws/q9t5s3a7/vllm-cpu-release-repo:latest",
		},
	}

	if err := validateCreateKthenaReq(context.Background(), req, util.JWTMessage{}); err != nil {
		t.Fatalf("validateCreateKthenaReq() error = %v", err)
	}
	if req.Replicas != 1 {
		t.Fatalf("Replicas = %d, want 1", req.Replicas)
	}
	if req.Worker.Replicas != 1 {
		t.Fatalf("Worker.Replicas = %d, want 1", req.Worker.Replicas)
	}
	if req.Worker.Config["served-model-name"] != "Qwen2.5-0.5B-Instruct" {
		t.Fatalf("served-model-name = %q", req.Worker.Config["served-model-name"])
	}
}

func TestValidateCreateKthenaReqRejectsUnsupportedV1Backend(t *testing.T) {
	t.Parallel()

	for _, backendType := range []string{"SGLang", "MindIE", "vLLMDisaggregated"} {
		t.Run(backendType, func(t *testing.T) {
			t.Parallel()
			req := &CreateKthenaReq{
				Name:        kthenaInferenceTestServiceName,
				ModelSource: inferenceModelSourceExternal,
				ModelURI:    kthenaInferenceTestModelURI,
				BackendType: backendType,
				Worker: KthenaWorkerReq{
					Image: kthenaInferenceTestImage,
				},
			}

			if err := validateCreateKthenaReq(context.Background(), req, util.JWTMessage{}); err == nil {
				t.Fatal("validateCreateKthenaReq() error = nil, want unsupported backend error")
			}
		})
	}
}

func TestBuildModelBoosterObjectUsesKthenaV1Replicas(t *testing.T) {
	t.Parallel()

	req := &CreateKthenaReq{
		Name:        kthenaInferenceTestServiceName,
		ModelSource: inferenceModelSourceExternal,
		ModelURI:    kthenaInferenceTestModelURI,
		BackendType: kthenaBackendVLLM,
		CacheURI:    kthenaDefaultCacheURI,
		Replicas:    3,
		Worker: KthenaWorkerReq{
			Image:    kthenaInferenceTestImage,
			Replicas: 1,
			Pods:     1,
			CPU:      "2",
			Memory:   kthenaDefaultWorkerMemory,
			Config:   map[string]string{"served-model-name": kthenaConversationTestService},
		},
	}

	obj := buildModelBoosterObject(req, util.JWTMessage{
		UserID:    1,
		AccountID: 2,
		Username:  kthenaConversationTestUsername,
	}, "crater-workspace")
	replicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "backend", "replicas")
	if err != nil || !found {
		t.Fatalf("spec.backend.replicas not found: found=%t err=%v", found, err)
	}
	if replicas != 3 {
		t.Fatalf("spec.backend.replicas = %d, want 3", replicas)
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "backend", "minReplicas"); found {
		t.Fatal("legacy spec.backend.minReplicas is present")
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "backend", "maxReplicas"); found {
		t.Fatal("legacy spec.backend.maxReplicas is present")
	}
	workers, found, err := unstructured.NestedSlice(obj.Object, "spec", "backend", "workers")
	if err != nil || !found || len(workers) != 1 {
		t.Fatalf("spec.backend.workers invalid: found=%t count=%d err=%v", found, len(workers), err)
	}
	worker, ok := workers[0].(map[string]any)
	if !ok {
		t.Fatalf("spec.backend.workers[0] type = %T, want map[string]any", workers[0])
	}
	if _, found := worker["affinity"]; found {
		t.Fatal("empty worker affinity is present")
	}
	if username := obj.GetAnnotations()[inferenceServiceAnnotationUsername]; username != kthenaConversationTestUsername {
		t.Fatalf("creator annotation = %q, want alice", username)
	}
}

func TestKthenaServiceOwnerUsesCreatorAnnotation(t *testing.T) {
	t.Parallel()

	obj := &unstructured.Unstructured{}
	obj.SetAnnotations(map[string]string{
		inferenceServiceAnnotationUsername: "  alice  ",
	})

	owner, userInfo := kthenaServiceOwner(obj)
	if owner != kthenaConversationTestUsername {
		t.Fatalf("owner = %q, want alice", owner)
	}
	if userInfo.Username != kthenaConversationTestUsername {
		t.Fatalf("userInfo.username = %q, want alice", userInfo.Username)
	}
	if userInfo.Nickname != "" {
		t.Fatalf("userInfo.nickname = %q, want empty", userInfo.Nickname)
	}
}

func TestModelBoosterToRespIncludesOwnerUserInfo(t *testing.T) {
	t.Parallel()

	req := &CreateKthenaReq{
		Name:        kthenaInferenceTestServiceName,
		ModelSource: inferenceModelSourceExternal,
		ModelURI:    kthenaInferenceTestModelURI,
		BackendType: kthenaBackendVLLM,
		CacheURI:    kthenaDefaultCacheURI,
		Replicas:    1,
		Worker: KthenaWorkerReq{
			Image:    kthenaInferenceTestImage,
			Replicas: 1,
			Pods:     1,
			CPU:      "2",
			Memory:   kthenaDefaultWorkerMemory,
			Config:   map[string]string{"served-model-name": kthenaConversationTestService},
		},
	}
	obj := buildModelBoosterObject(req, util.JWTMessage{Username: kthenaConversationTestUsername}, "crater-workspace")
	mgr := &KthenaMgr{client: controllerfake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()}

	resp, err := mgr.modelBoosterToResp(context.Background(), obj)
	if err != nil {
		t.Fatalf("modelBoosterToResp() error = %v", err)
	}
	if resp.Owner != kthenaConversationTestUsername {
		t.Fatalf("response owner = %q, want alice", resp.Owner)
	}
	if resp.UserInfo.Username != kthenaConversationTestUsername {
		t.Fatalf("response userInfo.username = %q, want alice", resp.UserInfo.Username)
	}
}

func TestRuntimeAwareInferencePhase(t *testing.T) {
	t.Parallel()

	resources := []KthenaResource{{Kind: kthenaKindModelServing}}
	if got := runtimeAwareInferencePhase(kthenaPhaseReady, resources, nil); got != kthenaPhaseDegraded {
		t.Fatalf("runtimeAwareInferencePhase() = %q, want Degraded", got)
	}
	if got := runtimeAwareInferencePhase(kthenaPhaseReady, resources, []KthenaRuntimePod{{Ready: true}}); got != kthenaPhaseReady {
		t.Fatalf("runtimeAwareInferencePhase() = %q, want Ready", got)
	}
	if got := runtimeAwareInferencePhase(kthenaPhaseProgressing, resources, nil); got != kthenaPhaseProgressing {
		t.Fatalf("runtimeAwareInferencePhase() = %q, want Progressing", got)
	}
}

func TestKthenaProxyHTTPStatus(t *testing.T) {
	t.Parallel()

	err := apierrors.NewNotFound(schema.GroupResource{Resource: "modelservers"}, "qwen")
	if got := kthenaProxyHTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("kthenaProxyHTTPStatus() = %d, want %d", got, http.StatusNotFound)
	}
	if got := kthenaProxyHTTPStatus(context.DeadlineExceeded); got != http.StatusBadGateway {
		t.Fatalf("kthenaProxyHTTPStatus() = %d, want %d", got, http.StatusBadGateway)
	}
}

func TestDiagnosticsFromPodEventsIgnoresStalePodEvents(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "qwen3test-backend1-0-leader-0-0",
		Namespace: "crater-workspace",
		UID:       types.UID("current-pod"),
	}}
	staleEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "stale-warning", Namespace: pod.Namespace},
		InvolvedObject: corev1.ObjectReference{
			Name: pod.Name,
			UID:  types.UID("previous-pod"),
		},
		Type:    corev1.EventTypeWarning,
		Reason:  "BackOff",
		Message: "Back-off restarting failed container engine",
	}
	currentEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "current-warning", Namespace: pod.Namespace},
		InvolvedObject: corev1.ObjectReference{
			Name: pod.Name,
			UID:  pod.UID,
		},
		Type:    corev1.EventTypeWarning,
		Reason:  "FailedScheduling",
		Message: "temporary scheduling warning",
	}
	mgr := &KthenaMgr{kubeClient: fake.NewSimpleClientset(staleEvent, currentEvent)}

	diagnostics := mgr.diagnosticsFromPodEvents(context.Background(), pod)
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics count = %d, want 1", len(diagnostics))
	}
	if diagnostics[0].Reason != currentEvent.Reason {
		t.Fatalf("diagnostic reason = %q, want %q", diagnostics[0].Reason, currentEvent.Reason)
	}
}

func TestKthenaInferenceRoutesRejectRequestsWhenFeatureIsDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:kthena_inference_routes_disabled?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}); err != nil {
		t.Fatal(err)
	}

	mgr := &KthenaMgr{configService: service.NewConfigService(query.Use(db))}
	router := gin.New()
	mgr.RegisterProtected(router.Group("/v1/kthena"))
	mgr.RegisterAdmin(router.Group("/v1/admin/kthena"))

	requests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/kthena/inference-services"},
		{http.MethodPost, "/v1/kthena/inference-services"},
		{http.MethodGet, "/v1/kthena/inference-services/qwen"},
		{http.MethodGet, "/v1/kthena/inference-services/qwen/yaml"},
		{http.MethodDelete, "/v1/kthena/inference-services/qwen"},
		{http.MethodPost, "/v1/kthena/inference-services/qwen/openai/v1/chat/completions"},
		{http.MethodGet, "/v1/admin/kthena/inference-services"},
		{http.MethodGet, "/v1/admin/kthena/inference-services/qwen"},
		{http.MethodGet, "/v1/admin/kthena/inference-services/qwen/yaml"},
		{http.MethodDelete, "/v1/admin/kthena/inference-services/qwen"},
	}
	for _, item := range requests {
		t.Run(item.method+" "+item.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), item.method, item.path, http.NoBody))
			if recorder.Code != http.StatusConflict {
				t.Fatalf("returned HTTP %d: %s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Code bizerr.BizCode `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != bizerr.Conflict.ResourceStatusError {
				t.Fatalf("response code = %d, want %d", response.Code, bizerr.Conflict.ResourceStatusError)
			}
		})
	}
}
