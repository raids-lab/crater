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
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// AddJobProfilingCapabilities enables in-job Python process sampling.
// Call this only for user workloads, not platform-managed auxiliary workloads.
func AddJobProfilingCapabilities(podSpec *corev1.PodSpec) {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.SecurityContext == nil {
			container.SecurityContext = &corev1.SecurityContext{}
		}
		if container.SecurityContext.Capabilities == nil {
			container.SecurityContext.Capabilities = &corev1.Capabilities{}
		}
		capabilities := container.SecurityContext.Capabilities
		if !slices.Contains(capabilities.Add, "SYS_PTRACE") {
			capabilities.Add = append(capabilities.Add, "SYS_PTRACE")
		}
	}
}

// SetJobSysAdminCapability applies the current user grant to regular job containers.
// Remove stale grants from templates when authorization has been revoked.
func SetJobSysAdminCapability(podSpec *corev1.PodSpec, enabled bool) {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.SecurityContext == nil {
			if !enabled {
				continue
			}
			container.SecurityContext = &corev1.SecurityContext{}
		}
		if container.SecurityContext.Capabilities == nil {
			if !enabled {
				continue
			}
			container.SecurityContext.Capabilities = &corev1.Capabilities{}
		}
		caps := container.SecurityContext.Capabilities
		caps.Add = slices.DeleteFunc(caps.Add, func(cap corev1.Capability) bool { return cap == "SYS_ADMIN" })
		if enabled {
			caps.Add = append(caps.Add, "SYS_ADMIN")
		}
	}
}
