package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/utils"
	"github.com/raids-lab/crater/pkg/vcjob/admission"
)

const (
	agentDiagnosticMaximumEvents       = 20
	agentDiagnosticMaximumPods         = 20
	agentDiagnosticMaximumEventMessage = 1_024
	agentDiagnosticMaximumEvidence     = 512
	agentExitCodeCommandNotExecutable  = 126
	agentExitCodeCommandNotFound       = 127
	agentExitCodeSIGKILL               = 137
)

// AgentJobDiagnosticService provides bounded, read-only diagnostic facts for
// an authorized Volcano job. It deliberately keeps authorization and cluster
// access in the backend instead of exposing either to the Agent runtime.
type AgentJobDiagnosticService struct {
	db         *gorm.DB
	kubeClient kubernetes.Interface
}

type AgentJobDiagnosticSnapshot struct {
	ObservedAt   time.Time                    `json:"observedAt"`
	Job          AgentDiagnosticJob           `json:"job"`
	Events       []AgentDiagnosticEvent       `json:"events"`
	Terminations []AgentDiagnosticTermination `json:"terminations,omitempty"`
	Metrics      map[string]float32           `json:"metrics,omitempty"`
	Findings     []AgentDiagnosticFinding     `json:"findings"`
	Scheduling   *AgentSchedulingDiagnosis    `json:"scheduling,omitempty"`
	Warnings     []string                     `json:"warnings,omitempty"`
}

type AgentDiagnosticJob struct {
	Name        string            `json:"name"`
	JobName     string            `json:"jobName"`
	Username    string            `json:"username,omitempty"`
	Nickname    string            `json:"nickname,omitempty"`
	JobType     string            `json:"jobType"`
	Status      string            `json:"status"`
	Resources   map[string]string `json:"resources"`
	CreatedAt   time.Time         `json:"createdAt"`
	StartedAt   time.Time         `json:"startedAt,omitempty"`
	CompletedAt time.Time         `json:"completedAt,omitempty"`
}

type AgentDiagnosticEvent struct {
	Reason     string    `json:"reason"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	Count      int32     `json:"count"`
	ObjectKind string    `json:"objectKind"`
	ObservedAt time.Time `json:"observedAt,omitempty"`
}

type AgentDiagnosticTermination struct {
	Reason     string    `json:"reason"`
	ExitCode   int32     `json:"exitCode"`
	Signal     int32     `json:"signal,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
}

type AgentDiagnosticFinding struct {
	Code       string   `json:"code"`
	Confidence string   `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

type AgentSchedulingDiagnosis struct {
	Source     string                         `json:"source"`
	TotalNodes int                            `json:"totalNodes"`
	Tasks      []AgentSchedulingTaskDiagnosis `json:"tasks"`
}

type AgentSchedulingTaskDiagnosis struct {
	Name                 string            `json:"name"`
	Replicas             int32             `json:"replicas"`
	Requests             map[string]string `json:"requests"`
	CompatibleNodes      int               `json:"compatibleNodes"`
	ResourceMatchedNodes int               `json:"resourceMatchedNodes"`
	ResourceShortages    map[string]int    `json:"resourceShortages,omitempty"`
}

func NewAgentJobDiagnosticService(
	db *gorm.DB,
	kubeClient kubernetes.Interface,
) *AgentJobDiagnosticService {
	return &AgentJobDiagnosticService{db: db, kubeClient: kubeClient}
}

func (s *AgentJobDiagnosticService) GetSnapshot(
	ctx context.Context,
	userID, accountID uint,
	identity AgentIdentity,
	jobName string,
) (*AgentJobDiagnosticSnapshot, error) {
	if strings.TrimSpace(jobName) == "" {
		return nil, bizerr.BadRequest.MissingParameter.New("job name is required")
	}

	job, err := s.getAuthorizedJob(ctx, userID, accountID, identity, jobName)
	if err != nil {
		return nil, err
	}

	snapshot := &AgentJobDiagnosticSnapshot{
		ObservedAt: time.Now(),
		Job: AgentDiagnosticJob{
			Name:        job.Name,
			JobName:     job.JobName,
			JobType:     string(job.JobType),
			Status:      string(job.Status),
			Resources:   utils.ToStringMap(job.Resources.Data()),
			CreatedAt:   job.CreationTimestamp,
			StartedAt:   job.RunningTimestamp,
			CompletedAt: job.CompletedTimestamp,
			Username:    job.User.Name,
			Nickname:    job.User.Nickname,
		},
	}
	if job.Events != nil {
		snapshot.Events = append(snapshot.Events, normalizeAgentDiagnosticEvents(job.Events.Data())...)
	}
	if job.TerminatedStates != nil {
		snapshot.Terminations = normalizeAgentDiagnosticTerminations(job.TerminatedStates.Data())
	}
	snapshot.Metrics = normalizeAgentDiagnosticMetrics(job)

	volcanoJob := job.Attributes.Data()
	if volcanoJob == nil || s.kubeClient == nil {
		snapshot.Warnings = append(snapshot.Warnings, "live Kubernetes diagnostic data is unavailable")
	} else {
		liveEvents, warnings := s.collectLiveEvents(ctx, volcanoJob)
		snapshot.Events = mergeAgentDiagnosticEvents(snapshot.Events, liveEvents)
		snapshot.Warnings = append(snapshot.Warnings, warnings...)
		if isAgentSchedulingPhase(job.Status) {
			scheduling, schedulingWarnings := s.collectSchedulingDiagnosis(ctx, volcanoJob)
			snapshot.Scheduling = scheduling
			snapshot.Warnings = append(snapshot.Warnings, schedulingWarnings...)
		}
	}
	snapshot.Findings = buildAgentDiagnosticFindings(snapshot)
	return snapshot, nil
}

func (s *AgentJobDiagnosticService) GetSchedulingDiagnosis(
	ctx context.Context,
	userID, accountID uint,
	identity AgentIdentity,
	jobName string,
) (*AgentSchedulingDiagnosis, error) {
	snapshot, err := s.GetSnapshot(ctx, userID, accountID, identity, jobName)
	if err != nil {
		return nil, err
	}
	if snapshot.Scheduling == nil {
		return nil, bizerr.Conflict.ResourceStatusError.New("job is not waiting for scheduling")
	}
	return snapshot.Scheduling, nil
}

func (s *AgentJobDiagnosticService) getAuthorizedJob(
	ctx context.Context,
	userID, accountID uint,
	identity AgentIdentity,
	jobName string,
) (*model.Job, error) {
	var job model.Job
	query := s.db.WithContext(ctx).Preload("User").Preload("Account").Where("job_name = ?", jobName)
	if identity == AgentIdentityUser {
		query = query.Where("user_id = ? AND account_id = ?", userID, accountID)
	} else if identity != AgentIdentityAdmin {
		return nil, bizerr.Forbidden.PermissionDenied.New("Agent identity cannot access job diagnostics")
	}
	err := query.First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.NotFound.DataBaseNotFound.New("job not found")
	}
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load job diagnostics")
	}
	return &job, nil
}

func (s *AgentJobDiagnosticService) collectLiveEvents(
	ctx context.Context,
	job *batch.Job,
) (events []AgentDiagnosticEvent, warnings []string) {
	warnings = make([]string, 0)
	events = make([]AgentDiagnosticEvent, 0)
	jobEvents, err := s.kubeClient.CoreV1().Events(job.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + job.Name,
	})
	if err != nil {
		warnings = append(warnings, "live job events are unavailable")
	} else {
		events = append(events, normalizeAgentDiagnosticEvents(jobEvents.Items)...)
	}
	pods, err := s.kubeClient.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: batch.JobNameKey + "=" + job.Name,
		Limit:         agentDiagnosticMaximumPods,
	})
	if err != nil {
		warnings = append(warnings, "live job pods are unavailable")
		return events, warnings
	}
	if pods.Continue != "" {
		warnings = append(warnings, "live pod events are limited to the first 20 job pods")
	}
	for index := range pods.Items {
		podEvents, err := s.kubeClient.CoreV1().Events(job.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + pods.Items[index].Name,
		})
		if err != nil {
			warnings = append(warnings, "some live pod events are unavailable")
			continue
		}
		events = append(events, normalizeAgentDiagnosticEvents(podEvents.Items)...)
	}
	return events, warnings
}

func (s *AgentJobDiagnosticService) collectSchedulingDiagnosis(
	ctx context.Context,
	job *batch.Job,
) (diagnosis *AgentSchedulingDiagnosis, warnings []string) {
	nodes, err := s.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, []string{"live node scheduling data is unavailable"}
	}
	pods, err := s.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, []string{"live pod resource requests are unavailable"}
	}

	usedByNode := make(map[string]v1.ResourceList)
	for index := range pods.Items {
		pod := &pods.Items[index]
		if pod.Spec.NodeName == "" || pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			continue
		}
		usedByNode[pod.Spec.NodeName] = utils.SumResources(
			usedByNode[pod.Spec.NodeName], utils.CalculateRequsetsByContainers(pod.Spec.Containers),
		)
	}

	diagnosis = &AgentSchedulingDiagnosis{
		Source:     "Kubernetes allocatable resources minus active Pod requests",
		TotalNodes: len(nodes.Items),
		Tasks:      make([]AgentSchedulingTaskDiagnosis, 0, len(job.Spec.Tasks)),
	}
	for index := range job.Spec.Tasks {
		task := &job.Spec.Tasks[index]
		if task.Replicas <= 0 {
			continue
		}
		requests := utils.CalculateRequsetsByContainers(task.Template.Spec.Containers)
		result := AgentSchedulingTaskDiagnosis{
			Name:              task.Name,
			Replicas:          task.Replicas,
			Requests:          utils.ToStringMap(requests),
			ResourceShortages: make(map[string]int),
		}
		for nodeIndex := range nodes.Items {
			node := &nodes.Items[nodeIndex]
			if !admission.NodeMatchesPodSchedulingConstraints(node, &task.Template.Spec) {
				continue
			}
			result.CompatibleNodes++
			available := utils.SubtractResource(node.Status.Allocatable, usedByNode[node.Name])
			if utils.ResourceListCovers(available, requests) {
				result.ResourceMatchedNodes++
				continue
			}
			for resourceName, requested := range requests {
				availableQuantity := available[resourceName]
				if availableQuantity.Cmp(requested) < 0 {
					result.ResourceShortages[string(resourceName)]++
				}
			}
		}
		if len(result.ResourceShortages) == 0 {
			result.ResourceShortages = nil
		}
		diagnosis.Tasks = append(diagnosis.Tasks, result)
	}
	return diagnosis, nil
}

func normalizeAgentDiagnosticEvents(events []v1.Event) []AgentDiagnosticEvent {
	result := make([]AgentDiagnosticEvent, 0, len(events))
	for index := range events {
		event := &events[index]
		observedAt := event.EventTime.Time
		if observedAt.IsZero() {
			observedAt = event.LastTimestamp.Time
		}
		if observedAt.IsZero() {
			observedAt = event.FirstTimestamp.Time
		}
		result = append(result, AgentDiagnosticEvent{
			Reason:     event.Reason,
			Type:       event.Type,
			Message:    truncateAgentDiagnosticText(event.Message, agentDiagnosticMaximumEventMessage),
			Count:      event.Count,
			ObjectKind: event.InvolvedObject.Kind,
			ObservedAt: observedAt,
		})
	}
	return result
}

func mergeAgentDiagnosticEvents(
	stored, live []AgentDiagnosticEvent,
) []AgentDiagnosticEvent {
	merged := make(map[string]AgentDiagnosticEvent, len(stored)+len(live))
	for _, event := range append(stored, live...) {
		key := event.Reason + "\x00" + event.Message + "\x00" + event.ObjectKind
		if existing, found := merged[key]; !found || event.ObservedAt.After(existing.ObservedAt) {
			merged[key] = event
		}
	}
	result := make([]AgentDiagnosticEvent, 0, len(merged))
	for _, event := range merged {
		result = append(result, event)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].ObservedAt.After(result[right].ObservedAt)
	})
	if len(result) > agentDiagnosticMaximumEvents {
		result = result[:agentDiagnosticMaximumEvents]
	}
	return result
}

func normalizeAgentDiagnosticTerminations(
	states []v1.ContainerStateTerminated,
) []AgentDiagnosticTermination {
	result := make([]AgentDiagnosticTermination, 0, len(states))
	for index := range states {
		state := &states[index]
		result = append(result, AgentDiagnosticTermination{
			Reason:     state.Reason,
			ExitCode:   state.ExitCode,
			Signal:     state.Signal,
			StartedAt:  state.StartedAt.Time,
			FinishedAt: state.FinishedAt.Time,
		})
	}
	return result
}

func normalizeAgentDiagnosticMetrics(job *model.Job) map[string]float32 {
	if job.ProfileData == nil || job.ProfileData.Data() == nil {
		return nil
	}
	profile := job.ProfileData.Data()
	metrics := make(map[string]float32)
	if profile.MemLimit != nil {
		metrics["memoryLimitMiB"] = *profile.MemLimit
	}
	if profile.CPUMemMax != nil {
		metrics["memoryUsageMaxMiB"] = *profile.CPUMemMax
	}
	if profile.GPUUtilMax != nil {
		metrics["gpuUtilizationMax"] = *profile.GPUUtilMax
	}
	if len(metrics) == 0 {
		return nil
	}
	return metrics
}

func buildAgentDiagnosticFindings(snapshot *AgentJobDiagnosticSnapshot) []AgentDiagnosticFinding {
	collector := newAgentDiagnosticFindingCollector()
	collector.addTerminations(snapshot.Terminations)
	collector.addEvents(snapshot.Events)
	collector.addScheduling(snapshot.Scheduling)
	return collector.findings
}

type agentDiagnosticFindingCollector struct {
	findings []AgentDiagnosticFinding
	seen     map[string]struct{}
}

func newAgentDiagnosticFindingCollector() *agentDiagnosticFindingCollector {
	return &agentDiagnosticFindingCollector{
		findings: make([]AgentDiagnosticFinding, 0),
		seen:     make(map[string]struct{}),
	}
}

func (collector *agentDiagnosticFindingCollector) add(code, confidence, evidence string) {
	if _, exists := collector.seen[code]; exists {
		return
	}
	collector.seen[code] = struct{}{}
	collector.findings = append(collector.findings, AgentDiagnosticFinding{
		Code:       code,
		Confidence: confidence,
		Evidence:   []string{truncateAgentDiagnosticText(evidence, agentDiagnosticMaximumEvidence)},
	})
}

func (collector *agentDiagnosticFindingCollector) addTerminations(
	terminations []AgentDiagnosticTermination,
) {
	for index := range terminations {
		state := &terminations[index]
		if state.Reason == "OOMKilled" {
			collector.add("oom-killed", "confirmed", "container termination reason is OOMKilled")
		} else if state.ExitCode == agentExitCodeSIGKILL {
			collector.add("sigkill-exit", "observed", "container exited with code 137; this alone does not establish OOM")
		}
		switch state.ExitCode {
		case agentExitCodeCommandNotExecutable:
			collector.add("command-not-executable", "observed", "container exited with code 126")
		case agentExitCodeCommandNotFound:
			collector.add("command-not-found", "observed", "container exited with code 127")
		}
	}
}

func (collector *agentDiagnosticFindingCollector) addEvents(events []AgentDiagnosticEvent) {
	for index := range events {
		event := &events[index]
		message := strings.ToLower(event.Message)
		reason := strings.ToLower(event.Reason)
		evidence := event.Reason + ": " + event.Message
		switch {
		case strings.Contains(reason, "failedscheduling") && strings.Contains(message, "insufficient"):
			collector.add("insufficient-resources", "confirmed", evidence)
		case strings.Contains(reason, "failedscheduling") &&
			(strings.Contains(message, "node selector") || strings.Contains(message, "affinity")):
			collector.add("node-selection-mismatch", "confirmed", evidence)
		case strings.Contains(reason, "failedscheduling") && strings.Contains(message, "taint"):
			collector.add("untolerated-taint", "confirmed", evidence)
		case strings.Contains(reason, "errimagepull") || strings.Contains(reason, "imagepullbackoff") ||
			strings.Contains(message, "failed to pull image"):
			collector.add("image-pull-failed", "confirmed", evidence)
		case strings.Contains(reason, "failedmount") || strings.Contains(message, "unable to attach or mount volumes"):
			collector.add("volume-mount-failed", "confirmed", evidence)
		case strings.Contains(reason, "evicted"):
			collector.add("pod-evicted", "confirmed", evidence)
		}
	}
}

func (collector *agentDiagnosticFindingCollector) addScheduling(scheduling *AgentSchedulingDiagnosis) {
	if scheduling == nil {
		return
	}
	for index := range scheduling.Tasks {
		task := &scheduling.Tasks[index]
		if task.CompatibleNodes > 0 && task.ResourceMatchedNodes == 0 && len(task.ResourceShortages) > 0 {
			collector.add(
				"estimated-resource-shortage",
				"observed",
				"no compatible node currently has enough requested resources for one task replica",
			)
		}
	}
}

func isAgentSchedulingPhase(phase batch.JobPhase) bool {
	return phase == batch.Pending || phase == batch.JobPhase("Prequeue")
}

func truncateAgentDiagnosticText(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum] + "..."
}
