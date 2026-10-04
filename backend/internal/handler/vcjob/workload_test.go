package vcjob

import (
	"context"
	"testing"
	"time"

	"github.com/raids-lab/crater/internal/kthena"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
)

func TestWorkloadFromModelServing(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	serving := testModelServing("qwen-service", "1", "2", createdAt)
	serving.Object["spec"].(map[string]any)["replicas"] = int64(2)
	resources := serving.Object["spec"].(map[string]any)["template"].(map[string]any)["roles"].([]any)[0].(map[string]any)["entryTemplate"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)[0].(map[string]any)["resources"].(map[string]any)
	requests := resources["requests"].(map[string]any)
	limits := resources["limits"].(map[string]any)
	delete(requests, "nvidia.com/gpu")
	limits["nvidia.com/gpu"] = "1"

	workload := workloadFromKthenaDeployment(serving)
	if workload.WorkloadID != "kthena-inference:qwen-service" {
		t.Fatalf("workloadID = %q", workload.WorkloadID)
	}
	if workload.WorkloadKind != workloadKindKthenaInference || workload.Scheduler != VolcanoSchedulerName {
		t.Fatalf("kind/scheduler = %q/%q", workload.WorkloadKind, workload.Scheduler)
	}
	if workload.DetailPath != "/portal/inference-services/qwen-service" {
		t.Fatalf("detailPath = %q", workload.DetailPath)
	}
	if workload.JobType != workloadJobTypeModelDeploy ||
		workload.Status != workloadStatusRunning ||
		workload.StatusDetail != workloadPhaseReady {
		t.Fatalf("type/status/detail = %q/%q/%q", workload.JobType, workload.Status, workload.StatusDetail)
	}
	if workload.Model != "Qwen/Qwen3-4B" || workload.Owner != "alice" || workload.Queue != "research" {
		t.Fatalf("model/owner/queue = %q/%q/%q", workload.Model, workload.Owner, workload.Queue)
	}
	if !workload.CreationTimestamp.Time.Equal(createdAt) {
		t.Fatalf("createdAt = %v, want %v", workload.CreationTimestamp.Time, createdAt)
	}
	if gpu, ok := workload.Resources["nvidia.com/gpu"]; !ok || gpu.Value() != 2 {
		got := int64(0)
		if ok {
			got = gpu.Value()
		}
		t.Fatalf("gpu resource = %d, want 2", got)
	}
}

func TestWorkloadFromModelServingRequiresCurrentGeneration(t *testing.T) {
	t.Parallel()
	serving := testModelServing("pending", "1", "2", time.Now())
	serving.SetGeneration(2)
	workload := workloadFromKthenaDeployment(serving)
	if workload.StatusDetail != "Progressing" || workload.Status != workloadStatusPending {
		t.Fatalf("status = %s/%s", workload.Status, workload.StatusDetail)
	}
}

func TestFindKthenaWorkloadsScopesByUserAndAccountLabels(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"), &unstructured.UnstructuredList{})
	manager := &VolcanojobMgr{
		client: controllerfake.NewClientBuilder().WithScheme(scheme).WithObjects(
			testModelServing("owned", "11", "22", time.Now()),
			testModelServing("other-account", "11", "23", time.Now()),
			testModelServing("other-user", "12", "22", time.Now()),
		).Build(),
		workloadNamespace: "crater-workspace",
	}
	userID, accountID := uint(11), uint(22)
	workloads, err := manager.findKthenaWorkloads(context.Background(), jobListScope{
		UserID: &userID, AccountID: &accountID,
	}, &jobListQuery{}, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(workloads) != 1 || workloads[0].Name != "owned" {
		t.Fatalf("scoped workloads = %#v", workloads)
	}
}

func TestFindWorkloadsHidesKthenaWhenFeatureIsDisabled(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"), &unstructured.UnstructuredList{})
	manager := &VolcanojobMgr{
		client: controllerfake.NewClientBuilder().WithScheme(scheme).WithObjects(
			testModelServing("hidden", "11", "22", time.Now()),
		).Build(),
		workloadNamespace: "crater-workspace",
	}
	request := &jobListQuery{
		JobTypes:      []string{workloadJobTypeModelDeploy},
		WorkloadKinds: []string{workloadKindKthenaInference},
	}

	workloads, err := manager.findWorkloads(context.Background(), jobListScope{}, request, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(workloads) != 0 {
		t.Fatalf("disabled Kthena feature returned workloads: %#v", workloads)
	}
}

func TestUnifiedWorkloadQueryAcceptsModelDeploymentFilters(t *testing.T) {
	t.Parallel()
	request := &jobListQuery{JobTypes: []string{workloadJobTypeModelDeploy}, WorkloadKinds: []string{workloadKindKthenaInference}}
	if err := validateJobListEnums(request); err != nil {
		t.Fatal(err)
	}
	if includesVolcanoJobType(request.JobTypes) || !includesKthenaJobType(request.JobTypes) {
		t.Fatal("model-deployment should select only Kthena workloads")
	}
	if !workloadMatchesQuery(&WorkloadResp{
		WorkloadKind: workloadKindKthenaInference,
		JobType:      workloadJobTypeModelDeploy,
		ScheduleType: model.ScheduleTypeNormal,
		Status:       workloadStatusRunning,
	}, request, -1) {
		t.Fatal("Kthena workload did not match its model-deployment filter")
	}
}

func testModelServing(name, userID, accountID string, createdAt time.Time) *unstructured.Unstructured {
	serving := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{"replicas": int64(1), "schedulerName": "volcano", "template": map[string]any{"roles": []any{map[string]any{
			"name": "server", "replicas": int64(1), "workerReplicas": int64(0), "entryTemplate": map[string]any{"metadata": map[string]any{"annotations": map[string]any{kthena.RoleConfigAnnotation: `{"execution":"single","gpuModel":"nvidia.com/gpu"}`}}, "spec": map[string]any{"containers": []any{map[string]any{
				"name": kthena.EngineContainer, "resources": map[string]any{"requests": map[string]any{"cpu": "4", "memory": "16Gi", "nvidia.com/gpu": "1"}, "limits": map[string]any{}},
			}}}},
		}}}},
	}}
	serving.SetGeneration(1)
	available := int64(2)
	serving.Object["status"] = map[string]any{"observedGeneration": int64(1), "availableReplicas": available}
	serving.SetGroupVersionKind(kthena.ServingGVK)
	serving.SetName(name)
	serving.SetNamespace("crater-workspace")
	serving.SetCreationTimestamp(metav1.NewTime(createdAt))
	serving.SetLabels(map[string]string{
		workloadKthenaManagedByLabel: workloadKthenaManagedByValue,
		kthena.LayoutLabel:           kthena.NativeLayout,
		workloadKthenaUserIDLabel:    userID,
		workloadKthenaAccountIDLabel: accountID,
	})
	serving.SetAnnotations(map[string]string{
		workloadKthenaUserAnnotation: "alice",
		kthena.ModelURIAnnotation:    "hf://Qwen/Qwen3-4B",
		kthena.ModelAnnotation:       "Qwen/Qwen3-4B",
		workloadKthenaAccountAnno:    "research",
	})
	return serving
}
