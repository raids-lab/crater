package kthena

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func deploymentFixture(name string, gvk schema.GroupVersionKind, native bool) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	obj.SetNamespace("jobs")
	labels := map[string]string{ManagedByLabel: ManagedByValue}
	if native {
		labels[LayoutLabel] = NativeLayout
	}
	obj.SetLabels(labels)
	return obj
}

func deploymentClient(objects ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	for _, gvk := range []schema.GroupVersionKind{ServingGVK} {
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(gvk.GroupVersion().WithKind(gvk.Kind+"List"), &unstructured.UnstructuredList{})
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestKthenaListOnlyIncludesManagedServing(t *testing.T) {
	managed := deploymentFixture("managed", ServingGVK, true)
	unrelated := deploymentFixture("unrelated", ServingGVK, false)
	c := deploymentClient(managed, unrelated)
	items, err := ListDeployments(t.Context(), c, "jobs", nil)
	if err != nil || len(items) != 1 || items[0].GetName() != managed.GetName() {
		t.Fatalf("deployments = %v, %v", items, err)
	}
	if obj, err := GetDeployment(t.Context(), c, client.ObjectKeyFromObject(unrelated)); !apierrors.IsNotFound(err) || obj != nil {
		t.Fatal("unrelated workload adopted")
	}
}

func TestKthenaMissingServingCRDReturnsError(t *testing.T) {
	c := interceptor.NewClient(
		deploymentClient(),
		interceptor.Funcs{List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return apierrors.NewNotFound(schema.GroupResource{Group: ServingGVK.Group, Resource: "modelservings"}, "")
		}},
	)
	if _, err := ListDeployments(t.Context(), c, "jobs", nil); !apierrors.IsNotFound(err) {
		t.Fatalf("missing CRD hidden: %v", err)
	}
}

func TestKthenaListDoesNotHideObservationFailures(t *testing.T) {
	c := interceptor.NewClient(
		deploymentClient(),
		interceptor.Funcs{List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return apierrors.NewForbidden(schema.GroupResource{Resource: "modelservings"}, "", nil)
		}},
	)
	if _, err := ListDeployments(t.Context(), c, "jobs", nil); !apierrors.IsForbidden(err) {
		t.Fatalf("permission failure swallowed: %v", err)
	}
}

func TestKthenaNativePhaseRequiresCurrentGenerationAndAllReplicas(t *testing.T) {
	obj := deploymentFixture("native", ServingGVK, true)
	obj.SetGeneration(2)
	obj.Object["spec"] = map[string]any{"replicas": int64(2)}
	for _, test := range []struct {
		observed, available int64
		expected            string
	}{
		{1, 2, "Progressing"}, {2, 0, "Pending"}, {2, 1, "Degraded"}, {2, 2, "Ready"},
	} {
		obj.Object["status"] = map[string]any{"observedGeneration": test.observed, "availableReplicas": test.available}
		if got := Phase(obj); got != test.expected {
			t.Fatalf("phase = %s, want %s", got, test.expected)
		}
	}
}
