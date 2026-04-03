package vcjob

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

//nolint:gochecknoinits // Register agent job submitter factory alongside the handler.
func init() {
	handler.RegisterJobMutationSubmitterFactory(NewAgentJobSubmitter)
}

// agentJobSubmitter implements handler.JobMutationSubmitter for the agent
// service. It reuses the same vcjob helper functions but returns errors
// directly instead of writing HTTP responses via resputil.
type agentJobSubmitter struct {
	mgr *VolcanojobMgr
}

// NewAgentJobSubmitter creates a JobMutationSubmitter backed by VolcanojobMgr
// internals.
func NewAgentJobSubmitter(conf *handler.RegisterConfig) handler.JobMutationSubmitter {
	return &agentJobSubmitter{
		mgr: &VolcanojobMgr{
			name:           "vcjobs",
			client:         conf.Client,
			config:         conf.KubeConfig,
			kubeClient:     conf.KubeClient,
			imagePacker:    conf.ImagePacker,
			imageRegistry:  conf.ImageRegistry,
			serviceManager: conf.ServiceManager,
		},
	}
}

func (s *agentJobSubmitter) SubmitJupyterJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateJupyterReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, fmt.Errorf("invalid jupyter request: %w", err)
	}

	if err := aitaskctl.CheckInteractiveLimitBeforeCreate(ctx, token.UserID, token.AccountID); err != nil {
		return nil, fmt.Errorf("interactive job limit reached: %v", err)
	}
	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %v", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %v", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("jpt", token.Username)
	baseURL := jobName[4:]

	jupyterCommand := fmt.Sprintf(
		"jupyter lab --ip=0.0.0.0 --no-browser --allow-root "+
			"--notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s/ "+
			"--ResourceUseDisplay.track_cpu_percent=True",
		token.Username, baseURL)

	commandArgs := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf("/usr/local/bin/unified-start.sh %s", jupyterCommand),
	}

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeJupyter, token, baseURL, req.Name, req.Template, req.AlertEnabled,
	)

	podSpec, err := generateInteractivePodSpec(
		ctx, token, &req.CreateJobCommon, req.Resource, req.Image,
		commandArgs, JupyterPort, string(CraterJobTypeJupyter), req.CpuPinningEnabled,
	)
	if err != nil {
		return nil, err
	}

	queueName := token.AccountName
	if token.AccountID != model.DefaultAccountID {
		queueName = vcqueue.GetUserQueueName(token.AccountID, token.UserID)
	}

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
			Plugins:                 volcanoPlugins,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   queueName,
			Policies: []batch.LifecyclePolicy{
				{Action: bus.RestartJobAction, Event: bus.PodEvictedEvent},
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

	if err = s.mgr.client.Create(ctx, &job); err != nil {
		return nil, err
	}

	port := &v1.ServicePort{
		Name:       "notebook",
		Port:       JupyterPort,
		TargetPort: intstr.FromInt(JupyterPort),
		Protocol:   v1.ProtocolTCP,
	}

	ingressPath, err := s.mgr.serviceManager.CreateIngressWithPrefix(
		ctx,
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(&job, batch.SchemeGroupVersion.WithKind("Job")),
		},
		labels, port, config.GetConfig().Host, baseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingress: %v", err)
	}
	log.Printf("Ingress created at path: %s", ingressPath)

	if err := s.mgr.CreateForwardIngresses(ctx, &job, req.Forwards, labels, token.Username); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *agentJobSubmitter) SubmitTrainingJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateCustomReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, fmt.Errorf("invalid training request: %w", err)
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %v", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %v", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("sg", token.Username)
	baseURL := jobName[3:]

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeCustom, token, baseURL, req.Name, req.Template, req.AlertEnabled,
	)

	podSpec, err := GenerateCustomPodSpec(ctx, token, &req)
	if err != nil {
		return nil, err
	}

	queueName := token.AccountName
	if token.AccountID != model.DefaultAccountID {
		queueName = vcqueue.GetUserQueueName(token.AccountID, token.UserID)
	}

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
			Queue:                   queueName,
			Plugins:                 volcanoPlugins,
			Policies: []batch.LifecyclePolicy{
				{Action: bus.RestartJobAction, Event: bus.PodEvictedEvent},
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
						{Action: bus.CompleteJobAction, Event: bus.TaskCompletedEvent},
					},
				},
			},
		},
	}

	if err = s.mgr.client.Create(ctx, &job); err != nil {
		return nil, err
	}

	if err := s.mgr.CreateForwardIngresses(ctx, &job, req.Forwards, labels, token.Username); err != nil {
		return nil, err
	}
	return &job, nil
}
