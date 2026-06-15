package api

// HTTP path 常量集中维护，业务方法禁止手写 "/api/..." 片段。

const (
	AuthPrefix          = "/api/auth"
	ImagesPrefix        = "/api/v1/images"
	ModelDownloadPrefix = "/api/v1/model-download/models"
	NodesPrefix         = "/api/v1/nodes"
	VCJobsPrefix        = "/api/v1/vcjobs"
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
