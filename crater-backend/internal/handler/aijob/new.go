package aijob

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/resputil"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/util"
)

type CreateTaskResp struct {
	TaskID uint
}

func (mgr *AIJobMgr) CreateJupyterJob(c *gin.Context) {
	token := interutil.GetToken(c)
	var vcReq CreateJupyterReq
	if err := c.ShouldBindJSON(&vcReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req CreateTaskReq

	req.TaskName = vcReq.Name
	req.Namespace = config.GetConfig().Namespaces.Job
	req.UserName = token.AccountName
	req.SLO = 1
	req.TaskType = model.EmiasJupyterTask
	req.Image = vcReq.Image.ImageLink
	req.ResourceRequest = vcReq.Resource

	taskModel := model.FormatTaskAttrToModel(&req.TaskAttr)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --notebook-dir=/home/%s"
	command := fmt.Sprintf(commandSchema, token.Username)

	// 1. Volume Mounts
	volumes, volumeMounts, err := vcjob.GenerateVolumeMounts(c, vcReq.VolumeMounts, token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 1.1 Support NGC images
	if !strings.Contains(req.Image, "jupyter") {
		volumes = append(volumes, v1.Volume{
			Name: "bash-script-volume",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "jupyter-start-configmap",
					},
					//nolint:mnd // 0755 is the default mode
					DefaultMode: ptr.To(int32(0755)),
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "bash-script-volume",
			MountPath: "/usr/bin/start.sh",
			ReadOnly:  false,
			SubPath:   "start.sh",
		})

		commandSchema := "/usr/bin/start.sh jupyter lab --allow-root --notebook-dir=/home/%s"
		command = fmt.Sprintf(commandSchema, token.Username)
	}

	// 2. Env Vars
	envs := vcjob.GenerateEnvs(c, token, vcReq.Envs)
	envs = append(
		envs,
		v1.EnvVar{Name: "GRANT_SUDO", Value: "1"},
		v1.EnvVar{Name: "CHOWN_HOME", Value: "1"},
	)

	// 3. Node Affinity and Tolerations
	baseAffinity := vcjob.GenerateNodeAffinity(vcReq.Selectors, nil)
	affinity := vcjob.GenerateArchitectureNodeAffinity(vcReq.Image, baseAffinity)

	tolerations := []v1.Toleration{} // AIJob doesn't use account tolerations

	imagePullSecrets := []v1.LocalObjectReference{}
	if config.GetConfig().Secrets.ImagePullSecretName != "" {
		imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{
			Name: config.GetConfig().Secrets.ImagePullSecretName,
		})
	}

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity:         affinity,
		Tolerations:      tolerations,
		Volumes:          volumes,
		ImagePullSecrets: imagePullSecrets,
		Containers: []v1.Container{
			{
				Name:    "jupyter-notebook",
				Image:   req.Image,
				Command: []string{"bash", "-c", command},
				Resources: v1.ResourceRequirements{
					Limits:   vcReq.Resource,
					Requests: vcReq.Resource,
				},
				WorkingDir: fmt.Sprintf("/home/%s", token.Username),

				Env: envs,
				Ports: []v1.ContainerPort{
					{ContainerPort: vcjob.JupyterPort, Name: "notebook-port", Protocol: v1.ProtocolTCP},
				},
				SecurityContext: &v1.SecurityContext{
					RunAsUser:  ptr.To(int64(0)),
					RunAsGroup: ptr.To(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
		EnableServiceLinks: ptr.To(false),
	}

	taskModel.PodTemplate = datatypes.NewJSONType(podSpec)
	taskModel.Owner = token.Username
	err = mgr.taskService.Create(taskModel)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	klog.Infof("create task success, taskID: %d", taskModel.ID)
	resp := CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.Success(c, resp)
}

// CreateCustom godoc
//
//	@Summary		CreateCustom a new AI job
//	@Description	CreateCustom a new AI job by client-go
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			job	body		any						true	"CreateCustom AI Job Request"
//	@Success		200	{object}	resputil.Response[any]	"CreateCustom AI Job Response"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs/training [post]
func (mgr *AIJobMgr) CreateCustom(c *gin.Context) {
	var vcReq CreateAIJobReq
	if err := c.ShouldBindJSON(&vcReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req CreateTaskReq

	token := interutil.GetToken(c)
	req.TaskName = vcReq.Name
	req.Namespace = config.GetConfig().Namespaces.Job
	req.UserName = token.AccountName
	req.SLO = vcReq.SLO
	req.TaskType = model.EmiasTrainingTask
	req.Image = vcReq.Image.ImageLink
	req.ResourceRequest = vcReq.Resource
	if vcReq.Command != nil {
		req.Command = *vcReq.Command
	}
	req.WorkingDir = vcReq.WorkingDir

	taskModel := model.FormatTaskAttrToModel(&req.TaskAttr)
	podSpec, err := vcjob.GenerateCustomPodSpec(c, token, &vcReq.CreateCustomReq)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("generate pod spec failed, err %v", err), resputil.NotSpecified)
		return
	}

	// Apply architecture-specific affinity
	baseAffinity := vcjob.GenerateNodeAffinity(vcReq.Selectors, nil)
	podSpec.Affinity = vcjob.GenerateArchitectureNodeAffinity(vcReq.Image, baseAffinity)
	// Tolerations remain unchanged (from GenerateCustomPodSpec)

	taskModel.PodTemplate = datatypes.NewJSONType(podSpec)
	taskModel.Owner = token.Username
	err = mgr.taskService.Create(taskModel)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	klog.Infof("create task success, taskID: %d", taskModel.ID)
	resp := CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.Success(c, resp)
}
