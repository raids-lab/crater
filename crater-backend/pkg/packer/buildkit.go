package packer

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *imagePacker) CreateFromDockerfile(c context.Context, data *BuildKitReq) error {
	initContainer := b.generateInitContainer(data)
	buildkitContainer := b.generateBuildKitContainer(data)
	volumes := b.generateVolumes()

	jobMeta := metav1.ObjectMeta{
		Name:      data.JobName,
		Namespace: data.Namespace,
		Annotations: map[string]string{
			"buildkit-data/UserID":      fmt.Sprint(data.UserID),
			"buildkit-data/ImageLink":   data.ImageLink,
			"buildkit-data/Dockerfile":  *data.Dockerfile,
			"buildkit-data/Description": *data.Description,
		},
	}

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      data.JobName,
				Namespace: data.Namespace,
				Annotations: map[string]string{
					"container.apparmor.security.beta.kubernetes.io/buildkit": "unconfined",
				},
			},
			Spec: corev1.PodSpec{
				RestartPolicy:  corev1.RestartPolicyNever,
				InitContainers: initContainer,
				Containers:     buildkitContainer,
				Volumes:        volumes,
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeUnconfined,
					},
					RunAsUser:  &runAsUerNumber,
					RunAsGroup: &runAsGroupNumber,
					FSGroup:    &fsAsGroupNumber,
				},
			},
		},
		TTLSecondsAfterFinished: &JobCleanTime,
		BackoffLimit:            &BackoffLimitNumber,
		Completions:             &CompletionNumber,
		Parallelism:             &ParallelismNumber,
	}

	job := &batchv1.Job{
		ObjectMeta: jobMeta,
		Spec:       jobSpec,
	}

	err := b.client.Create(c, job)
	return err
}

func (b *imagePacker) generateInitContainer(data *BuildKitReq) []corev1.Container {
	dockerfileSource := fmt.Sprintf("echo '%s' > /workspace/Dockerfile", *data.Dockerfile)
	initContainer := []corev1.Container{
		{
			Name:    "prepare",
			Image:   "***REMOVED***/docker.io/library/alpine:3.20",
			Command: []string{"/bin/sh"},
			Args:    []string{"-c", dockerfileSource},

			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
				},
			},
		},
	}
	return initContainer
}

func (b *imagePacker) generateVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "harborcredits",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: harborCreditSecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  ".dockerconfigjson",
							Path: "config.json",
						},
					},
				},
			},
		},
		{
			Name: "buildkitcerts",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: buildkitClientSecretName,
				},
			},
		},
	}
	return volumes
}

func (b *imagePacker) generateBuildKitContainer(data *BuildKitReq) []corev1.Container {
	output := fmt.Sprintf("type=image,name=%s,push=true", data.ImageLink)
	buildArgs := []string{
		"--addr", "tcp://buildkitd.crater-images:1234",
		"--tlscacert=/certs/ca.pem",
		"--tlscert=/certs/cert.pem",
		"--tlskey=/certs/key.pem",
		"build",
		"--frontend", "dockerfile.v0",
		"--progress", "plain",
		"--local", "context=/workspace",
		"--local", "dockerfile=/workspace",
		"--output", output,
	}
	buildkitContainer := []corev1.Container{
		{
			Name:  "buildkit",
			Image: "***REMOVED***/user-huangsy/buildkit/buildctl:v0.18.2",
			Args:  buildArgs,
			Env: []corev1.EnvVar{
				{
					Name:  "DOCKER_CONFIG",
					Value: "/.docker",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
					ReadOnly:  true,
				},
				{
					Name:      "buildkitcerts",
					MountPath: "/certs",
				},
				{
					Name:      "harborcredits",
					MountPath: "/.docker",
				},
			},
		},
	}
	return buildkitContainer
}
