//nolint:gocritic,gocyclo,lll,mnd,revive // Patrol orchestration intentionally centralizes policy execution and thresholds.
package patrol

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/datatypes"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/storageindex"
	"github.com/raids-lab/crater/pkg/util"
)

const (
	// 占卡检测任务
	TRIGGER_GPU_ANALYSIS_JOB = "trigger-gpu-analysis-job"
	// Billing 基础循环
	TRIGGER_BILLING_BASE_LOOP_JOB = "biling-base-loop"
	// 更新用户空间大小任务
	UPDATE_USER_SPACE_SIZE = "update-user-space-size"
	// 存储告警 AI 分析任务
	ANALYZE_STORAGE_ALERTS         = "analyze-storage-alerts"
	AUTO_SHRINK_STORAGE_EXPANSIONS = "auto-shrink-storage-expansions"
	REFRESH_PUBLIC_STORAGE_INDEX   = "refresh-public-storage-index-baseline"
	REFRESH_USER_STORAGE_INDEX     = "refresh-user-storage-index-daily"

	// AI 分析最大并发数
	defaultMaxConcurrentStorageAnalysis = 3
	autoShrinkToBufferThreshold         = 0.90
	autoShrinkRecoverThreshold          = 0.80
	autoShrinkObservationWindow         = time.Hour
	autoShrinkStageExpanded             = "expanded"
	autoShrinkStageBuffer               = "buffer_reduction"

	storageAnalysisConcurrencyEnv = "CRATER_STORAGE_ANALYSIS_MAX_CONCURRENCY"
)

type GpuAnalysisServiceInterface interface {
	TriggerAllJobsAnalysis(ctx context.Context) (int, error)
}

type BillingServiceInterface interface {
	RunBaseLoopOnce(ctx context.Context) (any, error)
}

// AgentDecision 是 LLM 存储扩容决策的结果，定义在 patrol 包以避免循环依赖。
type AgentDecision struct {
	AllowExpand   bool
	ExpandBytes   int64
	FreezeNewJobs bool
	Reason        string
	DecisionJobID string
}

// StorageAgentFunc 是调用 LLM 进行存储分析的函数签名。
type StorageAgentFunc func(tenantID string) (*AgentDecision, error)

type StorageAgentStartFunc func(ctx context.Context, tenantID string) (string, error)

type StorageAgentAwaitFunc func(ctx context.Context, tenantID string, jobID string) (*AgentDecision, error)

// Clients 包含巡检任务所需的客户端
type Clients struct {
	Client             client.Client
	KubeClient         kubernetes.Interface
	KubeConfig         *rest.Config
	PromClient         monitor.PrometheusInterface
	GpuAnalysisService GpuAnalysisServiceInterface
	BillingService     BillingServiceInterface
	RecordDecision     func(ctx context.Context, jobID string, action string, runErr error)
	StorageAgent       StorageAgentFunc // 注入的 LLM 分析函数，nil 时跳过 AI 分析
	StorageAgentStart  StorageAgentStartFunc
	StorageAgentAwait  StorageAgentAwaitFunc
	StorageIndex       *storageindex.Service
}

func storageAnalysisConcurrency() int {
	raw := os.Getenv(storageAnalysisConcurrencyEnv)
	if raw == "" {
		return defaultMaxConcurrentStorageAnalysis
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		klog.Warningf(
			"storageAnalysisConcurrency: invalid %s=%q, fallback to %d",
			storageAnalysisConcurrencyEnv,
			raw,
			defaultMaxConcurrentStorageAnalysis,
		)
		return defaultMaxConcurrentStorageAnalysis
	}

	return value
}

func NewPatrolClients(
	cli client.Client,
	kubeClient kubernetes.Interface,
	kubeConfig *rest.Config,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService GpuAnalysisServiceInterface,
	billingService BillingServiceInterface,
) *Clients {
	return &Clients{
		Client:             cli,
		KubeClient:         kubeClient,
		KubeConfig:         kubeConfig,
		PromClient:         promClient,
		GpuAnalysisService: gpuAnalysisService,
		BillingService:     billingService,
	}
}

// RunUpdateUserSpaceSize 更新用户空间大小
func RunUpdateUserSpaceSize(_ context.Context, clients *Clients) (any, error) {
	var users []model.User
	db := query.GetDB()
	if err := db.Find(&users).Error; err != nil {
		klog.Errorf("RunUpdateUserSpaceSize: 获取用户列表失败: %v", err)
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}
	klog.Infof("RunUpdateUserSpaceSize: 共有 %d 个用户", len(users))

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	updatedCount := 0
	for _, user := range users {
		if user.Space == "" {
			klog.Warningf("RunUpdateUserSpaceSize: 用户 %s 的空间路径为空，跳过", user.Name)
			continue
		}
		klog.Infof("RunUpdateUserSpaceSize: 正在获取用户 %s 的空间大小，路径: /user/%s", user.Name, user.Space)

		size, err := ceph.GetCephDirectorySize(clients.KubeClient, clients.KubeConfig, "rook-ceph", "/user/"+user.Space, prefixConfig)
		if err != nil {
			klog.Errorf("RunUpdateUserSpaceSize: 获取用户 %s 空间大小失败: %v", user.Name, err)
			continue
		}
		klog.Infof("RunUpdateUserSpaceSize: 用户 %s 空间大小: %d bytes", user.Name, size)

		// 检查是否需要记录历史数据
		var lastHistory model.TenantUsageHistory
		result := db.Where("tenant_id = ?", user.ID).Order("recorded_at DESC").First(&lastHistory)

		// 检查是否需要插入新记录
		needInsert := false
		if result.Error != nil {
			// 没有历史记录，需要插入
			needInsert = true
		} else {
			// 检查字节数差异是否大于 100MB，或者时间超过 1 小时
			byteDiff := size - lastHistory.UsageBytes
			if byteDiff < 0 {
				byteDiff = -byteDiff
			}
			timeDiff := time.Since(lastHistory.RecordedAt)
			if byteDiff > 100*1024*1024 || timeDiff.Hours() > 1 {
				needInsert = true
			}
		}

		// 插入历史记录
		if needInsert {
			history := model.TenantUsageHistory{
				TenantID:   user.ID,
				UsageBytes: size,
				RecordedAt: time.Now(),
			}
			if err := db.Create(&history).Error; err != nil {
				klog.Errorf("RunUpdateUserSpaceSize: 记录用户 %s 空间大小历史失败: %v", user.Name, err)
			} else {
				klog.Infof("RunUpdateUserSpaceSize: 记录用户 %s 空间大小历史成功", user.Name)
			}
		}

		var userSpaceSize model.UserSpaceSize
		result = db.Where("user_id = ?", user.ID).First(&userSpaceSize)
		if result.Error != nil {
			userSpaceSize = model.UserSpaceSize{
				UserID:   user.ID,
				Username: user.Name,
				Size:     size,
			}
			if err := db.Create(&userSpaceSize).Error; err != nil {
				klog.Errorf("RunUpdateUserSpaceSize: 创建用户 %s 空间大小记录失败: %v", user.Name, err)
				continue
			}
			klog.Infof("RunUpdateUserSpaceSize: 创建用户 %s 空间大小记录成功", user.Name)
		} else {
			userSpaceSize.Username = user.Name
			userSpaceSize.Size = size
			if err := db.Save(&userSpaceSize).Error; err != nil {
				klog.Errorf("RunUpdateUserSpaceSize: 更新用户 %s 空间大小记录失败: %v", user.Name, err)
				continue
			}
			klog.Infof("RunUpdateUserSpaceSize: 更新用户 %s 空间大小记录成功", user.Name)
		}

		updatedCount++
	}

	klog.Infof("RunUpdateUserSpaceSize: 完成，共更新了 %d 个用户的空间大小", updatedCount)
	return fmt.Sprintf("更新了 %d 个用户的空间大小", updatedCount), nil
}

// RunAnalyzeStorageAlerts 对超过90%理论配额且未临时扩容的用户并发执行 AI 分析，
// 并自动应用决策（冻结作业 / 临时扩容）。
//
//nolint:funlen // Storage alert analysis keeps candidate selection, LLM decision, and enforcement in one cron action.
func RunAnalyzeStorageAlerts(ctx context.Context, clients *Clients) (any, error) {
	db := query.GetDB()
	maxConcurrentStorageAnalysis := storageAnalysisConcurrency()

	// 查询有空间大小记录的用户，附带配额信息
	type userWithUsage struct {
		ID                 uint   `gorm:"column:id"`
		Name               string `gorm:"column:name"`
		Space              string `gorm:"column:space"`
		SpaceQuota         int64  `gorm:"column:space_quota"`
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
		CurrentSize        int64  `gorm:"column:current_size"`
	}

	var candidates []userWithUsage
	if err := db.Raw(`
		SELECT u.id, u.name, u.space, u.space_quota, u.original_space_quota, uss.size AS current_size
		FROM users u
		JOIN user_space_sizes uss ON uss.user_id = u.id
		WHERE u.deleted_at IS NULL
		  AND u.original_space_quota IS NULL
		  AND u.space_quota > 0
	`).Scan(&candidates).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 过滤出超过 90% 的用户
	var alertUsers []userWithUsage
	for _, u := range candidates {
		if float64(u.CurrentSize)/float64(u.SpaceQuota) >= 0.9 {
			alertUsers = append(alertUsers, u)
		}
	}

	klog.Infof("RunAnalyzeStorageAlerts: %d 个用户超过90%%配额，启动并发 AI 分析（最大并发 %d）",
		len(alertUsers), maxConcurrentStorageAnalysis)

	if len(alertUsers) == 0 {
		return "无超额用户，无需分析", nil
	}

	if clients.StorageAgent == nil {
		if clients.StorageAgentStart == nil || clients.StorageAgentAwait == nil {
			klog.Warningf("RunAnalyzeStorageAlerts: StorageAgent 未注入，跳过 AI 分析")
			return "StorageAgent 未配置", nil
		}
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	applyDecision := func(u userWithUsage, decision *AgentDecision) {
		klog.Infof("RunAnalyzeStorageAlerts: 用户 %s 决策: allow_expand=%v expand_bytes=%d freeze=%v reason=%s",
			u.Name, decision.AllowExpand, decision.ExpandBytes, decision.FreezeNewJobs, decision.Reason)

		recordDecision := func(action string, runErr error) {
			if clients.RecordDecision != nil && decision.DecisionJobID != "" {
				clients.RecordDecision(ctx, decision.DecisionJobID, action, runErr)
			}
		}
		if decision.AllowExpand && decision.ExpandBytes > 0 {
			newQuota := u.SpaceQuota + decision.ExpandBytes
			if err := db.Exec(
				"UPDATE users SET original_space_quota = space_quota, space_quota = ?, jobs_frozen = ? WHERE id = ? AND deleted_at IS NULL",
				newQuota, decision.FreezeNewJobs, u.ID,
			).Error; err != nil {
				klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s 写入扩容失败: %v", u.Name, err)
				recordDecision("expand_failed", err)
				return
			}
			if u.Space != "" {
				if cephErr := ceph.SetCephDirectoryQuota(
					clients.KubeClient, clients.KubeConfig, "rook-ceph",
					"/user/"+u.Space, prefixConfig, newQuota,
				); cephErr != nil {
					klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s Ceph 配额同步失败: %v", u.Name, cephErr)
				}
			}
			klog.Infof("RunAnalyzeStorageAlerts: 用户 %s 已临时扩容至 %d bytes", u.Name, newQuota)
			recordDecision("expand", nil)
			return
		}

		if decision.FreezeNewJobs {
			if err := db.Exec(
				"UPDATE users SET jobs_frozen = true WHERE id = ? AND deleted_at IS NULL", u.ID,
			).Error; err != nil {
				klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s 设置 jobs_frozen 失败: %v", u.Name, err)
				recordDecision("freeze_failed", err)
				return
			}

			recordDecision("freeze", nil)
			klog.Infof("RunAnalyzeStorageAlerts: 用户 %s 已冻结新作业创建", u.Name)
			return
		}

		recordDecision("observe", nil)
	}

	if clients.StorageAgentStart != nil && clients.StorageAgentAwait != nil {
		type pendingDecision struct {
			user  userWithUsage
			jobID string
		}

		sem := make(chan struct{}, maxConcurrentStorageAnalysis)
		var wg sync.WaitGroup
		var mu sync.Mutex
		pending := make([]pendingDecision, 0, len(alertUsers))

		for _, candidate := range alertUsers {
			u := candidate
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				klog.Infof("RunAnalyzeStorageAlerts: 开始派发用户 %s 的异步 AI 分析任务", u.Name)
				jobID, err := clients.StorageAgentStart(ctx, u.Name)
				if err != nil {
					klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s AI 分析任务派发失败: %v", u.Name, err)
					return
				}

				klog.Infof("RunAnalyzeStorageAlerts: 用户 %s AI 分析任务已派发 job_id=%s", u.Name, jobID)
				mu.Lock()
				pending = append(pending, pendingDecision{user: u, jobID: jobID})
				mu.Unlock()
			}()
		}
		wg.Wait()

		if len(pending) == 0 {
			return "没有成功派发任何 AI 分析任务", nil
		}

		klog.Infof(
			"RunAnalyzeStorageAlerts: %d 个用户的 AI 分析任务已派发完成，开始并发等待结果（最大并发 %d）",
			len(pending),
			maxConcurrentStorageAnalysis,
		)

		wg = sync.WaitGroup{}
		for _, item := range pending {
			pendingItem := item
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				decision, err := clients.StorageAgentAwait(ctx, pendingItem.user.Name, pendingItem.jobID)
				if err != nil {
					klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s 等待 AI 分析结果失败: %v", pendingItem.user.Name, err)
					return
				}

				applyDecision(pendingItem.user, decision)
			}()
		}
		wg.Wait()
		return fmt.Sprintf("分析完成，共处理 %d 个超额用户", len(alertUsers)), nil
	}

	// 使用 channel 实现有界并发
	sem := make(chan struct{}, maxConcurrentStorageAnalysis)
	var wg sync.WaitGroup
	for _, candidate := range alertUsers {
		u := candidate
		wg.Add(1)
		sem <- struct{}{} // 占槽（满时阻塞）
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // 释放槽

			klog.Infof("RunAnalyzeStorageAlerts: 开始分析用户 %s (size=%d quota=%d %.1f%%)",
				u.Name, u.CurrentSize, u.SpaceQuota, float64(u.CurrentSize)/float64(u.SpaceQuota)*100)

			decision, err := clients.StorageAgent(u.Name)
			if err != nil {
				klog.Errorf("RunAnalyzeStorageAlerts: 用户 %s AI 分析失败: %v", u.Name, err)
				return
			}

			applyDecision(u, decision)
		}()
	}

	wg.Wait()
	return fmt.Sprintf("分析完成，共处理 %d 个超额用户", len(alertUsers)), nil
}

// GetPatrolFunc 根据作业名称返回对应的巡检函数
// RunAutoShrinkStorageExpansions automatically recovers temporary storage expansions
// once a user's current usage has fallen below a conservative percentage of the
// original theoretical quota.
func RunAutoShrinkStorageExpansions(ctx context.Context, clients *Clients) (any, error) {
	db := query.GetDB()

	type expandedUser struct {
		ID                   uint       `gorm:"column:id"`
		Name                 string     `gorm:"column:name"`
		Space                string     `gorm:"column:space"`
		SpaceQuota           int64      `gorm:"column:space_quota"`
		OriginalSpaceQuota   int64      `gorm:"column:original_space_quota"`
		CurrentSize          int64      `gorm:"column:current_size"`
		ShrinkStage          string     `gorm:"column:shrink_stage"`
		ShrinkStageUpdatedAt *time.Time `gorm:"column:shrink_stage_updated_at"`
	}

	var users []expandedUser
	if err := db.Raw(`
		SELECT u.id, u.name, u.space, u.space_quota, u.original_space_quota, uss.size AS current_size,
		       u.shrink_stage, u.shrink_stage_updated_at
		FROM users u
		JOIN user_space_sizes uss ON uss.user_id = u.id
		WHERE u.deleted_at IS NULL
		  AND u.original_space_quota IS NOT NULL
	`).Scan(&users).Error; err != nil {
		return nil, fmt.Errorf("query expanded users failed: %w", err)
	}

	if len(users) == 0 {
		return "当前没有处于临时扩容状态的用户，无需执行自动缩容。", nil
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	shrunk := 0
	skipped := 0
	for _, user := range users {
		if user.OriginalSpaceQuota <= 0 {
			skipped++
			continue
		}

		usageRatio := float64(user.CurrentSize) / float64(user.OriginalSpaceQuota)
		stage := user.ShrinkStage
		if stage == "" {
			stage = autoShrinkStageExpanded
		}

		switch stage {
		case autoShrinkStageExpanded:
			if usageRatio >= autoShrinkToBufferThreshold {
				skipped++
				continue
			}

			bufferQuota := calculateShrinkBufferQuota(user.OriginalSpaceQuota, user.SpaceQuota)
			if bufferQuota <= user.OriginalSpaceQuota {
				bufferQuota = user.OriginalSpaceQuota
			}

			if err := db.Exec(
				"UPDATE users SET space_quota = ?, shrink_stage = ?, shrink_stage_updated_at = NOW() WHERE id = ? AND deleted_at IS NULL",
				bufferQuota, autoShrinkStageBuffer, user.ID,
			).Error; err != nil {
				klog.Errorf("RunAutoShrinkStorageExpansions: user=%s buffer shrink failed: %v", user.Name, err)
				skipped++
				continue
			}

			if user.Space != "" {
				if cephErr := ceph.SetCephDirectoryQuota(
					clients.KubeClient,
					clients.KubeConfig,
					"rook-ceph",
					"/user/"+user.Space,
					prefixConfig,
					bufferQuota,
				); cephErr != nil {
					klog.Errorf("RunAutoShrinkStorageExpansions: user=%s ceph buffer shrink failed: %v", user.Name, cephErr)
					skipped++
					continue
				}
			}

			shrunk++
			klog.Infof(
				"RunAutoShrinkStorageExpansions: user=%s moved to buffer stage quota=%d current_size=%d ratio=%.2f",
				user.Name,
				bufferQuota,
				user.CurrentSize,
				usageRatio,
			)
		case autoShrinkStageBuffer:
			if user.ShrinkStageUpdatedAt == nil || time.Since(*user.ShrinkStageUpdatedAt) < autoShrinkObservationWindow {
				skipped++
				continue
			}
			if usageRatio >= autoShrinkRecoverThreshold {
				skipped++
				continue
			}

			if err := db.Exec(
				"UPDATE users SET space_quota = ?, original_space_quota = NULL, jobs_frozen = false, shrink_stage = NULL, shrink_stage_updated_at = NULL WHERE id = ? AND deleted_at IS NULL",
				user.OriginalSpaceQuota, user.ID,
			).Error; err != nil {
				klog.Errorf("RunAutoShrinkStorageExpansions: user=%s final shrink failed: %v", user.Name, err)
				skipped++
				continue
			}

			if user.Space != "" {
				if cephErr := ceph.SetCephDirectoryQuota(
					clients.KubeClient,
					clients.KubeConfig,
					"rook-ceph",
					"/user/"+user.Space,
					prefixConfig,
					user.OriginalSpaceQuota,
				); cephErr != nil {
					klog.Errorf("RunAutoShrinkStorageExpansions: user=%s ceph final shrink failed: %v", user.Name, cephErr)
					skipped++
					continue
				}
			}

			shrunk++
			klog.Infof(
				"RunAutoShrinkStorageExpansions: user=%s fully restored quota=%d current_size=%d ratio=%.2f",
				user.Name,
				user.OriginalSpaceQuota,
				user.CurrentSize,
				usageRatio,
			)
		default:
			skipped++
		}
	}

	return fmt.Sprintf("自动缩容扫描完成：已处理 %d 个用户，跳过 %d 个用户。", shrunk, skipped), nil
}

func calculateShrinkBufferQuota(originalQuota, currentQuota int64) int64 {
	if currentQuota <= originalQuota {
		return originalQuota
	}

	delta := currentQuota - originalQuota
	bufferQuota := originalQuota + delta/2
	if bufferQuota <= originalQuota {
		return originalQuota
	}
	return bufferQuota
}

func RunRefreshPublicStorageIndexBaseline(ctx context.Context, clients *Clients) (any, error) {
	if clients.StorageIndex == nil {
		return nil, fmt.Errorf("storage index service is not initialized in patrol clients")
	}

	job, err := clients.StorageIndex.RefreshPublicBaseline(ctx)
	if err != nil {
		return nil, fmt.Errorf("refresh public storage index baseline failed: %w", err)
	}

	klog.Infof("RunRefreshPublicStorageIndexBaseline: scan_id=%s status=%s redundancy=%d",
		job.ScanID, job.Status, job.RedundancyCount)

	return map[string]any{
		"scan_id":          job.ScanID,
		"workspace_type":   job.WorkspaceType,
		"workspace_name":   job.WorkspaceName,
		"status":           job.Status,
		"entry_count":      job.EntryCount,
		"redundancy_count": job.RedundancyCount,
	}, nil
}

func RunRefreshUserStorageIndexDaily(ctx context.Context, clients *Clients) (any, error) {
	if clients.StorageIndex == nil {
		return nil, fmt.Errorf("storage index service is not initialized in patrol clients")
	}

	result, err := clients.StorageIndex.RefreshAllUserWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("refresh user storage index daily failed: %w", err)
	}

	klog.Infof("RunRefreshUserStorageIndexDaily: total=%v success=%v failed=%v",
		result["total"], result["success"], result["failed"])

	return result, nil
}

func GetPatrolFunc(jobName string, clients *Clients, jobConfig datatypes.JSON) (util.AnyFunc, error) {
	var f util.AnyFunc
	switch jobName {
	case TRIGGER_GPU_ANALYSIS_JOB:
		// TRIGGER_GPU_ANALYSIS_JOB 不需要 req 参数，但为了保持一致性，仍然定义了结构体
		req := &TriggerGpuAnalysisRequest{}
		if len(jobConfig) > 0 {
			if err := json.Unmarshal(jobConfig, req); err != nil {
				return nil, err
			}
		}
		f = func(ctx context.Context) (any, error) {
			return RunTriggerGpuAnalysis(ctx, clients)
		}
	case TRIGGER_BILLING_BASE_LOOP_JOB:
		f = func(ctx context.Context) (any, error) {
			return RunTriggerBillingBaseLoop(ctx, clients)
		}
	case UPDATE_USER_SPACE_SIZE:
		f = func(ctx context.Context) (any, error) {
			return RunUpdateUserSpaceSize(ctx, clients)
		}
	case ANALYZE_STORAGE_ALERTS:
		f = func(ctx context.Context) (any, error) {
			return RunAnalyzeStorageAlerts(ctx, clients)
		}
	case AUTO_SHRINK_STORAGE_EXPANSIONS:
		f = func(ctx context.Context) (any, error) {
			return RunAutoShrinkStorageExpansions(ctx, clients)
		}
	case REFRESH_PUBLIC_STORAGE_INDEX:
		f = func(ctx context.Context) (any, error) {
			return RunRefreshPublicStorageIndexBaseline(ctx, clients)
		}
	case REFRESH_USER_STORAGE_INDEX:
		f = func(ctx context.Context) (any, error) {
			return RunRefreshUserStorageIndexDaily(ctx, clients)
		}
	default:
		return nil, fmt.Errorf("unsupported patrol job name: %s", jobName)
	}
	return f, nil
}
