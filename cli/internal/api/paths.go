package api

// HTTP path 常量集中维护，业务方法禁止手写 "/api/..." 片段。

const (
	AuthPrefix          = "/api/auth"
	ModelDownloadPrefix = "/api/v1/model-download/models"
)

// AuthLoginPath 为登录接口路径（含模块前缀）。
const AuthLoginPath = AuthPrefix + "/login"

// ModelDownloadCreatePath creates a model or dataset download task.
const ModelDownloadCreatePath = ModelDownloadPrefix + "/download"

// ModelDownloadListPath lists model and dataset download tasks.
const ModelDownloadListPath = ModelDownloadPrefix + "/downloads"
