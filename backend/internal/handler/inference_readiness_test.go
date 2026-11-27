package handler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/kthena"
)

func TestServingReadinessRequiresCompleteTopology(t *testing.T) {
	roles := []kthena.RoleSpec{
		{Name: "prefill", Instances: 1, NodesPerInstance: 2},
		{Name: "decode", Instances: 1, NodesPerInstance: 1},
	}
	complete := []KthenaRuntimePod{
		{Group: "g", Role: "prefill", Instance: "0", Entry: true, Ready: true},
		{Group: "g", Role: "prefill", Instance: "0", Ready: true},
		{Group: "g", Role: "decode", Instance: "0", Entry: true, Ready: true},
	}
	for _, test := range []struct {
		name     string
		pods     []KthenaRuntimePod
		replicas int64
		want     string
	}{
		{"complete", complete, 1, kthenaPhaseReady},
		{"missing decode", complete[:2], 1, kthenaPhaseDegraded},
		{"missing entry", complete[1:], 1, kthenaPhaseDegraded},
		{"missing group", complete, 2, kthenaPhaseDegraded},
		{"missing worker", []KthenaRuntimePod{complete[0], complete[2]}, 1, kthenaPhaseDegraded},
		{"terminating old worker", append(append([]KthenaRuntimePod{}, complete...),
			KthenaRuntimePod{Group: "g", Role: "prefill", Instance: "0", Stage: runtimePodTerminating}), 1, kthenaPhaseReady},
		{"extra active group", append(append([]KthenaRuntimePod{}, complete...),
			KthenaRuntimePod{Group: "old", Role: "prefill", Instance: "0"}), 1, kthenaPhaseDegraded},
	} {
		t.Run(test.name, func(t *testing.T) {
			resp := &KthenaServiceResp{
				Phase: kthenaPhaseReady, Roles: roles, Replicas: test.replicas, RuntimePods: test.pods,
				Access: KthenaAccess{RouteName: "route", ServerName: "server"},
			}
			applyServingObservation(resp)
			if resp.Phase != test.want {
				t.Fatalf("phase=%s, want %s", resp.Phase, test.want)
			}
		})
	}
	complete[2].Ready = false
	groups, ready, _ := servingRuntimeGroups(roles, complete)
	if ready != 0 || groups[0].Roles[0].ReadyInstances != 1 || groups[0].Roles[1].ReadyInstances != 0 {
		t.Fatalf("unready decode was counted as available: %+v", groups)
	}
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
	}
	if isPodReady(pod) {
		t.Fatal("terminating pod accepted")
	}
	if got := kthenaRuntimePodFromPod(pod); got.Stage != runtimePodTerminating {
		t.Fatalf("terminating pod stage = %q", got.Stage)
	}
}

func TestServingReadinessIgnoresPreviousGeneration(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{"observedGeneration": int64(1), "availableReplicas": int64(2)},
	}}
	obj.SetGeneration(2)
	if got := observedReadyReplicas(obj); got != nil {
		t.Fatalf("reported %d replicas from an earlier generation", *got)
	}
	obj.SetGeneration(1)
	if got := observedReadyReplicas(obj); got == nil || *got != 2 {
		t.Fatalf("unexpected ready count: %v", got)
	}
}

func TestKthenaListUsesLivePodsForReadyCount(t *testing.T) {
	req := nativeTestRequest(t)
	req.Replicas = 1
	req.Roles[0].Instances = 1
	serving := nativeTestServing(t, req)
	serving.SetGeneration(1)
	serving.Object["status"] = map[string]any{"observedGeneration": int64(1), "availableReplicas": int64(1)}

	for _, tc := range []struct {
		name      string
		podPhase  corev1.PodPhase
		wantReady int64
		wantPhase string
	}{
		{name: "failed Pod with stale ready status", podPhase: corev1.PodFailed, wantPhase: kthenaPhaseDegraded},
		{name: "ready Pod", podPhase: corev1.PodRunning, wantReady: 1, wantPhase: kthenaPhaseReady},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "current-pod", Namespace: serving.GetNamespace(),
					Labels: map[string]string{
						kthenaServingNameLabel:            serving.GetName(),
						kthena.PodDeploymentLabel:         serving.GetLabels()[kthena.PodDeploymentLabel],
						kthena.GroupLabel:                 "group-0",
						kthena.RoleLabel:                  req.Roles[0].Name,
						kthena.EntryLabel:                 kthena.EntryValue,
						"modelserving.volcano.sh/role-id": "server-0",
					},
				},
				Status: corev1.PodStatus{Phase: tc.podPhase, Conditions: []corev1.PodCondition{{
					Type: corev1.PodReady, Status: corev1.ConditionTrue,
				}}},
			}
			oldPod := pod.DeepCopy()
			oldPod.Name = "old-pod"
			oldPod.Labels[kthena.PodDeploymentLabel] = "previous-deployment"
			oldPod.Labels[kthena.GroupLabel] = "old-group"
			oldPod.Status.Phase = corev1.PodRunning
			mgr := &KthenaMgr{client: nativeTestClient(serving), namespace: serving.GetNamespace(),
				kubeClient: kubefake.NewClientset(pod, oldPod)}
			services, err := mgr.listKthenaServices(t.Context(), client.MatchingLabels{
				inferenceServiceLabelManagedBy: inferenceServiceManagedByValue,
			})
			if err != nil || len(services) != 1 {
				t.Fatalf("list returned %d services, error = %v", len(services), err)
			}
			got := services[0]
			if got.ReadyReplicas == nil || *got.ReadyReplicas != tc.wantReady || got.Phase != tc.wantPhase {
				t.Fatalf("list returned ready=%v, phase=%s, want ready=%d, phase=%s",
					got.ReadyReplicas, got.Phase, tc.wantReady, tc.wantPhase)
			}
		})
	}
}

func TestServingRuntimeDistinguishesSchedulingDownloadingAndBlockedPods(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{
		Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: "Unschedulable", Message: "no free GPU",
	}}}}
	if got := kthenaRuntimePodFromPod(pod); got.Stage != runtimePodScheduling || got.Reason != "Unschedulable" || got.Message != "no free GPU" {
		t.Fatalf("unscheduled pod = %+v", got)
	}
	pod.Spec.NodeName = "gpu-worker"
	pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Name: "model-downloader", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
	}}
	if got := kthenaRuntimePodFromPod(pod); got.Stage != "downloading" {
		t.Fatalf("downloading pod stage = %q", got.Stage)
	}
	pod.Status.InitContainerStatuses[0].State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}
	if got := kthenaRuntimePodFromPod(pod); got.Stage != runtimePodBlocked || got.Reason != "ImagePullBackOff" {
		t.Fatalf("blocked pod = %+v", got)
	}
}
