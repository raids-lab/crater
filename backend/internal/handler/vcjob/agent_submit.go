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
// internals. We deliberately wire the full service set (configService,
// queueQuotaSvc, prequeueWatcher, billingService) so that scheduling helpers
// such as resolveJobScheduleMetadata work the same way they do on the normal
// /v1/vcjobs/* HTTP path; agent-submitted jobs should not silently bypass
// prequeue / backfill scheduling metadata.
func NewAgentJobSubmitter(conf *handler.RegisterConfig) handler.JobMutationSubmitter {
	return &agentJobSubmitter{
		mgr: &VolcanojobMgr{
			name:            "vcjobs",
			client:          conf.Client,
			config:          conf.KubeConfig,
			kubeClient:      conf.KubeClient,
			imagePacker:     conf.ImagePacker,
			imageRegistry:   conf.ImageRegistry,
			serviceManager:  conf.ServiceManager,
			configService:   conf.ConfigService,
			queueQuotaSvc:   conf.PrequeueService,
			prequeueWatcher: conf.PrequeueWatcher,
			billingService:  conf.BillingService,
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

	// Resolve scheduling metadata the same way the /v1/vcjobs/jupyter HTTP
	// handler does, so agent-submitted jobs carry the same prequeue /
	// backfill / tolerance annotations as user-submitted ones.
	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schedule metadata: %w", err)
	}

	if err := aitaskctl.CheckInteractiveLimitBeforeCreate(ctx, token.UserID, token.AccountID); err != nil {
		return nil, fmt.Errorf("interactive job limit reached: %w", err)
	}
	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %w", err)
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
		CraterJobTypeJupyter, token, baseURL, &req.CreateJobCommon, scheduleMetadata,
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
		return nil, fmt.Errorf("failed to create ingress: %w", err)
	}
	log.Printf("Ingress created at path: %s", ingressPath)

	if err := s.mgr.CreateForwardIngresses(ctx, &job, req.Forwards, labels, token.Username); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *agentJobSubmitter) SubmitWebIDEJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateJupyterReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, fmt.Errorf("invalid webide request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schedule metadata: %w", err)
	}

	if err := aitaskctl.CheckInteractiveLimitBeforeCreate(ctx, token.UserID, token.AccountID); err != nil {
		return nil, fmt.Errorf("interactive job limit reached: %w", err)
	}
	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %w", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("vsc", token.Username)
	baseURL := jobName[4:]
	webIDECommand := fmt.Sprintf("code-server --bind-addr 0.0.0.0:%d", JupyterPort)
	commandArgs := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf("/usr/local/bin/unified-start.sh %s", webIDECommand),
	}

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeWebIDE, token, baseURL, &req.CreateJobCommon, scheduleMetadata,
	)

	podSpec, err := generateInteractivePodSpec(
		ctx, token, &req.CreateJobCommon, req.Resource, req.Image,
		commandArgs, JupyterPort, string(CraterJobTypeWebIDE), req.CpuPinningEnabled,
	)
	if err != nil {
		return nil, err
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
			Queue:                   vcqueue.ResolveJobQueueName(token),
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

	if err = s.mgr.submitJob(ctx, token, &job); err != nil {
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

	// Match /v1/vcjobs/custom: resolve schedule metadata before quota checks.
	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schedule metadata: %w", err)
	}

	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %w", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("sg", token.Username)
	baseURL := jobName[3:]

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeCustom, token, baseURL, &req.CreateJobCommon, scheduleMetadata,
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

func (s *agentJobSubmitter) SubmitTensorflowJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateTensorflowReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, fmt.Errorf("invalid tensorflow request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(false)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schedule metadata: %w", err)
	}

	jobResources := v1.ResourceList{}
	for idx := range req.Tasks {
		jobResources = aitaskctl.AddResourceList(jobResources, req.Tasks[idx].Resource)
	}

	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %w", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("tf", token.Username)
	baseURL := jobName[3:]

	volumes, volumeMounts, err := GenerateVolumeMounts(ctx, req.VolumeMounts, token)
	if err != nil {
		return nil, err
	}
	baseAffinity := GenerateNodeAffinity(req.Selectors, jobResources)
	baseTolerations := GenerateTaintTolerationsForAccount(token)
	envs := GenerateEnvs(ctx, token, req.Envs)

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeTensorflow,
		token,
		baseURL,
		&req.CreateJobCommon,
		scheduleMetadata,
	)

	tasks := make([]batch.TaskSpec, len(req.Tasks))
	minAvailable := int32(0)
	for idx := range req.Tasks {
		task := &req.Tasks[idx]
		taskAffinity := GenerateArchitectureNodeAffinity(task.Image, baseAffinity)
		ports := make([]v1.ContainerPort, len(task.Ports))
		for portIdx, port := range task.Ports {
			ports[portIdx] = v1.ContainerPort{
				ContainerPort: port.Port,
				Name:          port.Name,
				Protocol:      v1.ProtocolTCP,
			}
		}

		podSpec := generatePodSpecForParallelJob(
			task,
			taskAffinity,
			baseTolerations,
			volumes,
			volumeMounts,
			envs,
			ports,
			req.CpuPinningEnabled,
		)

		taskSpec := batch.TaskSpec{
			Name:     task.Name,
			Replicas: task.Replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
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
		tasks[idx] = taskSpec
	}

	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(SevenDaySeconds),
			MinAvailable:            minAvailable,
			SchedulerName:           VolcanoSchedulerName,
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
			Queue: vcqueue.ResolveJobQueueName(token),
			Tasks: tasks,
		},
	}

	if err = s.mgr.submitJob(ctx, token, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *agentJobSubmitter) SubmitPytorchJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateTensorflowReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, fmt.Errorf("invalid pytorch request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(false)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schedule metadata: %w", err)
	}

	jobResources := v1.ResourceList{}
	for idx := range req.Tasks {
		jobResources = aitaskctl.AddResourceList(jobResources, req.Tasks[idx].Resource)
	}

	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return nil, fmt.Errorf("resource quota exceeded: %v", exceededResources)
	}

	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return nil, fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user queue exists: %w", err)
	}

	if req.AlertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return nil, fmt.Errorf("email not verified")
	}

	jobName := utils.GenerateJobName("pyt", token.Username)
	baseURL := jobName[4:]

	volumes, volumeMounts, err := GenerateVolumeMounts(ctx, req.VolumeMounts, token)
	if err != nil {
		return nil, err
	}
	baseAffinity := GenerateNodeAffinity(req.Selectors, jobResources)
	baseTolerations := GenerateTaintTolerationsForAccount(token)
	envs := GenerateEnvs(ctx, token, req.Envs)

	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypePytorch,
		token,
		baseURL,
		&req.CreateJobCommon,
		scheduleMetadata,
	)

	tasks := make([]batch.TaskSpec, len(req.Tasks))
	minAvailable := int32(0)
	for idx := range req.Tasks {
		task := &req.Tasks[idx]
		taskAffinity := GenerateArchitectureNodeAffinity(task.Image, baseAffinity)
		ports := make([]v1.ContainerPort, len(task.Ports))
		for portIdx, port := range task.Ports {
			ports[portIdx] = v1.ContainerPort{
				ContainerPort: port.Port,
				Name:          port.Name,
				Protocol:      v1.ProtocolTCP,
			}
		}

		podSpec := generatePodSpecForParallelJob(
			task,
			taskAffinity,
			baseTolerations,
			volumes,
			volumeMounts,
			envs,
			ports,
			req.CpuPinningEnabled,
		)

		taskSpec := batch.TaskSpec{
			Name:     task.Name,
			Replicas: task.Replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
		}

		switch task.Name {
		case "master":
			taskSpec.Policies = []batch.LifecyclePolicy{
				{
					Action: bus.CompleteJobAction,
					Event:  bus.TaskCompletedEvent,
				},
				{
					Action: bus.TerminateJobAction,
					Event:  bus.PodFailedEvent,
				},
			}
		case "worker":
			taskSpec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		}

		minAvailable += task.Replicas
		tasks[idx] = taskSpec
	}

	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(SevenDaySeconds),
			MinAvailable:            minAvailable,
			SchedulerName:           VolcanoSchedulerName,
			Plugins: map[string][]string{
				"pytorch": {"--master=master", "--worker=worker", "--port=23456"},
			},
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
			},
			Queue: vcqueue.ResolveJobQueueName(token),
			Tasks: tasks,
		},
	}

	if err = s.mgr.submitJob(ctx, token, &job); err != nil {
		return nil, err
	}
	return &job, nil
}
