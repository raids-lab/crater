package vcjob

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
)

func execTestPod(job *batch.Job) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-0", Namespace: job.Namespace, OwnerReferences: []metav1.OwnerReference{{
			APIVersion: batch.SchemeGroupVersion.String(), Kind: "Job", Name: job.Name, UID: job.UID, Controller: ptr.To(true),
		}}},
		Spec:   v1.PodSpec{Containers: []v1.Container{{Name: "main"}}},
		Status: v1.PodStatus{Phase: v1.PodRunning, ContainerStatuses: []v1.ContainerStatus{{Name: "main", State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}}}},
	}
}

func TestSelectExecTarget(t *testing.T) {
	job := &batch.Job{ObjectMeta: metav1.ObjectMeta{Name: "training", Namespace: "real-namespace", UID: types.UID("live-job")}}
	tests := []struct {
		name      string
		change    func(*v1.Pod)
		container string
		wantError error
	}{
		{name: "pure CPU pod"},
		{name: "ARM domestic accelerator", change: func(p *v1.Pod) {
			p.Spec.NodeSelector = map[string]string{"kubernetes.io/arch": "arm64"}
			p.Spec.Containers[0].Resources.Limits = v1.ResourceList{v1.ResourceName("huawei.com/Ascend910"): resource.MustParse("1")}
		}},
		{name: "other job UID rejected", change: func(p *v1.Pod) { p.OwnerReferences[0].UID = "old-job" }, wantError: bizerr.NotFound.Base},
		{name: "other namespace rejected", change: func(p *v1.Pod) { p.Namespace = "other" }, wantError: bizerr.NotFound.Base},
		{name: "non-controller owner rejected", change: func(p *v1.Pod) { p.OwnerReferences[0].Controller = ptr.To(false) }, wantError: bizerr.NotFound.Base},
		{name: "pending pod rejected", change: func(p *v1.Pod) { p.Status.Phase = v1.PodPending }, wantError: bizerr.Conflict.Base},
		{name: "terminated container rejected", change: func(p *v1.Pod) { p.Status.ContainerStatuses[0].State = v1.ContainerState{} }, wantError: bizerr.Conflict.Base},
		{name: "sidecar requires selection", change: func(p *v1.Pod) { p.Spec.Containers = append(p.Spec.Containers, v1.Container{Name: "sidecar"}) }, wantError: bizerr.BadRequest.Base},
		{name: "explicit main with sidecar", container: "main", change: func(p *v1.Pod) { p.Spec.Containers = append(p.Spec.Containers, v1.Container{Name: "sidecar"}) }},
		{name: "unknown container", container: "missing", wantError: bizerr.Conflict.Base},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := execTestPod(job)
			if tt.change != nil {
				tt.change(&pod)
			}
			got, container, err := SelectExecTarget(job, []v1.Pod{pod}, "", tt.container)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("error = %v, want %v", err, tt.wantError)
				}
				return
			}
			if err != nil || got.Name != pod.Name || container != "main" {
				t.Fatalf("target = %v/%s, error = %v", got, container, err)
			}
		})
	}
	t.Run("multiple pods require explicit selection", func(t *testing.T) {
		first := execTestPod(job)
		second := *first.DeepCopy()
		second.Name = "worker-1"
		pods := []v1.Pod{second, first}
		if _, _, err := SelectExecTarget(job, pods, "", ""); !errors.Is(err, bizerr.BadRequest.Base) {
			t.Fatalf("error = %v, want selection error", err)
		}
		pod, _, err := SelectExecTarget(job, pods, second.Name, "")
		if err != nil || pod.Name != second.Name {
			t.Fatalf("target = %v, error = %v", pod, err)
		}
	})
}
