//nolint:gocritic,gocyclo,mnd,unparam // Engine orchestration intentionally centralizes snapshot collection and decision execution.
package storagegovernance

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/llm"
	"github.com/raids-lab/crater/pkg/monitor"
)

type DecisionRequest struct {
	Username      string
	Source        model.StorageDecisionSource
	TriggerReason string
}

type StoredDecisionStatus struct {
	Status             string                   `json:"status"`
	Result             *llm.LLMDecisionResponse `json:"result,omitempty"`
	ErrorMsg           string                   `json:"error,omitempty"`
	ConstraintAdjusted bool                     `json:"constraint_adjusted,omitempty"`
	ConstraintBlocked  bool                     `json:"constraint_blocked,omitempty"`
}

type Engine struct {
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config
	promClient monitor.PrometheusInterface
	config     ConstraintConfig
}

func NewEngine(
	kubeClient kubernetes.Interface,
	kubeConfig *rest.Config,
	promClient monitor.PrometheusInterface,
	cfg ConstraintConfig,
) *Engine {
	if cfg.PolicyVersion == "" {
		cfg = DefaultConstraintConfig()
	}
	return &Engine{
		kubeClient: kubeClient,
		kubeConfig: kubeConfig,
		promClient: promClient,
		config:     cfg,
	}
}

func (e *Engine) StartAsyncDecision(ctx context.Context, req DecisionRequest) (string, error) {
	jobID, err := e.createPendingRecord(ctx, req)
	if err != nil {
		return "", err
	}

	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, _ = e.RunDecision(runCtx, jobID, req)
	}()

	return jobID, nil
}

func (e *Engine) DecideAndRecord(ctx context.Context, req DecisionRequest) (*llm.LLMDecisionResponse, string, error) {
	jobID, err := e.createPendingRecord(ctx, req)
	if err != nil {
		return nil, "", err
	}

	decision, err := e.RunDecision(ctx, jobID, req)
	if err != nil {
		return nil, jobID, err
	}

	return decision, jobID, nil
}

func (e *Engine) RunDecision(ctx context.Context, jobID string, req DecisionRequest) (*llm.LLMDecisionResponse, error) {
	startedAt := time.Now()
	_ = query.GetDB().WithContext(ctx).
		Model(&model.StorageDecisionRecord{}).
		Where("job_id = ?", jobID).
		Updates(map[string]any{
			"status":     model.StorageDecisionStatusRunning,
			"started_at": startedAt,
		}).Error

	snapshot, err := e.BuildSnapshot(ctx, req.Username)
	if err != nil {
		e.markError(ctx, jobID, startedAt, err)
		return nil, err
	}

	var rawDecision *llm.LLMDecisionResponse
	switch llm.GetStorageDecisionMode(ctx) {
	case llm.StorageDecisionModeDirect:
		snapshotJSON, marshalErr := json.Marshal(snapshot)
		if marshalErr != nil {
			e.persistFailure(ctx, jobID, snapshot, startedAt, marshalErr)
			return nil, marshalErr
		}
		rawDecision, err = llm.AskDirectDecision(ctx, string(snapshotJSON))
	default:
		rawDecision, err = llm.AskAgentForDecision(e.kubeClient, e.kubeConfig, req.Username, e.promClient)
	}
	if err != nil {
		e.persistFailure(ctx, jobID, snapshot, startedAt, err)
		return nil, err
	}

	finalDecision, evaluation := ApplySafetyConstraints(snapshot, *rawDecision, e.config, time.Now())
	if err := e.persistSuccess(ctx, jobID, req, snapshot, *rawDecision, finalDecision, evaluation, startedAt); err != nil {
		return nil, err
	}

	return &finalDecision, nil
}

func (e *Engine) BuildSnapshot(ctx context.Context, username string) (DecisionSnapshot, error) {
	var userRow struct {
		ID                 uint   `gorm:"column:id"`
		Name               string `gorm:"column:name"`
		Space              string `gorm:"column:space"`
		SpaceQuota         int64  `gorm:"column:space_quota"`
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
		JobsFrozen         bool   `gorm:"column:jobs_frozen"`
		ShrinkStage        string `gorm:"column:shrink_stage"`
	}

	if err := query.GetDB().WithContext(ctx).Raw(
		"SELECT id, name, space, space_quota, original_space_quota, jobs_frozen, shrink_stage FROM users WHERE name = ? AND deleted_at IS NULL",
		username,
	).Scan(&userRow).Error; err != nil {
		return DecisionSnapshot{}, fmt.Errorf("query user snapshot failed: %w", err)
	}
	if userRow.ID == 0 {
		return DecisionSnapshot{}, fmt.Errorf("user %s not found", username)
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	currentUsage, err := ceph.GetCephDirectorySize(
		e.kubeClient,
		e.kubeConfig,
		"rook-ceph",
		"/user/"+userRow.Space,
		prefixConfig,
	)
	if err != nil {
		currentUsage = ceph.UnknownSizeBytes
	}

	theoreticalQuota := userRow.SpaceQuota
	if userRow.OriginalSpaceQuota != nil {
		theoreticalQuota = *userRow.OriginalSpaceQuota
	}

	totalCapacity, usedCapacity, err := ceph.GetCraterStorageCapacity(e.kubeClient, e.kubeConfig, "rook-ceph")
	if err != nil {
		return DecisionSnapshot{}, fmt.Errorf("get platform capacity failed: %w", err)
	}

	runtimeFeatures, err := e.collectTenantRuntimeFeatures(ctx, username)
	if err != nil {
		return DecisionSnapshot{}, fmt.Errorf("collect tenant runtime features failed: %w", err)
	}

	var historyRows []model.TenantUsageHistory
	if err := query.GetDB().WithContext(ctx).
		Where("tenant_id = ?", userRow.ID).
		Order("recorded_at desc").
		Limit(10).
		Find(&historyRows).Error; err != nil {
		return DecisionSnapshot{}, fmt.Errorf("query usage history failed: %w", err)
	}

	recentHistory := make([]UsageHistoryPoint, 0, len(historyRows))
	for _, row := range historyRows {
		recentHistory = append(recentHistory, UsageHistoryPoint{
			RecordedAt: row.RecordedAt,
			UsageBytes: row.UsageBytes,
		})
	}
	slices.Reverse(recentHistory)

	var growthRate *float64
	if len(recentHistory) >= 2 {
		first := recentHistory[0]
		last := recentHistory[len(recentHistory)-1]
		hours := last.RecordedAt.Sub(first.RecordedAt).Hours()
		if hours > 0 {
			value := float64(last.UsageBytes-first.UsageBytes) / hours
			growthRate = &value
		}
	}

	var usageRatio float64
	if theoreticalQuota > 0 && currentUsage >= 0 {
		usageRatio = float64(currentUsage) / float64(theoreticalQuota)
	}
	availableCapacity := ceph.AvailableBytes(totalCapacity, usedCapacity)

	var lastExpandRecord model.StorageDecisionRecord
	var lastExpandAt *time.Time
	appliedExpandActions := []string{
		"expand",
		"manual_expand",
		"manual_expand_and_freeze",
	}
	if err := query.GetDB().WithContext(ctx).
		Where(
			"username = ? AND status = ? AND applied_action IN ?",
			username,
			model.StorageDecisionStatusDone,
			appliedExpandActions,
		).
		Order("updated_at desc").
		First(&lastExpandRecord).Error; err == nil {
		lastExpandAt = &lastExpandRecord.UpdatedAt
	}

	return DecisionSnapshot{
		Username:               username,
		UserID:                 userRow.ID,
		CurrentUsageBytes:      currentUsage,
		CurrentQuotaBytes:      userRow.SpaceQuota,
		TheoreticalQuotaBytes:  theoreticalQuota,
		UsageRatio:             usageRatio,
		GrowthRateBytesPerHour: growthRate,
		PlatformTotalBytes:     totalCapacity,
		PlatformUsedBytes:      usedCapacity,
		PlatformAvailableBytes: availableCapacity,
		IsCurrentlyExpanded:    userRow.OriginalSpaceQuota != nil,
		JobsFrozen:             userRow.JobsFrozen,
		ShrinkStage:            userRow.ShrinkStage,
		ActivePodCount:         runtimeFeatures.ActivePodCount,
		ActiveGPUPodCount:      runtimeFeatures.ActiveGPUPodCount,
		ActiveGPURequestTotal:  runtimeFeatures.ActiveGPURequestTotal,
		ActiveCPURequestCores:  runtimeFeatures.ActiveCPURequestCores,
		ActiveMemoryRequestMB:  runtimeFeatures.ActiveMemoryRequestMB,
		RealtimeCPUCores:       runtimeFeatures.RealtimeCPUCores,
		RealtimeMemoryMB:       runtimeFeatures.RealtimeMemoryMB,
		RealtimeGPUUtilPercent: runtimeFeatures.RealtimeGPUUtilPercent,
		RealtimeGPUMemoryMB:    runtimeFeatures.RealtimeGPUMemoryMB,
		GPUDataAvailable:       runtimeFeatures.GPUDataAvailable,
		MaxGPUHistoryPercent:   runtimeFeatures.MaxGPUHistoryPercent,
		LastExpandAt:           lastExpandAt,
		RecentHistory:          recentHistory,
	}, nil
}

type tenantRuntimeFeatures struct {
	ActivePodCount         int
	ActiveGPUPodCount      int
	ActiveGPURequestTotal  int
	ActiveCPURequestCores  float64
	ActiveMemoryRequestMB  float64
	RealtimeCPUCores       float64
	RealtimeMemoryMB       float64
	RealtimeGPUUtilPercent float64
	RealtimeGPUMemoryMB    float64
	GPUDataAvailable       bool
	MaxGPUHistoryPercent   float64
}

func (e *Engine) collectTenantRuntimeFeatures(ctx context.Context, username string) (*tenantRuntimeFeatures, error) {
	features := &tenantRuntimeFeatures{}
	jobNamespace := config.GetConfig().Namespaces.Job

	pods, err := e.kubeClient.CoreV1().Pods(jobNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "crater.raids.io/task-user=" + username,
	})
	if err != nil {
		return nil, err
	}

	gpuUtilSum := 0.0
	gpuUtilCount := 0
	gpuMemSum := 0.0

	for _, pod := range pods.Items {
		phase := string(pod.Status.Phase)
		if phase != "Running" && phase != "Pending" {
			continue
		}

		features.ActivePodCount++
		podGPURequests := 0
		for _, container := range pod.Spec.Containers {
			for resourceName, quantity := range container.Resources.Requests {
				if resourceName == corev1.ResourceCPU {
					features.ActiveCPURequestCores += float64(quantity.MilliValue()) / 1000
				}
				if resourceName == corev1.ResourceMemory {
					features.ActiveMemoryRequestMB += float64(quantity.Value()) / 1024 / 1024
				}
				if strings.Contains(string(resourceName), "nvidia.com/") {
					podGPURequests += int(quantity.Value())
					features.ActiveGPURequestTotal += int(quantity.Value())
				}
			}
		}
		if podGPURequests > 0 {
			features.ActiveGPUPodCount++
		}

		if e.promClient != nil {
			runtimeMetrics, err := e.queryPodRealtimeMetrics(pod.Name)
			if err == nil {
				features.RealtimeCPUCores += runtimeMetrics.CPUCores
				features.RealtimeMemoryMB += runtimeMetrics.MemoryMB
				if runtimeMetrics.GPUDataAvailable {
					features.GPUDataAvailable = true
					gpuUtilSum += runtimeMetrics.GPUUtilPercent
					gpuUtilCount++
					gpuMemSum += runtimeMetrics.GPUMemoryMB
				}
			}

			if podGPURequests > 0 {
				historyMetrics, err := e.queryPodGPUHistory(pod.Name, 24)
				if err == nil && historyMetrics.MaxUtil > features.MaxGPUHistoryPercent {
					features.MaxGPUHistoryPercent = historyMetrics.MaxUtil
				}
			}
		}
	}

	if gpuUtilCount > 0 {
		features.RealtimeGPUUtilPercent = gpuUtilSum / float64(gpuUtilCount)
		features.RealtimeGPUMemoryMB = gpuMemSum / float64(gpuUtilCount)
	}

	return features, nil
}

type podRealtimeMetrics struct {
	CPUCores         float64
	MemoryMB         float64
	GPUUtilPercent   float64
	GPUMemoryMB      float64
	GPUDataAvailable bool
}

func (e *Engine) queryPodRealtimeMetrics(podName string) (*podRealtimeMetrics, error) {
	jobNamespace := config.GetConfig().Namespaces.Job
	result := &podRealtimeMetrics{}

	if v, ok, err := e.promClient.QueryInstant(
		fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=%q,container!=""}[5m]))`, podName),
	); err == nil && ok {
		result.CPUCores = v
	}
	if v, ok, err := e.promClient.QueryInstant(
		fmt.Sprintf(`sum(container_memory_usage_bytes{pod=%q,container!=""})`, podName),
	); err == nil && ok {
		result.MemoryMB = v / 1024 / 1024
	}

	for _, query := range []string{
		fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q})`, jobNamespace, podName),
		fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{pod=%q})`, podName),
	} {
		if v, ok, err := e.promClient.QueryInstant(query); err == nil && ok {
			result.GPUUtilPercent = v
			result.GPUDataAvailable = true
			break
		}
	}

	for _, query := range []string{
		fmt.Sprintf(`avg(DCGM_FI_DEV_FB_USED{namespace=%q,pod=%q})`, jobNamespace, podName),
		fmt.Sprintf(`avg(DCGM_FI_DEV_FB_USED{pod=%q})`, podName),
	} {
		if v, ok, err := e.promClient.QueryInstant(query); err == nil && ok {
			result.GPUMemoryMB = v
			break
		}
	}

	return result, nil
}

type podGPUHistory struct {
	MaxUtil float64
}

func (e *Engine) queryPodGPUHistory(podName string, durationHours float64) (*podGPUHistory, error) {
	jobNamespace := config.GetConfig().Namespaces.Job
	duration := fmt.Sprintf("%.0fh", durationHours)
	if durationHours < 1 {
		duration = fmt.Sprintf("%.0fm", durationHours*60)
	}

	result := &podGPUHistory{}
	for _, query := range []string{
		fmt.Sprintf(`max_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[%s])`, jobNamespace, podName, duration),
		fmt.Sprintf(`max_over_time(DCGM_FI_DEV_GPU_UTIL{pod=%q}[%s])`, podName, duration),
	} {
		if v, ok, err := e.promClient.QueryInstant(query); err == nil && ok {
			result.MaxUtil = v
			return result, nil
		}
	}

	return result, nil
}

func GetDecisionStatus(ctx context.Context, jobID string) (*StoredDecisionStatus, error) {
	var record model.StorageDecisionRecord
	if err := query.GetDB().WithContext(ctx).Where("job_id = ?", jobID).First(&record).Error; err != nil {
		return nil, err
	}

	status := &StoredDecisionStatus{
		Status:             string(record.Status),
		ErrorMsg:           record.ErrorMessage,
		ConstraintAdjusted: record.ConstraintAdjusted,
		ConstraintBlocked:  record.ConstraintBlocked,
	}

	if len(record.FinalDecision) > 0 {
		var decision llm.LLMDecisionResponse
		if err := json.Unmarshal(record.FinalDecision, &decision); err == nil {
			status.Result = &decision
		}
	}

	return status, nil
}

func MarkDecisionExecution(ctx context.Context, jobID, action string, runErr error) error {
	updates := map[string]any{
		"applied_action": action,
	}
	if runErr != nil {
		updates["error_message"] = runErr.Error()
	} else {
		updates["error_message"] = ""
	}

	return query.GetDB().WithContext(ctx).
		Model(&model.StorageDecisionRecord{}).
		Where("job_id = ?", jobID).
		Updates(updates).Error
}

func (e *Engine) createPendingRecord(ctx context.Context, req DecisionRequest) (string, error) {
	var userRow struct {
		ID uint `gorm:"column:id"`
	}
	if err := query.GetDB().WithContext(ctx).Raw(
		"SELECT id FROM users WHERE name = ? AND deleted_at IS NULL",
		req.Username,
	).Scan(&userRow).Error; err != nil {
		return "", err
	}
	if userRow.ID == 0 {
		return "", fmt.Errorf("user %s not found", req.Username)
	}

	jobID := NewJobID()
	record := model.StorageDecisionRecord{
		JobID:         jobID,
		UserID:        userRow.ID,
		Username:      req.Username,
		Source:        req.Source,
		Status:        model.StorageDecisionStatusPending,
		TriggerReason: req.TriggerReason,
		StartedAt:     nil,
	}
	if err := query.GetDB().WithContext(ctx).Create(&record).Error; err != nil {
		return "", err
	}
	return jobID, nil
}

func (e *Engine) persistFailure(
	ctx context.Context,
	jobID string,
	snapshot DecisionSnapshot,
	startedAt time.Time,
	runErr error,
) {
	updates := map[string]any{
		"status":        model.StorageDecisionStatusError,
		"error_message": runErr.Error(),
		"finished_at":   time.Now(),
		"latency_ms":    time.Since(startedAt).Milliseconds(),
	}
	if data, err := json.Marshal(snapshot); err == nil {
		updates["snapshot"] = datatypes.JSON(data)
	}
	_ = query.GetDB().WithContext(ctx).
		Model(&model.StorageDecisionRecord{}).
		Where("job_id = ?", jobID).
		Updates(updates).Error
}

func (e *Engine) markError(ctx context.Context, jobID string, startedAt time.Time, runErr error) {
	_ = query.GetDB().WithContext(ctx).
		Model(&model.StorageDecisionRecord{}).
		Where("job_id = ?", jobID).
		Updates(map[string]any{
			"status":        model.StorageDecisionStatusError,
			"error_message": runErr.Error(),
			"finished_at":   time.Now(),
			"latency_ms":    time.Since(startedAt).Milliseconds(),
		}).Error
}

func (e *Engine) persistSuccess(
	ctx context.Context,
	jobID string,
	req DecisionRequest,
	snapshot DecisionSnapshot,
	rawDecision llm.LLMDecisionResponse,
	finalDecision llm.LLMDecisionResponse,
	evaluation ConstraintEvaluation,
	startedAt time.Time,
) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	rawDecisionJSON, err := json.Marshal(rawDecision)
	if err != nil {
		return err
	}
	finalDecisionJSON, err := json.Marshal(finalDecision)
	if err != nil {
		return err
	}
	evaluationJSON, err := json.Marshal(evaluation)
	if err != nil {
		return err
	}

	return query.GetDB().WithContext(ctx).
		Model(&model.StorageDecisionRecord{}).
		Where("job_id = ?", jobID).
		Updates(map[string]any{
			"user_id":               snapshot.UserID,
			"username":              req.Username,
			"source":                req.Source,
			"status":                model.StorageDecisionStatusDone,
			"snapshot":              datatypes.JSON(snapshotJSON),
			"raw_decision":          datatypes.JSON(rawDecisionJSON),
			"final_decision":        datatypes.JSON(finalDecisionJSON),
			"constraint_result":     datatypes.JSON(evaluationJSON),
			"raw_allow_expand":      rawDecision.AllowExpand,
			"raw_expand_bytes":      rawDecision.ExpandBytes,
			"raw_freeze_new_jobs":   rawDecision.FreezeNewJobs,
			"final_allow_expand":    finalDecision.AllowExpand,
			"final_expand_bytes":    finalDecision.ExpandBytes,
			"final_freeze_new_jobs": finalDecision.FreezeNewJobs,
			"constraint_adjusted":   evaluation.Adjusted,
			"constraint_blocked":    evaluation.Blocked,
			"constraint_version":    evaluation.PolicyVersion,
			"applied_action":        "",
			"error_message":         "",
			"finished_at":           time.Now(),
			"latency_ms":            time.Since(startedAt).Milliseconds(),
		}).Error
}

func NewJobID() string {
	return fmt.Sprintf("sd-%d", time.Now().UnixNano())
}
