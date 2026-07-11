package api

// HTTP path 常量集中维护，业务方法禁止手写 "/api/..." 片段。

const (
	AuthPrefix          = "/api/auth"
	AccountsPrefix      = "/api/v1/accounts"
	AdminAccountsPrefix = "/api/v1/admin/accounts"
	ApprovalOrderPrefix = "/api/v1/approvalorder"
	AdminApprovalPrefix = "/api/v1/admin/approvalorder"
	ContextPrefix       = "/api/v1/context"
	DatasetPrefix       = "/api/v1/dataset"
	AdminDatasetPrefix  = "/api/v1/admin/dataset"
	ImagesPrefix        = "/api/v1/images"
	AdminImagesPrefix   = "/api/v1/admin/images"
	JobTemplatePrefix   = "/api/v1/jobtemplate"
	ModelDownloadPrefix = "/api/v1/model-download/models"
	ModelDownloadRoot   = "/api/v1/model-download"
	AdminModelDLPfx     = "/api/v1/admin/model-download"
	NamespacesPrefix    = "/api/v1/namespaces"
	NodesPrefix         = "/api/v1/nodes"
	ResourcesPrefix     = "/api/v1/resources"
	AdminResourcesPfx   = "/api/v1/admin/resources"
	AdminOperationsPfx  = "/api/v1/admin/operations"
	AdminOperationLogs  = "/api/v1/admin/operation-logs"
	AdminQueueQuotasPfx = "/api/v1/admin/queue-quotas"
	AdminGPUAnalysisPfx = "/api/v1/admin/gpu-analysis"
	SystemConfigPrefix  = "/api/v1/system-config"
	AdminSysConfigPfx   = "/api/v1/admin/system-config"
	UsersPrefix         = "/api/v1/users"
	AdminUsersPrefix    = "/api/v1/admin/users"
	AIJobsPrefix        = "/api/v1/aijobs"
	SPJobsPrefix        = "/api/v1/spjobs"
	VCJobsPrefix        = "/api/v1/vcjobs"
	AdminVCJobsPrefix   = "/api/v1/admin/vcjobs"
)

// AuthLoginPath 为登录接口路径（含模块前缀）。
const AuthLoginPath = AuthPrefix + "/login"

// ModelDownloadCreatePath creates a model or dataset download task.
const ModelDownloadCreatePath = ModelDownloadPrefix + "/download"

// ModelDownloadListPath lists model and dataset download tasks.
const ModelDownloadListPath = ModelDownloadPrefix + "/downloads"

const (
	ImageAvailablePath = ImagesPrefix + "/available"
	ImageListPath      = ImagesPrefix + "/image"
	NodeListPath       = NodesPrefix
	VCJobListPath      = VCJobsPrefix
)
