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

package utils

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestAddJobProfilingCapabilities(t *testing.T) {
	t.Parallel()
	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "configured", SecurityContext: &corev1.SecurityContext{
				RunAsUser: ptr.To(int64(1001)), RunAsGroup: ptr.To(int64(1002)),
				AllowPrivilegeEscalation: ptr.To(false), ReadOnlyRootFilesystem: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				Capabilities:   &corev1.Capabilities{Add: []corev1.Capability{"IPC_LOCK", "SYS_PTRACE"}, Drop: []corev1.Capability{"NET_RAW"}},
			}},
			{Name: "empty-context", SecurityContext: &corev1.SecurityContext{RunAsNonRoot: ptr.To(true)}},
			{Name: "image-defaults"},
		},
		InitContainers: []corev1.Container{{Name: "init"}},
	}
	want := pod.DeepCopy()
	want.Containers[0].SecurityContext.Capabilities.Add = []corev1.Capability{"IPC_LOCK", "SYS_PTRACE"}
	want.Containers[1].SecurityContext.Capabilities = &corev1.Capabilities{Add: []corev1.Capability{"SYS_PTRACE"}}
	want.Containers[2].SecurityContext = &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"SYS_PTRACE"}},
	}
	AddJobProfilingCapabilities(&pod)
	AddJobProfilingCapabilities(&pod)
	if !reflect.DeepEqual(&pod, want) {
		t.Fatalf("profiling changed unrelated settings or duplicated capabilities: got %+v, want %+v", pod, want)
	}
	// Containers with no context must not share mutable capability slices.
	pod.Containers[1].SecurityContext.Capabilities.Add[0] = "NET_BIND_SERVICE"
	if pod.Containers[2].SecurityContext.Capabilities.Add[0] != "SYS_PTRACE" {
		t.Fatal("containers share profiling capabilities")
	}
}

func TestSetJobSysAdminCapabilityPreservesSecurityContext(t *testing.T) {
	t.Parallel()
	pod := corev1.PodSpec{Containers: []corev1.Container{{Name: "main", SecurityContext: &corev1.SecurityContext{
		RunAsUser: ptr.To(int64(1001)), AllowPrivilegeEscalation: ptr.To(false),
		Capabilities:   &corev1.Capabilities{Add: []corev1.Capability{"IPC_LOCK", "SYS_ADMIN", "SYS_ADMIN"}, Drop: []corev1.Capability{"NET_RAW"}},
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}}}, InitContainers: []corev1.Container{{Name: "init"}}}
	want := pod.DeepCopy()
	want.Containers[0].SecurityContext.Capabilities.Add = []corev1.Capability{"IPC_LOCK", "SYS_ADMIN"}
	SetJobSysAdminCapability(&pod, true)
	SetJobSysAdminCapability(&pod, true)
	if !reflect.DeepEqual(&pod, want) {
		t.Fatal("grant changed unrelated settings or duplicated SYS_ADMIN")
	}
	want.Containers[0].SecurityContext.Capabilities.Add = []corev1.Capability{"IPC_LOCK"}
	SetJobSysAdminCapability(&pod, false)
	if !reflect.DeepEqual(&pod, want) {
		t.Fatal("revoke changed unrelated settings")
	}
}
