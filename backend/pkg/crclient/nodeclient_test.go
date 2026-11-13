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
	"encoding/json"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

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

func TestPodJobStatusJSON(t *testing.T) {
	tests := []struct {
		name      string
		jobStatus *batch.JobPhase
		want      string
	}{
		{name: "no matched job"},
		{name: "job initializing", jobStatus: ptr.To(batch.JobPhase("")), want: `""`},
		{name: "job completed", jobStatus: ptr.To(batch.Completed), want: `"Completed"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(Pod{Status: corev1.PodSucceeded, JobStatus: tc.jobStatus})
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatal(err)
			}
			if got := string(payload["jobStatus"]); got != tc.want {
				t.Errorf("JSON jobStatus = %q, want %q", got, tc.want)
			}
			if got := string(payload["status"]); got != `"Succeeded"` {
				t.Errorf("JSON status = %q, want Succeeded", got)
			}
		})
	}
}
