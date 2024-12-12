package buildkit

import (
	"context"
	"fmt"
	"sync"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type buildkitMgr struct {
	client client.Client
	BuildKitInterface
}

var (
	once             sync.Once
	b                *buildkitMgr
	runAsUerNumber   int64 = 1000
	runAsGroupNumber int64 = 1000
	fsAsGroupNumber  int64 = 1000

	buildkitSecretName   string = "buildkit-secret"
	buildkitCachePVCName string = "buildkit-cache"

	JobCleanTime       int32 = 259200
	BackoffLimitNumber int32 = 0
	CompletionNumber   int32 = 1
	ParallelismNumber  int32 = 1
)

func GetBuildKitMgr(cli client.Client) BuildKitInterface {
	once.Do(func() {
		b = &buildkitMgr{
			client: cli,
		}
	})
	return b
}

func (b *buildkitMgr) CreateFromDockerfile(c context.Context, data *BuildKitData) error {
	initContainer := b.generateInitContainer(data)
	buildkitContainer := b.generateBuildKitContainer(data)
	volumes := b.generateVolumes()

	jobMeta := metav1.ObjectMeta{
		Name:      data.JobName,
		Namespace: data.Namespace,
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

func (b *buildkitMgr) generateInitContainer(data *BuildKitData) []corev1.Container {
	dockerfileSource := fmt.Sprintf("echo '%s' > /workspace/Dockerfile", data.Dockerfile)
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

func (b *buildkitMgr) generateVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "buildkitd",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
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
		{
			Name: "cache",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: buildkitCachePVCName,
				},
			},
		},
	}
	return volumes
}

func (b *buildkitMgr) generateBuildKitContainer(data *BuildKitData) []corev1.Container {
	output := fmt.Sprintf("type=image,name=%s,push=true", data.ImageLink)
	buildArgs := []string{
		"build",
		"--frontend", "dockerfile.v0",
		"--local", "context=/workspace",
		"--local", "dockerfile=/workspace",
		"--export-cache", "type=local,dest=/cache",
		"--import-cache", "type=local,src=/cache",
		"--output", output,
	}
	buildkitContainer := []corev1.Container{
		{
			Name:    "buildkit",
			Image:   "moby/buildkit:master-rootless",
			Command: []string{"buildctl-daemonless.sh"},
			Env: []corev1.EnvVar{
				{
					Name:  "BUILDKITD_FLAGS",
					Value: "--oci-worker-no-process-sandbox",
				},
				{
					Name:  "DOCKER_CONFIG",
					Value: "/.docker",
				},
			},
			Args: buildArgs,
			SecurityContext: &corev1.SecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeUnconfined,
				},
				RunAsUser:  &runAsUerNumber,
				RunAsGroup: &runAsGroupNumber,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
					ReadOnly:  true,
				},
				{
					Name:      "buildkitd",
					MountPath: "/home/user/.local/share/buildkit",
				},
				{
					Name:      "secret",
					MountPath: "/.docker",
				},
				{
					Name:      "cache",
					MountPath: "/cache",
				},
			},
		},
	}
	return buildkitContainer
}
