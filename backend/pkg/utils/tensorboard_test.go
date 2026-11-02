/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestResolveTensorboardLogDirEnv(t *testing.T) {
	t.Parallel()

	envs := []corev1.EnvVar{
		{Name: "UNCHANGED", Value: TensorboardJobNamePlaceholder},
		{
			Name:  TensorboardLogDirEnv,
			Value: "/home/test-user/tensorboard-runs/" + TensorboardJobNamePlaceholder,
		},
	}

	resolved := ResolveTensorboardLogDirEnv(envs, "sg-test-user-a1b2c")
	if got := resolved[0].Value; got != TensorboardJobNamePlaceholder {
		t.Fatalf("unrelated environment variable = %q, want unchanged value", got)
	}
	want := "/home/test-user/tensorboard-runs/sg-test-user-a1b2c"
	if got := resolved[1].Value; got != want {
		t.Fatalf("TensorBoard log directory = %q, want %q", got, want)
	}
	if envs[1].Value == resolved[1].Value {
		t.Fatal("ResolveTensorboardLogDirEnv mutated the caller's environment slice")
	}

	deferred := ResolveTensorboardLogDirEnv(envs, "")
	if got := deferred[1].Value; got != envs[1].Value {
		t.Fatalf("deferred TensorBoard log directory = %q, want %q", got, envs[1].Value)
	}

	custom := []corev1.EnvVar{{
		Name:  TensorboardLogDirEnv,
		Value: "/mnt/training/custom-events",
	}}
	resolvedCustom := ResolveTensorboardLogDirEnv(custom, "sg-test-user-a1b2c")
	if got := resolvedCustom[0].Value; got != custom[0].Value {
		t.Fatalf("custom TensorBoard log directory = %q, want %q", got, custom[0].Value)
	}
}

func TestResolveTensorboardLogDirInPodSpec(t *testing.T) {
	t.Parallel()

	podSpec := corev1.PodSpec{Containers: []corev1.Container{{
		Env: []corev1.EnvVar{{
			Name:  TensorboardLogDirEnv,
			Value: "/home/test-user/tensorboard-runs/" + TensorboardJobNamePlaceholder,
		}},
	}}}

	ResolveTensorboardLogDirInPodSpec(&podSpec, "legacy-job-240101-42")
	want := "/home/test-user/tensorboard-runs/legacy-job-240101-42"
	if got := podSpec.Containers[0].Env[0].Value; got != want {
		t.Fatalf("TensorBoard log directory in PodSpec = %q, want %q", got, want)
	}
}
