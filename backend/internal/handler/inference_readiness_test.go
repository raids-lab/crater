package handler

import (
	"testing"

	"github.com/raids-lab/crater/internal/kthena"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := runtimeAwareInferencePhase(kthenaPhaseReady, roles, test.replicas, test.pods); got != test.want {
				t.Fatalf("phase=%s, want %s", got, test.want)
			}
		})
	}
	complete[2].Ready = false
	if runtimeAwareInferencePhase(kthenaPhaseReady, roles, 1, complete) != kthenaPhaseDegraded {
		t.Fatal("unready decode accepted")
	}
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
	}
	if isPodReady(pod) {
		t.Fatal("terminating pod accepted")
	}
}
