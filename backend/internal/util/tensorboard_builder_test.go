package util

import (
	"strings"
	"testing"
)

func TestBuildDeploymentUsesPrebuiltTensorboardImage(t *testing.T) {
	t.Parallel()

	deployment := NewBuilder("jobs").BuildDeployment(
		"panel-id",
		"test-user",
		"/home/test-user/tensorboard-runs/test-job",
		"/tensorboard/panel-id",
		24,
		nil,
		nil,
	)

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != DefaultTensorboardImage {
		t.Fatalf("image = %q, want %q", container.Image, DefaultTensorboardImage)
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
