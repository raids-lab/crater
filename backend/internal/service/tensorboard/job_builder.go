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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	annotationKeyExpirationTime        = "crater.raids.io/expiration-time"
	labelKeyTensorboardID              = "crater.raids.io/tensorboard-id"
	labelKeyTypeTensorboard            = "tensorboard"
	tensorboardPort                    = 6006
	tensorboardSchedulerName           = "volcano"
	tensorboardTaskName                = "tensorboard"
	tensorboardMaxRuntimeSeconds int64 = 4 * 24 * 60 * 60
	tensorboardCleanupTTLSeconds int32 = 5 * 60
)

type workloadConfig struct {
	Image            string
	ImagePullPolicy  corev1.PullPolicy
	ImagePullSecrets []corev1.LocalObjectReference
	NodeSelector     map[string]string
	Tolerations      []corev1.Toleration
	Affinity         *corev1.Affinity
}

type jobBuilder struct {
	Namespace      string
	WorkloadConfig *workloadConfig
}

func newJobBuilder(namespace string, workloadConfig *workloadConfig) *jobBuilder {
	return &jobBuilder{
		Namespace:      namespace,
		WorkloadConfig: workloadConfig,
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (b *jobBuilder) buildJob(
	tbID string,
	username string,
	queueName string,
	logDir string,
	ingressPath string,
	volumes []corev1.Volume,
	volumeMounts []corev1.VolumeMount,
) *batch.Job {
	labels := map[string]string{
		"app":                     "tensorboard",
		labelKeyTensorboardID:     tbID,
		crclient.LabelKeyTaskUser: username,
		crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
	}

	// Resource Limits (Important to prevent OOM)
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}

	// The command binds TensorBoard to all interfaces and configures its dynamic path prefix.
	cmd := []string{
		"/bin/bash",
		"-lc",
		fmt.Sprintf(
			"python -m tensorboard.main --logdir %s --host 0.0.0.0 --port %d --path_prefix %s",
			shellQuote(logDir),
			tensorboardPort,
			shellQuote(ingressPath),
		),
	}

	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("tb-%s", tbID),
			Namespace: b.Namespace,
			Labels:    labels,
		},
		Spec: batch.JobSpec{
			SchedulerName:           tensorboardSchedulerName,
			Queue:                   queueName,
			MinAvailable:            1,
			MaxRetry:                0,
			TTLSecondsAfterFinished: ptr.To(tensorboardCleanupTTLSeconds),
			Tasks: []batch.TaskSpec{{
				Name:     tensorboardTaskName,
				Replicas: 1,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: labels},
					Spec: corev1.PodSpec{
						SchedulerName:         tensorboardSchedulerName,
						RestartPolicy:         corev1.RestartPolicyNever,
						ActiveDeadlineSeconds: ptr.To(tensorboardMaxRuntimeSeconds),
						ImagePullSecrets:      b.WorkloadConfig.ImagePullSecrets,
						NodeSelector:          b.WorkloadConfig.NodeSelector,
						Tolerations:           b.WorkloadConfig.Tolerations,
						Affinity:              b.WorkloadConfig.Affinity,
						Containers: []corev1.Container{
							{
								Name:            "tensorboard",
								Image:           b.WorkloadConfig.Image,
								ImagePullPolicy: b.WorkloadConfig.ImagePullPolicy,
								Command:         cmd,
								VolumeMounts:    volumeMounts,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: tensorboardPort,
										Name:          "http",
									},
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: ingressPath + "/",
											Port: intstr.FromInt(tensorboardPort),
										},
									},
									InitialDelaySeconds: 2,
									PeriodSeconds:       5,
									TimeoutSeconds:      2,
									FailureThreshold:    24,
								},
								Resources: resources,
							},
						},
						Volumes: volumes,
					},
				},
			}},
		},
	}

	return job
}
