package packer

import (
	"context"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/pkg/config"
)

func (b *imagePacker) CreateFromEnvd(c context.Context, data *EnvdReq) error {
	envdContainer := b.generateEnvdContainer(data)
	var err error
	var configMap *corev1.ConfigMap
	if configMap, err = b.createEnvdConfigMap(c, data); err != nil {
		return err
	}
	volumes := b.generateVolumes(data.JobName)
	var job *batchv1.Job
	if job, err = b.createEnvdJob(c, data, envdContainer, volumes); err != nil {
		return err
	}

	if err := b.updateOwnerReference(c, configMap, job); err != nil {
		return err
	}

	return nil
}

func (b *imagePacker) generateEnvdContainer(data *EnvdReq) []corev1.Container {
	output := fmt.Sprintf("type=image,name=%s,push=true", data.ImageLink)
	buildkitdService := fmt.Sprintf("%s.%s", buildkitdAmdName, config.GetConfig().Namespaces.Image)
	backendHost := config.GetConfig().Host
	// 构建完整的命令，先创建context，再执行build
	cmd := fmt.Sprintf(`
                envd context create \
                --name buildkitd \
                --builder tcp \
                --builder-address %s:1234 \
                --use && \
                envd build --platform linux/amd64 --output %s
        `, buildkitdService, output)

	setupCommands := []string{
		"/bin/sh",
		"-c",
		cmd,
	}
	envVars := []corev1.EnvVar{
		{
			Name:  "DOCKER_CONFIG",
			Value: "/.docker",
		},
		{
			Name:  "NO_PROXY",
			Value: fmt.Sprintf("$(NO_PROXY),%s,%s", buildkitdService, backendHost),
		},
	}
	httpsProxy := config.GetConfig().Registry.BuildTools.ProxyConfig.HTTPSProxy
	if httpsProxy != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: httpsProxy,
		})
	}
	httpProxy := config.GetConfig().Registry.BuildTools.ProxyConfig.HTTPProxy
	if httpProxy != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: httpProxy,
		})
	}
	envdContainer := []corev1.Container{
		{
			Name:  "buildkit",
			Image: config.GetConfig().Registry.BuildTools.Images.Envd,
			Args:  setupCommands,
			Env:   envVars,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "harborcredits",
					MountPath: "/.docker",
				},
				{
					Name:      "configmap-volume",
					MountPath: "/workspace",
					ReadOnly:  true,
				},
			},
		},
	}
	return envdContainer
}

func (b *imagePacker) createEnvdConfigMap(c context.Context, data *EnvdReq) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.JobName,
			Namespace: data.Namespace,
		},
		Data: map[string]string{
			"build.envd": *data.Envd,
		},
	}
	err := b.client.Create(c, configMap)
	return configMap, err
}

func (b *imagePacker) createEnvdJob(
	c context.Context,
	data *EnvdReq,
	envdContainer []corev1.Container,
	volumes []corev1.Volume,
) (*batchv1.Job, error) {
	tagsString, _ := json.Marshal(data.Tags)
	archString, _ := json.Marshal(data.Archs)
	jobMeta := metav1.ObjectMeta{
		Name:      data.JobName,
		Namespace: data.Namespace,
		Annotations: map[string]string{
			AnnotationKeyUserID:      fmt.Sprint(data.UserID),
			AnnotationKeyImageLink:   data.ImageLink,
			AnnotationKeyScript:      *data.Envd,
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
			},
			Spec: corev1.PodSpec{
				RestartPolicy:      corev1.RestartPolicyNever,
				Containers:         envdContainer,
				Volumes:            volumes,
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
