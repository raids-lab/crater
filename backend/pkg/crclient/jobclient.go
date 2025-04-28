package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"

	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
)

type JobControl struct {
	client.Client
	KubeClient kubernetes.Interface
	mu         sync.Mutex
}

const (
	jupyterPort   = 8888
	ServicePrefix = "svc-"
)

func (c *JobControl) GetJobStatus(task *model.AITask) (aijobapi.JobPhase, error) {
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

func (c *JobControl) DeleteJobFromTask(task *model.AITask) error {
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
	if task.TaskType == model.EmiasJupyterTask {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServicePrefix + task.JobName,
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
	ingress, err := ingressClient.Get(context.TODO(), config.GetConfig().Workspace.IngressName, metav1.GetOptions{})
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

func (c *JobControl) createTrainingJobFromTask(task *model.AITask) (jobname string, err error) {
	podSpec := task.PodTemplate.Data()
	if len(podSpec.Containers) == 0 {
		err = fmt.Errorf("no container in pod spec")
		return "", err
	}

	// convert metadata to lower case
	taskName := strings.ToLower(task.TaskName)
	jobname = fmt.Sprintf("%s-%d", taskName, task.ID)
	jobname = strings.ReplaceAll(jobname, "_", "-")
	//nolint:gosec // taskID is safe
	taskID := strconv.Itoa(int(task.ID))

	// set labels and annotations
	labels := map[string]string{
		aijobapi.LabelKeyTaskID:        taskID,
		aijobapi.LabelKeyTaskUser:      task.UserName,
		aijobapi.LabelKeyTaskType:      task.TaskType,
		aijobapi.LabelKeyTaskSLO:       strconv.FormatUint(uint64(task.SLO), 10),
		aijobapi.JobNameLabel:          jobname,
		aijobapi.LabelKeyEstimatedTime: strconv.FormatUint(uint64(task.EsitmatedTime), 10),
		LabelKeyBaseURL:                taskID,
		LabelKeyTaskType:               task.TaskType,
		LabelKeyTaskUser:               task.Owner,
	}

	annotations := make(map[string]string)
	annotations[aijobapi.AnnotationKeyProfileStat] = task.ProfileStat
	if task.ProfileStatus == model.EmiasProfileFinish {
		annotations[aijobapi.AnnotationKeyPreemptCount] = "0"
	}

	// set priority class
	if task.SLO == 0 {
		podSpec.PriorityClassName = model.EmiasPriorityClassBestEffort
	} else {
		podSpec.PriorityClassName = model.EmiasPriorityClassGauranteed
	}

	if podSpec.Affinity != nil {
		podSpec.SchedulerName = "default-scheduler"
	} else {
		podSpec.SchedulerName = "kube-gpu-colocate-scheduler"
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
				Spec: podSpec,
			},
			ResourceRequest: podSpec.Containers[0].Resources.Requests,
		},
	}
	err = c.Create(context.Background(), job)
	if err != nil {
		err = fmt.Errorf("create job %s: %w", task.TaskName, err)
		return "", err
	}
	return jobname, nil
}

func (c *JobControl) createJupyterJobFromTask(task *model.AITask) (jobname string, err error) {
	podSpec := task.PodTemplate.Data()
	if len(podSpec.Containers) == 0 {
		err = fmt.Errorf("no container in pod spec")
		return "", err
	}

	// convert metadata to lower case
	taskName := strings.ToLower(task.TaskName)
	jobname = fmt.Sprintf("%s-%d", taskName, task.ID)
	jobname = strings.ReplaceAll(jobname, "_", "-")
	//nolint:gosec // taskID is safe
	taskID := strconv.Itoa(int(task.ID))

	// set labels and annotations
	labels := map[string]string{
		aijobapi.LabelKeyTaskID:        taskID,
		aijobapi.LabelKeyTaskUser:      task.UserName,
		aijobapi.LabelKeyTaskType:      task.TaskType,
		aijobapi.LabelKeyTaskSLO:       strconv.FormatUint(uint64(task.SLO), 10),
		aijobapi.JobNameLabel:          jobname,
		aijobapi.LabelKeyEstimatedTime: strconv.FormatUint(uint64(task.EsitmatedTime), 10),
		LabelKeyBaseURL:                taskID,
		LabelKeyTaskType:               task.TaskType,
		LabelKeyTaskUser:               task.Owner,
	}

	annotations := make(map[string]string)
	annotations[aijobapi.AnnotationKeyProfileStat] = task.ProfileStat
	if task.ProfileStatus == model.EmiasProfileFinish {
		annotations[aijobapi.AnnotationKeyPreemptCount] = "0"
	}

	// set priority class
	podSpec.PriorityClassName = model.EmiasPriorityClassGauranteed

	if podSpec.Affinity != nil {
		podSpec.SchedulerName = "default-scheduler"
	} else {
		podSpec.SchedulerName = "kube-gpu-colocate-scheduler"
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
				Spec: podSpec,
			},
			ResourceRequest: podSpec.Containers[0].Resources.Requests,
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
			Name:      ServicePrefix + jobname,
			Namespace: task.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "aisystem.github.com/v1alpha1",
					Kind:               "AIJob",
					Name:               job.Name,
					UID:                job.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			},
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
			Type:            corev1.ServiceTypeNodePort,
		},
	}

	err = c.Create(context.Background(), svc)
	if err != nil {
		err = fmt.Errorf("create service %s: %w", task.TaskName, err)
		return "", err
	}

	return jobname, nil
}

// task.TaskType 目前有两种类型：training 和 jupyter，如果是 jupyter，则同时创建随机的端口转发
func (c *JobControl) CreateJobFromTask(task *model.AITask) (jobname string, err error) {
	switch task.TaskType {
	case model.EmiasTrainingTask:
		return c.createTrainingJobFromTask(task)
	case model.EmiasJupyterTask:
		return c.createJupyterJobFromTask(task)
	default:
		err = fmt.Errorf("task type is not valid: %v", task.TaskType)
		return
	}
}

func GenVolumeAndMountsFromAITask(task *model.AITask) ([]corev1.Volume, []corev1.VolumeMount, error) {
	// set volumes
	// task.UserName is in format of "userid-projectid"
	splited := strings.Split(task.UserName, "-")
	if len(splited) != 2 {
		return nil, nil, fmt.Errorf("user name is not valid: %v", task.UserName)
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "personal-volume",
			MountPath: "/home/" + task.UserName,
			SubPath:   "public",
		},
		{
			Name:      "cache-volume",
			MountPath: "/dev/shm",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "personal-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.GetConfig().Workspace.RWXPVCName,
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
		taskShareDir := model.JSONStringToVolumes(task.ShareDirs)
		if taskShareDir == nil {
			logutils.Log.Errorf("parse task share dir: %v", task.ShareDirs)
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

func (c *JobControl) GetNodeNameFromTask(ctx context.Context, task *model.AITask) (string, error) {
	//nolint:gosec // taskID is safe
	taskID := strconv.Itoa(int(task.ID))
	podLabels := map[string]string{
		LabelKeyBaseURL:  taskID,
		LabelKeyTaskType: task.TaskType,
		LabelKeyTaskUser: task.Owner,
	}
	podList := &corev1.PodList{}
	err := c.List(ctx, podList, client.InNamespace(task.Namespace), client.MatchingLabels(podLabels))
	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found for task %d", task.ID)
	}

	return podList.Items[0].Spec.NodeName, nil
}
