package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	// 需要引入 controller-runtime client
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/config" // 需要引入 config 获取 namespace
)

//nolint:gochecknoinits // init is used to register handler routes
func init() {
	Registers = append(Registers, NewGpuAnalysisMgr)
}

type GpuAnalysisMgr struct {
	name          string
	client        client.Client // [新增] 添加 controller-runtime client 以便操作 CRD
	service       *service.GpuAnalysisService
	configService *service.ConfigService
}

func NewGpuAnalysisMgr(conf *RegisterConfig) Manager {
	return &GpuAnalysisMgr{
		name:          "gpu-analysis",
		client:        conf.Client,
		service:       conf.GpuAnalysisService,
		configService: conf.ConfigService,
	}
}

func (mgr *GpuAnalysisMgr) GetName() string                      { return mgr.name }
func (mgr *GpuAnalysisMgr) RegisterPublic(_ *gin.RouterGroup)    {}
func (mgr *GpuAnalysisMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *GpuAnalysisMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListAnalyses)
	g.PUT("/:id/review", mgr.UpdateReviewStatus)

	// [新增] 确认并停止作业的接口
	g.POST("/:id/confirm-stop", mgr.ConfirmAndStopJob)

	g.POST("/trigger/pod", mgr.TriggerPodAnalysis)
	g.POST("/trigger/job", mgr.TriggerJobAnalysis)
	g.POST("/trigger/all-jobs", mgr.TriggerAllJobsAnalysis)
}

type TriggerAnalysisReq struct {
	Namespace string `json:"namespace" binding:"required"`
	PodName   string `json:"podName" binding:"required"`
}

// TriggerPodAnalysis godoc
// @Summary		手动触发对单个 Pod 的 GPU 分析
// @Description	用于测试和演示。提供 Pod 的 namespace 和 name 来启动一次完整的两阶段分析。
// @Tags			GpuAnalysis
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		TriggerAnalysisReq		true	"Pod 信息"
// @Success		200		{object}	resputil.Response[model.GpuAnalysis] "分析成功，返回创建的分析记录"
// @Failure		400		{object}	resputil.Response[any] "请求参数错误"
// @Failure		500		{object}	resputil.Response[any] "分析过程中发生错误"
// @Router			/v1/admin/gpu-analysis/trigger [post]
func (mgr *GpuAnalysisMgr) TriggerPodAnalysis(c *gin.Context) {
	// 【修改】使用 BusinessLogicError，以便前端组件捕获并显示“功能未开启”的引导提示，而非全局报错
	if !mgr.configService.IsGpuAnalysisEnabled(c.Request.Context()) {
		resputil.Error(c, "GPU Analysis feature is currently disabled.", resputil.BusinessLogicError)
		return
	}

	var req TriggerAnalysisReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	analysisResult, err := mgr.service.AnalyzePodByName(c.Request.Context(), req.Namespace, req.PodName)
	if err != nil {
		klog.Errorf("Manual analysis trigger failed for pod %s: %v", req.PodName, err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, analysisResult)
}

// GpuAnalysisWithJobInfo 是 ListAnalyses 接口的返回结构体
// 它在 GpuAnalysis 的基础上增加了 Job 的相关信息
type GpuAnalysisWithJobInfo struct {
	model.GpuAnalysis
	Name            string
	JobType         model.JobType
	Resources       datatypes.JSONType[v1.ResourceList]
	Nodes           datatypes.JSONType[[]string]
	Status          batch.JobPhase `json:"status"`
	LockedTimestamp time.Time      `json:"lockedTimestamp"`
}

// ListAnalyses godoc
// @Summary		列出所有 GPU 分析记录（包含作业信息）
// @Description	按创建时间降序列出所有 GPU 分析记录，并附带关联作业的类型、资源和节点信息
// @Tags			GpuAnalysis
// @Produce		json
// @Security		Bearer
// @Success		200	{object}	resputil.Response[[]GpuAnalysisWithJobInfo] "成功"
// @Failure		500	{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/gpu-analysis [get]
func (mgr *GpuAnalysisMgr) ListAnalyses(c *gin.Context) {
	var results []GpuAnalysisWithJobInfo
	ga := query.GpuAnalysis
	j := query.Job

	err := ga.WithContext(c.Request.Context()).
		LeftJoin(j, ga.JobName.EqCol(j.JobName)).
		Order(ga.Phase2Score.Desc(), ga.CreatedAt.Desc()).
		Select(ga.ALL, j.Name, j.JobType, j.Resources, j.Nodes, j.Status, j.LockedTimestamp).
		Scan(&results)

	if err != nil {
		resputil.Error(c, fmt.Sprintf("list gpu analyses failed: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, results)
}

type UpdateReviewStatusReq struct {
	ReviewStatus model.ReviewStatus `json:"reviewStatus" binding:"required,min=1"`
}

// UpdateReviewStatus godoc
// @Summary		更新管理员对分析记录的审核状态
// @Description	管理员在前端审核后，调用此接口将记录标记为“已确认占卡”或“已忽略”
// @Tags			GpuAnalysis
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int						true	"分析记录ID"
// @Param			data	body		UpdateReviewStatusReq	true	"新的审核状态"
// @Success		200		{object}	resputil.Response[string] "更新成功"
// @Failure		400		{object}	resputil.Response[any] "请求参数错误"
// @Failure		500		{object}	resputil.Response[any] "服务器错误"
// @Router			/v1/admin/gpu-analysis/{id}/review [put]
func (mgr *GpuAnalysisMgr) UpdateReviewStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resputil.BadRequestError(c, "Invalid ID format")
		return
	}
	if id <= 0 {
		resputil.BadRequestError(c, "ID must be a positive integer")
		return
	}

	var req UpdateReviewStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.ReviewStatus != model.ReviewStatusConfirmed && req.ReviewStatus != model.ReviewStatusIgnored {
		resputil.Error(c, "Invalid review status provided. Must be Confirmed(2) or Ignored(3).", http.StatusBadRequest)
		return
	}

	ga := query.GpuAnalysis
	info, err := ga.WithContext(c.Request.Context()).
		Where(ga.ID.Eq(uint(id))).
		Updates(model.GpuAnalysis{ReviewStatus: req.ReviewStatus})

	if err != nil {
		resputil.Error(c, fmt.Sprintf("update review status failed: %v", err), resputil.NotSpecified)
		return
	}
	if info.RowsAffected == 0 {
		resputil.Error(c, fmt.Sprintf("analysis record with id %d not found", id), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "success")
}

type TriggerJobAnalysisReq struct {
	JobName string `json:"jobName" binding:"required"`
}

// TriggerJobAnalysis godoc
// @Summary		手动触发对单个 Job 的 GPU 分析
// @Description	提供 Job 的 name 来启动一次完整的分析。如果该 Job 已有分析记录，则更新；否则创建新记录。
// @Tags			GpuAnalysis
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		TriggerJobAnalysisReq		true	"Job 信息"
// @Success		200		{object}	resputil.Response[model.GpuAnalysis] "分析成功，返回创建或更新后的分析记录"
// @Failure		400		{object}	resputil.Response[any] "请求参数错误"
// @Failure		500		{object}	resputil.Response[any] "分析过程中发生错误"
// @Router			/v1/admin/gpu-analysis/trigger/job [post]
func (mgr *GpuAnalysisMgr) TriggerJobAnalysis(c *gin.Context) {
	// 【修改】使用 BusinessLogicError
	if !mgr.configService.IsGpuAnalysisEnabled(c.Request.Context()) {
		resputil.Error(c, "GPU Analysis feature is currently disabled.", resputil.BusinessLogicError)
		return
	}

	var req TriggerJobAnalysisReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	analysisResult, err := mgr.service.AnalyzeJobByName(c.Request.Context(), req.JobName)
	if err != nil {
		klog.Errorf("Manual job analysis trigger failed for job %s: %v", req.JobName, err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, analysisResult)
}

type TriggerAllJobsAnalysisResponse struct {
	QueuedJobs int    `json:"queuedJobs"`
	Message    string `json:"message"`
}

// TriggerAllJobsAnalysis godoc
// @Summary		触发对所有运行中 Job 的异步 GPU 分析
// @Description	此接口会查找所有符合条件的运行中 Job，并将它们加入后台分析队列。接口会立即返回，分析将在后台异步进行。
// @Tags			GpuAnalysis
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[TriggerAllJobsAnalysisResponse] "成功，返回入队的 Job 数量"
// @Failure		500		{object}	resputil.Response[any] "服务器错误或队列已满"
// @Router			/v1/admin/gpu-analysis/trigger/all-jobs [post]
func (mgr *GpuAnalysisMgr) TriggerAllJobsAnalysis(c *gin.Context) {
	// 【修改】使用 BusinessLogicError
	if !mgr.configService.IsGpuAnalysisEnabled(c.Request.Context()) {
		resputil.Error(c, "GPU Analysis feature is currently disabled.", resputil.BusinessLogicError)
		return
	}

	count, err := mgr.service.TriggerAllJobsAnalysis(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	responseData := TriggerAllJobsAnalysisResponse{
		QueuedJobs: count,
		Message:    fmt.Sprintf("%d jobs have been queued for analysis.", count),
	}

	resputil.Success(c, responseData)
}

// ConfirmAndStopJob godoc
// @Summary		确认占卡并停止作业
// @Description	将分析记录标记为“已确认”，并立即停止（删除）对应的 Volcano Job。
// @Tags			GpuAnalysis
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int						true	"分析记录ID"
// @Success		200		{object}	resputil.Response[string] "操作成功"
// @Failure		400		{object}	resputil.Response[any] "请求参数错误"
// @Failure		500		{object}	resputil.Response[any] "操作失败"
// @Router			/v1/admin/gpu-analysis/{id}/confirm-stop [post]
func (mgr *GpuAnalysisMgr) ConfirmAndStopJob(c *gin.Context) {
	// 1. 解析 ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		resputil.BadRequestError(c, "Invalid ID format")
		return
	}

	ctx := c.Request.Context()
	ga := query.GpuAnalysis

	// 2. 获取分析记录以拿到 JobName
	analysis, err := ga.WithContext(ctx).Where(ga.ID.Eq(uint(id))).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("analysis record not found: %v", err), resputil.NotSpecified)
		return
	}

	// 3. 更新分析记录状态为“已确认”(ReviewStatusConfirmed)
	if _, err := ga.WithContext(ctx).
		Where(ga.ID.Eq(uint(id))).
		Updates(model.GpuAnalysis{ReviewStatus: model.ReviewStatusConfirmed}); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update review status: %v", err), resputil.NotSpecified)
		return
	}

	// 4. 调用复用的停止作业逻辑
	// 注意：这里复用了 VolcanojobMgr 中的 StopJobByName 逻辑
	if err := mgr.stopJobByName(ctx, analysis.JobName); err != nil {
		resputil.Error(c, fmt.Sprintf("review confirmed but failed to stop job '%s': %v", analysis.JobName, err), resputil.ServiceError)
		return
	}

	resputil.Success(c, fmt.Sprintf("Job '%s' stopped and analysis confirmed.", analysis.JobName))
}

// stopJobByName 是从 VolcanojobMgr.StopJobByName 复用（移植）过来的逻辑
// 它负责更新数据库状态并删除 K8s 中的 Job 资源
func (mgr *GpuAnalysisMgr) stopJobByName(ctx context.Context, jobName string) error {
	j := query.Job

	// 1. 获取数据库记录
	_, err := j.WithContext(ctx).Where(j.JobName.Eq(jobName)).First()
	if err != nil {
		return fmt.Errorf("job record not found in db: %w", err)
	}

	// 2. 尝试从 K8s 获取实体
	job := &batch.Job{}
	namespace := config.GetConfig().Namespaces.Job
	err = mgr.client.Get(ctx, client.ObjectKey{Name: jobName, Namespace: namespace}, job)

	exists := err == nil
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get job from k8s: %w", err)
	}

	// 3. 更新数据库状态为 Deleted
	if _, err := j.WithContext(ctx).Where(j.JobName.Eq(jobName)).Updates(model.Job{
		Status:             model.Deleted,
		CompletedTimestamp: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to update job status in db: %w", err)
	}

	// 4. 如果 K8s 中存在该 Job，则删除它（OwnerReference 会处理关联的 Pod/Svc/Ingress）
	if exists {
		// PropagationPolicy: Background 是默认行为，但显式写出更清晰
		deleteOptions := &client.DeleteOptions{}
		if err := mgr.client.Delete(ctx, job, deleteOptions); err != nil {
			// 如果删除时发现由于某种原因已经被删除了，忽略错误
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete job from k8s: %w", err)
			}
		}
	}

	return nil
}
