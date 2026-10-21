package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	agentDefaultJobPageSize = 20
	agentMaximumJobPageSize = 50
	agentMaximumJobSearch   = 128
)

type AgentPlatformReadService struct {
	db         *gorm.DB
	kubeClient kubernetes.Interface
}

type AgentJobListOptions struct {
	Page      int
	PageSize  int
	Search    string
	Username  string
	Statuses  []string
	Identity  AgentIdentity
	UserID    uint
	AccountID uint
}

type AgentJobSummary struct {
	Name        string            `json:"name"`
	JobName     string            `json:"jobName"`
	Username    string            `json:"username,omitempty"`
	Nickname    string            `json:"nickname,omitempty"`
	JobType     string            `json:"jobType"`
	Status      string            `json:"status"`
	Nodes       []string          `json:"nodes,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	StartedAt   time.Time         `json:"startedAt,omitempty"`
	CompletedAt time.Time         `json:"completedAt,omitempty"`
}

type AgentJobListPage struct {
	Items    []AgentJobSummary `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type AgentNodeCapacity struct {
	Name               string            `json:"name"`
	Role               string            `json:"role"`
	Architecture       string            `json:"architecture"`
	Ready              bool              `json:"ready"`
	Schedulable        bool              `json:"schedulable"`
	Capacity           map[string]string `json:"capacity"`
	Allocatable        map[string]string `json:"allocatable"`
	Requested          map[string]string `json:"requested"`
	EstimatedAvailable map[string]string `json:"estimatedAvailable"`
	Accelerators       map[string]string `json:"accelerators,omitempty"`
	Workloads          int               `json:"workloads"`
	ObservedAt         time.Time         `json:"observedAt"`
}

type AgentClusterCapacity struct {
	NodeCount          int               `json:"nodeCount"`
	ReadyNodes         int               `json:"readyNodes"`
	SchedulableNodes   int               `json:"schedulableNodes"`
	Allocatable        map[string]string `json:"allocatable"`
	Requested          map[string]string `json:"requested"`
	EstimatedAvailable map[string]string `json:"estimatedAvailable"`
	Accelerators       map[string]string `json:"accelerators,omitempty"`
	ObservedAt         time.Time         `json:"observedAt"`
}

func NewAgentPlatformReadService(
	db *gorm.DB,
	kubeClient kubernetes.Interface,
) *AgentPlatformReadService {
	return &AgentPlatformReadService{db: db, kubeClient: kubeClient}
}

func (s *AgentPlatformReadService) ListJobs(
	ctx context.Context,
	options *AgentJobListOptions,
) (*AgentJobListPage, error) {
	if err := normalizeAgentJobListOptions(options); err != nil {
		return nil, err
	}

	query, err := s.authorizedAgentJobQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	query, err = filterAgentJobs(query, options)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to count Agent jobs")
	}
	var jobs []model.Job
	offset := (options.Page - 1) * options.PageSize
	if err := query.Order("creation_timestamp DESC").Offset(offset).Limit(options.PageSize).Find(&jobs).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to list Agent jobs")
	}
	items := agentJobSummaries(jobs, options.Identity)
	return &AgentJobListPage{Items: items, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func normalizeAgentJobListOptions(options *AgentJobListOptions) error {
	if options.Page < 1 {
		options.Page = 1
	}
	if options.PageSize < 1 {
		options.PageSize = agentDefaultJobPageSize
	} else if options.PageSize > agentMaximumJobPageSize {
		options.PageSize = agentMaximumJobPageSize
	}
	options.Search = strings.TrimSpace(options.Search)
	options.Username = strings.TrimSpace(options.Username)
	if len([]rune(options.Search)) > agentMaximumJobSearch {
		return bizerr.BadRequest.ParameterError.New("search accepts at most 128 characters")
	}
	return nil
}

func (s *AgentPlatformReadService) authorizedAgentJobQuery(
	ctx context.Context,
	options *AgentJobListOptions,
) (*gorm.DB, error) {
	query := s.db.WithContext(ctx).Model(&model.Job{}).Preload("User").Preload("Account")
	switch options.Identity {
	case AgentIdentityUser:
		if options.Username != "" {
			return nil, bizerr.Forbidden.PermissionDenied.New("users cannot query another user's jobs")
		}
		query = query.Where("user_id = ? AND account_id = ?", options.UserID, options.AccountID)
	case AgentIdentityAdmin:
		if options.Username != "" {
			var user model.User
			if err := s.db.WithContext(ctx).Where("name = ?", options.Username).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, bizerr.NotFound.DataBaseNotFound.New("user not found")
				}
				return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to resolve job owner")
			}
			query = query.Where("user_id = ?", user.ID)
		}
	default:
		return nil, bizerr.Forbidden.PermissionDenied.New("Agent identity cannot list jobs")
	}
	return query, nil
}

func filterAgentJobs(query *gorm.DB, options *AgentJobListOptions) (*gorm.DB, error) {
	if options.Search != "" {
		pattern := "%" + strings.ToLower(options.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(job_name) LIKE ?", pattern, pattern)
	}
	if len(options.Statuses) > 0 {
		for _, status := range options.Statuses {
			if !validAgentJobStatus(status) {
				return nil, bizerr.BadRequest.ParameterError.New("unsupported job status " + status)
			}
		}
		query = query.Where("status IN ?", options.Statuses)
	}
	return query, nil
}

func agentJobSummaries(jobs []model.Job, identity AgentIdentity) []AgentJobSummary {
	items := make([]AgentJobSummary, 0, len(jobs))
	for index := range jobs {
		job := &jobs[index]
		item := AgentJobSummary{
			Name:        job.Name,
			JobName:     job.JobName,
			JobType:     string(job.JobType),
			Status:      string(job.Status),
			Nodes:       job.Nodes.Data(),
			Resources:   utils.ToStringMap(job.Resources.Data()),
			CreatedAt:   job.CreationTimestamp,
			StartedAt:   job.RunningTimestamp,
			CompletedAt: job.CompletedTimestamp,
		}
		if identity == AgentIdentityAdmin {
			item.Username = job.User.Name
			item.Nickname = job.User.Nickname
		}
		items = append(items, item)
	}
	return items
}

func (s *AgentPlatformReadService) ListNodes(ctx context.Context) ([]AgentNodeCapacity, error) {
	if s.kubeClient == nil {
		return nil, bizerr.ServiceError.ServiceUnavailable.New("Kubernetes node data is unavailable")
	}
	nodes, err := s.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, bizerr.ServiceError.ServiceUnavailable.Wrap(err, "failed to read Kubernetes nodes")
	}
	pods, err := s.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, bizerr.ServiceError.ServiceUnavailable.Wrap(err, "failed to read Kubernetes pod requests")
	}
	requestedByNode := make(map[string]v1.ResourceList)
	workloadsByNode := make(map[string]int)
	for index := range pods.Items {
		pod := &pods.Items[index]
		if pod.Spec.NodeName == "" || pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			continue
		}
		requestedByNode[pod.Spec.NodeName] = utils.SumResources(
			requestedByNode[pod.Spec.NodeName], utils.CalculateRequsetsByContainers(pod.Spec.Containers),
		)
		workloadsByNode[pod.Spec.NodeName]++
	}
	observedAt := time.Now()
	result := make([]AgentNodeCapacity, 0, len(nodes.Items))
	for index := range nodes.Items {
		node := &nodes.Items[index]
		requested := requestedByNode[node.Name]
		available := nonNegativeResources(utils.SubtractResource(node.Status.Allocatable, requested))
		result = append(result, AgentNodeCapacity{
			Name:               node.Name,
			Role:               agentNodeRole(node),
			Architecture:       node.Status.NodeInfo.Architecture,
			Ready:              agentNodeReady(node),
			Schedulable:        !node.Spec.Unschedulable,
			Capacity:           agentResourceStrings(node.Status.Capacity),
			Allocatable:        agentResourceStrings(node.Status.Allocatable),
			Requested:          agentResourceStrings(requested),
			EstimatedAvailable: agentResourceStrings(available),
			Accelerators:       agentAcceleratorStrings(node.Status.Allocatable),
			Workloads:          workloadsByNode[node.Name],
			ObservedAt:         observedAt,
		})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name < result[right].Name })
	return result, nil
}

func (s *AgentPlatformReadService) GetNodeCapacity(
	ctx context.Context,
	name string,
) (*AgentNodeCapacity, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, bizerr.BadRequest.MissingParameter.New("node name is required")
	}
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	for index := range nodes {
		if nodes[index].Name == name {
			return &nodes[index], nil
		}
	}
	return nil, bizerr.NotFound.K8sResourceNotFound.New("node not found")
}

func (s *AgentPlatformReadService) GetClusterCapacity(ctx context.Context) (*AgentClusterCapacity, error) {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	allocatable := make(v1.ResourceList)
	requested := make(v1.ResourceList)
	available := make(v1.ResourceList)
	accelerators := make(v1.ResourceList)
	result := &AgentClusterCapacity{NodeCount: len(nodes), ObservedAt: time.Now()}
	for index := range nodes {
		node := &nodes[index]
		if node.Ready {
			result.ReadyNodes++
		}
		if node.Ready && node.Schedulable {
			result.SchedulableNodes++
		} else {
			continue
		}
		allocatable = utils.SumResources(allocatable, parseAgentResourceStrings(node.Allocatable))
		requested = utils.SumResources(requested, parseAgentResourceStrings(node.Requested))
		available = utils.SumResources(available, parseAgentResourceStrings(node.EstimatedAvailable))
		accelerators = utils.SumResources(accelerators, parseAgentResourceStrings(node.Accelerators))
	}
	result.Allocatable = agentResourceStrings(allocatable)
	result.Requested = agentResourceStrings(requested)
	result.EstimatedAvailable = agentResourceStrings(available)
	result.Accelerators = agentResourceStrings(accelerators)
	return result, nil
}

//nolint:misspell // Volcano exposes this phase as "Cancelled".
func validAgentJobStatus(status string) bool {
	switch status {
	case "Prequeue", "Pending", "Aborting", "Aborted", "Running", "Restarting", "Completing",
		"Completed", "Terminating", "Terminated", "Failed", "Deleted", "Freed", "Cancelled":
		return true
	default:
		return false
	}
}

func agentNodeReady(node *v1.Node) bool {
	for index := range node.Status.Conditions {
		condition := &node.Status.Conditions[index]
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func agentNodeRole(node *v1.Node) string {
	if _, found := node.Labels["node-role.kubernetes.io/control-plane"]; found {
		return "control-plane"
	}
	if _, found := node.Labels["node-role.kubernetes.io/master"]; found {
		return "control-plane"
	}
	return "worker"
}

func agentResourceStrings(resources v1.ResourceList) map[string]string {
	result := make(map[string]string)
	for name, quantity := range resources {
		if !agentVisibleResource(name) || quantity.IsZero() {
			continue
		}
		result[string(name)] = quantity.String()
	}
	return result
}

func agentAcceleratorStrings(resources v1.ResourceList) map[string]string {
	result := make(map[string]string)
	for name, quantity := range resources {
		if !strings.Contains(string(name), "/") || quantity.IsZero() {
			continue
		}
		result[string(name)] = quantity.String()
	}
	return result
}

func agentVisibleResource(name v1.ResourceName) bool {
	return name == v1.ResourceCPU || name == v1.ResourceMemory || name == v1.ResourceEphemeralStorage ||
		strings.Contains(string(name), "/")
}

func nonNegativeResources(resources v1.ResourceList) v1.ResourceList {
	result := make(v1.ResourceList, len(resources))
	for name, quantity := range resources {
		if quantity.Sign() < 0 {
			result[name] = *resource.NewQuantity(0, quantity.Format)
			continue
		}
		result[name] = quantity.DeepCopy()
	}
	return result
}

func parseAgentResourceStrings(values map[string]string) v1.ResourceList {
	result := make(v1.ResourceList, len(values))
	for name, value := range values {
		quantity, err := resource.ParseQuantity(value)
		if err == nil {
			result[v1.ResourceName(name)] = quantity
		}
	}
	return result
}
