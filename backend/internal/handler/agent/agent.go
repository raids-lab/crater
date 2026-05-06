package agent

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/packer"
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
	agentToolListImageBuilds   = "list_image_builds"
	agentToolGetImageBuild     = "get_image_build_detail"
	agentToolGetImageAccess    = "get_image_access_detail"
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
	agentToolListStoragePVCs   = "list_storage_pvcs"
	agentToolGetPVCDetail      = "get_pvc_detail"
	agentToolGetPVCEvents      = "get_pvc_events"
	agentToolInspectJobStorage = "inspect_job_storage"
	agentToolStorageCapacity   = "get_storage_capacity_overview"
	agentToolNodeNetwork       = "get_node_network_summary"
	agentToolDiagnoseJobNet    = "diagnose_distributed_job_network"
	agentToolWebSearch         = "web_search"
	agentToolFetchURL          = "fetch_url"
	agentToolSandboxGrep       = "sandbox_grep"
	agentToolRuntimeSummary    = "get_agent_runtime_summary"
	agentToolK8sListNodes      = "k8s_list_nodes"
	agentToolK8sListPods       = "k8s_list_pods"
	agentToolK8sGetEvents      = "k8s_get_events"
	agentToolK8sDescribe       = "k8s_describe_resource"
	agentToolK8sPodLogs        = "k8s_get_pod_logs"
	agentToolK8sGetService     = "k8s_get_service"
	agentToolK8sGetEndpoints   = "k8s_get_endpoints"
	agentToolK8sGetIngress     = "k8s_get_ingress"
	agentToolPromQuery         = "prometheus_query"
	agentToolHarborCheck       = "harbor_check"

	// AIOps audit tools (admin)
	toolGetLatestAuditReport = "get_latest_audit_report"
	toolListAuditItems       = "list_audit_items"
	toolSaveAuditReport      = "save_audit_report"
	toolMarkAuditHandled     = "mark_audit_handled"
	toolBatchStopJobs        = "batch_stop_jobs"
	toolNotifyJobOwner       = "notify_job_owner"

	// Approval tools
	toolGetApprovalHistory = "get_approval_history"

	// Write tools that require user confirmation before execution
	agentToolResubmitJob      = "resubmit_job"
	agentToolStopJob          = "stop_job"
	agentToolDeleteJob        = "delete_job"
	agentToolCreateJupyter    = "create_jupyter_job"
	agentToolCreateWebIDE     = "create_webide_job"
	agentToolCreateTrain      = "create_training_job"
	agentToolCreateCustom     = "create_custom_job"
	agentToolCreatePytorch    = "create_pytorch_job"
	agentToolCreateTensorflow = "create_tensorflow_job"
	agentToolCreateImage      = "create_image_build"
	agentToolManageBuild      = "manage_image_build"
	agentToolRegisterImage    = "register_external_image"
	agentToolManageAccess     = "manage_image_access"
	agentToolCordonNode       = "cordon_node"
	agentToolUncordonNode     = "uncordon_node"
	agentToolDrainNode        = "drain_node"
	agentToolDeletePod        = "delete_pod"
	agentToolRestartWL        = "restart_workload"
	agentToolK8sScaleWL       = "k8s_scale_workload"
	agentToolK8sLabelNode     = "k8s_label_node"
	agentToolK8sTaintNode     = "k8s_taint_node"
	agentToolRunKubectl       = "run_kubectl"
	agentToolAdminCommand     = "execute_admin_command"
)

type AgentMgr struct {
	name          string
	client        client.Client
	kubeClient    kubernetes.Interface
	nodeClient    *crclient.NodeClient
	promClient    monitor.PrometheusInterface
	agentService  *service.AgentService
	jobSubmitter  handler.JobMutationSubmitter
	imagePacker   packer.ImagePackerInterface
	imageRegistry imageregistry.ImageRegistryInterface
	httpClient    *http.Client

	localToolCatalogMu        sync.RWMutex
	localToolCatalog          []agentLocalToolCatalogEntry
	localToolCatalogExpiresAt time.Time
}

func NewAgentMgr(conf *handler.RegisterConfig) handler.Manager {
	return &AgentMgr{
		name:          "agent",
		client:        conf.Client,
		kubeClient:    conf.KubeClient,
		nodeClient:    &crclient.NodeClient{Client: conf.Client, KubeClient: conf.KubeClient, PrometheusClient: conf.PrometheusClient},
		promClient:    conf.PrometheusClient,
		agentService:  service.NewAgentService(),
		jobSubmitter:  handler.NewJobMutationSubmitter(conf),
		imagePacker:   conf.ImagePacker,
		imageRegistry: conf.ImageRegistry,
		// Do not use Client.Timeout for agent SSE streaming. The Python side may
		// legitimately take minutes for multi-agent runs, and the per-request
		// context in python_proxy.go provides the actual upper bound.
		httpClient: &http.Client{},
	}
}

func (mgr *AgentMgr) GetName() string { return mgr.name }

func (mgr *AgentMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/tools/execute", mgr.ExecuteTool)
	g.GET("/k8s-ownership", mgr.K8sOwnershipCheck)
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

	// Feedback
	g.PUT("/feedbacks", mgr.UpsertFeedback)
	g.POST("/feedbacks/submit", mgr.SubmitFeedback)
	g.POST("/feedbacks/quick-submit", mgr.QuickSubmitFeedback)
	g.PUT("/feedbacks/enrich", mgr.EnrichFeedback)
	g.GET("/feedbacks", mgr.ListFeedbacks)
}

func (mgr *AgentMgr) RegisterInternal(g *gin.RouterGroup) {
	g.POST("/quality-evals", mgr.ReceiveQualityEvalResult)
	g.GET("/sessions/:sessionId/messages", mgr.GetInternalSessionMessages)
	g.GET("/sessions/:sessionId/tool-calls", mgr.GetInternalSessionToolCalls)
	g.GET("/sessions/:sessionId/turns", mgr.GetInternalSessionTurns)
	g.GET("/turns/:turnId/tool-calls", mgr.GetInternalTurnToolCalls)
	g.GET("/turns/:turnId/events", mgr.GetInternalTurnEvents)
}

func (mgr *AgentMgr) RegisterAdmin(g *gin.RouterGroup) {
	// Session / turn audit (group is already /api/v1/admin/agent, do NOT re-add /agent)
	g.GET("/sessions", mgr.ListAdminSessions)
	g.GET("/sessions/:sessionId/detail", mgr.GetAdminSessionDetail)
	g.GET("/sessions/:sessionId/messages", mgr.GetAdminSessionMessages)
	g.GET("/sessions/:sessionId/tool-calls", mgr.GetAdminSessionToolCalls)
	g.GET("/sessions/:sessionId/turns", mgr.GetAdminSessionTurns)
	g.GET("/sessions/:sessionId/feedbacks", mgr.ListAdminSessionFeedbacks)
	g.GET("/turns/:turnId/events", mgr.GetAdminTurnEvents)

	// Manual quality eval trigger (added in Task 3)
	g.POST("/sessions/:sessionId/trigger-eval", mgr.TriggerSessionQualityEval)

	// Ops reports
	g.GET("/ops-reports", mgr.ListOpsReports)
	g.GET("/ops-reports/latest", mgr.GetLatestOpsReport)
	g.GET("/ops-reports/:id", mgr.GetOpsReportDetail)
	g.GET("/ops-reports/:id/items", mgr.GetOpsReportItems)

	// Feedback stats
	g.GET("/feedbacks/stats", mgr.GetFeedbackStats)

	// Quality eval records
	g.GET("/quality-evals", mgr.ListQualityEvals)
}
