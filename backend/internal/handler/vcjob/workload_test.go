package vcjob

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
)

var workloadModelBoosterGVK = schema.GroupVersionKind{
	Group:   workloadModelBoosterListGVK.Group,
	Version: workloadModelBoosterListGVK.Version,
	Kind:    "ModelBooster",
}

func TestWorkloadFromModelBooster(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	booster := testModelBooster("qwen-service", "1", "2", workloadPhaseReady, createdAt)
	booster.Object["spec"].(map[string]any)["backend"].(map[string]any)["replicas"] = int64(2)
	resources := booster.Object["spec"].(map[string]any)["backend"].(map[string]any)["workers"].([]any)[0].(map[string]any)["resources"].(map[string]any)
	requests := resources["requests"].(map[string]any)
	limits := resources["limits"].(map[string]any)
	delete(requests, "nvidia.com/gpu")
	limits["nvidia.com/gpu"] = "1"

	workload := workloadFromModelBooster(booster)
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

func TestWorkloadFromModelBoosterRecognizesFailedCondition(t *testing.T) {
	t.Parallel()
	booster := testModelBooster("failed-service", "1", "2", "", time.Now())
	booster.Object["status"] = map[string]any{
		"conditions": []any{map[string]any{"type": workloadStatusFailed, workloadFieldStatus: workloadConditionTrue}},
	}

	workload := workloadFromModelBooster(booster)
	if workload.StatusDetail != workloadStatusFailed || workload.Status != workloadStatusFailed {
		t.Fatalf("status/detail = %q/%q, want Failed/Failed", workload.Status, workload.StatusDetail)
	}
}

func TestFindKthenaWorkloadsScopesByUserAndAccountLabels(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(workloadModelBoosterGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(workloadModelBoosterListGVK, &unstructured.UnstructuredList{})
	manager := &VolcanojobMgr{
		client: controllerfake.NewClientBuilder().WithScheme(scheme).WithObjects(
			testModelBooster("owned", "11", "22", "Ready", time.Now()),
			testModelBooster("other-account", "11", "23", "Ready", time.Now()),
			testModelBooster("other-user", "12", "22", "Ready", time.Now()),
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
	scheme.AddKnownTypeWithName(workloadModelBoosterGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(workloadModelBoosterListGVK, &unstructured.UnstructuredList{})
	manager := &VolcanojobMgr{
		client: controllerfake.NewClientBuilder().WithScheme(scheme).WithObjects(
			testModelBooster("hidden", "11", "22", "Ready", time.Now()),
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

func testModelBooster(name, userID, accountID, phase string, createdAt time.Time) *unstructured.Unstructured {
	booster := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"backend": map[string]any{
				"schedulerName": "volcano",
				"modelURI":      "hf://Qwen/Qwen3-4B",
				"replicas":      int64(1),
				"workers": []any{map[string]any{
					"replicas": int64(1),
					"pods":     int64(1),
					"config":   map[string]any{"served-model-name": "Qwen/Qwen3-4B"},
					"resources": map[string]any{"requests": map[string]any{
						"cpu":            "4",
						"memory":         "16Gi",
						"nvidia.com/gpu": "1",
					}, "limits": map[string]any{}},
				}},
			},
		},
		workloadFieldStatus: map[string]any{"phase": phase},
	}}
	booster.SetGroupVersionKind(workloadModelBoosterGVK)
	booster.SetName(name)
	booster.SetNamespace("crater-workspace")
	booster.SetCreationTimestamp(metav1.NewTime(createdAt))
	booster.SetLabels(map[string]string{
		workloadKthenaManagedByLabel: workloadKthenaManagedByValue,
		workloadKthenaUserIDLabel:    userID,
		workloadKthenaAccountIDLabel: accountID,
	})
	booster.SetAnnotations(map[string]string{
		workloadKthenaUserAnnotation: "alice",
		workloadKthenaAccountAnno:    "research",
	})
	return booster
}
