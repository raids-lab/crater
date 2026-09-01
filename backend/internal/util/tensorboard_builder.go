package util

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	AnnotationKeyExpirationTime = "crater.raids.io/expiration-time"
	LabelKeyTensorboardID       = "crater.raids.io/tensorboard-id"
	LabelKeyTypeTensorboard     = "tensorboard"
	DefaultTensorboardImage     = "crater-harbor.act.buaa.edu.cn/user-gederong27/tensorboard:2.20.0"
	TensorboardPort             = 6006
)

// Builder defines the utility to build K8s resources for Tensorboard
type Builder struct {
	Namespace string
}

// NewBuilder creates a new Builder
func NewBuilder(namespace string) *Builder {
	return &Builder{
		Namespace: namespace,
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// BuildDeployment creates a Deployment specification for Tensorboard Server.
func (b *Builder) BuildDeployment(
	tbID string,
	username string,
	logDir string,
	ingressPath string,
	ttlHours int32,
	volumes []corev1.Volume,
	volumeMounts []corev1.VolumeMount,
) *appsv1.Deployment {
	labels := map[string]string{
		"app":                     "tensorboard",
		LabelKeyTensorboardID:     tbID,
		crclient.LabelKeyTaskUser: username,
		crclient.LabelKeyTaskType: LabelKeyTypeTensorboard,
	}

	expirationTime := time.Now().Add(time.Duration(ttlHours) * time.Hour).Format(time.RFC3339)

	annotations := map[string]string{
		AnnotationKeyExpirationTime: expirationTime,
	}

	replicas := int32(1)

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
			TensorboardPort,
			shellQuote(ingressPath),
		),
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("tb-%s", tbID),
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         "tensorboard",
							Image:        DefaultTensorboardImage,
							Command:      cmd,
							VolumeMounts: volumeMounts,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: TensorboardPort,
									Name:          "http",
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: ingressPath + "/",
										Port: intstr.FromInt(TensorboardPort),
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
		},
	}

	return deploy
}
