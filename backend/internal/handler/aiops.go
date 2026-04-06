package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/prompts"
)

const (
	chatResponseTypeText       = "text"
	chatResponseTypeSuggestion = "suggestion"
	errorKeyAIOPsJobNotOwned   = "aiops.error.jobNotOwned"
	errorKeyAIOPsJobNotFound   = "aiops.error.jobNotFound"
	errorKeyAIOPsJobAmbiguous  = "aiops.error.jobAmbiguous"
	errorKeyAIOPsQueryFailed   = "aiops.error.queryFailed"

	diagnosisConfidenceHigh = "high"
	diagnosisSeverityError  = "error"
	diagnosisSeverityWarn   = "warning"
	diagnosisSeverityCrit   = "critical"

	topFailureReasonsLimit = 5
	maxEvidenceEventsLimit = 5
	jobCandidatesLimit     = 5
	recentFailedJobsLimit  = 50
	chatTopReasonsLimit    = 5

	llmRequestTimeout    = 120 * time.Second
	userIssueWarnPercent = 30.0
	percentBase          = 100.0
)

var errJobNotOwned = errors.New("job does not belong to requester")

type apiErrorDetails struct {
	Code   resputil.ErrorCode `json:"code"`
	MsgKey string             `json:"msgKey"`
	Msg    string             `json:"msg"`
}

//nolint:gochecknoinits // Handler managers are registered during package initialization.
func init() {
	Registers = append(Registers, NewAIOPsMgr)
}

type AIOPsMgr struct {
	name          string
	client        client.Client
	kubeClient    kubernetes.Interface
	configService *service.ConfigService
	httpClient    *http.Client
}

func NewAIOPsMgr(conf *RegisterConfig) Manager {
	return &AIOPsMgr{
		name:          "aiops",
		client:        conf.Client,
		kubeClient:    conf.KubeClient,
		configService: conf.ConfigService,
		httpClient:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (mgr *AIOPsMgr) GetName() string { return mgr.name }

func (mgr *AIOPsMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIOPsMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/health-overview", mgr.GetHealthOverview)
	g.POST("/chat", mgr.ChatMessage)
	g.POST("/llmchat", mgr.ChatMessageLLM)
	g.GET("/diagnose/:jobName", mgr.DiagnoseJob)
}

func (mgr *AIOPsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/health-overview", mgr.GetHealthOverviewAdmin)
	g.POST("/chat", mgr.ChatMessage)
	g.POST("/llmchat", mgr.ChatMessageLLM)
	g.GET("/diagnose/:jobName", mgr.DiagnoseJob)
}

// Health Overview Response
type HealthOverviewResp struct {
	TotalJobs    int     `json:"totalJobs"`
	FailedJobs   int     `json:"failedJobs"`
	PendingJobs  int     `json:"pendingJobs"`
	RunningJobs  int     `json:"runningJobs"`
	FailureRate  float64 `json:"failureRate"`
	FailureTrend []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"failureTrend"`
	TopFailureReasons []struct {
		Reason string `json:"reason"`
		Count  int    `json:"count"`
	} `json:"topFailureReasons"`
}

// GetHealthOverview godoc
// @Summary Get AIOps health overview
// @Description Get job health overview for current user. days=0 means all time.
// @Tags aiops
// @Produce json
// @Param days query int false "Lookback days (default 7, 0 means all time)"
// @Success 200 {object} resputil.Response[HealthOverviewResp]
// @Router /api/v1/aiops/health-overview [get]
// GetHealthOverview returns health overview for current user
//
//nolint:gocyclo // This endpoint aggregates multiple query branches for a single response.
func (mgr *AIOPsMgr) GetHealthOverview(c *gin.Context) {
	type QueryParams struct {
		Days *int `form:"days"` // nil for default (7 days), 0 for all time, >0 for specific days
	}
	var req QueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Set default value if not provided
	days := 7 // default to 7 days
	if req.Days != nil {
		days = *req.Days
	}

	token := util.GetToken(c)
	j := query.Job

	now := time.Now()
	var lookback time.Time

	// Build query
	baseQuery := j.WithContext(c).Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))

	// Add time filter if days > 0 (0 means all time, no filter)
	if days > 0 {
		lookback = now.AddDate(0, 0, -days)
		baseQuery = baseQuery.Where(j.CreationTimestamp.Gte(lookback))
	}

	// Get all jobs in time range
	allJobs, err := baseQuery.Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp := HealthOverviewResp{
		TotalJobs: len(allJobs),
	}

	// Count by status
	statusCount := make(map[string]int)
	for _, job := range allJobs {
		statusCount[string(job.Status)]++
	}

	resp.FailedJobs = statusCount[string(batch.Failed)]
	resp.PendingJobs = statusCount[string(batch.Pending)]
	resp.RunningJobs = statusCount[string(batch.Running)]

	if resp.TotalJobs > 0 {
		resp.FailureRate = float64(resp.FailedJobs) / float64(resp.TotalJobs) * 100
	}

	// Get failure trend (group by date) - fill all dates in range
	failureTrend := make(map[string]int)

	// Determine date range for trend
	var startDate, endDate time.Time
	endDate = now
	if days > 0 {
		startDate = lookback
	} else {
		if len(allJobs) == 0 {
			startDate = endDate
		} else {
			startDate = allJobs[0].CreationTimestamp
			for i := 1; i < len(allJobs); i++ {
				if allJobs[i].CreationTimestamp.Before(startDate) {
					startDate = allJobs[i].CreationTimestamp
				}
			}
		}
	}

	// Initialize all dates with 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		failureTrend[dateStr] = 0
	}

	// Count failures by creation date (consistent with total jobs filter)
	for _, job := range allJobs {
		if job.Status == batch.Failed {
			dateStr := job.CreationTimestamp.Format("2006-01-02")
			if _, exists := failureTrend[dateStr]; exists {
				failureTrend[dateStr]++
			}
		}
	}

	// Convert map to sorted slice
	for date, count := range failureTrend {
		resp.FailureTrend = append(resp.FailureTrend, struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		}{Date: date, Count: count})
	}
	sort.Slice(resp.FailureTrend, func(i, j int) bool {
		return resp.FailureTrend[i].Date < resp.FailureTrend[j].Date
	})

	// Get top failure reasons - use same time filter as total jobs (CreationTimestamp)
	failedQuery := j.WithContext(c).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.Status.Eq(string(batch.Failed)))

	if days > 0 {
		failedQuery = failedQuery.Where(j.CreationTimestamp.Gte(lookback))
	}

	failedJobs, failedErr := failedQuery.Find()
	if failedErr != nil {
		resputil.Error(c, failedErr.Error(), resputil.NotSpecified)
		return
	}

	reasonCount := make(map[string]int)
	for _, job := range failedJobs {
		reason := CategorizeFailure(job).TypeName
		reasonCount[reason]++
	}

	for reason, count := range reasonCount {
		resp.TopFailureReasons = append(resp.TopFailureReasons, struct {
			Reason string `json:"reason"`
			Count  int    `json:"count"`
		}{Reason: reason, Count: count})
	}
	sort.Slice(resp.TopFailureReasons, func(i, j int) bool {
		return resp.TopFailureReasons[i].Count > resp.TopFailureReasons[j].Count
	})
	if len(resp.TopFailureReasons) > topFailureReasonsLimit {
		resp.TopFailureReasons = resp.TopFailureReasons[:topFailureReasonsLimit]
	}

	resputil.Success(c, resp)
}

// GetHealthOverviewAdmin godoc
// @Summary Get AIOps health overview (admin)
// @Description Get job health overview for all users. days=0 means all time.
// @Tags aiops
// @Produce json
// @Param days query int false "Lookback days (default 7, 0 means all time)"
// @Success 200 {object} resputil.Response[HealthOverviewResp]
// @Router /api/v1/admin/aiops/health-overview [get]
// GetHealthOverviewAdmin returns health overview for all users (admin only)
//
//nolint:gocyclo // This endpoint mirrors user overview logic with admin-wide query scope.
func (mgr *AIOPsMgr) GetHealthOverviewAdmin(c *gin.Context) {
	type QueryParams struct {
		Days *int `form:"days"` // nil for default (7 days), 0 for all time, >0 for specific days
	}
	var req QueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Set default value if not provided
	days := 7 // default to 7 days
	if req.Days != nil {
		days = *req.Days
	}

	j := query.Job

	now := time.Now()
	var lookback time.Time

	// Build query
	baseQuery := j.WithContext(c)

	// Add time filter if days > 0 (0 means all time, no filter)
	if days > 0 {
		lookback = now.AddDate(0, 0, -days)
		baseQuery = baseQuery.Where(j.CreationTimestamp.Gte(lookback))
	}

	allJobs, err := baseQuery.Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp := HealthOverviewResp{
		TotalJobs: len(allJobs),
	}

	statusCount := make(map[string]int)
	for _, job := range allJobs {
		statusCount[string(job.Status)]++
	}

	resp.FailedJobs = statusCount[string(batch.Failed)]
	resp.PendingJobs = statusCount[string(batch.Pending)]
	resp.RunningJobs = statusCount[string(batch.Running)]

	if resp.TotalJobs > 0 {
		resp.FailureRate = float64(resp.FailedJobs) / float64(resp.TotalJobs) * 100
	}

	// Get failure trend (group by date) - fill all dates in range
	failureTrend := make(map[string]int)

	// Determine date range for trend
	var startDate, endDate time.Time
	endDate = now
	if days > 0 {
		startDate = lookback
	} else {
		if len(allJobs) == 0 {
			startDate = endDate
		} else {
			startDate = allJobs[0].CreationTimestamp
			for i := 1; i < len(allJobs); i++ {
				if allJobs[i].CreationTimestamp.Before(startDate) {
					startDate = allJobs[i].CreationTimestamp
				}
			}
		}
	}

	// Initialize all dates with 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		failureTrend[dateStr] = 0
	}

	// Count failures by creation date (consistent with total jobs filter)
	for _, job := range allJobs {
		if job.Status == batch.Failed {
			dateStr := job.CreationTimestamp.Format("2006-01-02")
			if _, exists := failureTrend[dateStr]; exists {
				failureTrend[dateStr]++
			}
		}
	}

	// Convert map to sorted slice
	for date, count := range failureTrend {
		resp.FailureTrend = append(resp.FailureTrend, struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		}{Date: date, Count: count})
	}
	sort.Slice(resp.FailureTrend, func(i, j int) bool {
		return resp.FailureTrend[i].Date < resp.FailureTrend[j].Date
	})

	// Get top failure reasons - use same time filter as total jobs (CreationTimestamp)
	failedQuery := j.WithContext(c).Where(j.Status.Eq(string(batch.Failed)))

	if days > 0 {
		failedQuery = failedQuery.Where(j.CreationTimestamp.Gte(lookback))
	}

	failedJobs, failedErr := failedQuery.Find()
	if failedErr != nil {
		resputil.Error(c, failedErr.Error(), resputil.NotSpecified)
		return
	}

	reasonCount := make(map[string]int)
	for _, job := range failedJobs {
		reason := CategorizeFailure(job).TypeName
		reasonCount[reason]++
	}

	for reason, count := range reasonCount {
		resp.TopFailureReasons = append(resp.TopFailureReasons, struct {
			Reason string `json:"reason"`
			Count  int    `json:"count"`
		}{Reason: reason, Count: count})
	}
	sort.Slice(resp.TopFailureReasons, func(i, j int) bool {
		return resp.TopFailureReasons[i].Count > resp.TopFailureReasons[j].Count
	})
	if len(resp.TopFailureReasons) > topFailureReasonsLimit {
		resp.TopFailureReasons = resp.TopFailureReasons[:topFailureReasonsLimit]
	}

	resputil.Success(c, resp)
}

// Diagnosis Response
type DiagnosisResp struct {
	JobName    string `json:"jobName"`
	Status     string `json:"status"`
	Category   string `json:"category"`
	Diagnosis  string `json:"diagnosis"`
	Solution   string `json:"solution"`
	Confidence string `json:"confidence"` // high, medium, low
	Severity   string `json:"severity"`   // critical, error, warning, info
	Evidence   struct {
		ExitCode   int32    `json:"exitCode,omitempty"`
		ExitReason string   `json:"exitReason,omitempty"`
		Events     []string `json:"events,omitempty"`
	} `json:"evidence"`
}

// DiagnoseJob performs rule-based diagnosis on a specific job
// @Summary Diagnose a job
// @Description Run rule-based diagnosis for a job by jobName.
// @Tags aiops
// @Produce json
// @Param jobName path string true "Job name"
// @Success 200 {object} resputil.Response[DiagnosisResp]
// @Router /api/v1/aiops/diagnose/{jobName} [get]
// @Router /api/v1/admin/aiops/diagnose/{jobName} [get]
func (mgr *AIOPsMgr) DiagnoseJob(c *gin.Context) {
	type URI struct {
		JobName string `uri:"jobName" binding:"required"`
	}
	var uri URI
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	job, err := mgr.findJobByInput(c, token, uri.JobName)
	if err != nil {
		lookupErr := classifyAIOPsLookupError(err)
		resputil.ErrorWithKey(c, lookupErr.MsgKey, lookupErr.Msg, lookupErr.Code)
		return
	}

	// Perform diagnosis
	diagnosis := PerformDiagnosis(job)
	resputil.Success(c, diagnosis)
}

// performDiagnosis applies diagnostic rules to a job
//
//nolint:gocyclo,funlen // Rule-driven diagnosis intentionally keeps category handling in one switch.
func PerformDiagnosis(job *model.Job) DiagnosisResp {
	resp := DiagnosisResp{
		JobName: job.JobName,
		Status:  string(job.Status),
	}

	// Rule-based diagnosis
	result := CategorizeFailure(job)
	resp.Category = result.TypeName

	// Apply diagnostic rules based on category
	switch result.TypeName {
	case "OOMKilled":
		resp.Diagnosis = "作业因内存溢出（OOM）被终止"
		resp.Solution = "建议：1) 增加内存请求和限制；2) 优化代码减少内存使用；3) 检查是否有内存泄漏"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit
		if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
			resp.Evidence.ExitCode = job.TerminatedStates.Data()[0].ExitCode
			resp.Evidence.ExitReason = job.TerminatedStates.Data()[0].Reason
		}

	case "ImagePullError":
		resp.Diagnosis = "镜像拉取失败"
		resp.Solution = "建议：1) 检查镜像名称和标签是否正确；2) 检查镜像仓库认证；3) 检查网络连接；4) 确认镜像是否存在"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError

	case "SchedulingInsufficientResources":
		resp.Diagnosis = "集群资源不足，无法调度"
		resp.Solution = "建议：1) 降低资源请求；2) 等待其他作业完成释放资源；3) 联系管理员扩容集群"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn

	case "SchedulingNodeSelectorMismatch":
		resp.Diagnosis = "节点选择器不匹配"
		resp.Solution = "建议：1) 检查节点标签配置；2) 修改作业的节点选择器；3) 联系管理员确认节点可用性"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError

	case "SchedulingTaintMismatch":
		resp.Diagnosis = "节点污点与容忍度不匹配"
		resp.Solution = "建议：1) 检查节点 taint；2) 在作业配置中补充 tolerations；3) 与管理员确认调度策略"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError

	case "SchedulingFailed":
		resp.Diagnosis = "作业调度失败"
		resp.Solution = "建议：1) 查看 FailedScheduling 事件原文；2) 检查资源请求、节点选择器与污点容忍；3) 必要时联系管理员"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError

	case "CrashLoopBackOff":
		resp.Diagnosis = "容器持续崩溃重启"
		resp.Solution = "建议：1) 查看容器日志确定崩溃原因；2) 检查启动命令；3) 检查配置文件；4) 可能是资源不足"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit

	case "VolumeMountFailed":
		resp.Diagnosis = "存储卷挂载失败"
		resp.Solution = "建议：1) 检查 PVC/PV 绑定状态；2) 核对存储类与访问模式；3) 检查挂载路径和权限"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError

	case "JobDeadlineExceeded":
		resp.Diagnosis = "作业超出截止时间被终止"
		resp.Solution = "建议：1) 评估并调大 activeDeadlineSeconds；2) 优化作业耗时；3) 检查资源瓶颈"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn

	case "ContainerError":
		resp.Diagnosis = "容器运行时错误"
		resp.Solution = "建议查看容器日志获取详细错误信息。常见原因包括：代码异常、依赖缺失、权限问题等"
		resp.Confidence = "medium"
		resp.Severity = diagnosisSeverityError
		if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
			resp.Evidence.ExitCode = job.TerminatedStates.Data()[0].ExitCode
			resp.Evidence.ExitReason = job.TerminatedStates.Data()[0].Reason
			if title, suggestion, ok := exitCodeDiagnosis(resp.Evidence.ExitCode); ok {
				resp.Diagnosis = fmt.Sprintf("容器运行时错误（Exit %d：%s）", resp.Evidence.ExitCode, title)
				resp.Solution = suggestion
			}
		}

	case "CommandNotFound":
		resp.Diagnosis = "启动命令或文件不存在"
		resp.Solution = "建议：1) 检查启动命令是否正确；2) 检查镜像内目标文件/脚本是否存在；3) 检查工作目录与PATH配置"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
		if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
			resp.Evidence.ExitCode = job.TerminatedStates.Data()[0].ExitCode
			resp.Evidence.ExitReason = job.TerminatedStates.Data()[0].Reason
		}

	case "GracefulTermination":
		resp.Diagnosis = "作业收到终止信号并退出"
		resp.Solution = "建议：1) 结合事件时间线确认是否为人工停止或调度回收；2) 如非预期，检查控制器与资源策略"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
		if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
			resp.Evidence.ExitCode = job.TerminatedStates.Data()[0].ExitCode
			resp.Evidence.ExitReason = job.TerminatedStates.Data()[0].Reason
		}

	case "Evicted":
		resp.Diagnosis = "作业所在 Pod 被节点驱逐（Evicted）"
		resp.Solution = "建议：1) 查看节点资源压力与驱逐原因；2) 检查请求/限制是否合理；3) 与管理员确认节点状态"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn

	case "SegmentationFault":
		resp.Diagnosis = "段错误（Segmentation Fault）"
		resp.Solution = "建议：1) C/C++程序内存访问错误；2) 检查指针使用；3) 可能是依赖库版本不兼容"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit

	case "JobAbortedOrTerminated":
		resp.Diagnosis = "作业被中止或终止"
		resp.Solution = "建议：1) 检查是否有人工停止或控制器回收；2) 结合事件与调度记录确认触发原因"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn

	default:
		resp.Diagnosis = "未能自动诊断出具体原因"
		resp.Solution = "建议查看作业日志和 Kubernetes 事件进行人工分析"
		resp.Confidence = "low"
		resp.Severity = diagnosisSeverityError
		if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
			resp.Evidence.ExitCode = job.TerminatedStates.Data()[0].ExitCode
			resp.Evidence.ExitReason = job.TerminatedStates.Data()[0].Reason
			if title, suggestion, ok := exitCodeDiagnosis(resp.Evidence.ExitCode); ok {
				resp.Diagnosis = fmt.Sprintf("退出异常（Exit %d：%s）", resp.Evidence.ExitCode, title)
				resp.Solution = suggestion
				resp.Confidence = "medium"
			}
		}
	}

	// Collect event evidence
	if job.Events != nil {
		events := job.Events.Data()
		for i := range events {
			if events[i].Type == "Warning" || events[i].Type == "Error" {
				resp.Evidence.Events = append(resp.Evidence.Events, events[i].Message)
			}
		}
		if len(resp.Evidence.Events) > maxEvidenceEventsLimit {
			resp.Evidence.Events = resp.Evidence.Events[:maxEvidenceEventsLimit]
		}
	}

	return resp
}

func exitCodeDiagnosis(exitCode int32) (title, suggestion string, ok bool) {
	mapping := map[int32]struct {
		title      string
		suggestion string
	}{
		1: {
			title:      "应用错误",
			suggestion: "容器因应用程序错误而停止。建议优先查看「基本信息/日志」中的错误堆栈，重点排查代码异常、依赖缺失、路径错误。",
		},
		125: {
			title:      "容器未能运行",
			suggestion: "容器运行命令未能成功执行。请检查运行环境、镜像参数及容器启动配置。",
		},
		126: {
			title:      "命令调用错误",
			suggestion: "无法调用镜像中指定命令。请确认命令路径正确，且具备可执行权限。",
		},
		127: {
			title:      "命令或文件不存在",
			suggestion: "找不到镜像中指定命令或文件。请检查启动命令、工作目录、文件路径与镜像内容是否一致。",
		},
		128: {
			title:      "退出参数无效",
			suggestion: "退出由无效退出码触发。请检查应用的退出处理逻辑。",
		},
		134: {
			title:      "异常终止（SIGABRT）",
			suggestion: "进程主动 abort 触发终止。请检查应用内部异常处理与断言逻辑。",
		},
		137: {
			title:      "立即终止（SIGKILL）",
			suggestion: "通常为内存不足或被系统强制终止。建议增加内存申请并结合日志确认是否发生 OOM。",
		},
		139: {
			title:      "分段错误（SIGSEGV）",
			suggestion: "进程发生非法内存访问。请检查依赖兼容性、底层算子或 C/C++ 扩展代码。",
		},
		143: {
			title:      "优雅终止（SIGTERM）",
			suggestion: "进程收到终止信号后退出，可能是调度或人工停止触发。请结合事件时间线判断是否为预期行为。",
		},
		255: {
			title:      "退出状态超出范围",
			suggestion: "退出码超出常规范围。建议重点检查容器日志与启动脚本。",
		},
	}
	if item, exists := mapping[exitCode]; exists {
		return item.title, item.suggestion, true
	}
	return "", "", false
}

// Chat Request & Response
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	JobName string `json:"jobName,omitempty"` // optional: if analyzing specific job
}

type ChatResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"` // text, diagnosis, suggestion
	Data    any    `json:"data,omitempty"`
}

func classifyAIOPsLookupError(err error) apiErrorDetails {
	if errors.Is(err, errJobNotOwned) {
		return apiErrorDetails{
			Code:   resputil.UserNotAllowed,
			MsgKey: errorKeyAIOPsJobNotOwned,
			Msg:    "The requested job does not belong to your account.",
		}
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "未找到作业") {
		return apiErrorDetails{
			Code:   resputil.BusinessLogicError,
			MsgKey: errorKeyAIOPsJobNotFound,
			Msg:    "Job not found.",
		}
	}
	if strings.Contains(errMsg, "无法唯一定位") {
		return apiErrorDetails{
			Code:   resputil.BusinessLogicError,
			MsgKey: errorKeyAIOPsJobAmbiguous,
			Msg:    "Multiple jobs matched the input. Please use a unique jobName.",
		}
	}
	return apiErrorDetails{
		Code:   resputil.ServiceError,
		MsgKey: errorKeyAIOPsQueryFailed,
		Msg:    "Failed to query job information.",
	}
}

func containsExitCode(message, code string) bool {
	pattern := fmt.Sprintf(`(?:\bexit\s*[:=]?\s*%s\b|\bexit%s\b)`, regexp.QuoteMeta(code), regexp.QuoteMeta(code))
	re := regexp.MustCompile(pattern)
	return re.MatchString(message)
}

func extractJobNameFromMessage(message, jobName string) string {
	name := strings.TrimSpace(jobName)
	if name != "" {
		return name
	}
	patterns := []string{
		`(?:作业|job)[：:]\s*([a-zA-Z0-9-]+)`,
		`(?:分析|诊断|查看)(?:作业)?\s+([a-zA-Z0-9-]+)`,
		`\b(jpt-[a-zA-Z0-9-]+)\b`,
		`^([a-zA-Z0-9-]{8,})$`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(message); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func canAccessAllJobs(c *gin.Context, token util.JWTMessage) bool {
	return token.RolePlatform == model.RoleAdmin && strings.HasPrefix(c.Request.URL.Path, "/api/v1/admin/")
}

//nolint:gocyclo // Access checks and fallback lookup branches are intentionally colocated.
func (mgr *AIOPsMgr) findJobByInput(c *gin.Context, token util.JWTMessage, jobName string) (*model.Job, error) {
	j := query.Job
	notOwnerMsg := fmt.Sprintf(
		"作业 %q 不属于你的账户，无法查看。\n\n请检查：\n\n• 作业名是否正确\n\n• 是否使用了你自己的作业 name（jobName）\n\n管理员账号可前往 Admin 页面使用 Chat 诊断（/admin/aiops）。",
		jobName,
	)
	q := j.WithContext(c).Where(j.JobName.Eq(jobName))
	var job *model.Job
	var err error
	if canAccessAllJobs(c, token) {
		job, err = q.First()
	} else {
		job, err = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).First()
	}
	if err == nil {
		return job, nil
	}
	if !canAccessAllJobs(c, token) {
		if total, totalErr := j.WithContext(c).Where(j.JobName.Eq(jobName)).Count(); totalErr == nil && total > 0 {
			return nil, fmt.Errorf("%w: %s", errJobNotOwned, notOwnerMsg)
		}
	}
	qName := j.WithContext(c).Where(j.Name.Eq(jobName))
	if !canAccessAllJobs(c, token) {
		qName = qName.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	count, countErr := qName.Count()
	if countErr != nil {
		return nil, fmt.Errorf("查询作业 \"%s\" 失败：%w", jobName, countErr)
	}
	if count == 0 {
		if !canAccessAllJobs(c, token) {
			if totalByName, totalByNameErr := j.WithContext(c).Where(j.Name.Eq(jobName)).Count(); totalByNameErr == nil && totalByName > 0 {
				return nil, fmt.Errorf("%w: %s", errJobNotOwned, notOwnerMsg)
			}
		}
		return nil, fmt.Errorf("未找到作业 \"%s\"。\n\n请检查：\n• 作业名是否正确\n• 建议使用作业详情路由中的 name 参数（即 jobName）", jobName)
	}
	if count > 1 {
		candidates, listErr := qName.Order(j.CreationTimestamp.Desc()).Limit(jobCandidatesLimit).Find()
		if listErr == nil {
			names := make([]string, 0, len(candidates))
			for i := range candidates {
				names = append(names, candidates[i].JobName)
			}
			return nil, fmt.Errorf(
				"找到 %d 个同名作业 %q，无法唯一定位。请改用作业详情路由里的 name 参数（即 jobName）进行查询，例如：\n• 作业:%s",
				count,
				jobName,
				strings.Join(names, "\n• 作业:"),
			)
		}
		return nil, err
	}
	job, err = qName.First()
	if err != nil {
		return nil, fmt.Errorf("查询作业 \"%s\" 失败：%w", jobName, err)
	}
	return job, nil
}

func (mgr *AIOPsMgr) llmChatCompletion(c *gin.Context, systemPrompt, userPrompt string) (string, error) {
	if mgr.configService == nil {
		return "", fmt.Errorf("LLM 配置服务未初始化")
	}
	cfg, err := mgr.configService.GetLLMConfig(c)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.ModelName) == "" {
		return "", fmt.Errorf("LLM 配置不完整，请先在系统配置中设置 BaseURL 和 Model")
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), llmRequestTimeout)
	defer cancel()
	return prompts.CallLLMText(
		mgr.httpClient,
		ctx,
		cfg.GetChatCompletionURL(),
		cfg.APIKey,
		cfg.ModelName,
		systemPrompt,
		userPrompt,
	)
}

// ChatMessageLLM godoc
// @Summary LLM chat for AIOps
// @Description Chat with AIOps assistant in LLM mode, optionally with a target job context.
// @Tags aiops
// @Accept json
// @Produce json
// @Param request body ChatRequest true "Chat request"
// @Success 200 {object} resputil.Response[ChatResponse]
// @Router /api/v1/aiops/llmchat [post]
// @Router /api/v1/admin/aiops/llmchat [post]
func (mgr *AIOPsMgr) ChatMessageLLM(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := util.GetToken(c)
	jobName := extractJobNameFromMessage(req.Message, req.JobName)
	systemPrompt := strings.Join([]string{
		"你是 Crater 智能运维助手。",
		"请围绕 Crater 平台作业排障回答，重点结合：退出码、K8s 事件、作业状态、调度与资源配置。",
		"回答要求：",
		"1) 不要编造未提供的数据或日志；",
		"2) 先给结论，再给依据，再给下一步操作；",
		"3) 优先给用户可执行步骤，必要时标注需要管理员介入；",
		"4) 对 Exit 1/127/137 等常见问题给出针对性排查路径；",
		"5) 保持简洁，避免泛化空话。",
	}, "\n")
	userPrompt := req.Message
	if jobName != "" {
		job, err := mgr.findJobByInput(c, token, jobName)
		if err != nil {
			lookupErr := classifyAIOPsLookupError(err)
			data := map[string]any{
				"engine": "llm",
				"error":  lookupErr,
			}
			if errors.Is(err, errJobNotOwned) {
				data["adminHint"] = true
			}
			resputil.Success(c, ChatResponse{
				Message: fmt.Sprintf("无法定位作业 %q：%v", jobName, err),
				Type:    chatResponseTypeText,
				Data:    data,
			})
			return
		}
		diagnosis := PerformDiagnosis(job)
		diagJSON, _ := json.Marshal(diagnosis)
		userPrompt = fmt.Sprintf(
			"用户问题：%s\n\n作业名：%s\n作业状态：%s\n机器诊断结果(JSON)：%s\n\n请基于以上信息给出结论、可能原因和下一步排查建议。",
			req.Message,
			job.JobName,
			job.Status,
			string(diagJSON),
		)
	}
	reply, err := mgr.llmChatCompletion(c, systemPrompt, userPrompt)
	if err != nil {
		resputil.Success(c, ChatResponse{
			Message: fmt.Sprintf("LLM 模式调用失败：%v", err),
			Type:    chatResponseTypeText,
			Data: map[string]any{
				"engine": "llm",
			},
		})
		return
	}
	resputil.Success(c, ChatResponse{
		Message: reply,
		Type:    chatResponseTypeText,
		Data: map[string]any{
			"engine": "llm",
			"mode":   "llmchat",
		},
	})
}

// ChatMessage handles chatbot interactions with rule matching
//
// @Summary Rule-based chat for AIOps
// @Description Chat with AIOps assistant in rule-based mode, optionally with a target job.
// @Tags aiops
// @Accept json
// @Produce json
// @Param request body ChatRequest true "Chat request"
// @Success 200 {object} resputil.Response[ChatResponse]
// @Router /api/v1/aiops/chat [post]
// @Router /api/v1/admin/aiops/chat [post]
//
//nolint:gocyclo,funlen // Chat routing intentionally uses ordered keyword rules and preset responses.
func (mgr *AIOPsMgr) ChatMessage(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	message := strings.ToLower(strings.TrimSpace(req.Message))

	// Rule-based chat responses
	var resp ChatResponse

	jobName := extractJobNameFromMessage(req.Message, req.JobName)

	// Rule 1: Greeting
	if strings.Contains(message, "你好") || strings.Contains(message, "hello") || strings.Contains(message, "hi") ||
		message == "help" || strings.Contains(message, "帮助") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "👋 你好！我是 Crater 智能运维助手。\n\n我可以帮助你：\n• 分析作业失败原因\n• 查看系统健康状况\n• 提供故障排查建议\n\n💡 **重要提示**：遇到 Exit 1/127 等问题时，建议先检查代码、启动命令和日志。\n\n🔍 **快速开始**：\n• 问我 \"最近失败的原因是什么？\"\n• 问我 \"容器错误怎么办？\"\n• 告诉我作业名来诊断，如 \"作业:jpt-xxx-xxx\""
		resp.Type = chatResponseTypeText
		resputil.Success(c, resp)
		return
	}

	// Rule 2: Analyze specific job
	if jobName != "" {
		job, err := mgr.findJobByInput(c, token, jobName)
		if err != nil {
			lookupErr := classifyAIOPsLookupError(err)
			resp.Message = err.Error()
			resp.Type = chatResponseTypeText
			resp.Data = map[string]any{"error": lookupErr}
			if errors.Is(err, errJobNotOwned) {
				resp.Data = map[string]any{"adminHint": true, "error": lookupErr}
			}
			resputil.Success(c, resp)
			return
		}

		diagnosis := PerformDiagnosis(job)
		resp.Message = fmt.Sprintf("✅ 已完成对作业 %s 的诊断分析", jobName)
		resp.Type = "diagnosis"
		resp.Data = diagnosis
		resputil.Success(c, resp)
		return
	}

	// Rule 3: How to reduce failure rate (check before general failure queries)
	if strings.Contains(message, "降低") && (strings.Contains(message, "失败率") || strings.Contains(message, "失败")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "📉 **降低作业失败率的方法**：\n\n**1. 优先处理用户代码与运行环境问题：**\n• 提交前在本地复现启动命令\n• 检查依赖版本和镜像环境是否一致\n• 仔细核对文件路径与配置文件\n\n**2. 资源配置按需申请：**\n• 根据实际峰值设置 CPU/GPU/内存\n• 避免过度申请导致长期排队\n• 出现 OOM 时先定位峰值再调参\n\n**3. 保证镜像与仓库可用：**\n• 使用稳定基础镜像并固定关键依赖\n• 提前构建并验证镜像可拉取\n\n**4. 调度与存储配置前置检查：**\n• 提交前检查节点选择器/容忍度配置\n• 检查 PVC、StorageClass 与挂载路径\n\n💡 **建议**：先用“最近失败的原因”看你当前主要失败类型，再做针对性优化。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 4: Ask about Exit 127 (Command Not Found)
	if containsExitCode(message, "127") || strings.Contains(message, "命令未找到") ||
		strings.Contains(message, "command not found") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "⚙️ **命令未找到（Exit 127）解决方案**\n\n**典型表现：**\n• 启动时提示 `command not found`\n• 脚本/可执行文件路径不存在（`No such file or directory`）\n\n**常见原因：**\n\n**1. 命令或脚本写错**\n• `python` / `python3`、`pip` / `pip3` 混用\n• 启动命令中有拼写错误\n\n**2. 镜像里没有安装该命令**\n• Dockerfile 没装依赖或工具链\n• 运行镜像与本地测试镜像不一致\n\n**3. 路径与权限问题**\n• 可执行文件不在 PATH\n• 脚本没有执行权限（`chmod +x`）\n• 使用了相对路径但工作目录不对\n\n**🔍 排查步骤：**\n1. 在作业日志里定位第一条 `command not found` 或 `No such file or directory`\n2. 进入同版本镜像执行 `which <cmd>`、`echo $PATH`\n3. 用绝对路径替换入口命令并重试\n\n💡 **建议**：优先修正启动命令和镜像内容，通常不属于资源配额问题。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 5: Ask about Exit 1 (Container Error) - check before general failure queries
	if containsExitCode(message, "1") || strings.Contains(message, "容器错误") ||
		(strings.Contains(message, "容器") && strings.Contains(message, "怎么办")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "🐛 **容器错误（Exit 1）详解**\n\n**典型表现：**\n• 程序启动后很快退出并返回 Exit 1\n• 日志里出现异常堆栈（如 Python Traceback）\n\n**常见原因：**\n• **ImportError / ModuleNotFoundError**：依赖缺失或版本不匹配\n• **FileNotFoundError**：配置/数据文件路径错误\n• **SyntaxError / RuntimeError**：代码运行异常\n• **业务逻辑失败**：输入参数、权限、外部服务调用异常\n\n**🔍 如何排查：**\n1. 打开作业日志，从最后一个报错堆栈往上看\n2. 搜索关键词：`Traceback`、`Exception`、`ImportError`、`FileNotFoundError`\n3. 对照镜像环境检查依赖版本与启动参数\n\n**常见修复方法：**\n• 在镜像中补齐依赖并固定版本\n• 修正文件路径与配置挂载\n• 先在本地/开发环境复现同样命令再提交\n\n⚠️ **注意**：Exit 1 是“程序自身错误”信号，不等同于 Exit 127（命令不存在）或 Exit 137（OOM）。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 6: Ask about OOM
	if strings.Contains(message, "内存") || strings.Contains(message, "oom") ||
		(containsExitCode(message, "137") || strings.Contains(message, "溢出")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "💥 **内存溢出（OOM，常见为 Exit 137）处理建议**\n\n**先确认是否真 OOM：**\n• 终止原因里出现 `OOMKilled`\n• 日志里出现 `out of memory` / `Killed`\n\n**处理方法（按优先级）：**\n\n**1. 先止血：提高内存配额**\n• 适当提高 memory request/limit\n• 小步调整并观察峰值\n\n**2. 再治本：降低内存占用**\n• 减小 batch size\n• 分块加载数据，避免一次性全量读入\n• 及时释放中间变量\n\n**3. 排查泄漏与异常增长**\n• 观察迭代过程中内存是否持续攀升\n• 检查缓存、列表累积、循环引用\n\n💡 **提示**：Exit 137 也可能是被系统强杀，不一定都等于 OOM，需结合 `OOMKilled` 与日志一起判断。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 7: Ask about image pull errors
	if strings.Contains(message, "镜像") || strings.Contains(message, "image") || strings.Contains(message, "pull") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "🐳 **镜像拉取失败解决方案**\n\n**常见原因及解决方法：**\n\n**1. 镜像名称或标签错误**\n• 检查镜像名称拼写\n• 确认标签（tag）是否正确\n• 示例：`nginx:latest` vs `nginx:1.21`\n\n**2. 镜像不存在**\n• 确认镜像已推送到仓库\n• 使用 `docker images` 或 Web UI 检查\n• 检查镜像仓库地址是否正确\n\n**3. 认证失败**\n• 私有镜像需要配置 imagePullSecrets\n• 检查仓库凭证是否正确\n• 联系管理员配置凭证\n\n**4. 网络问题**\n• 检查集群与镜像仓库的连接\n• 可能需要配置代理\n• 联系管理员检查网络\n\n**5. 权限问题**\n• 确认有权限访问该镜像\n• 私有镜像需要正确的访问权限\n\n**🔍 如何调试：**\n• 在作业详情的「事件」标签页查看具体错误\n• 尝试在本地拉取镜像：`docker pull your-image:tag`\n\n💡 **提示**：查看事件详情可以看到具体的错误信息，如 \"unauthorized\" 或 \"not found\"。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 8: Ask about node eviction
	if strings.Contains(message, "节点驱逐") || strings.Contains(message, "evict") ||
		strings.Contains(message, "node eviction") || strings.Contains(message, "驱逐") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "🔧 **节点驱逐问题详解**\n\n节点驱逐通常**不是用户代码问题**，多与节点状态或集群资源有关。\n\n**什么是节点驱逐？**\n当节点出现压力或异常时，Kubernetes 会将该节点上的 Pod 驱逐并尝试重新调度。\n\n**常见原因：**\n\n**1. 节点故障或维护**\n• 节点宕机、重启或维护\n• 节点不可用\n\n**2. 节点资源压力**\n• 磁盘、内存压力过高\n• 节点被系统保护机制驱逐 Pod\n\n**3. 节点污点（Taint）与调度策略变化**\n• 节点被打污点，现有 Pod 不再满足策略\n\n**✅ 建议：**\n\n**对于用户：**\n• 先观察是否自动重调度成功\n• 若长时间未恢复，重新提交作业\n\n**对于管理员：**\n• 检查节点状态与事件：`kubectl get nodes` / `kubectl describe node <node-name>`\n• 修复异常节点或调整调度策略\n\n💡 **提示**：若频繁发生驱逐，应优先排查集群健康与容量。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 9: Ask about volume mount
	if strings.Contains(message, "存储") || strings.Contains(message, "挂载") || strings.Contains(message, "mount") ||
		strings.Contains(message, "volume") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "💾 **存储卷挂载失败解决方案**\n\n存储挂载失败通常是资源对象或权限配置问题。\n\n**常见原因：**\n\n**1. PVC/PV 异常**\n• PVC 未绑定成功\n• PV 容量、访问模式不匹配\n\n**2. StorageClass 或 CSI 组件问题**\n• StorageClass 名称错误\n• CSI 驱动异常或未就绪\n\n**3. 配置对象缺失**\n• 依赖的 ConfigMap/Secret 不存在\n• 挂载路径配置错误\n\n**4. 权限问题**\n• 目录权限与运行用户不匹配\n• 存储后端访问受限\n\n**🔍 如何排查：**\n• 在作业详情「事件」中查看 `FailedMount` 原因\n• 检查 PVC：`kubectl get pvc -n crater-workspace`\n• 查看详情：`kubectl describe pvc <pvc-name>`\n\n💡 **提示**：多数存储挂载问题需要管理员协同处理。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 10: Ask about scheduling or resource issues
	if strings.Contains(message, "调度") || strings.Contains(message, "资源不足") ||
		strings.Contains(message, "scheduling") || strings.Contains(message, "insufficient") ||
		strings.Contains(message, "排队") || strings.Contains(message, "pending") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "⏳ **调度和资源问题解决方案**\n\n**常见调度失败原因：**\n\n**1. 资源不足（最常见）**\n• 集群 CPU/GPU/内存不足\n• 其他作业占用了资源\n\n**解决方法：**\n• 降低资源请求（如果实际用不了那么多）\n• 等待其他作业完成释放资源\n• 选择资源充足的时段提交\n• 联系管理员扩容集群\n\n**2. 节点选择器不匹配**\n• NodeSelector 配置的标签在集群中不存在\n• 示例：`gpu-type: v100` 但集群没有 V100 GPU\n\n**解决方法：**\n• 检查节点标签配置\n• 修改或移除节点选择器\n• 联系管理员确认可用节点类型\n\n**3. 污点和容忍度不匹配**\n• 节点有污点（Taint），但作业没有对应的容忍度\n\n**解决方法：**\n• 添加容忍度配置\n• 选择没有污点的节点\n\n**4. 资源配额限制**\n• 达到命名空间的资源配额上限\n• 同时运行的作业数量限制\n\n**解决方法：**\n• 等待其他作业完成\n• 联系管理员调整配额\n\n**🔍 如何查看：**\n• 在作业详情的「事件」标签查看调度失败原因\n• 查看集群资源：`kubectl top nodes`\n\n💡 **提示**：作业长时间 Pending 时，查看事件可以了解具体原因。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 11: Ask about how to view logs
	if strings.Contains(message, "日志") || strings.Contains(message, "log") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "📝 **查看作业日志的方法**\n\n**方法1：Web 界面（推荐）**\n1. 进入作业列表页\n2. 点击作业名称进入详情页\n3. 切换到「日志」标签\n4. 实时查看日志输出\n\n**方法2：命令行**\n```bash\n# 查看 Pod 日志\nkubectl logs <pod-name> -n crater-workspace\n\n# 实时跟踪日志\nkubectl logs -f <pod-name> -n crater-workspace\n\n# 查看之前的日志（容器重启后）\nkubectl logs <pod-name> -n crater-workspace --previous\n```\n\n**💡 日志排查技巧：**\n\n**对于 Exit 1 错误（代码错误）：**\n• 搜索关键词：`Error`、`Exception`、`Failed`、`Traceback`\n• Python: 查找 `ImportError`、`FileNotFoundError`\n• CUDA: 查找 `CUDA out of memory`\n\n**对于 Exit 127 错误（命令未找到）：**\n• 查找 `command not found`\n• 查找 `No such file or directory`\n\n**对于其他错误：**\n• 从日志末尾往前看，找最后的错误信息\n• 注意 `WARNING` 和 `ERROR` 级别的消息\n\n⚠️ **注意**：Pod 删除后日志会丢失，建议及时查看或配置日志持久化。\n\n🔍 **看不懂日志？** 把关键错误信息告诉我，我可以帮你分析！"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 12: Ask about failure reasons or statistics (moved to end - check after specific error questions)
	if strings.Contains(message, "失败") || strings.Contains(message, "错误") || strings.Contains(message, "原因") ||
		strings.Contains(message, "为什么") || strings.Contains(message, "统计") {
		// Get recent failed jobs
		j := query.Job
		now := time.Now()
		lookback := now.AddDate(0, 0, -7)

		failedJobs, err := j.WithContext(c).
			Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
			Where(j.Status.Eq(string(batch.Failed))).
			Where(j.CreationTimestamp.Gte(lookback)).
			Limit(recentFailedJobsLimit).
			Find()

		if err != nil || len(failedJobs) == 0 {
			resp.Message = "✅ 很好！最近7天没有失败的作业。\n\n保持这个状态！如果以后遇到问题，随时来找我。"
			resp.Type = chatResponseTypeText
			resputil.Success(c, resp)
			return
		}

		// Categorize failures
		reasonCount := make(map[string]int)
		userIssueCount := 0
		for _, job := range failedJobs {
			reason := CategorizeFailure(job).TypeName
			reasonCount[reason]++
			// Count user code issues
			if reason == "ContainerError" || reason == "CommandNotFound" {
				userIssueCount++
			}
		}

		// Build response message
		msg := fmt.Sprintf("📊 **最近7天失败作业统计**（共 %d 个）：\n\n", len(failedJobs))

		// Sort by count
		type reasonPair struct {
			reason string
			count  int
		}
		var pairs []reasonPair
		for reason, count := range reasonCount {
			pairs = append(pairs, reasonPair{reason, count})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].count > pairs[j].count
		})

		// Top 5
		limit := chatTopReasonsLimit
		if len(pairs) < limit {
			limit = len(pairs)
		}
		for i := 0; i < limit; i++ {
			msg += fmt.Sprintf(
				"%d. **%s**: %d 次 (%.1f%%)\n",
				i+1,
				friendlyReasonName(pairs[i].reason),
				pairs[i].count,
				float64(pairs[i].count)/float64(len(failedJobs))*percentBase,
			)
		}

		// Add user issue warning if significant
		if userIssueCount > 0 {
			userIssueRate := float64(userIssueCount) / float64(len(failedJobs)) * percentBase
			if userIssueRate > userIssueWarnPercent {
				msg += fmt.Sprintf("\n⚠️ **重要**：其中 %.1f%% 可能是用户代码或环境问题（Exit 1/127），建议优先检查：\n• 容器日志中的错误信息\n• 依赖是否正确安装\n• 启动命令是否正确", userIssueRate)
			}
		}

		msg += "\n\n💡 **下一步**：\n• 输入作业名诊断具体问题，如 \"作业:jpt-xxx-xxx\"\n• 问我 \"容器错误怎么办\" 了解常见问题"

		resp.Message = msg
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Default: general help with specific examples
	resp.Message = "🤖 我是 Crater 智能运维助手，我可以帮你分析作业失败原因和排查问题。\n\n" +
		"**📋 查询失败统计**\n" +
		"试试问我：\n" +
		"• \"最近失败的原因是什么？\"\n" +
		"• \"为什么作业总是失败？\"\n" +
		"• \"如何降低失败率？\"\n\n" +
		"**🔍 诊断具体作业**\n" +
		"告诉我作业名称，我会帮你诊断：\n" +
		"• \"作业:jpt-username-251221-xxxxx\"\n" +
		"• \"分析作业 jpt-username-251221-xxxxx\"\n\n" +
		"**💡 常见问题（示例）**\n" +
		"• \"容器错误（Exit 1）怎么办？\"\n" +
		"• \"节点驱逐是什么原因？\"\n" +
		"• \"存储挂载失败怎么解决？\"\n" +
		"• \"命令未找到（Exit 127）\"\n" +
		"• \"内存溢出（OOM）如何处理？\"\n\n" +
		"**🛠️ 其他常见问题**\n" +
		"• \"如何查看日志？\"\n" +
		"• \"镜像拉取失败怎么办？\"\n" +
		"• \"调度失败是什么原因？\"\n" +
		"• \"资源不足怎么办？\"\n\n" +
		"💬 直接输入你的问题，我会尽力帮助你！"
	resp.Type = chatResponseTypeText
	resputil.Success(c, resp)
}

// friendlyReasonName converts internal reason names to user-friendly names
func friendlyReasonName(reason string) string {
	mapping := map[string]string{
		"OOMKilled":                       "内存溢出",
		"ImagePullError":                  "镜像拉取失败",
		"SchedulingInsufficientResources": "资源不足",
		"SchedulingNodeSelectorMismatch":  "节点选择器不匹配",
		"SchedulingTaintMismatch":         "节点污点不匹配",
		"SchedulingFailed":                "调度失败",
		"CrashLoopBackOff":                "容器崩溃循环",
		"Evicted":                         "节点驱逐",
		"ContainerError":                  "容器错误",
		"CommandNotFound":                 "命令未找到",
		"GracefulTermination":             "优雅终止",
		"SegmentationFault":               "段错误",
		"VolumeMountFailed":               "存储卷挂载失败",
		"JobDeadlineExceeded":             "作业超时",
		"JobAbortedOrTerminated":          "作业被中止",
		"UnknownFailure":                  "未知错误",
	}
	if friendly, ok := mapping[reason]; ok {
		return friendly
	}
	return reason
}
