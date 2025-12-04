package vcjob

import (
	"fmt"
	"log"
	"regexp"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/utils"
)

// CreateWebIDEJob godoc
//
//	@Summary		Create a WebIDE job
//	@Description	Create a WebIDE job
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			CreateJupyterReq	body		CreateJupyterReq		true	"Create WebIDE Job Request"
//	@Success		200					{object}	resputil.Response[any]	"Success"
//	@Failure		400					{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500					{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/webide [post]
func (mgr *VolcanojobMgr) CreateWebIDEJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := aitaskctl.CheckInteractiveLimitBeforeCreate(c, token.UserID, token.AccountID); err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.AccountID, req.Resource)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.ServiceError)
		return
	}

	// If email alert is enabled, check if email is verified
	if req.AlertEnabled && !utils.CheckUserEmail(c, token.UserID) {
		resputil.Error(c, "Email not verified", resputil.UserEmailNotVerified)
		return
	}

	// Generate job name with type prefix (RFC 1035 compliant)
	jobName := utils.GenerateJobName("vsc", token.Username)
	// baseURL for ingress paths (without type prefix)
	baseURL := jobName[4:] // Remove "vsc-" prefix
	randomSuffix := fmt.Sprintf("%s-%d", jobName[len(jobName)-5:], token.UserID)

	// Unified jupyter start command
	webideCommand := fmt.Sprintf("code-server --bind-addr 0.0.0.0:%d", JupyterPort)

	commandArgs := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf("/usr/local/bin/unified-start.sh %s", webideCommand),
	}

	// 4. Labels and Annotations
	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeWebIDE,
		token,
		baseURL,
		req.Name,
		req.Template,
		req.AlertEnabled,
	)

	// 5. Create the pod spec
	podSpec, err := generateInteractivePodSpec(
		c,
		token,
		&req.CreateJobCommon,
		req.Resource,
		req.Image,
		commandArgs,
		JupyterPort,
		string(CraterJobTypeWebIDE),
	)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 6. Create volcano job
	//nolint:dupl // TODO: refactor to reduce duplicate code
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			// 3 days
			TTLSecondsAfterFinished: ptr.To(ThreeDaySeconds),
			MinAvailable:            1,
			MaxRetry:                1,
			Plugins:                 volcanoPlugins,
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
							Annotations: podAnnotations,
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

	// create jupyter notebook ingress
	port := &v1.ServicePort{
		Name:       "webide",
		Port:       JupyterPort,
		TargetPort: intstr.FromInt(JupyterPort),
		Protocol:   v1.ProtocolTCP,
	}

	ingressPath, err := mgr.serviceManager.CreateNamedIngress(
		c,
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(&job, batch.SchemeGroupVersion.WithKind("Job")),
		},
		labels,
		port,
		config.GetConfig().Host,
		token.Username,
		randomSuffix,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create ingress: %v", err), resputil.NotSpecified)
		return
	}

	log.Printf("Ingress created at path: %s", ingressPath)

	// create forward ing rules in template
	if err := mgr.CreateForwardIngresses(c, &job, req.Forwards, labels, token.Username); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

// GetWebIDESecret godoc
//
//	@Summary		Get the password of the WebIDE job
//	@Description	Get the password of the WebIDE job by reading config file in the pod
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string							true	"Job Name"
//	@Success		200		{object}	resputil.Response[JupyterTokenResp]	"Success"
//	@Failure		400		{object}	resputil.Response[any]			"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]			"Other errors"
//	@Router			/v1/vcjobs/{name}/secret [get]
func (mgr *VolcanojobMgr) GetWebIDESecret(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.JobType != model.JobTypeWebIDE {
		resputil.Error(c, "Job type is not WebIDE", resputil.NotSpecified)
		return
	}

	vcjob := &batch.Job{}
	namespace := config.GetConfig().Namespaces.Job
	if err := mgr.client.Get(c, client.ObjectKey{
		Namespace: namespace,
		Name:      req.JobName,
	}, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Check if the job is running
	status := vcjob.Status.State.Phase
	if status != batch.Running {
		resputil.Error(c, "Job is not running", resputil.NotSpecified)
		return
	}

	baseURL := vcjob.Labels[crclient.LabelKeyBaseURL]
	randomPrefix := fmt.Sprintf("%s-%d", req.JobName[len(req.JobName)-5:], token.UserID)

	podName, _ := getPodNameAndLabelFromJob(vcjob)

	// Construct the full URL directly
	host := config.GetConfig().Host
	fullURL := fmt.Sprintf("https://%s.%s", randomPrefix, host)

	// Check if password has been cached in the job annotations
	password, ok := vcjob.Annotations[AnnotationKeyWebIDE]
	if ok {
		resputil.Success(c, JupyterTokenResp{
			BaseURL:   baseURL,
			Token:     password,
			FullURL:   fullURL,
			PodName:   podName,
			Namespace: namespace,
		})
		return
	}

	// Fetch the password from the pod
	// Command: cat /home/<username>/.config/code-server/config.yaml
	cmd := []string{"cat", fmt.Sprintf("/home/%s/.config/code-server/config.yaml", token.Username)}
	output, err := mgr.execCommandInPod(c, namespace, podName, string(CraterJobTypeWebIDE), cmd)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to exec command in pod: %v", err), resputil.NotSpecified)
		return
	}

	// Parse the password
	re := regexp.MustCompile(`password: (.+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		password = matches[1]
	} else {
		resputil.Error(c, "failed to find password in config file", resputil.NotSpecified)
		return
	}

	// Cache the password in the job annotations
	vcjob.Annotations[AnnotationKeyWebIDE] = password
	if err := mgr.client.Update(c, vcjob); err != nil {
		// Just log the error, do not fail the request
		log.Printf("failed to update job annotation: %v", err)
	}

	resputil.Success(c, JupyterTokenResp{
		BaseURL:   baseURL,
		Token:     password,
		FullURL:   fullURL,
		PodName:   podName,
		Namespace: namespace,
	})
}
