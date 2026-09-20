package util

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBuildDeploymentUsesConfiguredTensorboardImage(t *testing.T) {
	t.Parallel()

	imageConfig := TensorboardImageConfig{
		Image:           "registry.example.com/observability/tensorboard:2.20.0",
		ImagePullPolicy: corev1.PullNever,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "tensorboard-registry"},
		},
	}
	deployment := NewBuilder("jobs", imageConfig).BuildDeployment(
		"panel-id",
		"test-user",
		"/home/test-user/tensorboard-runs/test-job",
		"/tensorboard/panel-id",
		24,
		nil,
		nil,
	)

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != imageConfig.Image {
		t.Fatalf("image = %q, want %q", container.Image, imageConfig.Image)
	}
	if container.ImagePullPolicy != imageConfig.ImagePullPolicy {
		t.Fatalf("image pull policy = %q, want %q", container.ImagePullPolicy, imageConfig.ImagePullPolicy)
	}
	if got := deployment.Spec.Template.Spec.ImagePullSecrets; len(got) != 1 || got[0].Name != "tensorboard-registry" {
		t.Fatalf("image pull secrets = %#v, want tensorboard-registry", got)
	}

	command := strings.Join(container.Command, " ")
	if strings.Contains(command, "pip install") {
		t.Fatalf("command installs TensorBoard at runtime: %q", command)
	}
	if !strings.Contains(command, "python -m tensorboard.main") {
		t.Fatalf("command does not start TensorBoard: %q", command)
	}
	if !strings.Contains(command, "--logdir '/home/test-user/tensorboard-runs/test-job'") {
		t.Fatalf("command does not contain the expected log directory: %q", command)
	}
	if !strings.Contains(command, "--path_prefix '/tensorboard/panel-id'") {
		t.Fatalf("command does not contain the expected path prefix: %q", command)
	}
}
