package vcjob

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
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

func agentSubmitErrorf(format string, args ...any) error {
	return bizerr.Internal.ServiceError.New(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...))
}

func buildAgentParallelTaskSpecs(
	req *CreateTensorflowReq,
	baseAffinity *v1.Affinity,
	baseTolerations []v1.Toleration,
	volumes []v1.Volume,
	volumeMounts []v1.VolumeMount,
	envs []v1.EnvVar,
	labels map[string]string,
	podAnnotations map[string]string,
	jobType CraterJobType,
) (tasks []batch.TaskSpec, minAvailable int32) {
	tasks = make([]batch.TaskSpec, len(req.Tasks))
	minAvailable = 0
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

		applyAgentParallelTaskPolicy(&taskSpec, jobType)
		minAvailable += task.Replicas
		tasks[idx] = taskSpec
	}
	return tasks, minAvailable
}

func applyAgentParallelTaskPolicy(taskSpec *batch.TaskSpec, jobType CraterJobType) {
	switch jobType {
	case CraterJobTypeTensorflow:
		if taskSpec.Name == volcanoTaskWorker {
			taskSpec.Policies = []batch.LifecyclePolicy{
				{
					Action: bus.CompleteJobAction,
					Event:  bus.TaskCompletedEvent,
				},
			}
		}
	case CraterJobTypePytorch:
		switch taskSpec.Name {
		case volcanoTaskMaster:
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
		case volcanoTaskWorker:
			taskSpec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		}
	}
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

func (s *agentJobSubmitter) preCheckJobCreate(
	ctx context.Context,
	token util.JWTMessage,
	scheduleType model.ScheduleType,
	requireInteractiveLimit bool,
) error {
	if s.mgr.billingService != nil {
		if err := s.mgr.billingService.OnJobCreateCheck(ctx, token.UserID, token.AccountID, &scheduleType); err != nil {
			return err
		}
	}
	if requireInteractiveLimit {
		if err := aitaskctl.CheckInteractiveLimitBeforeCreate(ctx, token.UserID, token.AccountID); err != nil {
			return agentSubmitErrorf("interactive job limit reached: %v", err)
		}
	}
	exceededResources, err := aitaskctl.CheckResourcesBeforeCreateJob(ctx, token.UserID, token.AccountID)
	if err != nil {
		return agentSubmitErrorf("failed to check resources: %w", err)
	}
	if len(exceededResources) > 0 {
		return agentSubmitErrorf("resource quota exceeded: %v", exceededResources)
	}
	return nil
}

func (s *agentJobSubmitter) prepareJobCreate(
	ctx context.Context,
	token util.JWTMessage,
	scheduleType model.ScheduleType,
	requireInteractiveLimit bool,
	alertEnabled bool,
) error {
	if err := s.preCheckJobCreate(ctx, token, scheduleType, requireInteractiveLimit); err != nil {
		return err
	}
	if err := vcqueue.EnsureAccountQueueExists(ctx, s.mgr.client, token, token.AccountID); err != nil {
		return agentSubmitErrorf("failed to ensure account queue exists: %v", err)
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, s.mgr.client, token, token.AccountID, token.UserID); err != nil {
		return agentSubmitErrorf("failed to ensure user queue exists: %v", err)
	}
	if alertEnabled && !utils.CheckUserEmail(ctx, token.UserID) {
		return agentSubmitErrorf("email not verified")
	}
	return nil
}

func (s *agentJobSubmitter) DeleteJob(ctx context.Context, token util.JWTMessage, jobName string) (any, error) {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return nil, agentSubmitErrorf("job_name is required")
	}

	jobRecord, err := getJob(ctx, jobName, &token)
	if err != nil {
		return nil, agentSubmitErrorf("job not found or access denied: %w", err)
	}
	plan, err := s.mgr.buildDeleteJobPlan(ctx, jobRecord)
	if err != nil {
		return nil, err
	}
	if err := s.mgr.applyDeleteJobPlan(ctx, jobRecord, plan); err != nil {
		return nil, err
	}
	if err := s.mgr.deleteClusterJob(ctx, plan); err != nil {
		return nil, err
	}
	s.mgr.notifyDeletedPrequeue(plan.shouldDeleteRecord)

	return map[string]any{
		"jobName":           jobRecord.JobName,
		"status":            "deleted",
		"deletedRecord":     plan.shouldDeleteRecord,
		"deletedClusterJob": plan.shouldDeleteJob,
	}, nil
}

func (s *agentJobSubmitter) StopJob(ctx context.Context, token util.JWTMessage, jobName string) (any, error) {
	result, err := s.DeleteJob(ctx, token, jobName)
	if err != nil {
		return nil, err
	}
	if resultMap, ok := result.(map[string]any); ok {
		resultMap["status"] = "stopped"
	}
	return result, nil
}

//nolint:gocyclo // Resubmit sanitizes cloned Volcano jobs and applies optional resource overrides.
func (s *agentJobSubmitter) ResubmitJob(ctx context.Context, token util.JWTMessage, rawReq json.RawMessage) (any, error) {
	var req struct {
		JobName  string  `json:"job_name"`
		Name     *string `json:"name"`
		CPU      *string `json:"cpu"`
		Memory   *string `json:"memory"`
		GPUCount *int    `json:"gpu_count"`
		GPUModel *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, agentSubmitErrorf("invalid resubmit request: %w", err)
	}
	req.JobName = strings.TrimSpace(req.JobName)
	normalizeAgentSubmitOptionalString(&req.Name)
	normalizeAgentSubmitOptionalString(&req.CPU)
	normalizeAgentSubmitOptionalString(&req.Memory)
	normalizeAgentSubmitOptionalString(&req.GPUModel)
	if req.JobName == "" {
		return nil, agentSubmitErrorf("job_name is required")
	}
	if token.Username == "" {
		return nil, agentSubmitErrorf("user identity is unavailable for resubmit")
	}

	jobRecord, err := getJob(ctx, req.JobName, &token)
	if err != nil {
		return nil, agentSubmitErrorf("job not found or access denied: %w", err)
	}
	sourceJob := jobRecord.Attributes.Data()
	if sourceJob == nil {
		return nil, agentSubmitErrorf("job spec is unavailable for resubmit")
	}

	clonedJob := sourceJob.DeepCopy()
	appliedOverrides, err := applyAgentSubmitResubmitOverrides(clonedJob, req.CPU, req.Memory, req.GPUCount, req.GPUModel)
	if err != nil {
		return nil, err
	}

	prefix := agentSubmitJobNamePrefix(jobRecord.JobName)
	newJobName := utils.GenerateJobName(prefix, token.Username)
	baseURL := agentSubmitBaseURLFromJobName(newJobName)

	clonedJob.ObjectMeta = metav1.ObjectMeta{
		Name:        newJobName,
		Namespace:   config.GetConfig().Namespaces.Job,
		Labels:      copyAgentSubmitStringMap(clonedJob.Labels),
		Annotations: copyAgentSubmitStringMap(clonedJob.Annotations),
	}
	clonedJob.Status = batch.JobStatus{}
	clonedJob.ResourceVersion = ""
	clonedJob.UID = ""
	clonedJob.CreationTimestamp = metav1.Time{}
	clonedJob.ManagedFields = nil
	clonedJob.OwnerReferences = nil
	clonedJob.Finalizers = nil
	clonedJob.DeletionTimestamp = nil

	if clonedJob.Labels == nil {
		clonedJob.Labels = map[string]string{}
	}
	clonedJob.Labels[crclient.LabelKeyBaseURL] = baseURL
	if clonedJob.Annotations == nil {
		clonedJob.Annotations = map[string]string{}
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		clonedJob.Annotations[AnnotationKeyTaskName] = strings.TrimSpace(*req.Name)
		appliedOverrides["name"] = strings.TrimSpace(*req.Name)
	} else if clonedJob.Annotations[AnnotationKeyTaskName] == "" {
		clonedJob.Annotations[AnnotationKeyTaskName] = jobRecord.Name
	}

	for idx := range clonedJob.Spec.Tasks {
		task := &clonedJob.Spec.Tasks[idx]
		task.Template.ResourceVersion = ""
		task.Template.UID = ""
		task.Template.CreationTimestamp = metav1.Time{}
		task.Template.ManagedFields = nil
		if task.Template.Labels == nil {
			task.Template.Labels = map[string]string{}
		}
		task.Template.Labels[crclient.LabelKeyBaseURL] = baseURL
		task.Template.Labels[crclient.LabelKeyTaskType] = clonedJob.Labels[crclient.LabelKeyTaskType]
		task.Template.Labels[crclient.LabelKeyTaskUser] = clonedJob.Labels[crclient.LabelKeyTaskUser]
		if accountName := clonedJob.Labels[crclient.LalbeKeyTaskAccount]; accountName != "" {
			task.Template.Labels[crclient.LalbeKeyTaskAccount] = accountName
		}
	}

	if err := s.mgr.client.Create(ctx, clonedJob); err != nil {
		return nil, agentSubmitErrorf("failed to create resubmitted job: %w", err)
	}
	if err := s.ensureResubmittedJobAccess(ctx, clonedJob); err != nil {
		return map[string]any{
			"sourceJobName": jobRecord.JobName,
			"jobName":       newJobName,
			"status":        "created",
			"warning":       err.Error(),
		}, nil
	}

	return map[string]any{
		"sourceJobName": jobRecord.JobName,
		"jobName":       newJobName,
		"displayName":   clonedJob.Annotations[AnnotationKeyTaskName],
		"status":        "created",
		"overrides":     appliedOverrides,
	}, nil
}

func (s *agentJobSubmitter) SubmitJupyterJob(
	ctx context.Context,
	token util.JWTMessage,
	rawReq json.RawMessage,
) (any, error) {
	var req CreateJupyterReq
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return nil, agentSubmitErrorf("invalid jupyter request: %w", err)
	}

	// Resolve scheduling metadata the same way the /v1/vcjobs/jupyter HTTP
	// handler does, so agent-submitted jobs carry the same prequeue /
	// backfill / tolerance annotations as user-submitted ones.
	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, agentSubmitErrorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, agentSubmitErrorf("failed to resolve schedule metadata: %w", err)
	}

	if err := s.prepareJobCreate(ctx, token, scheduleType, true, req.AlertEnabled); err != nil {
		return nil, err
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
		commandArgs, string(CraterJobTypeJupyter), req.CpuPinningEnabled,
	)
	if err != nil {
		return nil, err
	}

	queueName := token.AccountName
	if token.AccountID != model.DefaultAccountID {
		queueName = vcqueue.GetUserQueueName(token.AccountID, token.UserID)
	}

	job := buildInteractiveVolcanoJob(jobName, labels, jobAnnotations, podAnnotations, &podSpec, queueName)

	if err := s.mgr.client.Create(ctx, &job); err != nil {
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
		return nil, agentSubmitErrorf("failed to create ingress: %v", err)
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
		return nil, agentSubmitErrorf("invalid webide request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, agentSubmitErrorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, agentSubmitErrorf("failed to resolve schedule metadata: %w", err)
	}

	if err := s.prepareJobCreate(ctx, token, scheduleType, true, req.AlertEnabled); err != nil {
		return nil, err
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
		commandArgs, string(CraterJobTypeWebIDE), req.CpuPinningEnabled,
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
			TTLSecondsAfterFinished: ptr.To(utils.ThreeDaySeconds),
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

	if err := s.mgr.submitJob(ctx, token, &job); err != nil {
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
		return nil, agentSubmitErrorf("invalid training request: %w", err)
	}

	// Match /v1/vcjobs/custom: resolve schedule metadata before quota checks.
	scheduleType, err := req.validateScheduleOptions(true)
	if err != nil {
		return nil, agentSubmitErrorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, agentSubmitErrorf("failed to resolve schedule metadata: %w", err)
	}

	if err := s.prepareJobCreate(ctx, token, scheduleType, false, req.AlertEnabled); err != nil {
		return nil, err
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

	job := buildTrainingVolcanoJob(jobName, labels, jobAnnotations, podAnnotations, &podSpec, queueName)

	if err := s.mgr.client.Create(ctx, &job); err != nil {
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
		return nil, agentSubmitErrorf("invalid tensorflow request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(false)
	if err != nil {
		return nil, agentSubmitErrorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, agentSubmitErrorf("failed to resolve schedule metadata: %w", err)
	}

	jobResources := utils.CalculateReplicatedResources(
		req.Tasks,
		func(task TaskReq) v1.ResourceList {
			return task.Resource
		},
		func(task TaskReq) int32 {
			return task.Replicas
		},
	)

	if err := s.prepareJobCreate(ctx, token, scheduleType, false, req.AlertEnabled); err != nil {
		return nil, err
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

	tasks, minAvailable := buildAgentParallelTaskSpecs(
		&req, baseAffinity, baseTolerations, volumes, volumeMounts, envs, labels, podAnnotations, CraterJobTypeTensorflow,
	)

	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(utils.SevenDaySeconds),
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

	if err := s.mgr.submitJob(ctx, token, &job); err != nil {
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
		return nil, agentSubmitErrorf("invalid pytorch request: %w", err)
	}

	scheduleType, err := req.validateScheduleOptions(false)
	if err != nil {
		return nil, agentSubmitErrorf("invalid schedule options: %w", err)
	}
	scheduleMetadata, err := s.mgr.resolveJobScheduleMetadata(ctx, scheduleType)
	if err != nil {
		return nil, agentSubmitErrorf("failed to resolve schedule metadata: %w", err)
	}

	jobResources := utils.CalculateReplicatedResources(
		req.Tasks,
		func(task TaskReq) v1.ResourceList {
			return task.Resource
		},
		func(task TaskReq) int32 {
			return task.Replicas
		},
	)

	if err := s.prepareJobCreate(ctx, token, scheduleType, false, req.AlertEnabled); err != nil {
		return nil, err
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

	tasks, minAvailable := buildAgentParallelTaskSpecs(
		&req, baseAffinity, baseTolerations, volumes, volumeMounts, envs, labels, podAnnotations, CraterJobTypePytorch,
	)

	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			TTLSecondsAfterFinished: ptr.To(utils.SevenDaySeconds),
			MinAvailable:            minAvailable,
			SchedulerName:           VolcanoSchedulerName,
			Plugins: map[string][]string{
				string(CraterJobTypePytorch): {pytorchPluginMasterArg, pytorchPluginWorkerArg, pytorchPluginPortArg},
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

	if err := s.mgr.submitJob(ctx, token, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func normalizeAgentSubmitOptionalString(value **string) {
	if value == nil || *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}

func agentSubmitJobNamePrefix(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "job"
}

func agentSubmitBaseURLFromJobName(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return jobName
}

func copyAgentSubmitStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (s *agentJobSubmitter) ensureResubmittedJobAccess(ctx context.Context, job *batch.Job) error {
	if job == nil || s.mgr.serviceManager == nil || len(job.Spec.Tasks) == 0 {
		return nil
	}

	labels := copyAgentSubmitStringMap(job.Labels)
	taskType := labels[crclient.LabelKeyTaskType]
	baseURL := labels[crclient.LabelKeyBaseURL]
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
	}

	switch taskType {
	case string(model.JobTypeJupyter):
		_, err := s.mgr.serviceManager.CreateIngressWithPrefix(
			ctx,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "notebook",
				Port:       JupyterPort,
				TargetPort: intstr.FromInt(JupyterPort),
				Protocol:   v1.ProtocolTCP,
			},
			config.GetConfig().Host,
			baseURL,
		)
		return err
	case string(model.JobTypeWebIDE):
		username := labels[crclient.LabelKeyTaskUser]
		randomPrefix := uuid.New().String()[:5]
		_, err := s.mgr.serviceManager.CreateNamedIngress(
			ctx,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "webide",
				Port:       JupyterPort,
				TargetPort: intstr.FromInt(JupyterPort),
				Protocol:   v1.ProtocolTCP,
			},
			config.GetConfig().Host,
			username,
			randomPrefix,
		)
		return err
	default:
		return nil
	}
}

//nolint:gocyclo // Resubmit overrides walk task/container resource specs and preserve GPU model semantics.
func applyAgentSubmitResubmitOverrides(
	job *batch.Job,
	cpu *string,
	memory *string,
	gpuCount *int,
	gpuModel *string,
) (map[string]any, error) {
	if job == nil {
		return nil, agentSubmitErrorf("job spec is unavailable for override")
	}

	applied := make(map[string]any)
	for taskIdx := range job.Spec.Tasks {
		task := &job.Spec.Tasks[taskIdx]
		for containerIdx := range task.Template.Spec.Containers {
			container := &task.Template.Spec.Containers[containerIdx]
			if cpu != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*cpu))
				if err != nil {
					return nil, agentSubmitErrorf("invalid cpu override: %w", err)
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceCPU] = quantity
				container.Resources.Limits[v1.ResourceCPU] = quantity
				applied["cpu"] = quantity.String()
			}
			if memory != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*memory))
				if err != nil {
					return nil, agentSubmitErrorf("invalid memory override: %w", err)
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceMemory] = quantity
				container.Resources.Limits[v1.ResourceMemory] = quantity
				applied["memory"] = quantity.String()
			}

			gpuResourceName, changed, err := overrideAgentSubmitGPUResourceRequirements(
				&container.Resources,
				gpuCount,
				gpuModel,
			)
			if err != nil {
				return nil, err
			}
			if changed {
				if gpuCount != nil {
					applied["gpu_count"] = *gpuCount
				}
				if gpuResourceName != "" {
					applied["gpu_resource_name"] = gpuResourceName
					if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
						applied["gpu_model"] = normalizeAgentSubmitGPUModelName(*gpuModel)
					}
				}
			}
		}
	}

	if len(applied) == 0 {
		applied["inherit"] = "original_spec"
	}
	return applied, nil
}

//nolint:gocyclo // GPU resource overrides must handle count, model rename, zeroing and inherited resource names.
func overrideAgentSubmitGPUResourceRequirements(
	requirements *v1.ResourceRequirements,
	gpuCount *int,
	gpuModel *string,
) (gpuResourceName string, changed bool, err error) {
	if requirements == nil {
		return "", false, nil
	}
	currentGPUKey := detectAgentSubmitGPUResourceName(requirements.Requests)
	if currentGPUKey == "" {
		currentGPUKey = detectAgentSubmitGPUResourceName(requirements.Limits)
	}
	if currentGPUKey == "" && gpuCount == nil && gpuModel == nil {
		return "", false, nil
	}

	targetGPUKey := currentGPUKey
	if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
		targetGPUKey = normalizeAgentSubmitGPUResourceName(currentGPUKey, *gpuModel)
	}
	if targetGPUKey == "" && gpuCount != nil && *gpuCount > 0 {
		targetGPUKey = normalizeAgentSubmitGPUResourceName(currentGPUKey, "gpu")
	}
	if targetGPUKey == "" {
		return "", false, nil
	}

	changed = false
	if requirements.Requests == nil {
		requirements.Requests = v1.ResourceList{}
	}
	if requirements.Limits == nil {
		requirements.Limits = v1.ResourceList{}
	}
	if currentGPUKey != "" && currentGPUKey != targetGPUKey {
		moveAgentSubmitResourceQuantity(requirements.Requests, currentGPUKey, targetGPUKey)
		moveAgentSubmitResourceQuantity(requirements.Limits, currentGPUKey, targetGPUKey)
		changed = true
	}
	if removeAgentSubmitGPUResourcesExcept(requirements.Requests, targetGPUKey) {
		changed = true
	}
	if removeAgentSubmitGPUResourcesExcept(requirements.Limits, targetGPUKey) {
		changed = true
	}

	if gpuCount != nil {
		if *gpuCount < 0 {
			return "", false, agentSubmitErrorf("gpu_count must be non-negative")
		}
		if *gpuCount == 0 {
			if removeAgentSubmitGPUResourcesExcept(requirements.Requests, "") {
				changed = true
			}
			if removeAgentSubmitGPUResourcesExcept(requirements.Limits, "") {
				changed = true
			}
			return "", changed, nil
		}
		quantity := *resource.NewQuantity(int64(*gpuCount), resource.DecimalSI)
		requirements.Requests[targetGPUKey] = quantity
		requirements.Limits[targetGPUKey] = quantity
		changed = true
	}

	return string(targetGPUKey), changed, nil
}

func detectAgentSubmitGPUResourceName(resources v1.ResourceList) v1.ResourceName {
	for name := range resources {
		if isAgentSubmitGPUResourceName(string(name)) {
			return name
		}
	}
	return ""
}

func normalizeAgentSubmitGPUModelName(input string) string {
	gpuModelName := strings.TrimSpace(strings.ToLower(input))
	return strings.ReplaceAll(gpuModelName, " ", "-")
}

func normalizeAgentSubmitGPUResourceName(current v1.ResourceName, gpuModel string) v1.ResourceName {
	gpuModelName := normalizeAgentSubmitGPUModelName(gpuModel)
	if gpuModelName == "" {
		return current
	}
	if strings.Contains(gpuModelName, "/") {
		return v1.ResourceName(gpuModelName)
	}
	vendor := "nvidia.com"
	if current != "" {
		parts := strings.SplitN(string(current), "/", 2)
		if len(parts) == 2 && parts[0] != "" {
			vendor = parts[0]
		}
	}
	return v1.ResourceName(fmt.Sprintf("%s/%s", vendor, gpuModelName))
}

func moveAgentSubmitResourceQuantity(resources v1.ResourceList, oldName, newName v1.ResourceName) {
	if resources == nil || oldName == "" || oldName == newName {
		return
	}
	if quantity, ok := resources[oldName]; ok {
		resources[newName] = quantity
		delete(resources, oldName)
	}
}

func removeAgentSubmitGPUResourcesExcept(resources v1.ResourceList, keep v1.ResourceName) bool {
	if resources == nil {
		return false
	}
	changed := false
	for name := range resources {
		if name == keep || !isAgentSubmitGPUResourceName(string(name)) {
			continue
		}
		delete(resources, name)
		changed = true
	}
	return changed
}

func isAgentSubmitGPUResourceName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "nvidia.com/") || strings.Contains(normalized, "/gpu") || strings.Contains(normalized, "gpu") {
		return true
	}
	for _, gpuModelName := range []string{"v100", "a100", "h100", "l40s", "rtx4090"} {
		if normalized == gpuModelName || strings.HasSuffix(normalized, "/"+gpuModelName) {
			return true
		}
	}
	return false
}
