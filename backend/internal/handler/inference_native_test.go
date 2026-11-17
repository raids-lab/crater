package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
)

func nativeTestRequest(t *testing.T) *CreateKthenaReq {
	t.Helper()
	req := &CreateKthenaReq{
		Name:        "native-qwen",
		ModelSource: inferenceModelSourceExternal,
		ModelURI:    kthenaInferenceTestModelURI,
		BackendType: "vLLM",
		Layout:      "combined",
		Replicas:    2,
		Port:        9000,
		Roles: []kthena.RoleSpec{
			{
				Name:             "server",
				Instances:        3,
				Execution:        "single",
				NodesPerInstance: 1,
				TensorParallel:   1,
				PipelineParallel: 1,
				EngineOptions:    map[string]string{},
				Entry: kthena.PodProfile{
					Image:    kthenaInferenceTestImage,
					CPU:      "2",
					Memory:   "4Gi",
					GPU:      "1",
					GPUModel: "nvidia.com/gpu",
				},
			},
		},
	}
	if err := validateCreateKthenaReq(t.Context(), req, util.JWTMessage{}); err != nil {
		t.Fatal(err)
	}
	return req
}

func nativeTestClient(objects ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	for _, gvk := range []schema.GroupVersionKind{kthena.ServingGVK, kthena.ServerGVK, kthena.RouteGVK} {
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(gvk.GroupVersion().WithKind(gvk.Kind+"List"), &unstructured.UnstructuredList{})
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func nativeTestServing(t *testing.T, req *CreateKthenaReq) *unstructured.Unstructured {
	t.Helper()
	obj, err := buildModelServing(req, util.JWTMessage{UserID: 1, AccountID: 2, Username: kthenaConversationTestUsername}, "jobs", "")
	if err != nil {
		t.Fatal(err)
	}
	obj.SetUID(types.UID("native-uid"))
	return obj
}

//nolint:gocyclo // Check the cross-resource contract together, including pod/route/probe consistency.
func TestKthenaNativeResourceContract(t *testing.T) {
	req := nativeTestRequest(t)
	req.Roles[0].Entry.Selectors = []corev1.NodeSelectorRequirement{{Key: "gpu", Operator: corev1.NodeSelectorOpExists}}
	req.Roles[0].Entry.Tolerations = []corev1.Toleration{{Key: "gpu", Operator: corev1.TolerationOpExists}}
	serving := nativeTestServing(t, req).DeepCopy()
	backend, err := kthena.ReadDeployment(serving)
	if err != nil {
		t.Fatal(err)
	}
	if backend.Replicas != int64(2) {
		t.Fatalf("backend = %v", backend)
	}
	roles, _, _ := unstructured.NestedSlice(serving.Object, "spec", "template", "roles")
	role := roles[0].(map[string]any)
	spec, _, _ := unstructured.NestedMap(role, "entryTemplate", "spec")
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var pod corev1.PodSpec
	if err := json.Unmarshal(raw, &pod); err != nil {
		t.Fatal(err)
	}
	if len(pod.SchedulingGates) != 0 {
		t.Fatal("disabled billing must not block scheduling")
	}
	engine := pod.Containers[0]
	if role["replicas"] != int64(3) || role["workerReplicas"] != int64(0) {
		t.Fatalf("role replicas = %v", role)
	}
	if engine.ReadinessProbe.HTTPGet.Port.IntVal != 9000 || engine.StartupProbe.HTTPGet.Port.IntVal != 9000 {
		t.Fatal("engine probes do not follow configured port")
	}
	if pod.Affinity == nil || len(pod.Tolerations) != 1 || len(pod.InitContainers) != 1 {
		t.Fatalf("pod = %+v", pod)
	}
	download := pod.InitContainers[0]
	if !download.Resources.Requests.Cpu().Equal(*engine.Resources.Requests.Cpu()) ||
		!download.Resources.Requests.Memory().Equal(*engine.Resources.Requests.Memory()) ||
		len(download.Resources.Requests) != 2 {
		t.Fatal("downloader has an unaccounted resource budget")
	}
	children := buildKthenaNetworking(serving, req)
	for _, child := range children {
		child = child.DeepCopy()
		if !isRelatedKthenaObject(child, serving) {
			t.Fatalf("%s missing exact ownership", child.GetKind())
		}
	}
	port, _, _ := unstructured.NestedInt64(children[0].Object, "spec", "workloadPort", "port")
	if port != 9000 {
		t.Fatalf("ModelServer port = %d", port)
	}
	selectors, _, _ := unstructured.NestedStringMap(children[0].Object, "spec", "workloadSelector", "matchLabels")
	labels, _, _ := unstructured.NestedStringMap(role, "entryTemplate", "metadata", "labels")
	if selectors[kthena.PodDeploymentLabel] == "" || selectors[kthena.PodDeploymentLabel] != labels[kthena.PodDeploymentLabel] {
		t.Fatal("pod selection mismatch")
	}
	// An optional export supports checking these exact objects against the pinned upstream CRDs.
	if dir := os.Getenv("CRATER_KTHENA_MANIFEST_DIR"); dir != "" {
		for _, obj := range append([]*unstructured.Unstructured{serving}, children...) {
			data, err := json.MarshalIndent(obj.Object, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			// #nosec G703 -- Opt-in test export, fixed resource kinds and a local test-runner directory.
			if err := os.WriteFile(filepath.Join(dir, obj.GetKind()+".json"), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestKthenaNativePVCModelUsesMountedSource(t *testing.T) {
	req := nativeTestRequest(t)
	req.ModelSource = inferenceModelSourcePlatform
	req.ModelPVC = "models"
	req.ModelSubPath = "users/alice/qwen"
	req.CacheURI = "pvc://models"
	req.ModelURI = "pvc:///models/users/alice/qwen"
	serving := nativeTestServing(t, req)
	roles, _, _ := unstructured.NestedSlice(serving.Object, "spec", "template", "roles")
	spec, _, _ := unstructured.NestedMap(roles[0].(map[string]any), "entryTemplate", "spec")
	init, _, _ := unstructured.NestedSlice(spec, "initContainers")
	if len(init) != 0 {
		t.Fatal("existing PVC model was downloaded again")
	}
	containers, _, _ := unstructured.NestedSlice(spec, "containers")
	command, _, _ := unstructured.NestedSlice(containers[0].(map[string]any), "command")
	mounts, _, _ := unstructured.NestedSlice(containers[0].(map[string]any), "volumeMounts")
	modelMount := mounts[0].(map[string]any)
	if modelMount["subPath"] != req.ModelSubPath || modelMount["readOnly"] != true {
		t.Fatalf("model mount must isolate its read-only directory: %v", modelMount)
	}
	var launch struct {
		Argv []string `json:"argv"`
	}
	if err := json.Unmarshal([]byte(command[4].(string)), &launch); err != nil {
		t.Fatal(err)
	}
	if launch.Argv[4] != "/model-cache" {
		t.Fatalf("command = %v", command)
	}
	req.ModelSubPath = "../other/qwen"
	if _, err := buildModelServing(req, util.JWTMessage{}, "jobs", ""); err == nil {
		t.Fatal("unmounted PVC path accepted")
	}
}

func TestKthenaNativeLifecycleAndConversation(t *testing.T) {
	req := nativeTestRequest(t)
	base := nativeTestClient()
	c := interceptor.NewClient(base,
		interceptor.Funcs{Create: func(ctx context.Context,
			c client.WithWatch,
			obj client.Object,
			opts ...client.CreateOption) error {
			if obj.GetObjectKind().GroupVersionKind() == kthena.ServingGVK {
				obj.SetUID(types.UID("created-uid"))
			}
			return c.Create(ctx, obj, opts...)
		}})
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	obj, err := mgr.createNativeKthenaService(t.Context(), req, util.JWTMessage{UserID: 1, AccountID: 2})
	if err != nil {
		t.Fatal(err)
	}
	listed,
		err := mgr.listKthenaServices(t.Context(),
		client.MatchingLabels{inferenceServiceLabelManagedBy: inferenceServiceManagedByValue})
	if err != nil || len(listed) != 1 || len(listed[0].Resources) != 0 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	raw, err := mgr.rawKthenaResources(t.Context(), obj)
	if err != nil || len(raw["items"].([]any)) != 3 {
		t.Fatalf("raw resources = %v, %v", raw, err)
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	util.SetJWTContext(ctx, util.JWTMessage{UserID: 1, AccountID: 2})
	scope, err := kthenaConversationScopeFromDeployment(ctx, obj)
	if err != nil || scope.RouteModelName != req.Name || scope.ModelName != req.ServedModel {
		t.Fatalf("scope = %+v, %v", scope, err)
	}
	loaded, err := kthena.GetDeployment(t.Context(), c, client.ObjectKeyFromObject(obj))
	if err != nil || loaded.GetKind() != kthena.ServingKind {
		t.Fatalf("load = %v, %v", loaded, err)
	}
}

func TestKthenaNativeCreationConflictRollsBackOnlyNewRoot(t *testing.T) {
	for _, gvk := range []schema.GroupVersionKind{kthena.ServerGVK, kthena.RouteGVK} {
		t.Run(gvk.Kind, func(t *testing.T) {
			req := nativeTestRequest(t)
			foreign := &unstructured.Unstructured{}
			foreign.SetGroupVersionKind(gvk)
			foreign.SetName(req.Name)
			foreign.SetNamespace("jobs")
			foreign.SetUID(types.UID("foreign-uid"))
			c := nativeTestClient(foreign)
			mgr := &KthenaMgr{client: c, namespace: "jobs"}
			if _, err := mgr.createNativeKthenaService(t.Context(), req, util.JWTMessage{}); !apierrors.IsAlreadyExists(err) {
				t.Fatalf("create error = %v", err)
			}
			root := &unstructured.Unstructured{}
			root.SetGroupVersionKind(kthena.ServingGVK)
			if err := c.Get(t.Context(), client.ObjectKeyFromObject(foreign), root); !apierrors.IsNotFound(err) {
				t.Fatalf("new root not rolled back: %v", err)
			}
			preserved := &unstructured.Unstructured{}
			preserved.SetGroupVersionKind(gvk)
			if err := c.Get(t.Context(),
				client.ObjectKeyFromObject(foreign),
				preserved); err != nil ||
				preserved.GetUID() != foreign.GetUID() {
				t.Fatal("foreign resource changed")
			}
		})
	}
}

func TestKthenaDoesNotOverwriteExistingDeployment(t *testing.T) {
	req := nativeTestRequest(t)
	old := nativeTestServing(t, req)
	c := nativeTestClient(old)
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	if _,
		err := mgr.createNativeKthenaService(t.Context(),
		req,
		util.JWTMessage{UserID: 1,
			AccountID: 2}); !apierrors.IsAlreadyExists(err) {
		t.Fatalf("deployment conflict = %v", err)
	}
	loaded, err := kthena.GetDeployment(t.Context(), c, client.ObjectKeyFromObject(old))
	if err != nil || !reflect.DeepEqual(loaded.Object["spec"], old.Object["spec"]) {
		t.Fatal("existing deployment changed")
	}
}

func TestKthenaNativeOwnerUIDMustMatch(t *testing.T) {
	req := nativeTestRequest(t)
	serving := nativeTestServing(t, req)
	route := buildKthenaNetworking(serving, req)[1]
	route.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: serving.GetAPIVersion(),
		Kind: serving.GetKind(),
		Name: serving.GetName(),
		UID:  types.UID("old-uid")}})
	if isRelatedKthenaObject(route, serving) {
		t.Fatal("stale owner matched")
	}
}

func TestKthenaNativeConversationTurnRoutesByDeploymentName(t *testing.T) {
	req := nativeTestRequest(t)
	req.Name = kthenaConversationTestService
	serving := nativeTestServing(t, req)
	serving.SetNamespace(kthenaConversationTestNamespace)
	router, _, _ := newKthenaConversationTestRouter(
		t,
		func(_ context.Context, _, _ string, body []byte, _ http.Header) ([]byte, error) {
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["model"] != req.Name {
				t.Fatalf("routed model = %v", payload["model"])
			}
			return []byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`), nil
		},
		serving,
	)
	response := requestKthenaConversationTurn(t, router,
		"/v1/kthena/inference-services/"+req.Name+"/conversations/turns",
		`{"content":"hello","clientTurnId":"native-turn"}`)
	if response.Conversation.SessionID == "" {
		t.Fatal("native conversation was not persisted")
	}
}

func TestKthenaNativeDeletionUsesRootIdentity(t *testing.T) {
	req := nativeTestRequest(t)
	serving := nativeTestServing(t, req)
	c := nativeTestClient(serving)
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/", http.NoBody)
	ctx.Params = gin.Params{{Key: "name", Value: req.Name}}
	util.SetJWTContext(ctx, util.JWTMessage{UserID: 1, AccountID: 2})
	mgr.deleteKthenaService(ctx, false)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", recorder.Code, recorder.Body.String())
	}
	if _, err := kthena.GetDeployment(t.Context(), c, client.ObjectKeyFromObject(serving)); !apierrors.IsNotFound(err) {
		t.Fatalf("root still exists: %v", err)
	}
}

func TestKthenaNativeMissingRouteCannotReportReady(t *testing.T) {
	req := nativeTestRequest(t)
	serving := nativeTestServing(t, req)
	serving.Object["status"] = map[string]any{"availableReplicas": req.Replicas}
	mgr := &KthenaMgr{client: nativeTestClient(serving)}
	response, err := mgr.inferenceServiceToResp(t.Context(), serving)
	if err != nil {
		t.Fatal(err)
	}
	if response.Phase != kthenaPhaseProgressing {
		t.Fatalf("phase without routing = %s", response.Phase)
	}
}
