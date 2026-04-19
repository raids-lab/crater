package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/samber/lo"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
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
	ID                    uint              `json:"id"`
	Name                  string            `json:"name"`
	Enabled               bool              `json:"enabled"`
	PrequeueCandidateSize int               `json:"prequeueCandidateSize"`
	Quota                 map[string]string `json:"quota"`
}

type QueueQuotaConfig struct {
	Quotas []QueueQuotaConfigItem `json:"quotas"`
}

type ResolvedQueueQuota struct {
	Name                  string
	Enabled               bool
	PrequeueCandidateSize int
	Quota                 map[string]string
}

type UserResourceUsageSummaryItem struct {
	Resource string
	Used     string
	Limit    string
	HasLimit bool
}

type UserResourceUsageSummary struct {
	QueueName    string
	QuotaEnabled bool
	OccupiedJobs int
	Resources    map[string]UserResourceUsageSummaryItem
}

type PrequeueService struct {
	q             *query.Query
	configService *ConfigService
}

type resourceUsageMetric struct {
	usedMilli int64
	usedFmt   apiresource.Format
	hasUsed   bool
	limit     string
	limitFmt  apiresource.Format
	hasLimit  bool
}

func NewPrequeueService(q *query.Query, configService *ConfigService) *PrequeueService {
	return &PrequeueService{q: q, configService: configService}
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

func QueueQuotaOccupiedJobPhases() []string {
	return []string{
		string(batch.Running),
		string(batch.Pending),
		string(batch.Restarting),
		string(batch.Completing),
		string(batch.Aborting),
		string(batch.Terminating),
	}
}

func queueQuotaPhaseSet() map[batch.JobPhase]struct{} {
	phases := make(map[batch.JobPhase]struct{}, len(QueueQuotaOccupiedJobPhases()))
	for _, phase := range QueueQuotaOccupiedJobPhases() {
		phases[batch.JobPhase(phase)] = struct{}{}
	}
	return phases
}

func isSupportedQueueResource(resourceName string) bool {
	return resourceName == string(v1.ResourceCPU) ||
		resourceName == string(v1.ResourceMemory) ||
		strings.Contains(resourceName, "/")
}

func buildResourceUsageSummary(
	queueName string,
	quotaEnabled bool,
	occupiedJobs int,
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
		resources[resourceName] = UserResourceUsageSummaryItem{
			Resource: resourceName,
			Used:     used,
			Limit:    metric.limit,
			HasLimit: metric.hasLimit,
		}
	}

	return &UserResourceUsageSummary{
		QueueName:    queueName,
		QuotaEnabled: quotaEnabled,
		OccupiedJobs: occupiedJobs,
		Resources:    resources,
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

func applyLimitQuantity(metric *resourceUsageMetric, quantity apiresource.Quantity) {
	metric.limit = quantity.String()
	metric.limitFmt = quantity.Format
	metric.hasLimit = true
}

func BuildQueueResourceUsageSummary(
	queueName string,
	allocated v1.ResourceList,
	capability v1.ResourceList,
	occupiedJobs int,
) *UserResourceUsageSummary {
	metrics := make(map[string]*resourceUsageMetric)

	for name, quantity := range allocated {
		applyUsedQuantity(ensureResourceUsageMetric(metrics, string(name)), quantity)
	}

	for name, quantity := range capability {
		applyLimitQuantity(ensureResourceUsageMetric(metrics, string(name)), quantity)
	}

	return buildResourceUsageSummary(queueName, len(capability) > 0, occupiedJobs, metrics)
}

func buildUserResourceUsageSummary(
	resolved *ResolvedQueueQuota,
	jobs []*model.Job,
) *UserResourceUsageSummary {
	if resolved == nil {
		resolved = &ResolvedQueueQuota{}
	}

	quotaEnabled := resolved.Enabled && len(resolved.Quota) > 0
	activePhases := queueQuotaPhaseSet()
	metrics := make(map[string]*resourceUsageMetric)
	occupiedJobs := 0

	for _, job := range jobs {
		if job == nil {
			continue
		}
		queueName := resolveQueueQuotaName(job.Queue, job.AccountID, job.UserID)
		if queueName != resolved.Name {
			continue
		}
		if _, ok := activePhases[job.Status]; !ok {
			continue
		}
		if job.Status == batch.Running || job.Status == batch.Pending {
			occupiedJobs++
		}
		for name, quantity := range job.Resources.Data() {
			applyUsedQuantity(ensureResourceUsageMetric(metrics, string(name)), quantity)
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

	return buildResourceUsageSummary(resolved.Name, quotaEnabled, occupiedJobs, metrics)
}

func (s *PrequeueService) isQueueQuotaGloballyEnabled(ctx context.Context) (bool, error) {
	if s.configService == nil {
		return false, fmt.Errorf("config service is not initialized")
	}

	cfg, err := s.configService.GetPrequeueConfig(ctx)
	if err != nil {
		return false, err
	}

	return cfg.QueueQuotaEnabled, nil
}

func (s *PrequeueService) ResolveQueueQuota(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
) (*ResolvedQueueQuota, error) {
	queueQuotaEnabled, err := s.isQueueQuotaGloballyEnabled(ctx)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedQueueQuota{
		Name:                  resolveQueueQuotaName(queueName, accountID, userID),
		PrequeueCandidateSize: model.DefaultPrequeueCandidateSize,
		Quota:                 map[string]string{},
	}

	qql := s.q.QueueQuotaLimit
	record, err := qql.WithContext(ctx).Where(qql.Name.Eq(resolved.Name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resolved, nil
		}
		return nil, err
	}

	resolved.Enabled = queueQuotaEnabled && record.Enabled
	if record.PrequeueCandidateSize > 0 {
		resolved.PrequeueCandidateSize = record.PrequeueCandidateSize
	}
	resolved.Quota = sanitizeQueueQuota(record.Quota.Data())
	return resolved, nil
}

func (s *PrequeueService) GetConfig(ctx context.Context) (*QueueQuotaConfig, error) {
	qql := s.q.QueueQuotaLimit
	quotas, err := qql.WithContext(ctx).
		Order(qql.CreatedAt.Asc()).
		Find()
	if err != nil {
		return nil, err
	}

	items := lo.Map(quotas, func(quota *model.QueueQuotaLimit, _ int) QueueQuotaConfigItem {
		size := lo.Ternary(
			quota.PrequeueCandidateSize > 0,
			quota.PrequeueCandidateSize,
			model.DefaultPrequeueCandidateSize,
		)
		return QueueQuotaConfigItem{
			ID:                    quota.ID,
			Name:                  quota.Name,
			Enabled:               quota.Enabled,
			PrequeueCandidateSize: size,
			Quota:                 sanitizeQueueQuota(quota.Quota.Data()),
		}
	})

	return &QueueQuotaConfig{
		Quotas: items,
	}, nil
}

func (s *PrequeueService) CreateConfig(
	ctx context.Context,
	item *QueueQuotaConfigItem,
) (*QueueQuotaConfigItem, error) {
	var err error
	if item, err = validateAndSanitize(item); err != nil {
		return nil, err
	}
	item.Quota = sanitizeQueueQuota(item.Quota)
	item.PrequeueCandidateSize = lo.Ternary(
		item.PrequeueCandidateSize > 0,
		item.PrequeueCandidateSize,
		model.DefaultPrequeueCandidateSize,
	)

	qql := s.q.QueueQuotaLimit
	if _, err := qql.WithContext(ctx).Where(qql.Name.Eq(item.Name)).First(); err == nil {
		return nil, ErrQueueQuotaNameConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	record := &model.QueueQuotaLimit{
		Name:                  item.Name,
		Enabled:               item.Enabled,
		PrequeueCandidateSize: item.PrequeueCandidateSize,
		Quota:                 datatypes.NewJSONType(item.Quota),
	}
	if err := s.q.QueueQuotaLimit.WithContext(ctx).Create(record); err != nil {
		return nil, err
	}

	return &QueueQuotaConfigItem{
		ID:                    record.ID,
		Name:                  record.Name,
		Enabled:               record.Enabled,
		PrequeueCandidateSize: record.PrequeueCandidateSize,
		Quota:                 item.Quota,
	}, nil
}

func (s *PrequeueService) UpdateConfig(
	ctx context.Context,
	item *QueueQuotaConfigItem,
) (*QueueQuotaConfigItem, error) {
	var err error
	if item, err = validateAndSanitize(item); err != nil {
		return nil, err
	}
	item.Quota = sanitizeQueueQuota(item.Quota)
	item.PrequeueCandidateSize = lo.Ternary(
		item.PrequeueCandidateSize > 0,
		item.PrequeueCandidateSize,
		model.DefaultPrequeueCandidateSize,
	)

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
			UpdateSimple(
				qql.Name.Value(item.Name),
				qql.Enabled.Value(item.Enabled),
				qql.PrequeueCandidateSize.Value(item.PrequeueCandidateSize),
				qql.Quota.Value(datatypes.NewJSONType(item.Quota)),
			)
		if err != nil {
			return err
		}
		if info.RowsAffected > 0 {
			updated = &QueueQuotaConfigItem{
				ID:                    item.ID,
				Name:                  item.Name,
				Enabled:               item.Enabled,
				PrequeueCandidateSize: item.PrequeueCandidateSize,
				Quota:                 item.Quota,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *PrequeueService) DeleteQueueQuota(ctx context.Context, id uint) error {
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

func (s *PrequeueService) GetUserResourceUsageSummary(
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

func (s *PrequeueService) getUserResourceUsageSummaryWithResolved(
	ctx context.Context,
	userID,
	accountID uint,
	resolved *ResolvedQueueQuota,
) (*UserResourceUsageSummary, error) {
	jobs, err := s.listUserQueueOccupiedNormalJobs(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return buildUserResourceUsageSummary(resolved, jobs), nil
}

func (s *PrequeueService) listUserQueueOccupiedNormalJobs(
	ctx context.Context,
	userID,
	accountID uint,
) ([]*model.Job, error) {
	j := s.q.Job
	jobs, err := j.WithContext(ctx).Where(
		j.UserID.Eq(userID),
		j.AccountID.Eq(accountID),
		j.Status.In(QueueQuotaOccupiedJobPhases()...),
		j.ScheduleType.Eq(int(model.ScheduleTypeNormal)),
	).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	return jobs, nil
}

func (s *PrequeueService) CountAccountRunningJobs(ctx context.Context, accountID uint) (int, error) {
	j := s.q.Job
	count, err := j.WithContext(ctx).Where(
		j.AccountID.Eq(accountID),
		j.Status.Eq(string(batch.Running)),
	).Count()
	if err != nil {
		return 0, fmt.Errorf("failed to count account running jobs: %w", err)
	}

	return int(count), nil
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
	used := make(map[string]int64, len(summary.Resources))
	for resourceName, item := range summary.Resources {
		usedQty, parseErr := apiresource.ParseQuantity(item.Used)
		if parseErr != nil {
			continue
		}
		used[resourceName] = usedQty.MilliValue()
	}

	for name, valStr := range requestedResources {
		qty, parseErr := apiresource.ParseQuantity(valStr)
		if parseErr != nil {
			continue
		}
		used[name] += qty.MilliValue()
	}

	var details []ResourceLimitDetail
	anyExceeded := false
	for resourceName, limitStr := range resolved.Quota {
		limitQty, parseErr := apiresource.ParseQuantity(limitStr)
		if parseErr != nil {
			continue
		}
		usedMilli := used[resourceName]
		limitMilli := limitQty.MilliValue()
		exceeded := usedMilli > limitMilli
		usedQty := apiresource.NewMilliQuantity(usedMilli, limitQty.Format)
		details = append(details, ResourceLimitDetail{
			Resource: resourceName,
			Used:     usedQty.String(),
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

func (s *PrequeueService) CheckUserResourceLimit(
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

	jobs, err := s.listUserQueueOccupiedNormalJobs(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	return buildResourceLimitCheckResult(resolved, jobs, requestedResources), nil
}

func (s *PrequeueService) GetPrequeueCandidateSize(
	ctx context.Context,
	userID,
	accountID uint,
	queueName string,
) (int, error) {
	resolved, err := s.ResolveQueueQuota(ctx, userID, accountID, queueName)
	if err != nil {
		return model.DefaultPrequeueCandidateSize, err
	}
	return resolved.PrequeueCandidateSize, nil
}
