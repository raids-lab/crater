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

package crclient

import (
	"context"
	"errors"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/indexer"
)

type nodeSnapshotTestCache struct {
	client.Client
	pods        map[string][]corev1.Pod
	listedNodes []string
	err         error
}

func (cache *nodeSnapshotTestCache) List(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
	options := (&client.ListOptions{}).ApplyOptions(opts)
	if options.FieldSelector == nil {
		return errors.New("missing node Pod index selector")
	}
	nodeName, exact := options.FieldSelector.RequiresExactMatch(indexer.PodNodeNameIndex)
	if !exact {
		return errors.New("missing exact node Pod index selector")
	}
	cache.listedNodes = append(cache.listedNodes, nodeName)
	if cache.err != nil {
		return cache.err
	}
	list.(*corev1.PodList).Items = cache.pods[nodeName]
	return nil
}

func snapshotUsagePod(namespace, cpu string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(cpu),
			}},
		}}},
	}
}

func TestListNodesSnapshotUsesCachedPodIndex(t *testing.T) {
	jobNamespace := config.GetConfig().Namespaces.Job
	cache := &nodeSnapshotTestCache{pods: map[string][]corev1.Pod{
		"node-a": {snapshotUsagePod(jobNamespace, "1"), snapshotUsagePod("kube-system", "250m")},
		"node-b": {snapshotUsagePod(jobNamespace, "2")},
		"node-c": {snapshotUsagePod(jobNamespace, "100")},
	}}
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "nodes", func(action clienttesting.Action) (bool, runtime.Object, error) {
		options := action.(clienttesting.ListActionImpl).GetListOptions()
		if options.Limit != 1 {
			t.Errorf("node page limit = %d, want 1", options.Limit)
		}
		name, next := "node-a", "next-node"
		if options.Continue == "next-node" {
			name, next = "node-b", ""
		}
		return true, &corev1.NodeList{
			ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: next},
			Items:    []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: name}}},
		}, nil
	})
	nc := &NodeClient{Client: cache, KubeClient: kubeClient}
	for _, page := range []struct {
		cursor, name, next string
		cpu                int64
	}{{"", "node-a", "next-node", 1250}, {"next-node", "node-b", "", 2000}} {
		items, next, version, err := nc.ListNodesSnapshot(t.Context(), &metav1.ListOptions{Limit: 1, Continue: page.cursor})
		if err != nil || len(items) != 1 || next != page.next || version != "100" {
			t.Fatalf("unexpected node page: items=%+v next=%q version=%q err=%v", items, next, version, err)
		}
		used := items[0].Used[corev1.ResourceCPU]
		if items[0].Name != page.name || used.MilliValue() != page.cpu || items[0].Workloads != 1 {
			t.Fatalf("unexpected usage on page for %s: %+v", page.name, items[0])
		}
	}
	if !slices.Equal(cache.listedNodes, []string{"node-a", "node-b"}) {
		t.Fatalf("cached Pod reads = %v, want only the current pages' nodes", cache.listedNodes)
	}
	for _, action := range kubeClient.Actions() {
		if action.GetResource().Resource != "nodes" {
			t.Fatalf("unexpected API request during node paging: %+v", action)
		}
	}
}

func TestListNodesSnapshotPropagatesCachedPodReadFailure(t *testing.T) {
	readErr := errors.New("node Pod cache unavailable")
	cache := &nodeSnapshotTestCache{err: readErr}
	nc := &NodeClient{
		Client:     cache,
		KubeClient: fake.NewSimpleClientset(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}}),
	}
	items, _, _, err := nc.ListNodesSnapshot(t.Context(), &metav1.ListOptions{Limit: 1})
	if !errors.Is(err, readErr) || items != nil {
		t.Fatalf("cache failure returned items=%+v err=%v, want no successful usage data", items, err)
	}
}
