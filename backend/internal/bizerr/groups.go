package bizerr

// badRequestGroup 400xx - 客户端请求错误
// 场景：客户端发送的请求本身有问题，修改请求参数后可以重试。
// 区别：400 侧重于“输入验证”，409 侧重于“业务状态冲突”。
type badRequestGroup struct {
	Base *BizError
	// InvalidRequest: 无法解析请求体（如 JSON 格式错误、Base64 编码非法等）
	InvalidRequest BizCode `code:"40001"`
	// ParameterError: 结构化校验失败（如 Validator 标签触发的 email 格式错误、字符串长度不足、数值范围不对）
	ParameterError BizCode `code:"40002"`
	// MissingParameter: 缺少 API 文档中定义的必传字段
	MissingParameter BizCode `code:"40003"`
}

// authGroup 401xx - 身份验证相关
// 场景：不知道你是谁，或者你的身份凭证已失效。
type authGroup struct {
	Base *BizError
	// TokenExpired: JWT 或 Session 已过期，需要重新登录
	TokenExpired BizCode `code:"40101"`
	// TokenInvalid: Token 签名错误、被篡改或格式完全不对
	TokenInvalid BizCode `code:"40102"`
	// MustRegister: 身份认证通过（如微信授权成功）但尚未绑定本系统账号
	MustRegister BizCode `code:"40103"`
	// RegisterTimeout: 注册流程中的验证码或临时状态超时
	RegisterTimeout BizCode `code:"40104"`
	// RegisterNotFound: 尝试用未注册的凭证进行登录操作
	RegisterNotFound BizCode `code:"40105"`
	// InvalidCredentials: 账号不存在或密码/验证码匹配失败
	InvalidCredentials BizCode `code:"40106"`
	// AccountLocked: 因多次输错密码或其他原因导致的账号临时封禁
	AccountLocked BizCode `code:"40107"`
	// MFARequired: 需要额外的二次验证（如短信验证码、谷歌验证器）
	MFARequired BizCode `code:"40108"`
}

// forbiddenGroup 403xx - 权限不足
// 场景：我知道你是谁，但你没权操作这个资源。通常与 RBAC 相关。
type forbiddenGroup struct {
	Base *BizError
	// PermissionDenied: 用户已登录，但其角色不具备访问该接口或该 ID 资源的权限
	PermissionDenied BizCode `code:"40301"`
	// UserEmailNotVerified: 强制性要求（如必须验证邮箱后才能发布内容）
	UserEmailNotVerified BizCode `code:"40302"`
}

// notFoundGroup 404xx - 资源不存在
// 场景：请求路径正确，但路径中指向的具体 ID 或名称找不到。
type notFoundGroup struct {
	Base *BizError
	// DataBaseNotFound: 数据库中查不到对应的记录（如 GET /user/100，100 不存在）
	DataBaseNotFound BizCode `code:"40401"`
	// ServiceSshdNotFound: 特定业务场景下的资源丢失（如指定的 SSH 服务不存在）
	ServiceSshdNotFound BizCode `code:"40402"`
	// K8sResourceNotFound: 集群中找不到指定的 Pod, Deployment 或 Namespace
	K8sResourceNotFound BizCode `code:"40403"`
}

// methodNotAllowedGroup 405xx - 方法不允许
type methodNotAllowedGroup struct {
	Base *BizError
	// MethodNotAllowed: 接口只支持 POST，客户端用了 GET
	MethodNotAllowed BizCode `code:"40501"`
}

// conflictGroup 409xx - 业务状态冲突
// 场景：请求参数没问题，但由于“服务器当前数据状态”导致操作无法完成。
// 区别：400 是“你填错了”，409 是“虽然你填对了，但现在不让你这么干”。
type conflictGroup struct {
	Base *BizError
	// ResourceAlreadyExists: 唯一索引冲突（如：用户名已注册、手机号已被占用）
	ResourceAlreadyExists BizCode `code:"40901"`
	// ResourceStatusError: 状态机限制（如：订单已支付不能再修改金额、任务运行中不能直接删除）
	ResourceStatusError BizCode `code:"40902"`
	// DuplicateOperation: 幂等性检查失败（如：1秒内重复点击提交按钮，导致重复请求）
	DuplicateOperation BizCode `code:"40903"`
	// DependencyConflict: 级联删除限制（如：该角色下还有用户，不能删除该角色）
	DependencyConflict BizCode `code:"40904"`
}

// payloadTooLargeGroup 413xx - 数据过大
type payloadTooLargeGroup struct {
	Base *BizError
	// PayloadTooLarge: 上传文件超过 Nginx 或后端限制的 MaxBodySize
	PayloadTooLarge BizCode `code:"41301"`
}

// rateLimitGroup 429xx - 请求过多
type rateLimitGroup struct {
	Base *BizError
	// TooManyRequests: 触发了 IP 限流、用户每分钟请求数限流或 API 总并发限流
	TooManyRequests BizCode `code:"42901"`
}

// internalGroup 500xx - 服务端错误
// 场景：服务器自己的代码没写好，或者是环境问题。客户端重试不一定有效，通常需要开发介入。
type internalGroup struct {
	Base *BizError
	// ServiceError: 未归类的通用内部错误
	ServiceError BizCode `code:"50001"`
	// DatabaseError: 数据库连接超时、SQL 语法错误、连接池满等（非业务数据问题）
	DatabaseError BizCode `code:"50002"`
	// CacheError: Redis 连接失败、序列化失败或内存溢出
	CacheError BizCode `code:"50003"`
	// ThirdPartyApiError: 外部依赖服务（如支付网关、短信平台）返回了 5xx 错误
	ThirdPartyApiError BizCode `code:"50004"`
	// FileSystemError: 磁盘空间满、权限不足导致无法写入日志或上传文件
	FileSystemError BizCode `code:"50005"`
	// K8sServiceError: 调用 K8s API Server 返回非 404 的错误
	K8sServiceError BizCode `code:"50006"`
	// InternalPanic: 代码触发了 runtime panic 被 recover 捕获
	InternalPanic BizCode `code:"50007"`
	// VolcanoServiceError: 调度系统内部逻辑异常
	VolcanoServiceError BizCode `code:"50008"`
}

// badGatewayGroup 502xx - 网关错误
// 场景：通常是代理（Nginx/Ingress）找不到下游 Pod，或者 Pod 进程已挂。
type badGatewayGroup struct {
	Base *BizError
	// BadGateway: 上游服务进程崩溃、重启中或无法建立 TCP 连接
	BadGateway BizCode `code:"50201"`
}

// serviceErrorGroup 503xx - 服务不可用
// 场景：服务器能响应，但主动告知由于过载或维护无法处理。
type serviceErrorGroup struct {
	Base *BizError
	// ServiceUnavailable: 服务器正在维护模式，或系统负载过高拒绝服务
	ServiceUnavailable BizCode `code:"50301"`
}

// gatewayTimeoutGroup 504xx - 网关超时
// 场景：下游服务执行太慢，超过了网关设置的 Timeout 时间。
type gatewayTimeoutGroup struct {
	Base *BizError
	// GatewayTimeout: 数据库慢查询、下游 RPC 响应慢导致的链路超时
	GatewayTimeout BizCode `code:"50401"`
}
