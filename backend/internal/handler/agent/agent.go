package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
)

//nolint:gochecknoinits // Handler managers are registered during package initialization.
func init() {
	handler.Registers = append(handler.Registers, NewAgentMgr)
}

const (
	agentDefaultPythonServiceURL = "http://localhost:8000"
	agentChatMessageMaxRunes     = 4000

	agentToolStatusSuccess              = "success"
	agentToolStatusError                = "error"
	agentToolStatusConfirmationRequired = "confirmation_required"
	agentToolStatusAwaitConfirm         = "await_confirm"

	// Read-only tool names
	agentToolGetJobDetail      = "get_job_detail"
	agentToolGetJobEvents      = "get_job_events"
	agentToolGetJobLogs        = "get_job_logs"
	agentToolDiagnoseJob       = "diagnose_job"
	agentToolGetDiagnosticCtx  = "get_diagnostic_context"
	agentToolSearchSimilarFail = "search_similar_failures"
	agentToolQueryJobMetrics   = "query_job_metrics"
	agentToolAnalyzeQueue      = "analyze_queue_status"
	agentToolRealtimeCapacity  = "get_realtime_capacity"
	agentToolListImages        = "list_available_images"
	agentToolListCudaBase      = "list_cuda_base_images"
	agentToolListGPUModels     = "list_available_gpu_models"
	agentToolRecommendImages   = "recommend_training_images"
	agentToolCheckQuota        = "check_quota"
	agentToolGetHealthOverview = "get_health_overview"
	agentToolListUserJobs      = "list_user_jobs"
	agentToolGetClusterHealth  = "get_cluster_health_overview"
	agentToolListClusterJobs   = "list_cluster_jobs"
	agentToolListClusterNodes  = "list_cluster_nodes"

	// New read-only tools
	agentToolDetectIdleJobs    = "detect_idle_jobs"
	agentToolGetJobTemplates   = "get_job_templates"
	agentToolGetFailureStats   = "get_failure_statistics"
	agentToolGetClusterReport  = "get_cluster_health_report"
	agentToolResourceRecommend = "get_resource_recommendation"
	agentToolGetNodeDetail     = "get_node_detail"
	agentToolGetAdminOpsReport = "get_admin_ops_report"

	// AIOps audit tools (admin)
	toolGetLatestAuditReport = "get_latest_audit_report"
	toolListAuditItems       = "list_audit_items"
	toolSaveAuditReport      = "save_audit_report"
	toolMarkAuditHandled     = "mark_audit_handled"
	toolBatchStopJobs        = "batch_stop_jobs"
	toolNotifyJobOwner       = "notify_job_owner"

	// Write tools that require user confirmation before execution
	agentToolResubmitJob   = "resubmit_job"
	agentToolStopJob       = "stop_job"
	agentToolDeleteJob     = "delete_job"
	agentToolCreateJupyter = "create_jupyter_job"
	agentToolCreateTrain   = "create_training_job"
)

type AgentMgr struct {
	name         string
	client       client.Client
	kubeClient   kubernetes.Interface
	nodeClient   *crclient.NodeClient
	promClient   monitor.PrometheusInterface
	agentService *service.AgentService
	jobSubmitter handler.JobMutationSubmitter
	httpClient   *http.Client
}

func NewAgentMgr(conf *handler.RegisterConfig) handler.Manager {
	return &AgentMgr{
		name:         "agent",
		client:       conf.Client,
		kubeClient:   conf.KubeClient,
		nodeClient:   &crclient.NodeClient{Client: conf.Client, KubeClient: conf.KubeClient, PrometheusClient: conf.PrometheusClient},
		promClient:   conf.PrometheusClient,
		agentService: service.NewAgentService(),
		jobSubmitter: handler.NewJobMutationSubmitter(conf),
		// Do not use Client.Timeout for agent SSE streaming. The Python side may
		// legitimately take minutes for multi-agent runs, and the per-request
		// context in python_proxy.go provides the actual upper bound.
		httpClient: &http.Client{},
	}
}

func (mgr *AgentMgr) GetName() string { return mgr.name }

func (mgr *AgentMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/tools/execute", mgr.ExecuteTool)
}

func (mgr *AgentMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/chat", mgr.Chat)
	g.POST("/chat/confirm", mgr.ConfirmToolExecution)
	g.POST("/chat/resume", mgr.ResumeAfterConfirmation)
	g.GET("/config-summary", mgr.GetAgentConfigSummary)
	g.GET("/sessions", mgr.ListSessions)
	g.PUT("/sessions/:sessionId/pin", mgr.UpdateSessionPin)
	g.DELETE("/sessions/:sessionId", mgr.DeleteSession)
	g.GET("/sessions/:sessionId/messages", mgr.GetSessionMessages)
	g.GET("/sessions/:sessionId/tool-calls", mgr.GetSessionToolCalls)
	g.GET("/sessions/:sessionId/turns", mgr.GetSessionTurns)
	g.GET("/turns/:turnId/events", mgr.GetTurnEvents)
	g.POST("/chat/parameter-update", mgr.HandleParameterUpdate)
}

func (mgr *AgentMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/ops-reports", mgr.ListOpsReports)
	g.GET("/ops-reports/latest", mgr.GetLatestOpsReport)
	g.GET("/ops-reports/:id", mgr.GetOpsReportDetail)
	g.GET("/ops-reports/:id/items", mgr.GetOpsReportItems)
}
