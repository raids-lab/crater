package service

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/kthena"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKthenaServingUsageIncludesAllReplicasAndScopesAccount(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"), &unstructured.UnstructuredList{})
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": kthena.ServingGVK.GroupVersion().String(), "kind": kthena.ServingKind,
		"metadata": map[string]any{"name": "active", "namespace": "jobs", "labels": map[string]any{
			kthena.ManagedByLabel: kthena.ManagedByValue, kthena.LayoutLabel: kthena.NativeLayout, "crater.raids.io/user-id": "1", "crater.raids.io/account-id": "2"}},
		"spec": map[string]any{"replicas": int64(2), "template": map[string]any{"roles": []any{map[string]any{
			"replicas": int64(
				3,
			), "entryTemplate": map[string]any{"metadata": map[string]any{"annotations": map[string]any{kthena.RoleConfigAnnotation: `{"execution":"single","gpuModel":"nvidia.com/gpu"}`}}, "spec": map[string]any{"containers": []any{map[string]any{
				"name": kthena.EngineContainer, "resources": map[string]any{"requests": map[string]any{"cpu": "500m", "memory": "1Gi", "nvidia.com/gpu": "1"}}}}}}}}}},
	}}
	queued := obj.DeepCopy()
	queued.SetName("queued")
	_ = unstructured.SetNestedField(queued.Object, int64(0), "spec", "replicas")
	foreign := obj.DeepCopy()
	foreign.SetName("foreign")
	labels := foreign.GetLabels()
	labels["crater.raids.io/account-id"] = "3"
	foreign.SetLabels(labels)
	svc := &PrequeueService{}
	svc.SetServingClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj, queued, foreign).Build(), "jobs")
	jobs, err := svc.servingUsage(t.Context(), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("account usage includes queued or foreign deployment: %d", len(jobs))
	}
	resources := jobs[0].Resources.Data()
	if resources.Cpu().String() != "3" || resources.Memory().String() != "6Gi" {
		t.Fatalf("resources=%v", resources)
	}
	gpu := resources["nvidia.com/gpu"]
	if gpu.Value() != 6 {
		t.Fatalf("GPU=%v", gpu)
	}
}

func TestServingUsageRetainsDrainingAndOrphanPods(t *testing.T) {
	for _, test := range []struct {
		name        string
		root        bool
		replicas    int64
		exclude     bool
		replacement string
		want        string
	}{
		{"suspended", true, 0, false, "0", "3"},
		{"scaled down", true, 1, false, "0", "5"},
		{"deleted root", false, 0, false, "0", "3"},
		{"replacement overlap", true, 1, true, "2", "3"},
		{"replacement covers live", true, 1, true, "4", "3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
			scheme.AddKnownTypeWithName(
				kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"),
				&unstructured.UnstructuredList{},
			)
			labels := map[string]string{
				kthena.LayoutLabel:           kthena.NativeLayout,
				kthena.ManagedByLabel:        kthena.ManagedByValue,
				kthena.PodDeploymentLabel:    "deployment",
				"crater.raids.io/user-id":    "1",
				"crater.raids.io/account-id": "2",
			}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "old",
					Namespace:         "jobs",
					Labels:            labels,
					Finalizers:        []string{"test/hold"},
					DeletionTimestamp: ptr.To(metav1.Now()),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "engine",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("3")},
							},
						},
					},
				},
			}
			done := pod.DeepCopy()
			done.Name = "done"
			done.Status.Phase = corev1.PodSucceeded
			objects := []client.Object{pod, done}
			if test.root {
				obj := &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": kthena.ServingGVK.GroupVersion().String(),
						"kind":       kthena.ServingKind,
						"spec": map[string]any{
							"replicas": test.replicas,
							"template": map[string]any{
								"roles": []any{
									map[string]any{
										"name":     "server",
										"replicas": int64(1),
										"entryTemplate": map[string]any{
											"metadata": map[string]any{"annotations": map[string]any{kthena.RoleConfigAnnotation: `{ "execution":"single", "gpuModel":"nvidia.com/gpu" }`}},
											"spec": map[string]any{
												"containers": []any{
													map[string]any{
														"name":      kthena.EngineContainer,
														"resources": map[string]any{"requests": map[string]any{"cpu": "2"}},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}
				obj.SetName("root")
				obj.SetNamespace("jobs")
				obj.SetUID("root-uid")
				obj.SetLabels(labels)
				objects = append(objects, obj)
			}
			svc := &PrequeueService{}
			svc.SetServingClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(), "jobs")
			ctx := t.Context()
			if test.exclude {
				ctx = WithServingReplacementResources(
					ExcludeServingReservation(ctx, "root-uid"),
					corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(test.replacement)},
				)
			}
			jobs, err := svc.servingUsage(ctx, 1, 2)
			if err != nil {
				t.Fatal(err)
			}
			total := resource.MustParse("0")
			for _, job := range jobs {
				resources := job.Resources.Data()
				total.Add(*resources.Cpu())
			}
			if total.Cmp(resource.MustParse(test.want)) != 0 {
				t.Fatalf("CPU=%s want %s", total.String(), test.want)
			}
		})
	}
}
