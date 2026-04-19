package handler

import (
	"github.com/gin-gonic/gin"

	"strings"

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
	cronjobManager *cronjob.CronJobManager
	watcher        *prequeuewatcher.PrequeueWatcher
}

func NewSystemConfigMgr(conf *RegisterConfig) Manager {
	return &SystemConfigMgr{
		name:           "system-config",
		service:        conf.ConfigService,
		cronjobManager: conf.CronJobManager,
		watcher:        conf.PrequeueWatcher,
	}
}

func (mgr *SystemConfigMgr) GetName() string                      { return mgr.name }
func (mgr *SystemConfigMgr) RegisterPublic(_ *gin.RouterGroup)    {}
func (mgr *SystemConfigMgr) RegisterProtected(_ *gin.RouterGroup) {}

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
