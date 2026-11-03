package admission

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func TestAdmissionForCordonedNode(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "v100-node"},
		Spec:       v1.NodeSpec{Unschedulable: true},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{v1.ResourceCPU: resource.MustParse("56"), v1.ResourceMemory: resource.MustParse("504Gi"), "nvidia.com/v100": resource.MustParse("8")},
			Conditions:  []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionTrue}},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(node).Build()
	requests := v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse("40"),
		v1.ResourceMemory: resource.MustParse("400Gi"),
		"nvidia.com/v100": resource.MustParse("8"),
	}
	job := &batch.Job{Spec: batch.JobSpec{Tasks: []batch.TaskSpec{{
		Replicas: 1,
		Template: v1.PodTemplateSpec{Spec: v1.PodSpec{Containers: []v1.Container{{
			Name: "jupyter", Resources: v1.ResourceRequirements{Requests: requests},
		}}}},
	}}}}
	result, err := CheckJobAdmission(context.Background(), client, job)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted || result.Reason == "" {
		t.Fatalf("expected cordoned node rejection with reason, got %+v", result)
	}
	node.Spec.Unschedulable = false
	if err := client.Update(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	result, err = CheckJobAdmission(context.Background(), client, job)
	if err != nil || !result.Accepted {
		t.Fatalf("eligible node should admit job: %+v, %v", result, err)
	}
}
