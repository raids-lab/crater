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
	"slices"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/pkg/indexer"
)

func TestCPUModelFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name: "explicit model overrides ambiguous hardware identifiers",
			labels: map[string]string{
				LabelKeyCPUModel: "Intel_Xeon_Silver_4210",
				"feature.node.kubernetes.io/cpu-model.vendor_id": "Intel",
				"feature.node.kubernetes.io/cpu-model.family":    "6",
				"feature.node.kubernetes.io/cpu-model.id":        "85",
			},
			want: "Intel Xeon Silver 4210",
		},
		{
			name: "arm model",
			labels: map[string]string{
				LabelKeyCPUModel: "HUAWEI_Kunpeng_920_5250",
			},
			want: "HUAWEI Kunpeng 920 5250",
		},
		{
			name: "hardware identifiers alone do not identify a product",
			labels: map[string]string{
				"feature.node.kubernetes.io/cpu-model.vendor_id": "VendorUnknown",
				"feature.node.kubernetes.io/cpu-model.family":    "15",
				"feature.node.kubernetes.io/cpu-model.id":        "34336",
			},
			want: "",
		},
		{
			name: "no cpu labels",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Labels: tt.labels},
			}
			if got := cpuModelFromLabels(node); got != tt.want {
				t.Errorf("cpuModelFromLabels() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPodInfoFromKubePodIncludesStartTime(t *testing.T) {
	createdAt := metav1.NewTime(time.Date(2026, 8, 24, 2, 55, 51, 0, time.UTC))
	startedAt := metav1.NewTime(time.Date(2026, 8, 24, 17, 50, 32, 0, time.UTC))
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example-default0-0",
			Namespace:         "crater-workspace",
			CreationTimestamp: createdAt,
		},
		Status: corev1.PodStatus{
			Phase:     corev1.PodRunning,
			StartTime: &startedAt,
		},
	}

	got := podInfoFromKubePod(pod)
	if got.CreateTime != createdAt {
		t.Fatalf("CreateTime = %v, want %v", got.CreateTime, createdAt)
	}
	if got.StartTime == nil || !got.StartTime.Equal(&startedAt) {
		t.Fatalf("StartTime = %v, want %v", got.StartTime, startedAt)
	}
}

func TestPodInfoFromKubePodKeepsMissingStartTime(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}

	if got := podInfoFromKubePod(pod); got.StartTime != nil {
		t.Fatalf("StartTime = %v, want nil", got.StartTime)
	}
}

func TestListNodesReportsIndependentStatusFields(t *testing.T) {
	occupied := []corev1.Taint{{
		Key:    "crater.raids.io/account",
		Value:  "research",
		Effect: corev1.TaintEffectNoSchedule,
	}}
	ownerTolerations := []corev1.Toleration{{
		Key:      occupied[0].Key,
		Operator: corev1.TolerationOpEqual,
		Value:    occupied[0].Value,
		Effect:   occupied[0].Effect,
	}}
	diskPressure := corev1.Taint{
		Key: "node.kubernetes.io/disk-pressure", Effect: corev1.TaintEffectNoSchedule,
	}
	tests := []struct {
		name            string
		spec            corev1.NodeSpec
		conditions      []corev1.NodeCondition
		labels          map[string]string
		wantStatus      corev1.NodeConditionType
		wantReadiness   corev1.ConditionStatus
		wantPressures   []corev1.NodeConditionType
		tolerations     []corev1.Toleration
		wantSchedulable bool
	}{
		{
			name:          "ready exclusive node in the public account",
			spec:          corev1.NodeSpec{Taints: occupied},
			conditions:    []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			wantStatus:    NodeStatusOccupied,
			wantReadiness: corev1.ConditionTrue,
		},
		{
			name:            "ready exclusive node in the owning account",
			spec:            corev1.NodeSpec{Taints: occupied},
			conditions:      []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			wantStatus:      NodeStatusOccupied,
			wantReadiness:   corev1.ConditionTrue,
			tolerations:     ownerTolerations,
			wantSchedulable: true,
		},
		{
			name:          "exclusive node with untolerated disk taint",
			spec:          corev1.NodeSpec{Taints: append([]corev1.Taint{diskPressure}, occupied...)},
			conditions:    []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			wantStatus:    NodeStatusOccupied,
			wantReadiness: corev1.ConditionTrue,
			tolerations:   ownerTolerations,
		},
		{
			name:          "occupied and not ready before the taint is added",
			spec:          corev1.NodeSpec{Taints: occupied},
			conditions:    []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}},
			wantStatus:    NodeStatusOccupied,
			wantReadiness: corev1.ConditionFalse,
		},
		{
			name:          "occupied and cordoned before the taint is added",
			spec:          corev1.NodeSpec{Taints: occupied, Unschedulable: true},
			conditions:    []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			wantStatus:    NodeStatusOccupied,
			wantReadiness: corev1.ConditionTrue,
		},
		{
			name: "ready with memory and disk pressure",
			spec: corev1.NodeSpec{Taints: []corev1.Taint{diskPressure}},
			conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodeNetworkUnavailable, Status: corev1.ConditionUnknown},
			},
			wantStatus:    corev1.NodeReady,
			wantReadiness: corev1.ConditionTrue,
			wantPressures: []corev1.NodeConditionType{corev1.NodeDiskPressure, corev1.NodeMemoryPressure},
		},
		{
			name: "uncertain readiness with PID and network conditions",
			spec: corev1.NodeSpec{Unschedulable: true},
			conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionUnknown},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodeNetworkUnavailable, Status: corev1.ConditionTrue},
			},
			wantStatus:    NodeStatusUnschedulable,
			wantReadiness: corev1.ConditionUnknown,
			wantPressures: []corev1.NodeConditionType{corev1.NodeNetworkUnavailable, corev1.NodePIDPressure},
		},
		{
			name:          "virtual node without a Ready condition",
			labels:        map[string]string{"crater.raids.io/nodetype": "virtual"},
			wantStatus:    NodeStatusUnknown,
			wantReadiness: corev1.ConditionUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node", Labels: tt.labels},
				Spec:       tt.spec,
				Status:     corev1.NodeStatus{Conditions: tt.conditions},
			}
			c := &NodeClient{Client: fake.NewClientBuilder().
				WithObjects(node).
				WithIndex(&corev1.Pod{}, indexer.PodNodeNameIndex, func(obj client.Object) []string {
					return []string{obj.(*corev1.Pod).Spec.NodeName}
				}).
				Build()}
			nodes, err := c.ListNodes(t.Context(), tt.tolerations...)
			if err != nil || len(nodes) != 1 {
				t.Fatalf("ListNodes returned %d nodes, err = %v", len(nodes), err)
			}
			got := nodes[0]
			if got.SchedulableForAccount != tt.wantSchedulable {
				t.Errorf("schedulableForAccount = %t, want %t", got.SchedulableForAccount, tt.wantSchedulable)
			}
			if got.Status != tt.wantStatus || got.ReadyStatus != tt.wantReadiness ||
				got.Unschedulable != tt.spec.Unschedulable || !slices.Equal(got.Pressures, tt.wantPressures) {
				t.Errorf("got status=%q readyStatus=%q unschedulable=%t pressures=%v, want %q %q %t %v",
					got.Status, got.ReadyStatus, got.Unschedulable, got.Pressures,
					tt.wantStatus, tt.wantReadiness, tt.spec.Unschedulable, tt.wantPressures)
			}
		})
	}
}
