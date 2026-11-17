// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extender

import (
	"fmt"
	"sync"
	"testing"

	. "github.com/bytedance/mockey"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/kthena"
)

func servingFixture(t *testing.T, objects ...client.Object) (*unstructured.Unstructured, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"), &unstructured.UnstructuredList{})
	serving := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": kthena.ServingGVK.GroupVersion().String(), "kind": kthena.ServingKind,
		"spec": map[string]any{"replicas": int64(2), "template": map[string]any{"roles": []any{map[string]any{
			"name": "server", "replicas": int64(1),
			"entryTemplate": map[string]any{
				"metadata": map[string]any{"annotations": map[string]any{kthena.RoleConfigAnnotation: `{"execution":"single","gpuModel":"nvidia.com/gpu"}`}},
				"spec": map[string]any{"containers": []any{map[string]any{"name": kthena.EngineContainer,
					"resources": map[string]any{"requests": map[string]any{"cpu": "1", "memory": "1Gi"}}}}},
			},
		}}}},
	}}
	serving.SetName("model")
	serving.SetNamespace(testNamespace)
	serving.SetUID("model-uid")
	serving.SetLabels(map[string]string{kthena.ManagedByLabel: kthena.ManagedByValue,
		kthena.LayoutLabel: kthena.NativeLayout, kthena.PodDeploymentLabel: "deployment-id",
		"crater.raids.io/user-id": "7", "crater.raids.io/account-id": "1"})
	serving.SetAnnotations(map[string]string{scheduling.QueueNameAnnotationKey: publicQueue})
	return serving, fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestConcurrentWorkloadsCannotOversubscribeQuota(t *testing.T) {
	PatchConvey("concurrent callbacks check and reserve atomically", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		const candidates = 32
		views := make([]*jobView, 0, candidates)
		requests := make([]*requestJobInfo, 0, candidates)
		for i := range candidates {
			name := fmt.Sprintf("concurrent-%d", i)
			views = append(views, newView(name, rl("cpu", "1")))
			requests = append(requests, vcjobRequest(name))
		}
		s := seededServer(newSnapshot(views...))
		start := make(chan struct{})
		votes := make(chan int, candidates)
		var waiting sync.WaitGroup
		for _, req := range requests {
			waiting.Go(func() {
				<-start
				votes <- s.decide(t.Context(), req).status
			})
		}
		close(start)
		waiting.Wait()
		close(votes)
		accepted := 0
		for status := range votes {
			if status == voteAbstain {
				accepted++
			}
		}
		if accepted != 1 {
			t.Fatalf("concurrent admissions = %d, want exactly one CPU reserved", accepted)
		}
	})
}

func servingGroup(serving *unstructured.Unstructured, name string) *scheduling.PodGroup {
	return &scheduling.PodGroup{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace,
		CreationTimestamp: metav1.NewTime(fixedNow),
		OwnerReferences: []metav1.OwnerReference{{APIVersion: kthena.ServingGVK.GroupVersion().String(),
			Kind: kthena.ServingKind, Name: serving.GetName(), UID: serving.GetUID(), Controller: ptr.To(true)}}},
		Spec: scheduling.PodGroupSpec{Queue: publicQueue}}
}

func servingRequest(group *scheduling.PodGroup) *requestJobInfo {
	return &requestJobInfo{Namespace: testNamespace, PodGroup: &requestPodGroup{Metadata: group.ObjectMeta}}
}

func TestServingPodGroupsShareQuotaWithJobs(t *testing.T) {
	PatchConvey("serving groups and jobs reserve against one ledger", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "2"})
		serving, reader := servingFixture(t)
		if err := reader.Create(t.Context(), serving); err != nil {
			t.Fatal(err)
		}
		groups := &scheduling.PodGroupList{Items: []scheduling.PodGroup{
			*servingGroup(serving, "model-0"), *servingGroup(serving, "model-1"),
		}}
		snap := newSnapshot(newView(jobA, rl("cpu", "1"), admitted), newView(jobB, rl("cpu", "1")))
		s := seededServer(snap)
		s.reader = reader
		if err := s.addServingViews(t.Context(), snap, groups, testNamespace); err != nil {
			t.Fatal(err)
		}
		if got := s.decide(t.Context(), servingRequest(&groups.Items[0])); got.status != voteAbstain {
			t.Fatalf("first serving group vote = %+v, want one available CPU admitted", got)
		}
		for _, req := range []*requestJobInfo{servingRequest(&groups.Items[1]), vcjobRequest(jobB)} {
			if got := s.decide(t.Context(), req); got.status != voteReject || fieldValue(got, fieldCause) != causeQuota {
				t.Fatalf("quota vote = %+v, want rejection after serving group reservation", got)
			}
		}
	})
}

func TestServingRequestOwnershipAndMissingSnapshot(t *testing.T) {
	PatchConvey("only exact managed controller references participate", t, func() {
		stubJobNamespace()
		serving, reader := servingFixture(t)
		if err := reader.Create(t.Context(), serving); err != nil {
			t.Fatal(err)
		}
		s := seededServer(newSnapshot())
		s.reader = reader
		group := servingGroup(serving, "model-0")
		if got := s.decide(t.Context(), servingRequest(group)); got.status != voteReject {
			t.Fatalf("uncached managed group vote = %+v, want retry instead of unchecked admission", got)
		}
		group.OwnerReferences[0].UID = types.UID("another-owner")
		if got := s.decide(t.Context(), servingRequest(group)); got.status != voteReject {
			t.Fatalf("mismatched managed owner vote = %+v, want rejection", got)
		}
		group.OwnerReferences[0].UID = serving.GetUID()
		serving.SetLabels(map[string]string{kthena.ManagedByLabel: "external"})
		if err := reader.Update(t.Context(), serving); err != nil {
			t.Fatal(err)
		}
		if got := s.decide(t.Context(), servingRequest(group)); got.status != voteAbstain {
			t.Fatalf("unmanaged serving vote = %+v, want abstention", got)
		}
	})
}

func TestServingSnapshotRetainsAllocatedPods(t *testing.T) {
	for _, test := range []struct {
		name                     string
		paused, deleted, waiting bool
		wantCPU                  string
	}{
		{name: "admitted group and live pod are counted once", wantCPU: "3"},
		{name: "paused retains active and terminating pods", paused: true, wantCPU: "3"},
		{name: "deleted retains orphan pods", deleted: true, wantCPU: "3"},
		{name: "unadmitted pods do not reserve resources", waiting: true, wantCPU: "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			serving, reader := servingFixture(t)
			if test.paused {
				if err := unstructured.SetNestedField(serving.Object, int64(0), "spec", "replicas"); err != nil {
					t.Fatal(err)
				}
			}
			if !test.deleted {
				if err := reader.Create(t.Context(), serving); err != nil {
					t.Fatal(err)
				}
			}
			active := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "active", Namespace: testNamespace,
				Labels: serving.GetLabels(), Annotations: serving.GetAnnotations()},
				Spec: v1.PodSpec{NodeName: "allocated-node", Containers: []v1.Container{{Name: "engine",
					Resources: v1.ResourceRequirements{Requests: rl("cpu", "1")}}}}}
			terminating := active.DeepCopy()
			terminating.Name = "terminating"
			terminating.Spec.Containers[0].Resources.Requests = rl("cpu", "2")
			terminating.Finalizers = []string{"test/hold"}
			terminating.DeletionTimestamp = ptr.To(metav1.Now())
			group := servingGroup(serving, "model-0")
			group.Status.Phase = scheduling.PodGroupInqueue
			if test.waiting {
				active.Spec.NodeName = ""
				terminating.Spec.NodeName = ""
				group.Status.Phase = scheduling.PodGroupPending
			}
			foreign := active.DeepCopy()
			foreign.Name = "foreign"
			foreign.Labels = serving.GetLabels()
			foreign.Labels["crater.raids.io/account-id"] = "2"
			foreign.Annotations = map[string]string{scheduling.QueueNameAnnotationKey: "other-account"}
			for _, obj := range []client.Object{active, terminating, foreign} {
				if err := reader.Create(t.Context(), obj); err != nil {
					t.Fatal(err)
				}
			}
			snap := newSnapshot()
			s := seededServer(snap)
			s.reader = reader
			if err := s.addServingViews(t.Context(), snap, &scheduling.PodGroupList{Items: []scheduling.PodGroup{*group}}, testNamespace); err != nil {
				t.Fatal(err)
			}
			if got := cpuOf(snap.usageByOwner[ownerKey{userID: testUserID, queue: publicQueue}]); got != test.wantCPU {
				t.Fatalf("reserved CPU = %s, want %s", got, test.wantCPU)
			}
			if (test.paused || test.deleted) && len(snap.views) != 0 {
				t.Fatal("paused or deleted serving groups must not be admitted again")
			}
		})
	}
}
