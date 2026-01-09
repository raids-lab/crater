package packer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
)

func (b *imagePacker) CreateFromDockerfile(c context.Context, data *BuildKitReq) error {
	buildkitContainer := b.generateBuildKitContainer(data)
	volumes := b.generateVolumes(data.JobName)
	var configMap *corev1.ConfigMap
	var err error
	if configMap, err = b.createDockerfileConfigMap(c, data); err != nil {
		return err
	}
	var job *batchv1.Job
	if job, err = b.createJob(c, data, buildkitContainer, volumes); err != nil {
		return err
	}

	if err := b.updateOwnerReference(c, configMap, job); err != nil {
		return err
	}

	return nil
}

func (b *imagePacker) generateVolumes(jobName string) []corev1.Volume {
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
			Name: "configmap-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: jobName,
					},
				},
			},
		},
	}
	return volumes
}

func (b *imagePacker) generateBuildKitContainer(data *BuildKitReq) []corev1.Container {
	output := fmt.Sprintf("type=image,name=%s,push=true", data.ImageLink)
	archs := strings.Join(data.Archs, ",")

	// 判断archs字符串是否包含arm
	appendArmCmd := ""
	if strings.Contains(archs, "arm") {
		buildkitdArmNameSpace := fmt.Sprintf("%s.%s", buildkitdArmName, config.GetConfig().Namespaces.Image)
		appendArmCmd = fmt.Sprintf(`docker buildx create \
		--name multi-platform-builder \
		--append \
		--node arm-node \
		--driver remote tcp://%s:1234 && \
		`, buildkitdArmNameSpace)
	}

	buildkitdAmdNameSpace := fmt.Sprintf("%s.%s", buildkitdAmdName, config.GetConfig().Namespaces.Image)
	// 构建完整的命令
	cmd := fmt.Sprintf(`
		docker buildx create \
		--name multi-platform-builder \
		--node amd-node \
		--driver remote tcp://%s:1234 && \
		%s	docker buildx use multi-platform-builder && \
		docker buildx build --progress plain --platform %s --file /workspace/Dockerfile --output %s /workspace
	`, buildkitdAmdNameSpace, appendArmCmd, archs, output)

	setupCommands := []string{
		"/bin/sh",
		"-c",
		cmd,
	}

	buildkitContainer := []corev1.Container{
		{
			Name:  "buildkit",
			Image: config.GetConfig().Registry.BuildTools.Images.Buildx,
			Args:  setupCommands,
			Env: []corev1.EnvVar{
				{
					Name:  "DOCKER_CONFIG",
					Value: "/.docker",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "harborcredits",
					MountPath: "/.docker/config.json",
					SubPath:   "config.json", // 只挂载文件，不覆盖目录
					ReadOnly:  true,
				},
				{
					Name:      "configmap-volume",
					MountPath: "/workspace",
					ReadOnly:  true,
				},
			},
		},
	}
	return buildkitContainer
}

func (b *imagePacker) DeleteJob(c context.Context, jobName, ns string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: ns,
		},
	}
	deletePolicy := metav1.DeletePropagationForeground
	err := b.client.Delete(c, job, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
	return err
}

func (b *imagePacker) createDockerfileConfigMap(c context.Context, data *BuildKitReq) (*corev1.ConfigMap, error) {
	var requirements string
	if data.Requirements == nil {
		requirements = ""
	} else {
		requirements = *data.Requirements
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.JobName,
			Namespace: data.Namespace,
		},
		Data: map[string]string{
			"Dockerfile":       *data.Dockerfile,
			"requirements.txt": requirements,
		},
	}
	err := b.client.Create(c, configMap)
	return configMap, err
}

func (b *imagePacker) updateOwnerReference(c context.Context, configMap *corev1.ConfigMap, job *batchv1.Job) error {
	ownerReference := metav1.OwnerReference{
		APIVersion:         "batch/v1",
		Kind:               "Job",
		Name:               job.Name,
		UID:                job.UID,
		Controller:         ptr.To(true),
		BlockOwnerDeletion: ptr.To(true),
	}
	fmt.Printf("ownerReference: %+v\n", ownerReference)
	configMap.OwnerReferences = append(configMap.OwnerReferences, ownerReference)
	err := b.client.Update(c, configMap)
	fmt.Printf("configMap: %+v\n", configMap)
	return err
}

func (b *imagePacker) createJob(
	c context.Context,
	data *BuildKitReq,
	buildkitContainer []corev1.Container,
	volumes []corev1.Volume,
) (*batchv1.Job, error) {
	tagsString, _ := json.Marshal(data.Tags)
	archString, _ := json.Marshal(data.Archs)
	fmt.Printf("archString: %s\n", archString)
	jobMeta := metav1.ObjectMeta{
		Name:      data.JobName,
		Namespace: data.Namespace,
		Annotations: map[string]string{
			AnnotationKeyUserID:      fmt.Sprint(data.UserID),
			AnnotationKeyImageLink:   data.ImageLink,
			AnnotationKeyScript:      *data.Dockerfile,
			AnnotationKeyDescription: *data.Description,
			AnnotationKeyTags:        string(tagsString),
			AnnotationKeySource:      string(data.BuildSource),
			AnnotationKeyTemplate:    data.Template,
			AnnotationKeyArchs:       string(archString),
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
				RestartPolicy: corev1.RestartPolicyNever,
				Containers:    buildkitContainer,
				Volumes:       volumes,
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeUnconfined,
					},
					RunAsUser:  &runAsUerNumber,
					RunAsGroup: &runAsGroupNumber,
					FSGroup:    &fsAsGroupNumber,
				},
				EnableServiceLinks: ptr.To(false),
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "amd64",
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
	fmt.Printf("job: %+v\n", job)
	return job, err
}
