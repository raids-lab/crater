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

package tensorboard

import (
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

//nolint:gocyclo // This test intentionally verifies the complete generated resource in one place.
func TestBuildJobUsesConfiguredImageSchedulingAndLifecycle(t *testing.T) {
	t.Parallel()

	workloadConfig := workloadConfig{
		Image:              "registry.example.com/observability/tensorboard:2.20.0",
		ImagePullPolicy:    corev1.PullNever,
		IdleTimeoutSeconds: 3600,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "tensorboard-registry"},
		},
		NodeSelector: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		Tolerations: []corev1.Toleration{{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		}},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
					Weight: 1,
					Preference: corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "nvidia.com/gpu.present",
						Operator: corev1.NodeSelectorOpDoesNotExist,
					}}},
				}},
			},
		},
	}
	job := newJobBuilder("jobs", &workloadConfig).buildJob(
		"panel-id",
		"训练对比",
		"test-user",
		"q-a1-u2",
		"/home/test-user/tensorboard-runs/test-job",
		"/tensorboard/panel-id",
		nil,
		nil,
	)

	if job.Annotations[annotationKeyTensorboardName] != "训练对比" || job.Name != "tb-panel-id" {
		t.Fatalf("display name and resource ID are not independent: %#v", job.ObjectMeta)
	}
	if job.Spec.SchedulerName != tensorboardSchedulerName || job.Spec.Queue != "q-a1-u2" {
		t.Fatalf("unexpected Volcano scheduling config: %#v", job.Spec)
	}
	if job.Spec.MinAvailable != 1 || job.Spec.MaxRetry != 0 {
		t.Fatalf("unexpected gang or retry settings: %#v", job.Spec)
	}
	policies := job.Spec.Tasks[0].Policies
	if len(policies) != 1 || policies[0].Event != bus.TaskCompletedEvent || policies[0].Action != bus.CompleteJobAction {
		t.Fatalf("missing normal completion policy: %#v", policies)
	}
	if job.Spec.TTLSecondsAfterFinished == nil ||
		*job.Spec.TTLSecondsAfterFinished != tensorboardCleanupTTLSeconds {
		t.Fatalf("unexpected cleanup TTL: %#v", job.Spec.TTLSecondsAfterFinished)
	}

	podSpec := job.Spec.Tasks[0].Template.Spec
	container := podSpec.Containers[0]
	if container.Image != workloadConfig.Image {
		t.Fatalf("image = %q, want %q", container.Image, workloadConfig.Image)
	}
	if container.ImagePullPolicy != workloadConfig.ImagePullPolicy {
		t.Fatalf("image pull policy = %q, want %q", container.ImagePullPolicy, workloadConfig.ImagePullPolicy)
	}
	if podSpec.SchedulerName != tensorboardSchedulerName || podSpec.RestartPolicy != corev1.RestartPolicyNever {
		t.Fatalf("unexpected Pod scheduling config: %#v", podSpec)
	}
	if podSpec.ActiveDeadlineSeconds == nil ||
		*podSpec.ActiveDeadlineSeconds != tensorboardMaxRuntimeSeconds+tensorboardSafetyMarginSeconds {
		t.Fatalf("unexpected active deadline: %#v", podSpec.ActiveDeadlineSeconds)
	}
	if got := podSpec.ImagePullSecrets; len(got) != 1 || got[0].Name != "tensorboard-registry" {
		t.Fatalf("image pull secrets = %#v, want tensorboard-registry", got)
	}
	if _, ok := podSpec.NodeSelector["node-role.kubernetes.io/control-plane"]; !ok {
		t.Fatalf("node selector = %#v, want control-plane selector", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) != 1 || podSpec.Tolerations[0].Effect != corev1.TaintEffectNoSchedule {
		t.Fatalf("tolerations = %#v, want control-plane NoSchedule toleration", podSpec.Tolerations)
	}
	if !reflect.DeepEqual(podSpec.Affinity, workloadConfig.Affinity) {
		t.Fatalf("affinity = %#v, want %#v", podSpec.Affinity, workloadConfig.Affinity)
	}

	command := strings.Join(container.Command, " ")
	if strings.Contains(command, "pip install") {
		t.Fatalf("command installs TensorBoard at runtime: %q", command)
	}
	if !reflect.DeepEqual(container.Command[:3], []string{"python", "-u", "-c"}) || container.Command[3] != tensorboardLauncher {
		t.Fatal("command does not run the embedded launcher")
	}
	wantArgs := []string{
		"--logdir", "/home/test-user/tensorboard-runs/test-job",
		"--path-prefix", "/tensorboard/panel-id",
		"--port", "6006", "--upstream-port", "6007",
		"--idle-timeout-seconds", "3600", "--max-runtime-seconds", "345600",
	}
	if !reflect.DeepEqual(container.Args, wantArgs) {
		t.Fatalf("launcher args = %#v, want %#v", container.Args, wantArgs)
	}
	if container.ReadinessProbe.HTTPGet.Path != tensorboardHealthPath {
		t.Fatal("readiness probe must not count as user activity")
	}
}
