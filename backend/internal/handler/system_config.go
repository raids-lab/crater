package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"strings"

	"github.com/raids-lab/crater/dao/query"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/prequeuewatcher"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewSystemConfigMgr)
}

type SystemConfigMgr struct {
	name           string
	service        *service.ConfigService
	billingService *service.BillingService
	cronjobManager *cronjob.CronJobManager
	watcher        *prequeuewatcher.PrequeueWatcher
}

func NewSystemConfigMgr(conf *RegisterConfig) Manager {
	return &SystemConfigMgr{
		name:           "system-config",
		service:        conf.ConfigService,
		billingService: conf.BillingService,
		cronjobManager: conf.CronJobManager,
		watcher:        conf.PrequeueWatcher,
	}
}

func (mgr *SystemConfigMgr) GetName() string                   { return mgr.name }
func (mgr *SystemConfigMgr) RegisterPublic(_ *gin.RouterGroup) {}
func (mgr *SystemConfigMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/billing", mgr.GetBillingStatus)
}

func (mgr *SystemConfigMgr) RegisterAdmin(g *gin.RouterGroup) {
	// 路由组: /v1/admin/system-config
	g.GET("/llm", mgr.GetLLMConfig)
	g.PUT("/llm", mgr.UpdateLLMConfig)
	// 新增：重置 LLM 配置
	g.DELETE("/llm", mgr.ResetLLMConfig)

	g.GET("/gpu-analysis", mgr.GetGpuAnalysisStatus)
	g.PUT("/gpu-analysis", mgr.SetGpuAnalysisStatus)
	g.GET("/prequeue", mgr.GetPrequeueConfig)
	g.PUT("/prequeue", mgr.UpdatePrequeueConfig)

	g.GET("/billing", mgr.GetBillingStatus)
	g.PUT("/billing", mgr.SetBillingStatus)
	g.POST("/billing/reconcile", mgr.TriggerBillingBaseLoop)
	g.POST("/billing/reset-all", mgr.ResetAllBillingBalances)
	g.POST("/billing/extra-balance-all", mgr.GrantAllUsersExtraBalance)
}

// --- DTOs ---

type LLMConfigResp struct {
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey"`
	ModelName string `json:"modelName"`
}

type UpdateLLMConfigReq struct {
	BaseURL   string `json:"baseUrl" binding:"required"`
	APIKey    string `json:"apiKey"`
	ModelName string `json:"modelName" binding:"required"`
	Validate  bool   `json:"validate"` // 是否立即校验连接
}

type GpuAnalysisStatusResp struct {
	Enabled bool `json:"enabled"`
}

type SetGpuAnalysisStatusReq struct {
	Enable bool `json:"enable"`
}

type PrequeueConfigResp struct {
	BackfillEnabled                  bool  `json:"backfillEnabled"`
	QueueQuotaEnabled                bool  `json:"queueQuotaEnabled"`
	NormalJobWaitingToleranceSeconds int64 `json:"normalJobWaitingToleranceSeconds"`
	ActivateTickerIntervalSeconds    int64 `json:"activateTickerIntervalSeconds"`
	MaxTotalActivationsPerRound      int64 `json:"maxTotalActivationsPerRound"`
}

type UpdatePrequeueConfigReq struct {
	BackfillEnabled                  *bool  `json:"backfillEnabled" binding:"required"`
	QueueQuotaEnabled                *bool  `json:"queueQuotaEnabled" binding:"required"`
	NormalJobWaitingToleranceSeconds *int64 `json:"normalJobWaitingToleranceSeconds" binding:"required,gt=0"`
	ActivateTickerIntervalSeconds    *int64 `json:"activateTickerIntervalSeconds" binding:"required,gt=0"`
	MaxTotalActivationsPerRound      *int64 `json:"maxTotalActivationsPerRound" binding:"required,gt=0"`
}

type BillingStatusResp struct {
	FeatureEnabled                    bool    `json:"featureEnabled"`
	Active                            bool    `json:"active"`
	RunningSettlementEnabled          bool    `json:"runningSettlementEnabled"`
	RunningSettlementIntervalMinutes  int     `json:"runningSettlementIntervalMinutes"`
	JobFreeMinutes                    int     `json:"jobFreeMinutes"`
	DefaultIssueAmount                float64 `json:"defaultIssueAmount"`
	DefaultIssuePeriodMinutes         int     `json:"defaultIssuePeriodMinutes"`
	AccountIssueAmountOverrideEnabled bool    `json:"accountIssueAmountOverrideEnabled"`
	AccountIssuePeriodOverrideEnabled bool    `json:"accountIssuePeriodOverrideEnabled"`
	BaseLoopCronStatus                string  `json:"baseLoopCronStatus"`
	BaseLoopCronEnabled               bool    `json:"baseLoopCronEnabled"`
}

type SetBillingStatusReq struct {
	FeatureEnabled                    *bool                       `json:"featureEnabled"`
	Active                            *bool                       `json:"active"`
	RunningSettlementEnabled          *bool                       `json:"runningSettlementEnabled"`
	RunningSettlementIntervalMinutes  *int                        `json:"runningSettlementIntervalMinutes"`
	JobFreeMinutes                    *int                        `json:"jobFreeMinutes"`
	DefaultIssueAmount                *service.BillingAmountInput `json:"defaultIssueAmount"`
	DefaultIssuePeriodMinutes         *int                        `json:"defaultIssuePeriodMinutes"`
	AccountIssueAmountOverrideEnabled *bool                       `json:"accountIssueAmountOverrideEnabled"`
	AccountIssuePeriodOverrideEnabled *bool                       `json:"accountIssuePeriodOverrideEnabled"`
}

type ResetAllBillingBalancesResp struct {
	AccountsAffected     int       `json:"accountsAffected"`
	UserAccountsAffected int       `json:"userAccountsAffected"`
	IssuedAt             time.Time `json:"issuedAt"`
}

type GrantAllUsersExtraBalanceReq struct {
	Delta  service.BillingAmountInput `json:"delta" binding:"required"`
	Reason string                     `json:"reason"`
}

type GrantAllUsersExtraBalanceResp struct {
	UsersAffected int       `json:"usersAffected"`
	Delta         float64   `json:"delta"`
	Reason        string    `json:"reason"`
	IssuedAt      time.Time `json:"issuedAt"`
}

// --- Handlers ---

// GetLLMConfig godoc
// @Summary		获取 LLM 配置信息
// @Description	获取当前系统配置的 LLM BaseURL, ModelName。出于安全考虑，API Key 可能会被脱敏显示。
// @Tags			SystemConfig
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[LLMConfigResp] "配置信息"
// @Failure		500		{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/system-config/llm [get]
func (mgr *SystemConfigMgr) GetLLMConfig(c *gin.Context) {
	// service 返回的是解密后的明文配置（方便内部使用）
	cfg, err := mgr.service.GetLLMConfig(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	// 响应给前端时，将 Key 替换为固定掩码
	// 只要 Key 不为空，就显示 "********"
	displayKey := ""
	if cfg.APIKey != "" {
		displayKey = service.MaskedAPIKeyPlaceholder // 使用常量 "********"
	}

	resputil.Success(c, LLMConfigResp{
		BaseURL:   cfg.BaseURL,
		APIKey:    displayKey, // 前端拿到的是 "********"
		ModelName: cfg.ModelName,
	})
}

// UpdateLLMConfig godoc
// @Summary		更新 LLM 配置
// @Description	更新 LLM 的连接信息。如果 validate 为 true，会尝试连接 /check 接口，失败则不保存。
// @Tags			SystemConfig
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		UpdateLLMConfigReq		true	"配置信息"
// @Success		200		{object}	resputil.Response[string] "更新成功"
// @Failure		400		{object}	resputil.Response[any] "参数错误或校验失败"
// @Router			/v1/admin/system-config/llm [put]
func (mgr *SystemConfigMgr) UpdateLLMConfig(c *gin.Context) {
	var req UpdateLLMConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	serviceCfg := &service.LLMConfig{
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey, // 这里可能是 "********" 也可能是 "sk-newkey..."
		ModelName: req.ModelName,
	}

	// Service 内部会判断：如果 Key == "********" 则不更新 DB 中的 Key
	err := mgr.service.UpdateLLMConfig(c.Request.Context(), serviceCfg, req.Validate)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			resputil.Error(c, "LLM connection check failed. Please verify your settings.", resputil.BusinessLogicError)
			return
		}
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, "LLM configuration updated successfully")
}

// ResetLLMConfig godoc
// @Summary		重置 LLM 配置
// @Description	清空 LLM 配置（BaseURL, Key, Model）并强制关闭 GPU 分析功能
// @Tags			SystemConfig
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[string] "重置成功"
// @Failure		500		{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/system-config/llm [delete]
func (mgr *SystemConfigMgr) ResetLLMConfig(c *gin.Context) {
	err := mgr.service.ResetLLMConfig(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, "LLM configuration reset successfully")
}

// GetGpuAnalysisStatus godoc
// @Summary		获取 GPU 分析功能开关状态
// @Description	查询当前系统是否开启了自动 GPU 资源滥用检测
// @Tags			SystemConfig
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[GpuAnalysisStatusResp] "开关状态"
// @Router			/v1/admin/system-config/gpu-analysis [get]
func (mgr *SystemConfigMgr) GetGpuAnalysisStatus(c *gin.Context) {
	enabled := mgr.service.IsGpuAnalysisEnabled(c.Request.Context())
	resputil.Success(c, GpuAnalysisStatusResp{Enabled: enabled})
}

// SetGpuAnalysisStatus godoc
// @Summary		设置 GPU 分析功能开关
// @Description	开启或关闭 GPU 检测。注意：尝试开启时，系统会强制检查 LLM 连接，如果连接不通，将无法开启。
// @Tags			SystemConfig
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		SetGpuAnalysisStatusReq		true	"开关设置"
// @Success		200		{object}	resputil.Response[string] "设置成功"
// @Failure		400		{object}	resputil.Response[any] "LLM连接检查失败，无法开启"
// @Router			/v1/admin/system-config/gpu-analysis [put]
func (mgr *SystemConfigMgr) SetGpuAnalysisStatus(c *gin.Context) {
	var req SetGpuAnalysisStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	err := mgr.service.SetGpuAnalysisEnabled(c, req.Enable)
	if err != nil {
		// 返回特定错误码供前端翻译
		resputil.Success(c, "ERR_LLM_CONNECTION_FAILED")
		return
	}

	action := "disabled"
	if req.Enable {
		action = "enabled"
	}
	resputil.Success(c, "GPU analysis "+action)
}

// GetPrequeueConfig godoc
// @Summary		获取新版排队配置
// @Description	获取当前回填提交开关、Crater 队内资源配额开关、普通作业等待忍耐时间和 watcher 运行参数
// @Tags			SystemConfig
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[PrequeueConfigResp] "配置"
// @Failure		500		{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/system-config/prequeue [get]
func (mgr *SystemConfigMgr) GetPrequeueConfig(c *gin.Context) {
	cfg, err := mgr.service.GetPrequeueConfig(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, PrequeueConfigResp{
		BackfillEnabled:                  cfg.BackfillEnabled,
		QueueQuotaEnabled:                cfg.QueueQuotaEnabled,
		NormalJobWaitingToleranceSeconds: cfg.NormalJobWaitingToleranceSeconds,
		ActivateTickerIntervalSeconds:    cfg.ActivateTickerIntervalSeconds,
		MaxTotalActivationsPerRound:      cfg.MaxTotalActivationsPerRound,
	})
}

// UpdatePrequeueConfig godoc
// @Summary		更新新版排队配置
// @Description	更新回填提交开关、Crater 队内资源配额开关、普通作业等待忍耐时间和 watcher 运行参数
// @Tags			SystemConfig
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		UpdatePrequeueConfigReq	true	"配置"
// @Success		200		{object}	resputil.Response[string] "更新成功"
// @Failure		400		{object}	resputil.Response[any] "参数错误"
// @Failure		500		{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/system-config/prequeue [put]
func (mgr *SystemConfigMgr) UpdatePrequeueConfig(c *gin.Context) {
	var req UpdatePrequeueConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	cfg := &service.PrequeueRuntimeConfig{
		BackfillEnabled:                  *req.BackfillEnabled,
		QueueQuotaEnabled:                *req.QueueQuotaEnabled,
		NormalJobWaitingToleranceSeconds: *req.NormalJobWaitingToleranceSeconds,
		ActivateTickerIntervalSeconds:    *req.ActivateTickerIntervalSeconds,
		MaxTotalActivationsPerRound:      *req.MaxTotalActivationsPerRound,
	}
	if err := mgr.service.UpdatePrequeueConfig(c.Request.Context(), cfg); err != nil {
		if strings.Contains(err.Error(), "must be greater than 0") {
			resputil.BadRequestError(c, err.Error())
			return
		}
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	if mgr.watcher != nil {
		mgr.watcher.RequestFullScan()
	}

	resputil.Success(c, "Prequeue configuration updated successfully")
}

func (mgr *SystemConfigMgr) GetBillingStatus(c *gin.Context) {
	if mgr.billingService == nil {
		resputil.Error(c, "billing service is not initialized", resputil.ServiceError)
		return
	}
	status := mgr.billingService.GetStatus(c.Request.Context())
	if !status.FeatureEnabled {
		resputil.Success(c, BillingStatusResp{
			FeatureEnabled:      false,
			BaseLoopCronStatus:  status.BaseLoopCronStatus,
			BaseLoopCronEnabled: status.BaseLoopCronEnabled,
		})
		return
	}
	resputil.Success(c, BillingStatusResp{
		FeatureEnabled:                    status.FeatureEnabled,
		Active:                            status.Active,
		RunningSettlementEnabled:          status.RunningSettlementEnabled,
		RunningSettlementIntervalMinutes:  status.RunningSettlementIntervalMinutes,
		JobFreeMinutes:                    status.JobFreeMinutes,
		DefaultIssueAmount:                status.DefaultIssueAmount,
		DefaultIssuePeriodMinutes:         status.DefaultIssuePeriodMinutes,
		AccountIssueAmountOverrideEnabled: status.AccountIssueAmountOverrideEnabled,
		AccountIssuePeriodOverrideEnabled: status.AccountIssuePeriodOverrideEnabled,
		BaseLoopCronStatus:                status.BaseLoopCronStatus,
		BaseLoopCronEnabled:               status.BaseLoopCronEnabled,
	})
}

func (mgr *SystemConfigMgr) SetBillingStatus(c *gin.Context) {
	if mgr.billingService == nil {
		resputil.Error(c, "billing service is not initialized", resputil.ServiceError)
		return
	}
	var req SetBillingStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var defaultIssueAmount *int64
	if req.DefaultIssueAmount != nil {
		v := req.DefaultIssueAmount.MicroPoints()
		defaultIssueAmount = &v
	}
	if err := mgr.billingService.UpdateStatus(c.Request.Context(), service.BillingUpdate{
		FeatureEnabled:                    req.FeatureEnabled,
		Active:                            req.Active,
		RunningSettlementEnabled:          req.RunningSettlementEnabled,
		RunningSettlementIntervalMinutes:  req.RunningSettlementIntervalMinutes,
		JobFreeMinutes:                    req.JobFreeMinutes,
		DefaultIssueAmount:                defaultIssueAmount,
		DefaultIssuePeriodMinutes:         req.DefaultIssuePeriodMinutes,
		AccountIssueAmountOverrideEnabled: req.AccountIssueAmountOverrideEnabled,
		AccountIssuePeriodOverrideEnabled: req.AccountIssuePeriodOverrideEnabled,
	}); err != nil {
		resputil.Error(c, err.Error(), resputil.BusinessLogicError)
		return
	}
	resputil.Success(c, "Billing configuration updated")
}

func (mgr *SystemConfigMgr) TriggerBillingBaseLoop(c *gin.Context) {
	if mgr.billingService == nil {
		resputil.Error(c, "billing service is not initialized", resputil.ServiceError)
		return
	}
	result, err := mgr.billingService.RunBaseLoopOnce(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, result)
}

func (mgr *SystemConfigMgr) ResetAllBillingBalances(c *gin.Context) {
	if mgr.billingService == nil {
		resputil.Error(c, "billing service is not initialized", resputil.ServiceError)
		return
	}
	if !mgr.billingService.IsFeatureEnabled(c.Request.Context()) {
		resputil.Error(c, "billing feature is disabled", resputil.BusinessLogicError)
		return
	}
	result, err := mgr.billingService.ResetAllPeriodFreeBalances(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, ResetAllBillingBalancesResp{
		AccountsAffected:     result.AccountsAffected,
		UserAccountsAffected: result.UserAccountsAffected,
		IssuedAt:             result.IssuedAt,
	})
}

func (mgr *SystemConfigMgr) GrantAllUsersExtraBalance(c *gin.Context) {
	var req GrantAllUsersExtraBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	delta := req.Delta.MicroPoints()
	if delta <= 0 {
		resputil.BadRequestError(c, "delta must be > 0")
		return
	}

	issuedAt := time.Now()
	usersAffected := 0
	err := query.GetDB().WithContext(c).Transaction(func(tx *gorm.DB) error {
		txQuery := query.Use(tx)
		u := txQuery.User
		users, err := u.WithContext(c).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(u.DeletedAt.IsNull()).
			Find()
		if err != nil {
			return err
		}

		for i := range users {
			user := users[i]
			after := user.ExtraBalance + delta
			if _, err := u.WithContext(c).
				Where(u.ID.Eq(user.ID), u.DeletedAt.IsNull()).
				Update(u.ExtraBalance, after); err != nil {
				return err
			}
			usersAffected++
		}
		return nil
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("grant extra balance to all users failed: %v", err), resputil.NotSpecified)
		return
	}

	klog.Infof(
		"grant extra balance to all users success, usersAffected=%d delta=%d reason=%s",
		usersAffected,
		delta,
		req.Reason,
	)

	resputil.Success(c, GrantAllUsersExtraBalanceResp{
		UsersAffected: usersAffected,
		Delta:         service.ToDisplayPoints(delta),
		Reason:        req.Reason,
		IssuedAt:      issuedAt,
	})
}
