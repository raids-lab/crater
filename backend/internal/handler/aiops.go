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
		reason := categorizeFailure(job).typeName
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
		reason := categorizeFailure(job).typeName
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
// @Success 200 {object} resputil.Response
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
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Perform diagnosis
	diagnosis := performDiagnosis(job)
	resputil.Success(c, diagnosis)
}

// performDiagnosis applies diagnostic rules to a job
//
//nolint:gocyclo // Rule-driven diagnosis intentionally keeps category handling in one switch.
func performDiagnosis(job *model.Job) DiagnosisResp {
	resp := DiagnosisResp{
		JobName: job.JobName,
		Status:  string(job.Status),
	}

	// Rule-based diagnosis
	result := categorizeFailure(job)
	resp.Category = result.typeName

	// Apply diagnostic rules based on category
	switch result.typeName {
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

	case "CrashLoopBackOff":
		resp.Diagnosis = "容器持续崩溃重启"
		resp.Solution = "建议：1) 查看容器日志确定崩溃原因；2) 检查启动命令；3) 检查配置文件；4) 可能是资源不足"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit

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
// @Success 200 {object} resputil.Response
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
			data := map[string]any{
				"engine": "llm",
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
		diagnosis := performDiagnosis(job)
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
// @Success 200 {object} resputil.Response
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
		resp.Message = "👋 你好！我是 Crater 智能运维助手。\n\n我可以帮助你：\n• 分析作业失败原因\n• 查看系统健康状况\n• 提供故障排查建议\n\n💡 **重要提示**：根据统计，37.7% 的失败是由用户代码或环境配置问题导致的。遇到 Exit 1/127 错误时，建议先检查代码和日志。\n\n🔍 **快速开始**：\n• 问我 \"最近失败的原因是什么？\"\n• 问我 \"容器错误怎么办？\"\n• 告诉我作业名来诊断，如 \"作业:jpt-xxx-xxx\""
		resp.Type = chatResponseTypeText
		resputil.Success(c, resp)
		return
	}

	// Rule 2: Analyze specific job
	if jobName != "" {
		job, err := mgr.findJobByInput(c, token, jobName)
		if err != nil {
			resp.Message = err.Error()
			resp.Type = chatResponseTypeText
			if errors.Is(err, errJobNotOwned) {
				resp.Data = map[string]any{"adminHint": true}
			}
			resputil.Success(c, resp)
			return
		}

		diagnosis := performDiagnosis(job)
		resp.Message = fmt.Sprintf("✅ 已完成对作业 %s 的诊断分析", jobName)
		resp.Type = "diagnosis"
		resp.Data = diagnosis
		resputil.Success(c, resp)
		return
	}

	// Rule 3: How to reduce failure rate (check before general failure queries)
	if strings.Contains(message, "降低") && (strings.Contains(message, "失败率") || strings.Contains(message, "失败")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "📉 **降低作业失败率的方法**：\n\n**1. 针对用户代码问题（占37.7%）：**\n• 提交前在本地测试代码\n• 检查所有依赖是否在镜像中\n• 使用正确的 Python 版本和包版本\n• 仔细检查文件路径\n\n**2. 针对资源问题：**\n• 根据实际需求设置合理的资源配额\n• 避免过度申请资源导致排队\n• OOM 时增加内存或优化代码\n\n**3. 针对镜像问题：**\n• 使用稳定的基础镜像\n• 提前构建并测试镜像\n• 检查镜像仓库连接\n\n**4. 针对配置问题：**\n• 仔细检查作业配置\n• 确认存储卷路径正确\n• 检查节点选择器配置\n\n💡 **建议**：先通过 \"最近失败的原因\" 查看你的主要失败类型，然后针对性优化。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 4: Ask about Exit 1 (Container Error) - check before general failure queries
	if strings.Contains(message, "exit 1") || strings.Contains(message, "容器错误") ||
		(strings.Contains(message, "容器") && strings.Contains(message, "怎么办")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "🐛 **容器错误（Exit 1）详解**\n\n容器错误占失败作业的 **24.1%**，是最常见的失败类型。\n\n**常见原因：**\n• **ImportError**: Python 依赖包缺失或版本不匹配\n• **FileNotFoundError**: 找不到数据文件或脚本\n• **SyntaxError**: 代码语法错误\n• **CUDA out of memory**: GPU 显存不足\n• **逻辑错误**: 代码运行时异常\n\n**🔍 如何排查：**\n1. **查看容器日志**（作业详情页 → 日志标签）\n2. **搜索关键错误**：\n   - ImportError / ModuleNotFoundError\n   - FileNotFoundError / No such file\n   - CUDA out of memory\n   - Traceback（Python 错误堆栈）\n3. **常见修复方法**：\n   - 在镜像中安装缺失的依赖\n   - 检查文件路径是否正确\n   - 修复代码语法错误\n   - 减小 batch size 或降低模型精度\n\n⚠️ **重要**：这类问题通常需要查看日志并修改代码或镜像配置。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 5: Ask about OOM
	if strings.Contains(message, "内存") || strings.Contains(message, "oom") ||
		(strings.Contains(message, "exit 137") || strings.Contains(message, "溢出")) {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "💥 **内存溢出（OOM, Exit 137）解决方案**\n\n内存溢出占失败作业的 **4.2%**。\n\n**解决方法（按推荐顺序）：**\n\n**1. 增加内存配额**（最快）\n• 在作业配置中增加 memory 请求和限制\n• 建议先尝试增加 50%\n• 例如：从 8Gi 增加到 12Gi\n\n**2. 优化代码**（治本）\n• 减小 batch size（最有效）\n• 使用梯度累积代替大 batch\n• 及时释放不用的变量（del variable）\n• 使用生成器而非一次性加载全部数据\n• 使用混合精度训练（FP16）\n\n**3. 检查内存泄漏**\n• 使用 memory_profiler 工具分析\n• 检查是否有循环引用\n• 确认是否正确关闭文件\n\n**4. 使用内存映射**\n• 对于大数据集，使用 mmap\n• PyTorch 可用 DataLoader 的 pin_memory\n\n💡 **快速测试**：先临时增加内存配额，如果还失败则需要优化代码。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 6: Ask about Exit 127 (Command Not Found)
	if strings.Contains(message, "exit 127") || strings.Contains(message, "命令未找到") ||
		strings.Contains(message, "command not found") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "⚙️ **命令未找到（Exit 127）解决方案**\n\n命令未找到占失败作业的 **9.4%**。\n\n**常见原因：**\n\n**1. 命令拼写错误**\n• `python` vs `python3`\n• `pip` vs `pip3`\n• 检查命令是否有拼写错误\n\n**2. 命令未安装**\n• 镜像中缺少该命令\n• 需要在 Dockerfile 中安装\n• 例如：`apt-get install xxx` 或 `pip install xxx`\n\n**3. PATH 环境变量问题**\n• 命令不在 PATH 中\n• 可以使用绝对路径：`/usr/local/bin/python3`\n\n**4. 启动脚本路径错误**\n• 检查脚本文件是否存在\n• 使用绝对路径而非相对路径\n\n**🔍 如何调试：**\n```bash\n# 在本地测试镜像\ndocker run -it your-image:tag /bin/bash\nwhich python3  # 检查命令是否存在\necho $PATH     # 查看 PATH 变量\n```\n\n⚠️ **提示**：这通常是镜像环境配置问题，需要修改 Dockerfile 或启动命令。"
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
		resp.Message = "🔧 **节点驱逐问题详解**\n\n节点驱逐占失败作业的 **23.6%**，通常**不是用户问题**。\n\n**什么是节点驱逐？**\n当节点出现问题时，Kubernetes 会将该节点上的 Pod 驱逐（删除），并尝试在其他节点上重新调度。\n\n**常见原因：**\n\n**1. 节点故障或维护**\n• 节点宕机或重启\n• 管理员进行节点维护\n• 节点资源耗尽\n\n**2. 节点资源压力**\n• 节点磁盘空间不足\n• 节点内存压力过大\n• 节点进入不可调度状态\n\n**3. 节点污点（Taint）**\n• 管理员给节点添加了污点\n• 节点被标记为不可调度\n\n**✅ 解决方法：**\n\n**对于用户：**\n• **等待重新调度**：系统通常会自动重新调度到其他节点\n• **重新提交作业**：如果长时间未恢复\n• **配置容忍度**：允许作业在有污点的节点上运行\n\n**对于管理员：**\n• 检查节点状态：`kubectl get nodes`\n• 查看节点事件：`kubectl describe node <node-name>`\n• 修复或替换故障节点\n\n💡 **重要**：这是基础设施问题，无需自查代码。如果频繁发生，请联系管理员检查集群健康状况。"
		resp.Type = chatResponseTypeSuggestion
		resputil.Success(c, resp)
		return
	}

	// Rule 9: Ask about volume mount
	if strings.Contains(message, "存储") || strings.Contains(message, "挂载") || strings.Contains(message, "mount") ||
		strings.Contains(message, "volume") {
		//nolint:lll // Keep the preset response as one literal to preserve exact markdown output.
		resp.Message = "💾 **存储卷挂载失败解决方案**\n\n存储卷挂载失败占 **18.4%**，通常是配置问题。\n\n**常见原因：**\n\n**1. ConfigMap 不存在**\n• 作业依赖的 ConfigMap 未创建\n• 常见：`custom-start-configmap` 缺失\n• **解决**：联系管理员创建或检查配置\n\n**2. CSI 驱动未安装**\n• 存储驱动（如 rook-ceph）未正确安装\n• **解决**：联系管理员检查存储驱动\n\n**3. LXCFS 挂载路径问题**\n• LXCFS 服务未运行\n• 挂载路径不存在\n• **解决**：联系管理员检查节点 LXCFS 状态\n\n**4. 存储卷权限问题**\n• 没有权限访问存储卷\n• PVC（PersistentVolumeClaim）状态异常\n• **解决**：检查存储卷权限配置\n\n**5. 存储类（StorageClass）问题**\n• 指定的 StorageClass 不存在\n• StorageClass 配置错误\n• **解决**：确认可用的 StorageClass\n\n**🔍 如何调试：**\n• 在作业详情的「事件」标签查看具体错误\n• 检查 PVC 状态：`kubectl get pvc -n crater-workspace`\n• 查看存储卷详情：`kubectl describe pvc <pvc-name>`\n\n💡 **提示**：这类问题通常需要管理员协助解决，用户一般无法自行修复。请将错误信息提供给管理员。"
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
			reason := categorizeFailure(job).typeName
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
		"**💡 常见问题（Top 5）**\n" +
		"• \"容器错误（Exit 1）怎么办？\" - 占24.1%\n" +
		"• \"节点驱逐是什么原因？\" - 占23.6%\n" +
		"• \"存储挂载失败怎么解决？\" - 占18.4%\n" +
		"• \"命令未找到（Exit 127）\" - 占9.4%\n" +
		"• \"内存溢出（OOM）如何处理？\" - 占4.2%\n\n" +
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
