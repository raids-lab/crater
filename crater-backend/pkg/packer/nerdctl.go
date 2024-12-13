package packer

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/ptr"
)

func (b *imagePacker) CreateFromSnapshot(c context.Context, data *SnapshotReq) error {
	container := b.generateSnapshotContainer(data)
	volumes := b.generateSnapshotVolumes()
	jobName := data.PodName + "-snapshot"

	jobMeta := metav1.ObjectMeta{
		Name:      jobName,
		Namespace: data.Namespace,
	}

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: data.Namespace,
			},
			Spec: corev1.PodSpec{
				Containers:    []corev1.Container{container},
				Volumes:       volumes,
				RestartPolicy: corev1.RestartPolicyNever,
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  &runAsUerNumber,
					RunAsGroup: &runAsGroupNumber,
					FSGroup:    &fsAsGroupNumber,
				},
				Affinity: ptr.To(corev1.Affinity{
					NodeAffinity: ptr.To(corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: ptr.To(corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "kubernetes.io/hostname",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{data.NodeName},
										},
									},
								},
							},
						}),
					}),
				}),
			},
		},
		BackoffLimit:            &BackoffLimitNumber,
		TTLSecondsAfterFinished: &JobCleanTime,
	}

	job := &batchv1.Job{
		ObjectMeta: jobMeta,
		Spec:       jobSpec,
	}

	return b.client.Create(c, job)
}

func (b *imagePacker) generateSnapshotContainer(data *SnapshotReq) corev1.Container {
	args := []string{
		"/snapshot.sh",
		"--namespace", data.Namespace,
		"--pod-name", data.PodName,
		"--container-name", data.ContainerName,
		"--image-url", data.ImageLink,
		// 如有需要，可添加 "--size-limit" 参数
		"--size-limit", "20",
	}

	container := corev1.Container{
		Name:    "build-image",
		Image:   "***REMOVED***/nerdctl:2.0.1",
		Command: args,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "containerd-sock",
				MountPath: "/run/containerd/containerd.sock",
			},
			{
				Name:      "secret",
				MountPath: "/.docker",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "DOCKER_CONFIG",
				Value: "/.docker",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memoryLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuRequest),
				corev1.ResourceMemory: resource.MustParse(memoryRequest),
			},
		},
	}

	return container
}

func (b *imagePacker) generateSnapshotVolumes() []corev1.Volume {
	hostPathType := corev1.HostPathSocket
	volumes := []corev1.Volume{
		{
			Name: "containerd-sock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/containerd/containerd.sock",
					Type: &hostPathType,
				},
			},
		},
		{
			Name: "secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: buildkitSecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  ".dockerconfigjson",
							Path: "config.json",
						},
					},
				},
			},
		},
	}

	return volumes
}
