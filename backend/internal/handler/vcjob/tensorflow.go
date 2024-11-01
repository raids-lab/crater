package vcjob

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

type (
	PortReq struct {
		Name string `json:"name"`
		Port int32  `json:"port"`
	}

	TaskReq struct {
		Name       string          `json:"name"`
		Replicas   int32           `json:"replicas"`
		Resource   v1.ResourceList `json:"resource"`
		Image      string          `json:"image"`
		Command    *string         `json:"command"`
		WorkingDir *string         `json:"workingDir"`
		Ports      []PortReq       `json:"ports"`
	}

	CreateTensorflowReq struct {
		CreateJobCommon `json:",inline"`
		Name            string    `json:"name" binding:"required"`
		Tasks           []TaskReq `json:"tasks"`
	}
)

// CreateTrainingJob godoc
// @Summary Create a training job
// @Description Create a training job
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param CreateTrainingReq body any true "CreateTrainingReq"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/tensorflow [post]
func (mgr *VolcanojobMgr) CreateTensorflowJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateTensorflowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobResources := v1.ResourceList{}
	for i := range len(req.Tasks) {
		jobResources = aitaskctl.AddResourceList(jobResources, req.Tasks[i].Resource)
	}
	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.QueueID, jobResources)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.NotSpecified)
		return
	}

	// Ingress base URL
	jobName := fmt.Sprintf("tf-%s-%s", token.Username, uuid.New().String()[:5])

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
		LabelKeyTaskType: "tensorflow",
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
		task := &req.Tasks[i]
		// 4.1. Generate ports
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

		if task.Name == "worker" {
			taskSpec.Policies = []batch.LifecyclePolicy{
				{
					Action: bus.CompleteJobAction,
					Event:  bus.TaskCompletedEvent,
				},
			}
		}

		minAvailable += task.Replicas
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
				"env": {},
				"svc": {},
			},
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
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
