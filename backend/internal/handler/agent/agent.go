package agent

import (
	"net/http"
	"time"

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
	agentLLMPreflightTimeout     = 8 * time.Second

	agentToolStatusSuccess              = "success"
	agentToolStatusError                = "error"
	agentToolStatusConfirmationRequired = "confirmation_required"
	agentToolStatusAwaitConfirm         = "await_confirm"
	agentToolStatusAwaitingConfirmation = "awaiting_confirmation"

	agentTurnStatusRunning   = "running"
	agentTurnStatusCompleted = "completed"
	agentTurnStatusFailed    = "failed"
	agentTurnStatusCancelled = "cancelled" //nolint:misspell // API/frontend status currently uses British spelling.
	agentToolStatusRejected  = "rejected"

	agentRoleSingleAgent        = "single_agent"
	agentRoleCoordinator        = "coordinator"
	agentMessageRoleAssistant   = "assistant"
	agentSessionSourceUser      = "user"
	agentSessionSourceChat      = "chat"
	agentSessionSourceAdmin     = "admin"
	agentSessionSourceOpsAudit  = "ops_audit"
	agentSessionSourceSystem    = "system"
	agentToolAuditSourceBackend = "backend"

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
	agentToolListGPUModels     = "list_available_gpu_models"
	agentToolCheckQuota        = "check_quota"
	agentToolListUserJobs      = "list_user_jobs"

	// New read-only tools
	agentToolGetJobTemplates   = "get_job_templates"
	agentToolResourceRecommend = "get_resource_recommendation"

	// Write tools that require user confirmation before execution
	agentToolResubmitJob      = "resubmit_job"
	agentToolStopJob          = "stop_job"
	agentToolDeleteJob        = "delete_job"
	agentToolCreateJupyter    = "create_jupyter_job"
	agentToolCreateWebIDE     = "create_webide_job"
	agentToolCreateCustom     = "create_custom_job"
	agentToolCreatePytorch    = "create_pytorch_job"
	agentToolCreateTensorflow = "create_tensorflow_job"
)

type AgentMgr struct {
	name          string
	client        client.Client
	kubeClient    kubernetes.Interface
	nodeClient    *crclient.NodeClient
	promClient    monitor.PrometheusInterface
	agentService  *service.AgentService
	configService *service.ConfigService
	jobSubmitter  handler.JobMutationSubmitter
	jobReader     handler.JobInsightReader
	imageReader   handler.ImageInsightReader
	httpClient    *http.Client
}

func NewAgentMgr(conf *handler.RegisterConfig) handler.Manager {
	return &AgentMgr{
		name:          "agent",
		client:        conf.Client,
		kubeClient:    conf.KubeClient,
		nodeClient:    &crclient.NodeClient{Client: conf.Client, KubeClient: conf.KubeClient, PrometheusClient: conf.PrometheusClient},
		promClient:    conf.PrometheusClient,
		agentService:  service.NewAgentService(),
		configService: conf.ConfigService,
		jobSubmitter:  handler.NewJobMutationSubmitter(conf),
		jobReader:     handler.NewJobInsightReader(conf),
		imageReader:   handler.NewImageInsightReader(conf),
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
	g.POST("/ask/stream", mgr.Ask)
	g.POST("/chat/confirm", mgr.ConfirmToolExecution)
	g.POST("/chat/resume", mgr.ResumeAfterConfirmation)
	g.GET("/config-summary", mgr.GetAgentConfigSummary)
	g.GET("/sessions", mgr.ListSessions)
	g.PUT("/sessions/:sessionId/pin", mgr.UpdateSessionPin)
	g.PUT("/sessions/:sessionId/title", mgr.UpdateSessionTitle)
	g.DELETE("/sessions/:sessionId", mgr.DeleteSession)
	g.GET("/sessions/:sessionId/messages", mgr.GetSessionMessages)
	g.GET("/sessions/:sessionId/tool-calls", mgr.GetSessionToolCalls)
	g.GET("/sessions/:sessionId/turns", mgr.GetSessionTurns)
	g.GET("/turns/:turnId/events", mgr.GetTurnEvents)
	g.POST("/chat/parameter-update", mgr.HandleParameterUpdate)

	// Feedback
	g.PUT("/feedbacks", mgr.UpsertFeedback)
	g.POST("/feedbacks/submit", mgr.SubmitFeedback)
	g.POST("/feedbacks/quick-submit", mgr.QuickSubmitFeedback)
	g.PUT("/feedbacks/enrich", mgr.EnrichFeedback)
	g.GET("/feedbacks", mgr.ListFeedbacks)
}

func (mgr *AgentMgr) RegisterInternal(_ *gin.RouterGroup) {}

func (mgr *AgentMgr) RegisterAdmin(_ *gin.RouterGroup) {}
