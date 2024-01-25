package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type JobControl struct {
	client.Client
}

func (c *JobControl) GetJobStatus(task *models.AITask) (aijobapi.JobPhase, error) {
	// todo: 存储jobname到数据库
	ns := task.Namespace
	job := &aijobapi.AIJob{}
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: ns,
		Name:      task.JobName,
	}, job)
	if err != nil {
		return "", err
	}
	return job.Status.Phase, nil
}

func (c *JobControl) DeleteJobFromTask(task *models.AITask) error {

	ns := task.Namespace
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.JobName,
			Namespace: ns,
		},
	}
	err := c.Delete(context.Background(), job)

	// 对于 Jupyter 类型，还需要删除 Service
	if task.TaskType == models.JupyterTask {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.JobName,
				Namespace: ns,
			},
		}
		err = c.Delete(context.Background(), svc)
	}

	return err
}

func (c *JobControl) createTrainingJobFromTask(task *models.AITask) (jobname string, err error) {
	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		err = fmt.Errorf("resource request is not valid: %v", err)
		return
	}

	// convert metadata to lower case
	taskName := strings.ToLower(task.TaskName)
	jobname = fmt.Sprintf("%s-%d", taskName, task.ID)
	jobname = strings.Replace(jobname, "_", "-", -1)
	taskID := strconv.Itoa(int(task.ID))

	// set labels and annotations
	labels := map[string]string{
		aijobapi.LabelKeyTaskID:        taskID,
		aijobapi.LabelKeyTaskUser:      task.UserName,
		aijobapi.LabelKeyTaskType:      task.TaskType,
		aijobapi.LabelKeyTaskSLO:       strconv.FormatUint(uint64(task.SLO), 10),
		aijobapi.JobNameLabel:          jobname,
		aijobapi.LabelKeyEstimatedTime: strconv.FormatUint(uint64(task.EsitmatedTime), 10),
	}

	annotations := make(map[string]string)
	annotations[aijobapi.AnnotationKeyProfileStat] = task.ProfileStat
	if task.ProfileStatus == models.ProfileFinish {
		annotations[aijobapi.AnnotationKeyPreemptCount] = "0"
	}

	// set priority class
	var priorityClassName string
	if task.SLO == 0 {
		priorityClassName = models.PriorityClassBestEffort
	} else {
		priorityClassName = models.PriorityClassGauranteed
	}

	volumes, volumeMounts, err := GenVolumeAndMountsFromAITask(task)
	if err != nil {
		err = fmt.Errorf("gen volumes and mounts failed: %v", err)
		return
	}
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobname,
			Namespace:   task.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: aijobapi.JobSpec{
			Replicas: 1,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        jobname,
					Labels:      labels,
					Namespace:   task.Namespace,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					PriorityClassName: priorityClassName,
					RestartPolicy:     corev1.RestartPolicyNever,
					NodeSelector: map[string]string{
						"v100": "true", // todo: for test
					},
					Containers: []corev1.Container{
						{
							Image:   task.Image,
							Name:    "main",
							Command: []string{"/bin/bash", "-c", task.Command}, // todo:
							// Args:    models.JSONStringToList(task.Args),        // todo:
							Resources: corev1.ResourceRequirements{
								Limits:   resourceRequest,
								Requests: resourceRequest,
							},
							WorkingDir:   task.WorkingDir,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
			ResourceRequest: resourceRequest,
		},
	}
	err = c.Create(context.Background(), job)
	if err != nil {
		err = fmt.Errorf("create job %s failed: %v", task.TaskName, err)
		return
	}
	return
}

func (c *JobControl) createJupyterJobFromTask(task *models.AITask) (jobname string, err error) {
	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		err = fmt.Errorf("resource request is not valid: %v", err)
		return
	}

	// convert metadata to lower case
	username := strings.ToLower(task.UserName)
	jobname = fmt.Sprintf("jupyter-%s-singleuser-%d", username, task.ID)
	jobname = strings.Replace(jobname, "_", "-", -1)
	taskID := strconv.Itoa(int(task.ID))

	// set labels and annotations
	labels := map[string]string{
		aijobapi.LabelKeyTaskID:        taskID,
		aijobapi.LabelKeyTaskUser:      task.UserName,
		aijobapi.LabelKeyTaskType:      task.TaskType,
		aijobapi.LabelKeyTaskSLO:       strconv.FormatUint(uint64(task.SLO), 10),
		aijobapi.JobNameLabel:          jobname,
		aijobapi.LabelKeyEstimatedTime: strconv.FormatUint(uint64(task.EsitmatedTime), 10),
	}

	annotations := make(map[string]string)
	annotations[aijobapi.AnnotationKeyProfileStat] = task.ProfileStat
	if task.ProfileStatus == models.ProfileFinish {
		annotations[aijobapi.AnnotationKeyPreemptCount] = "0"
	}

	// set priority class
	priorityClassName := models.PriorityClassGauranteed

	volumes, volumeMounts, err := GenVolumeAndMountsFromAITask(task)
	if err != nil {
		err = fmt.Errorf("gen volumes and mounts failed: %v", err)
		return
	}
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobname,
			Namespace:   task.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: aijobapi.JobSpec{
			Replicas: 1,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        jobname,
					Namespace:   task.Namespace,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					PriorityClassName: priorityClassName,
					RestartPolicy:     corev1.RestartPolicyNever,
					// NodeSelector: map[string]string{
					// 	"v100": "true", // todo: for test
					// },
					Containers: []corev1.Container{
						{
							Image:           task.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "notebook",
							Command:         []string{"/bin/bash", "-c", task.Command},
							Env: []corev1.EnvVar{
								{Name: "GRANT_SUDO", Value: "1"},
								{Name: "CHOWN_HOME", Value: "1"},
								{Name: "NB_UID", Value: "1005"},
								{Name: "NB_USER", Value: username},
								// {Name: "NVIDIA_VISIBLE_DEVICES", Value: "GPU-5a51526f-bffa-b9d5-1dd5-346fe90b5abf"},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8888, Name: "notebook-port", Protocol: corev1.ProtocolTCP},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   resourceRequest,
								Requests: resourceRequest,
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: func(b bool) *bool { return &b }(true),
								RunAsUser:                func(i int64) *int64 { return &i }(0),
								RunAsGroup:               func(i int64) *int64 { return &i }(0),
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
							WorkingDir:               task.WorkingDir,
							VolumeMounts:             volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
			ResourceRequest: resourceRequest,
		},
	}
	err = c.Create(context.Background(), job)
	if err != nil {
		err = fmt.Errorf("create job %s failed: %v", task.TaskName, err)
		return
	}

	// 创建 Service，转发 Jupyter 端口
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobname,
			Namespace: task.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				aijobapi.LabelKeyTaskID: taskID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       jobname,
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8888),
					NodePort:   0, // Kubernetes will allocate a port
				},
			},
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeNodePort,
		},
	}

	err = c.Create(context.Background(), svc)
	return
}

// task.TaskType 目前有两种类型：training 和 jupyter，如果是 jupyter，则同时创建随机的端口转发
func (c *JobControl) CreateJobFromTask(task *models.AITask) (jobname string, err error) {
	if task.TaskType == models.TrainingTask {
		return c.createTrainingJobFromTask(task)
	} else if task.TaskType == models.JupyterTask {
		return c.createJupyterJobFromTask(task)
	} else {
		err = fmt.Errorf("task type is not valid: %v", task.TaskType)
		return
	}
}

func GenVolumeAndMountsFromAITask(task *models.AITask) ([]corev1.Volume, []corev1.VolumeMount, error) {

	// set volumes
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "user-volume",
			MountPath: "/home/" + task.UserName,
		},
		{
			Name:      "cache-volume",
			MountPath: "/dev/shm",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "user-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf(UserHomePVC, task.UserName),
				},
			},
		},
		{
			Name: "cache-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}
	if task.ShareDirs != "" {
		taskShareDir := models.JSONStringToVolumes(task.ShareDirs)
		if taskShareDir == nil {
			logrus.Errorf("parse task share dir failed： %v", task.ShareDirs)
			return nil, nil, fmt.Errorf("parse task share dir failed： %v", task.ShareDirs)
		}
		for pvc, mounts := range taskShareDir {

			volumes = append(volumes, corev1.Volume{
				Name: pvc,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			})
			for _, mount := range mounts {
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      pvc,
					MountPath: mount.MountPath,
					SubPath:   mount.SubPath,
				})
			}
		}
	}
	return volumes, volumeMounts, nil
}
