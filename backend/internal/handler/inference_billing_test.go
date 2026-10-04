package handler

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/raids-lab/crater/internal/kthena"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKthenaServingMeterExcludesPendingAndCapsDeletion(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Second)
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "pod-id", Labels: map[string]string{
		inferenceServiceLabelUserID: "1", inferenceServiceLabelAccountID: "2", kthena.PodDeploymentLabel: "deployment"}}}
	if usage, err := servingPodUsage(pod, start); err != nil || usage != nil {
		t.Fatalf("pending Pod billed: %v %v", usage, err)
	}
	pod.Spec.NodeName = "gpu-node"
	pod.Status.StartTime = &metav1.Time{Time: start}
	pod.Spec.Containers = []corev1.Container{
		{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}}},
	}
	pod.Spec.InitContainers = []corev1.Container{
		{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}}},
	}
	end := start.Add(time.Minute)
	pod.DeletionTimestamp = &metav1.Time{Time: end}
	usage, err := servingPodUsage(pod, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if usage.EndedAt == nil || !usage.ObservedUntil.Equal(end) {
		t.Fatalf("deleting Pod overcharged: %+v", usage)
	}
	resources := usage.Resources.Data()
	if resources.Cpu().String() != "2" {
		t.Fatal("init budget was not accounted")
	}
}

func TestKthenaMeterProtectsPodBeforeScheduling(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "metered", Namespace: "jobs"},
		Spec: corev1.PodSpec{SchedulingGates: []corev1.PodSchedulingGate{{Name: servingBillingGate}, {Name: "other.example/gate"}}}}
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()
	r := &KthenaReconciler{mgr: &KthenaMgr{client: kube}}
	if err := r.observeBillingPod(t.Context(), pod); err != nil {
		t.Fatal(err)
	}
	saved := &corev1.Pod{}
	if err := kube.Get(t.Context(), client.ObjectKeyFromObject(pod), saved); err != nil {
		t.Fatal(err)
	}
	if !controllerutil.ContainsFinalizer(saved, servingBillingFinalizer) || len(saved.Spec.SchedulingGates) != 1 ||
		saved.Spec.SchedulingGates[0].Name != "other.example/gate" {
		t.Fatalf("unsafe scheduling release: %+v", saved)
	}
	if err := r.observeBillingPod(t.Context(), saved); err != nil {
		t.Fatal(err)
	}
}
