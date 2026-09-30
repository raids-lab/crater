package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/samber/lo"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/utils"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

var (
	ErrQueueQuotaNameRequired = errors.New("queue quota name is required")
	ErrQueueQuotaNotFound     = errors.New("queue quota not found")
	ErrQueueQuotaNameConflict = errors.New("queue quota name already exists")
	ErrQueueQuotaInvalidQuota = errors.New("invalid quota value")
)

type ResourceLimitDetail struct {
	Resource string `json:"resource"`
	Used     string `json:"used"`
	Limit    string `json:"limit"`
	Exceeded bool   `json:"exceeded"`
}

type ResourceLimitCheckResult struct {
	Enabled  bool                  `json:"enabled"`
	Exceeded bool                  `json:"exceeded"`
	Details  []ResourceLimitDetail `json:"details"`
}

type QueueQuotaConfigItem struct {
	ID    uint              `json:"id"`
	Name  string            `json:"name"`
	Quota map[string]string `json:"quota"`
}

type QueueQuotaConfig struct {
	Quotas []QueueQuotaConfigItem `json:"quotas"`
}

type ResolvedQueueQuota struct {
	Name    string
	Enabled bool
	Quota   map[string]string
}

type UserResourceUsageSummaryItem struct {
	Resource string
	Used     string
	Running  string
	Pending  string
	Limit    string
	HasLimit bool
}

type UserResourceUsageSummary struct {
	RunningJobs int
	PendingJobs int
	Resources   map[string]UserResourceUsageSummaryItem
}

type QueueQuotaService struct {
	q             *query.Query
	configService *ConfigService
}

type resourceUsageMetric struct {
	usedMilli    int64
	usedFmt      apiresource.Format
	hasUsed      bool
	runningMilli int64
	runningFmt   apiresource.Format
	hasRunning   bool
	pendingMilli int64
	pendingFmt   apiresource.Format
	hasPending   bool
	limit        string
	limitFmt     apiresource.Format
	hasLimit     bool
}

func NewQueueQuotaService(q *query.Query, configService *ConfigService) *QueueQuotaService {
	return &QueueQuotaService{q: q, configService: configService}
}

func resolveQueueQuotaName(queueName string, accountID, userID uint) string {
	if queueName != "" {
		return queueName
	}

	queueName = vcqueue.PublicQueueName
	if accountID != model.DefaultAccountID {
		queueName = vcqueue.GetUserQueueName(accountID, userID)
	}
	return queueName
}

func validateAndSanitize(item *QueueQuotaConfigItem) (*QueueQuotaConfigItem, error) {
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return nil, ErrQueueQuotaNameRequired
	}
	item.Quota = sanitizeQueueQuota(item.Quota)
	for _, value := range item.Quota {
		if _, err := apiresource.ParseQuantity(value); err != nil {
			return nil, ErrQueueQuotaInvalidQuota
		}
	}
	return item, nil
}

func sanitizeQueueQuota(quota map[string]string) map[string]string {
	if quota == nil {
		quota = map[string]string{}
	}
	keys := lo.Filter(lo.Keys(quota), func(key string, _ int) bool {
		return strings.TrimSpace(key) != "" && strings.TrimSpace(quota[key]) != ""
	})
	sanitized := lo.SliceToMap(keys, func(key string) (string, string) {
		return strings.TrimSpace(key), strings.TrimSpace(quota[key])
	})
	return sanitized
}

// QueueQuotaOccupiedJobPhases lists the phases that always hold queue resources. A Pending job holds
// them only once volcano admitted its pod group, which utils.IsJobInqueue decides from both phases.
func QueueQuotaOccupiedJobPhases() []string {
	return []string{
		string(batch.Running),
		string(batch.Restarting),
		string(batch.Completing),
		string(batch.Aborting),
		string(batch.Terminating),
	}
}

// queueQuotaOccupied reports whether a job currently holds queue resources.
func queueQuotaOccupied(job *model.Job) bool {
	return slices.Contains(QueueQuotaOccupiedJobPhases(), string(job.Status)) ||
		utils.IsJobInqueue(job.Status, job.PodGroupPhase)
}

func isSupportedQueueResource(resourceName string) bool {
	return resourceName == string(v1.ResourceCPU) ||
		resourceName == string(v1.ResourceMemory) ||
		strings.Contains(resourceName, "/")
}

func buildResourceUsageSummary(
	runningJobs int,
	pendingJobs int,
	metrics map[string]*resourceUsageMetric,
) *UserResourceUsageSummary {
	resources := make(map[string]UserResourceUsageSummaryItem, len(metrics))
	for resourceName, metric := range metrics {
		if !isSupportedQueueResource(resourceName) {
			continue
		}
		format := metric.usedFmt
		if !metric.hasUsed && metric.hasLimit {
			format = metric.limitFmt
		}
		used := apiresource.NewMilliQuantity(metric.usedMilli, format).String()
		runningFormat := metric.runningFmt
		if !metric.hasRunning {
			runningFormat = format
		}
		pendingFormat := metric.pendingFmt
		if !metric.hasPending {
			pendingFormat = format
		}
		resources[resourceName] = UserResourceUsageSummaryItem{
			Resource: resourceName,
			Used:     used,
			Running:  apiresource.NewMilliQuantity(metric.runningMilli, runningFormat).String(),
			Pending:  apiresource.NewMilliQuantity(metric.pendingMilli, pendingFormat).String(),
			Limit:    metric.limit,
			HasLimit: metric.hasLimit,
		}
	}

	return &UserResourceUsageSummary{
		RunningJobs: runningJobs,
		PendingJobs: pendingJobs,
		Resources:   resources,
	}
}

func ensureResourceUsageMetric(
	metrics map[string]*resourceUsageMetric,
	resourceName string,
) *resourceUsageMetric {
	metric, ok := metrics[resourceName]
	if !ok {
		metric = &resourceUsageMetric{}
		metrics[resourceName] = metric
	}
	return metric
}

func applyUsedQuantity(metric *resourceUsageMetric, quantity apiresource.Quantity) {
	metric.usedMilli += quantity.MilliValue()
	metric.usedFmt = quantity.Format
	metric.hasUsed = true
}

func applyRunningQuantity(metric *resourceUsageMetric, quantity apiresource.Quantity) {
	metric.runningMilli += quantity.MilliValue()
	metric.runningFmt = quantity.Format
	metric.hasRunning = true
}

func applyPendingQuantity(metric *resourceUsageMetric, quantity apiresource.Quantity) {
	metric.pendingMilli += quantity.MilliValue()
	metric.pendingFmt = quantity.Format
	metric.hasPending = true
}

func applyLimitQuantity(metric *resourceUsageMetric, quantity apiresource.Quantity) {
	metric.limit = quantity.String()
	metric.limitFmt = quantity.Format
	metric.hasLimit = true
}

func buildUserResourceUsageSummary(
	resolved *ResolvedQueueQuota,
	jobs []*model.Job,
) *UserResourceUsageSummary {
	if resolved == nil {
		resolved = &ResolvedQueueQuota{}
	}

	quotaEnabled := resolved.Enabled && len(resolved.Quota) > 0
	metrics := make(map[string]*resourceUsageMetric)
	runningJobs := 0
	pendingJobs := 0

	for _, job := range jobs {
		if job == nil {
			continue
		}
		queueName := resolveQueueQuotaName(job.Queue, job.AccountID, job.UserID)
		if queueName != resolved.Name {
			continue
		}
		if !queueQuotaOccupied(job) {
			continue
		}
		inqueue := utils.IsJobInqueue(job.Status, job.PodGroupPhase)
		if job.Status == batch.Running {
			runningJobs++
		}
		if inqueue {
			pendingJobs++
		}
		for name, quantity := range job.Resources.Data() {
			metric := ensureResourceUsageMetric(metrics, string(name))
			applyUsedQuantity(metric, quantity)
			if job.Status == batch.Running {
				applyRunningQuantity(metric, quantity)
			}
			if inqueue {
				applyPendingQuantity(metric, quantity)
			}
		}
	}

	if quotaEnabled {
		for resourceName, limitStr := range resolved.Quota {
			limitQty, err := apiresource.ParseQuantity(limitStr)
			if err != nil {
				continue
			}
			applyLimitQuantity(ensureResourceUsageMetric(metrics, resourceName), limitQty)
		}
	}

	return buildResourceUsageSummary(runningJobs, pendingJobs, metrics)
}

func (s *QueueQuotaService) ResolveQueueQuota(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
) (*ResolvedQueueQuota, error) {
	if s.configService == nil {
		return nil, fmt.Errorf("config service is not initialized")
	}
	cfg, err := s.configService.GetSchedulerExtenderConfig(ctx)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedQueueQuota{
		Name:  resolveQueueQuotaName(queueName, accountID, userID),
		Quota: map[string]string{},
	}

	qql := s.q.QueueQuotaLimit
	record, err := qql.WithContext(ctx).Where(qql.Name.Eq(resolved.Name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resolved, nil
		}
		return nil, err
	}

	resolved.Quota = sanitizeQueueQuota(record.Quota.Data())
	resolved.Enabled = cfg.QueueQuotaEnabled && len(resolved.Quota) > 0
	return resolved, nil
}

// QueueQuotaSet holds every configured queue limit, so one round trip serves a whole scheduling
// decision instead of one query per inspected job.
type QueueQuotaSet struct {
	Enabled bool
	quotas  map[string]map[string]string
}

func (s *QueueQuotaService) LoadQuotaSet(
	ctx context.Context,
	cfg *model.SchedulerExtenderConfig,
) (*QueueQuotaSet, error) {
	qql := s.q.QueueQuotaLimit
	records, err := qql.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}

	set := &QueueQuotaSet{
		Enabled: cfg != nil && cfg.QueueQuotaEnabled,
		quotas:  make(map[string]map[string]string, len(records)),
	}
	for _, record := range records {
		set.quotas[record.Name] = sanitizeQueueQuota(record.Quota.Data())
	}
	return set, nil
}

func (set *QueueQuotaSet) Resolve(queueName string) *ResolvedQueueQuota {
	resolved := &ResolvedQueueQuota{Name: queueName, Quota: map[string]string{}}
	if set == nil {
		return resolved
	}
	quota, ok := set.quotas[queueName]
	if !ok {
		return resolved
	}
	resolved.Quota = quota
	resolved.Enabled = set.Enabled && len(quota) > 0
	return resolved
}

func (s *QueueQuotaService) GetConfig(ctx context.Context) (*QueueQuotaConfig, error) {
	qql := s.q.QueueQuotaLimit
	quotas, err := qql.WithContext(ctx).
		Order(qql.CreatedAt.Asc()).
		Find()
	if err != nil {
		return nil, err
	}

	items := lo.Map(quotas, func(quota *model.QueueQuotaLimit, _ int) QueueQuotaConfigItem {
		return QueueQuotaConfigItem{
			ID:    quota.ID,
			Name:  quota.Name,
			Quota: sanitizeQueueQuota(quota.Quota.Data()),
		}
	})

	return &QueueQuotaConfig{
		Quotas: items,
	}, nil
}

func (s *QueueQuotaService) CreateConfig(
	ctx context.Context,
	item *QueueQuotaConfigItem,
) (*QueueQuotaConfigItem, error) {
	var err error
	if item, err = validateAndSanitize(item); err != nil {
		return nil, err
	}
	item.Quota = sanitizeQueueQuota(item.Quota)

	qql := s.q.QueueQuotaLimit
	if _, err := qql.WithContext(ctx).Where(qql.Name.Eq(item.Name)).First(); err == nil {
		return nil, ErrQueueQuotaNameConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	record := &model.QueueQuotaLimit{
		Name:  item.Name,
		Quota: datatypes.NewJSONType(item.Quota),
	}
	if err := s.q.QueueQuotaLimit.WithContext(ctx).Create(record); err != nil {
		return nil, err
	}

	return &QueueQuotaConfigItem{
		ID:    record.ID,
		Name:  record.Name,
		Quota: item.Quota,
	}, nil
}

func (s *QueueQuotaService) UpdateConfig(
	ctx context.Context,
	item *QueueQuotaConfigItem,
) (*QueueQuotaConfigItem, error) {
	var err error
	if item, err = validateAndSanitize(item); err != nil {
		return nil, err
	}
	item.Quota = sanitizeQueueQuota(item.Quota)

	var updated *QueueQuotaConfigItem
	err = s.q.Transaction(func(tx *query.Query) error {
		qql := tx.QueueQuotaLimit
		_, err := qql.WithContext(ctx).Where(qql.ID.Eq(item.ID)).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQueueQuotaNotFound
			}
			return err
		}
		if q, err := qql.WithContext(ctx).Where(qql.Name.Eq(item.Name)).First(); err == nil {
			if q.ID != item.ID {
				return ErrQueueQuotaNameConflict
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		info, err := tx.
			QueueQuotaLimit.
			WithContext(ctx).
			Where(qql.ID.Eq(item.ID)).
			Updates(map[string]any{
				"name":  item.Name,
				"quota": datatypes.NewJSONType(item.Quota),
			})
		if err != nil {
			return err
		}
		if info.RowsAffected > 0 {
			updated = &QueueQuotaConfigItem{
				ID:    item.ID,
				Name:  item.Name,
				Quota: item.Quota,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *QueueQuotaService) DeleteQueueQuota(ctx context.Context, id uint) error {
	qql := s.q.QueueQuotaLimit
	result, err := qql.WithContext(ctx).Unscoped().Where(qql.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return ErrQueueQuotaNotFound
	}
	return nil
}

func (s *QueueQuotaService) GetUserResourceUsageSummary(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
) (*UserResourceUsageSummary, error) {
	resolved, err := s.ResolveQueueQuota(ctx, userID, accountID, queueName)
	if err != nil {
		return nil, err
	}

	return s.getUserResourceUsageSummaryWithResolved(ctx, userID, accountID, resolved)
}

func (s *QueueQuotaService) getUserResourceUsageSummaryWithResolved(
	ctx context.Context,
	userID,
	accountID uint,
	resolved *ResolvedQueueQuota,
) (*UserResourceUsageSummary, error) {
	jobs, err := s.listUserQueueOccupiedJobs(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return buildUserResourceUsageSummary(resolved, jobs), nil
}

func (s *QueueQuotaService) listUserQueueOccupiedJobs(
	ctx context.Context,
	userID,
	accountID uint,
) ([]*model.Job, error) {
	j := s.q.Job
	jobs, err := j.WithContext(ctx).Where(
		j.UserID.Eq(userID),
		j.AccountID.Eq(accountID),
		field.Or(
			j.Status.In(QueueQuotaOccupiedJobPhases()...),
			// Mirrors utils.IsJobInqueue on the database side.
			field.And(
				j.Status.In("", string(batch.Pending)),
				j.PodGroupPhase.NotIn("", string(scheduling.PodGroupPending)),
			),
		),
	).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	return jobs, nil
}

func buildResourceLimitCheckResult(
	resolved *ResolvedQueueQuota,
	jobs []*model.Job,
	requestedResources map[string]string,
) *ResourceLimitCheckResult {
	if resolved == nil || !resolved.Enabled || len(resolved.Quota) == 0 {
		return &ResourceLimitCheckResult{Enabled: false}
	}

	summary := buildUserResourceUsageSummary(resolved, jobs)
	projected := make(map[string]int64, len(summary.Resources))
	for resourceName, item := range summary.Resources {
		usedQty, parseErr := apiresource.ParseQuantity(item.Used)
		if parseErr != nil {
			continue
		}
		projected[resourceName] = usedQty.MilliValue()
	}
	addMilliQuantities(projected, requestedResources)

	return compareAgainstQuota(resolved, projected)
}

// EvaluateQueueQuota is the shared verdict core: the extender feeds cache-derived usage while the
// HTTP API feeds database-derived usage, so both sides answer with identical arithmetic.
func EvaluateQueueQuota(
	resolved *ResolvedQueueQuota,
	used v1.ResourceList,
	requested v1.ResourceList,
) *ResourceLimitCheckResult {
	if resolved == nil || !resolved.Enabled || len(resolved.Quota) == 0 {
		return &ResourceLimitCheckResult{Enabled: false}
	}

	projected := make(map[string]int64, len(used)+len(requested))
	addMilliQuantities(projected, utils.ToStringMap(used))
	addMilliQuantities(projected, utils.ToStringMap(requested))

	return compareAgainstQuota(resolved, projected)
}

func addMilliQuantities(projected map[string]int64, resources map[string]string) {
	for name, valStr := range resources {
		qty, err := apiresource.ParseQuantity(valStr)
		if err != nil {
			continue
		}
		projected[name] += qty.MilliValue()
	}
}

func compareAgainstQuota(resolved *ResolvedQueueQuota, projected map[string]int64) *ResourceLimitCheckResult {
	var details []ResourceLimitDetail
	anyExceeded := false
	for resourceName, limitStr := range resolved.Quota {
		limitQty, err := apiresource.ParseQuantity(limitStr)
		if err != nil {
			continue
		}
		usedMilli := projected[resourceName]
		exceeded := usedMilli > limitQty.MilliValue()
		details = append(details, ResourceLimitDetail{
			Resource: resourceName,
			Used:     apiresource.NewMilliQuantity(usedMilli, limitQty.Format).String(),
			Limit:    limitStr,
			Exceeded: exceeded,
		})
		if exceeded {
			anyExceeded = true
		}
	}

	return &ResourceLimitCheckResult{
		Enabled:  true,
		Exceeded: anyExceeded,
		Details:  details,
	}
}

func (s *QueueQuotaService) CheckUserResourceLimit(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
	requestedResources map[string]string,
) (*ResourceLimitCheckResult, error) {
	resolved, err := s.ResolveQueueQuota(ctx, userID, accountID, queueName)
	if err != nil {
		return nil, err
	}
	if !resolved.Enabled || len(resolved.Quota) == 0 {
		return &ResourceLimitCheckResult{Enabled: false}, nil
	}

	jobs, err := s.listUserQueueOccupiedJobs(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return buildResourceLimitCheckResult(resolved, jobs, requestedResources), nil
}

func (s *QueueQuotaService) CheckRequestedResourceLimit(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
	requestedResources map[string]string,
) (*ResourceLimitCheckResult, error) {
	resolved, err := s.ResolveQueueQuota(ctx, userID, accountID, queueName)
	if err != nil {
		return nil, err
	}
	if !resolved.Enabled || len(resolved.Quota) == 0 {
		return &ResourceLimitCheckResult{Enabled: false}, nil
	}
	return buildResourceLimitCheckResult(resolved, nil, requestedResources), nil
}
