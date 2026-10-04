package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raids-lab/crater/internal/kthena"

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

	req := nativeTestRequest(t)
	req.Replicas = 1
	req.Roles[0].Instances = 1

	if err := validateCreateKthenaReq(context.Background(), req, util.JWTMessage{}); err != nil {
		t.Fatalf("validateCreateKthenaReq() error = %v", err)
	}
	if req.Replicas != 1 {
		t.Fatalf("Replicas = %d, want 1", req.Replicas)
	}
	if req.Roles[0].Instances != 1 {
		t.Fatalf("Worker.Replicas = %d, want 1", req.Roles[0].Instances)
	}
	if req.ServedModel != "Qwen2.5-0.5B-Instruct" {
		t.Fatalf("served-model-name = %q", req.ServedModel)
	}
}

func TestValidateCreateKthenaReqRejectsUnsupportedV1Backend(t *testing.T) {
	t.Parallel()

	for _, backendType := range []string{"MindIE", "vLLMDisaggregated"} {
		t.Run(backendType, func(t *testing.T) {
			t.Parallel()
			req := nativeTestRequest(t)
			req.BackendType = backendType

			if err := validateCreateKthenaReq(context.Background(), req, util.JWTMessage{}); err == nil {
				t.Fatal("validateCreateKthenaReq() error = nil, want unsupported backend error")
			}
		})
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

func TestKthenaServiceToRespIncludesOwnerUserInfo(t *testing.T) {
	t.Parallel()

	req := nativeTestRequest(t)
	obj, err := buildModelServing(
		req,
		util.JWTMessage{Username: kthenaConversationTestUsername},
		"crater-workspace",
		"downloader:test",
	)
	if err != nil {
		t.Fatal(err)
	}
	mgr := &KthenaMgr{client: controllerfake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()}

	resp, err := mgr.inferenceServiceToResp(context.Background(), obj)
	if err != nil {
		t.Fatalf("inferenceServiceToResp() error = %v", err)
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

	resources := []kthena.RoleSpec{{Name: "server", Instances: 1, NodesPerInstance: 1}}
	if got := runtimeAwareInferencePhase(kthenaPhaseReady, resources, 1, nil); got != kthenaPhaseDegraded {
		t.Fatalf("runtimeAwareInferencePhase() = %q, want Degraded", got)
	}
	if got := runtimeAwareInferencePhase(kthenaPhaseReady, resources, 1, []KthenaRuntimePod{{Role: "server", Entry: true, Ready: true}}); got != kthenaPhaseReady {
		t.Fatalf("runtimeAwareInferencePhase() = %q, want Ready", got)
	}
	if got := runtimeAwareInferencePhase(kthenaPhaseProgressing, resources, 1, nil); got != kthenaPhaseProgressing {
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
