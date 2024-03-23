package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/models"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type JobControl struct {
	client.Client
	KubeClient kubernetes.Interface
	mu         sync.Mutex
}

const (
	jupyterPort = 8888
)

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
	if err != nil {
		err = fmt.Errorf("delete job %s failed: %w", task.JobName, err)
		return err
	}

	// 对于 Jupyter 类型，还需要删除 Service
	if task.TaskType == models.JupyterTask {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.JobName,
				Namespace: ns,
			},
		}
		err = c.Delete(context.Background(), svc)
		if err != nil {
			err = fmt.Errorf("delete service %s: %w", task.JobName, err)
			return err
		}
	}

	// 此外，还需要删除 Ingress

	// 添加锁
	c.mu.Lock()
	defer c.mu.Unlock()

	ingressClient := c.KubeClient.NetworkingV1().Ingresses(ns)

	// Get the existing Ingress
	ingress, err := ingressClient.Get(context.TODO(), "crater-jobs-ingress", metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("get crater-jobs-ingress: %w", err)
		return err
	}

	// Remove the path from the first rule
	for i, path := range ingress.Spec.Rules[0].HTTP.Paths {
		if strings.Contains(path.Path, task.JobName) {
			ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths[:i], ingress.Spec.Rules[0].HTTP.Paths[i+1:]...)
			break
		}
	}

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		err = fmt.Errorf("update ingress: %w", err)
		return err
	}

	return err
}

func (c *JobControl) createTrainingJobFromTask(task *models.AITask, selector map[string]string) (jobname string, err error) {
	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		err = fmt.Errorf("resource request is not valid: %w", err)
		return "", err
	}

	// convert metadata to lower case
	taskName := strings.ToLower(task.TaskName)
	jobname = fmt.Sprintf("%s-%d", taskName, task.ID)
	jobname = strings.ReplaceAll(jobname, "_", "-")
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
		err = fmt.Errorf("gen volumes and mounts: %w", err)
		return "", err
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
					SchedulerName:     task.SchedulerName,
					PriorityClassName: priorityClassName,
					RestartPolicy:     corev1.RestartPolicyNever,
					NodeSelector:      selector,
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
		err = fmt.Errorf("create job %s: %w", task.TaskName, err)
		return "", err
	}
	return jobname, nil
}

func (c *JobControl) createJupyterJobFromTask(task *models.AITask, selector map[string]string) (jobname string, err error) {
	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		err = fmt.Errorf("resource request is not valid: %w", err)
		return "", err
	}

	// convert metadata to lower case
	username := strings.ToLower(task.UserName)
	jobname = fmt.Sprintf("%s-%d", username, task.ID)
	jobname = strings.ReplaceAll(jobname, "_", "-")
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
		err = fmt.Errorf("gen volumes and mounts: %w", err)
		return "", err
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
					SchedulerName:     task.SchedulerName,
					PriorityClassName: priorityClassName,
					RestartPolicy:     corev1.RestartPolicyNever,
					NodeSelector:      selector,
					Containers: []corev1.Container{
						{
							Image:           task.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "notebook",
							Command:         []string{"/bin/bash", "-c", fmt.Sprintf(task.Command, jobname)},
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
		err = fmt.Errorf("create job %s: %w", task.TaskName, err)
		return "", err
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
					TargetPort: intstr.FromInt(jupyterPort),
					// NodePort:   0, // Kubernetes will allocate a port
				},
			},
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeClusterIP,
		},
	}

	err = c.Create(context.Background(), svc)
	if err != nil {
		err = fmt.Errorf("create service %s: %w", task.TaskName, err)
		return "", err
	}

	// 添加锁
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建 Ingress，转发 Jupyter 端口
	ingressClient := c.KubeClient.NetworkingV1().Ingresses(task.Namespace)

	// Get the existing Ingress
	ingress, err := ingressClient.Get(context.TODO(), "crater-jobs-ingress", metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("get ingress: %w", err)
		return "", err
	}

	// Add a new path to the first rule
	newPath := networkingv1.HTTPIngressPath{
		Path:     fmt.Sprintf("/jupyter/%s", jobname),
		PathType: func(s networkingv1.PathType) *networkingv1.PathType { return &s }(networkingv1.PathTypePrefix),
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: jobname,
				Port: networkingv1.ServiceBackendPort{
					Number: 80,
				},
			},
		},
	}

	ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths, newPath)

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		err = fmt.Errorf("update ingress: %w", err)
		return "", err
	}

	return jobname, nil
}

// task.TaskType 目前有两种类型：training 和 jupyter，如果是 jupyter，则同时创建随机的端口转发
func (c *JobControl) CreateJobFromTask(task *models.AITask) (jobname string, err error) {
	// analyze task node selector
	matchExpressions := map[string]string{}
	// get models array from task.GPUModels string
	if task.GPUModel != "" {
		matchExpressions["act.crater/model"] = task.GPUModel
	}

	switch task.TaskType {
	case models.TrainingTask:
		return c.createTrainingJobFromTask(task, matchExpressions)
	case models.JupyterTask:
		return c.createJupyterJobFromTask(task, matchExpressions)
	default:
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
			logrus.Errorf("parse task share dir: %v", task.ShareDirs)
			return nil, nil, fmt.Errorf("parse task share dir: %v", task.ShareDirs)
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
