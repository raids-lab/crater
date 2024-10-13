package vcjob

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

func (mgr *VolcanojobMgr) CreatePytorchJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateTensorflowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// base URL
	jobName := fmt.Sprintf("py-%s-%s", token.Username, uuid.New().String()[:5])

	// 1. Volume Mounts
	volumes, volumeMounts, err := GenerateVolumeMounts(c, token.UserID, req.VolumeMounts)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 2. TODO: Node Affinity for ARM64 Nodes
	affinity := GenerateNodeAffinity(req.Selectors)

	// 3. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "pytorch",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  jobName,
	}
	annotations := map[string]string{
		AnnotationKeyTaskName: req.Name,
	}

	// 4. Create the task spec
	tasks := make([]batch.TaskSpec, len(req.Tasks))
	minAvailable := int32(0)
	for i := range req.Tasks {
		// 4.1. Generate ports
		task := &req.Tasks[i]
		ports := make([]v1.ContainerPort, len(task.Ports))
		for j, port := range task.Ports {
			ports[j] = v1.ContainerPort{
				ContainerPort: port.Port,
				Name:          port.Name,
				Protocol:      v1.ProtocolTCP,
			}
		}
		// 4.2. Generate pod spec
		podSpec := generatePodSpec(task, affinity, volumes, volumeMounts, req.Envs, ports)

		// 4.3. Create task spec
		taskSpec := batch.TaskSpec{
			Name:     task.Name,
			Replicas: task.Replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: podSpec,
			},
		}

		if task.Name == "master" {
			taskSpec.Policies = []batch.LifecyclePolicy{
				{
					Action: bus.CompleteJobAction,
					Event:  bus.TaskCompletedEvent,
				},
			}
			minAvailable = task.Replicas
		} else if task.Name == "worker" {
			taskSpec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		}

		tasks[i] = taskSpec
	}

	// 5. Create volcano job
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batch.JobSpec{
			MinAvailable:  minAvailable,
			SchedulerName: VolcanoSchedulerName,
			Plugins: map[string][]string{
				"pytorch": {"--master=master", "--worker=worker", "--port=23456"},
			},
			Queue: token.QueueName,
			Tasks: tasks,
		},
	}

	if err = mgr.client.Create(c, &job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}
