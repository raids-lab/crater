package vcjob

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

const (
	ThreeDaySeconds int32 = 259200
	IngressLabelKey       = "ingress.crater.raids.io" // Annotation Ingress Key

)

type (
	CreateJupyterReq struct {
		CreateJobCommon `json:",inline"`
		Name            string          `json:"name" binding:"required"`
		Resource        v1.ResourceList `json:"resource"`
		Image           string          `json:"image" binding:"required"`
	}
)

func (mgr *VolcanojobMgr) CreateJupyterJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.QueueID, req.Resource)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.NotSpecified)
		return
	}
	// Ingress base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("jupyter-%s", baseURL)

	// useTensorBoard
	useTensorboard := fmt.Sprintf("%t", req.UseTensorBoard)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s"
	command := fmt.Sprintf(commandSchema, token.Username, baseURL)

	// 1. Volume Mounts
	volumes, volumeMounts, err := GenerateNewVolumeMounts(c, token.UserID, req.VolumeMounts)
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
					DefaultMode: lo.ToPtr(int32(0755)),
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "bash-script-volume",
			MountPath: "/usr/bin/start.sh",
			ReadOnly:  false,
			SubPath:   "start.sh",
		})

		commandSchema := "/usr/bin/start.sh jupyter lab --allow-root --notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s"
		command = fmt.Sprintf(commandSchema, token.Username, baseURL)
	}

	// 2. Env Vars
	//nolint:mnd // 4 is the number of default envs
	envs := make([]v1.EnvVar, len(req.Envs)+4)
	envs[0] = v1.EnvVar{Name: "GRANT_SUDO", Value: "1"}
	envs[1] = v1.EnvVar{Name: "CHOWN_HOME", Value: "1"}
	envs[2] = v1.EnvVar{Name: "NB_UID", Value: "1001"}
	envs[3] = v1.EnvVar{Name: "NB_USER", Value: token.Username}
	for i, env := range req.Envs {
		envs[i+4] = env
	}

	// 3. TODO: Node Affinity for ARM64 Nodes
	affinity := GenerateNodeAffinity(req.Selectors)

	// 4. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "jupyter",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}

	// 初始化 annotations 并添加 notebookIngress Annotation
	annotations := map[string]string{
		AnnotationKeyTaskName:       req.Name,
		AnnotationKeyUseTensorBoard: useTensorboard,
	}

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:    "jupyter-notebook",
				Image:   req.Image,
				Command: []string{"bash", "-c", command},
				Resources: v1.ResourceRequirements{
					Limits:   req.Resource,
					Requests: req.Resource,
				},
				WorkingDir: fmt.Sprintf("/home/%s", token.Username),

				Env: envs,
				Ports: []v1.ContainerPort{
					{ContainerPort: JupyterPort, Name: "notebook-port", Protocol: v1.ProtocolTCP},
				},
				SecurityContext: &v1.SecurityContext{
					AllowPrivilegeEscalation: lo.ToPtr(true),
					RunAsUser:                lo.ToPtr(int64(0)),
					RunAsGroup:               lo.ToPtr(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
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
			// 3 days
			TTLSecondsAfterFinished: lo.ToPtr(ThreeDaySeconds),
			MinAvailable:            1,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   token.QueueName,
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

// GetJobToken godoc
// @Summary Get the ingress base url and jupyter token of the job
// @Description Get the token of the job by logs
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobName path string true "Job Name"
// @Success 200 {object} resputil.Response[JobTokenResp] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/token [get]
func (mgr *VolcanojobMgr) GetJobToken(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	j := query.Job
	job, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Where(j.UserID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.JobType != model.JobTypeJupyter {
		resputil.Error(c, "Job type is not Jupyter", resputil.NotSpecified)
		return
	}

	vcjob := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if vcjob.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job do not belong to the user", resputil.NotSpecified)
		return
	}

	// Check if the job is running
	status := vcjob.Status.State.Phase
	if status != batch.Running {
		resputil.Error(c, "Job not running", resputil.NotSpecified)
		return
	}

	baseURL := vcjob.Labels[LabelKeyBaseURL]

	// Get the logs of the job pod
	var podName string
	for i := range vcjob.Spec.Tasks {
		task := &vcjob.Spec.Tasks[i]
		podName = fmt.Sprintf("%s-%s-0", vcjob.Name, task.Name)
		break
	}

	// Check if jupyter token has been cached in the job annotations
	jupyterToken, ok := vcjob.Annotations[AnnotationKeyJupyter]
	if ok {
		resputil.Success(c, JobTokenResp{
			BaseURL:   baseURL,
			Token:     jupyterToken,
			PodName:   podName,
			Namespace: namespace,
		})
		return
	}

	buf, err := mgr.getPodLog(c, namespace, podName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(buf.String())
	if len(matches) >= 2 {
		jupyterToken = matches[1]
	} else {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	if jupyterToken == "" {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	// Cache the jupyter token in the job annotations
	vcjob.Annotations[AnnotationKeyJupyter] = jupyterToken
	if err := mgr.client.Update(c, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 创建 jupyter-notebook 的 svc 和 ingress
	ctx := c.Request.Context()
	notebookIngress := crclient.PodIngress{
		Name:   "notebook",
		Port:   8888,                                // Notebook 在容器内的默认端口
		Prefix: fmt.Sprintf("/ingress/%s", baseURL), // 设置唯一的 URL 前缀
	}

	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if err := crclient.CreateCustomForwardingRule(ctx, mgr.client, &pod, notebookIngress); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, JobTokenResp{
		BaseURL:   baseURL,
		Token:     jupyterToken,
		PodName:   podName,
		Namespace: namespace,
	})
}
