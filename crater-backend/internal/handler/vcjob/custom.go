package vcjob

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
)

type (
	CreateCustomReq struct {
		CreateJobCommon `json:",inline"`
		Name            string          `json:"name" binding:"required"`
		Resource        v1.ResourceList `json:"resource"`
		Image           string          `json:"image" binding:"required"`
		Command         string          `json:"command" binding:"required"`
		WorkingDir      string          `json:"workingDir" binding:"required"`
		Template        string          `json:"template"`
		AlertEnabled    bool            `json:"alertEnabled"`
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
// @Router /v1/vcjobs/training [post]
func (mgr *VolcanojobMgr) CreateTrainingJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateCustomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.AccountID, req.Resource)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.NotSpecified)
		return
	}

	// base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("single-%s", baseURL)

	// useTensorBoard
	useTensorboard := fmt.Sprintf("%t", req.UseTensorBoard)

	// 4. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: string(model.JobTypeCustom),
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}
	annotations := map[string]string{
		AnnotationKeyTaskName:       req.Name,
		AnnotationKeyTaskTemplate:   req.Template,
		AnnotationKeyUseTensorBoard: useTensorboard,
		AnnotationKeyAlertEnabled:   strconv.FormatBool(req.AlertEnabled),
	}

	// 5. Create the pod spec
	podSpec, err := GenerateCustomPodSpec(c, token.UserID, &req)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 6. Create volcano job
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(ThreeDaySeconds),
			MinAvailable:            1,
			MaxRetry:                1,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   token.AccountName,
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
			},
			Tasks: []batch.TaskSpec{
				{
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      labels,
							Annotations: annotations,
						},
						Spec: podSpec,
					},
					Policies: []batch.LifecyclePolicy{
						{
							Action: bus.CompleteJobAction,
							Event:  bus.TaskCompletedEvent,
						},
					},
				},
			},
		},
	}

	if err = mgr.client.Create(c, &job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

func GenerateCustomPodSpec(
	ctx context.Context,
	userID uint,
	custom *CreateCustomReq,
) (podSpec v1.PodSpec, err error) {
	volumes, volumeMounts, err := GenerateVolumeMounts(ctx, userID, custom.VolumeMounts)
	if err != nil {
		return podSpec, err
	}
	affinity := GenerateNodeAffinity(custom.Selectors, false)

	podSpec = v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:       custom.Name,
				Image:      custom.Image,
				Command:    []string{"sh", "-c", custom.Command},
				WorkingDir: custom.WorkingDir,
				Resources: v1.ResourceRequirements{
					Limits:   custom.Resource,
					Requests: custom.Resource,
				},
				Env:   custom.Envs,
				Ports: []v1.ContainerPort{},
				SecurityContext: &v1.SecurityContext{
					RunAsUser:  ptr.To(int64(0)),
					RunAsGroup: ptr.To(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
		RestartPolicy: v1.RestartPolicyNever,
	}
	return podSpec, nil
}
