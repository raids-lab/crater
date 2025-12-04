package vcjob

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

type (
	CreateCustomReq struct {
		CreateJobCommon `json:",inline"`
		Resource        v1.ResourceList `json:"resource"`
		Image           ImageBaseInfo   `json:"image" binding:"required"`
		Shell           *string         `json:"shell"`
		Command         *string         `json:"command"`
		WorkingDir      string          `json:"workingDir" binding:"required"`
	}
)

// CreateTrainingJob godoc
//
//	@Summary		Create a training job
//	@Description	Create a training job
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			CreateTrainingReq	body		any						true	"CreateTrainingReq"
//	@Success		200					{object}	resputil.Response[any]	"Success"
//	@Failure		400					{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500					{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/training [post]
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

	// 如果希望接受邮件，则需要确保邮箱已验证
	if req.AlertEnabled && !utils.CheckUserEmail(c, token.UserID) {
		resputil.Error(c, "Email not verified", resputil.UserEmailNotVerified)
		return
	}

	// Generate job name with type prefix (RFC 1035 compliant)
	jobName := utils.GenerateJobName("sg", token.Username)
	// baseURL for ingress paths (without type prefix)
	baseURL := jobName[3:] // Remove "sg-" prefix

	// 4. Labels and Annotations
	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeCustom,
		token,
		baseURL,
		req.Name,
		req.Template,
		req.AlertEnabled,
	)

	// 5. Create the pod spec
	podSpec, err := GenerateCustomPodSpec(c, token, &req)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 6. Create volcano job
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(ThreeDaySeconds),
			MinAvailable:            1,
			MaxRetry:                1,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   token.AccountName,
			Plugins:                 volcanoPlugins,
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
							Annotations: podAnnotations,
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

	// create forward ing rules in template
	if err := mgr.CreateForwardIngresses(c, &job, req.Forwards, labels, token.Username); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

func GenerateCustomPodSpec(
	ctx context.Context,
	token util.JWTMessage,
	custom *CreateCustomReq,
) (podSpec v1.PodSpec, err error) {
	volumes, volumeMounts, err := GenerateVolumeMounts(ctx, custom.VolumeMounts, token)
	if err != nil {
		return podSpec, err
	}

	baseAffinity := GenerateNodeAffinity(custom.Selectors, custom.Resource)
	affinity := GenerateArchitectureNodeAffinity(custom.Image, baseAffinity)
	fmt.Printf("Affinity generated: %+v\n", affinity)
	tolerations := GenerateTaintTolerationsForAccount(token)
	envs := GenerateEnvs(ctx, token, custom.Envs)

	imagePullSecrets := []v1.LocalObjectReference{}
	if config.GetConfig().Secrets.ImagePullSecretName != "" {
		imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{
			Name: config.GetConfig().Secrets.ImagePullSecretName,
		})
	}

	podSpec = v1.PodSpec{
		Affinity:         affinity,
		Tolerations:      tolerations,
		Volumes:          volumes,
		ImagePullSecrets: imagePullSecrets,
		Containers: []v1.Container{
			{
				Name:       string(CraterJobTypeCustom),
				Image:      custom.Image.ImageLink,
				WorkingDir: custom.WorkingDir,
				Resources: v1.ResourceRequirements{
					Limits:   custom.Resource,
					Requests: custom.Resource,
				},
				Env:   envs,
				Ports: []v1.ContainerPort{},
				SecurityContext: &v1.SecurityContext{
					RunAsUser:  ptr.To(int64(0)),
					RunAsGroup: ptr.To(int64(0)),
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{"IPC_LOCK"},
					},
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
		RestartPolicy:      v1.RestartPolicyNever,
		EnableServiceLinks: ptr.To(false),
	}

	if custom.Command != nil && *custom.Command != "" {
		if custom.Shell == nil {
			custom.Shell = ptr.To("sh")
		}
		podSpec.Containers[0].Command = []string{*custom.Shell, "-c", *custom.Command}
	}

	return podSpec, nil
}
