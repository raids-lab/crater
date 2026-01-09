package packer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/pkg/config"

	"k8s.io/utils/ptr"
)

func (b *imagePacker) CreateFromSnapshot(c context.Context, data *SnapshotReq) error {
	container := b.generateSnapshotContainer(data)
	volumes := b.generateSnapshotVolumes()
	jobName := data.PodName + "-snapshot" + uuid.New().String()[:4]

	jobMeta := metav1.ObjectMeta{
		Name:      jobName,
		Namespace: config.GetConfig().Namespaces.Image,
		Annotations: map[string]string{
			AnnotationKeyUserID:      fmt.Sprint(data.UserID),
			AnnotationKeyImageLink:   data.ImageLink,
			AnnotationKeyScript:      "",
			AnnotationKeyDescription: data.Description,
			AnnotationKeyTags:        "",
			AnnotationKeySource:      string(data.BuildSource),
			AnnotationKeyTemplate:    "",
		},
	}

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: config.GetConfig().Namespaces.Image,
			},
			Spec: corev1.PodSpec{
				Containers:    []corev1.Container{container},
				Volumes:       volumes,
				RestartPolicy: corev1.RestartPolicyNever,
				// Allow caller to pass tolerations so snapshot job can be scheduled on tainted nodes
				Tolerations: data.Tolerations,
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
				EnableServiceLinks: ptr.To(false),
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
	}

	// For non-admin users, add size limit argument
	if !data.IsAdmin {
		args = append(args, "--size-limit", "40") // 40 GiB
	}

	container := corev1.Container{
		Name:    "build-image",
		Image:   config.GetConfig().Registry.BuildTools.Images.Nerdctl,
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
	}

	return volumes
}
