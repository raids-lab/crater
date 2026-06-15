package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/patrol"
	"github.com/raids-lab/crater/pkg/storagegovernance"
	"github.com/raids-lab/crater/pkg/storageindex"
)

// ---- LLM 任务状态存储 ----

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewStorageMgr)
}

type StorageMgr struct {
	name       string
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config
	promClient monitor.PrometheusInterface
}

// AutoScaleRequest 自动扩缩容请求
type AutoScaleRequest struct {
	MinQuota       int64   `json:"min_quota" binding:"required,min=-1"`               // 最小配额，-1 表示无限制
	MaxQuota       int64   `json:"max_quota" binding:"required,min=-1"`               // 最大配额，-1 表示无限制
	ScaleUpRatio   float64 `json:"scale_up_ratio" binding:"required,min=1"`           // 扩容比例，如 1.5 表示扩容到当前使用的 1.5 倍
	ScaleDownRatio float64 `json:"scale_down_ratio" binding:"required,min=0.1,max=1"` // 缩容比例，如 0.8 表示缩容到当前使用的 0.8 倍
}

func NewStorageMgr(conf *RegisterConfig) Manager {
	return &StorageMgr{
		name:       "storage",
		kubeClient: conf.KubeClient,
		kubeConfig: conf.KubeConfig,
		promClient: conf.PrometheusClient,
	}
}

func (mgr *StorageMgr) GetName() string { return mgr.name }

func (mgr *StorageMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *StorageMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/dirsize/*path", mgr.GetDirectorySize)
	g.GET("/my-quota", mgr.GetMyQuota)
}

func (mgr *StorageMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/user-spaces", mgr.GetAllUserSpaceSizes)
	g.PUT("/user-spaces/:user/quota", mgr.SetUserSpaceQuota)
	g.POST("/user-spaces/:user/autoscale", mgr.AutoScaleUserSpaceQuota)
	g.POST("/auto-shrink", mgr.RunAutoShrink)
	g.POST("/user-spaces/:user/apply-expansion", mgr.ApplyExpansion)
	g.POST("/user-spaces/:user/freeze-jobs", mgr.FreezeJobs)
	g.POST("/user-spaces/:user/revert-expansion", mgr.RevertExpansion)
	g.POST("/user-spaces/:user/unfreeze-jobs", mgr.UnfreezeJobs)
	g.GET("/decisions", mgr.ListStorageDecisions)
	g.POST("/decisions/replay", mgr.ReplayStorageDecisions)
	g.GET("/decisions/:job_id", mgr.GetStorageDecision)
	g.POST("/user-spaces/:user/llm-decision", mgr.TriggerLLMDecision)
	g.GET("/user-spaces/:user/llm-decision/:job_id", mgr.GetLLMDecisionStatus)
	g.POST("/index/scan", mgr.TriggerMetadataIndexScan)
	g.GET("/index/scans/:scan_id", mgr.GetMetadataIndexScan)
	g.GET("/index/workspaces/:workspace_type/:workspace_name/overview", mgr.GetMetadataWorkspaceOverview)
	g.GET("/index/workspaces/:workspace_type/:workspace_name/redundancy-hits", mgr.ListMetadataWorkspaceRedundancyHits)
	g.GET("/index/workspaces/:workspace_type/:workspace_name/candidates", mgr.ListMetadataWorkspaceCandidates)
	g.GET("/index/workspaces/:workspace_type/:workspace_name/candidate-files", mgr.ListMetadataWorkspaceCandidateFiles)
	g.POST("/index/compare-folders", mgr.CompareMetadataFolders)
	g.GET("/index/compare-folders/:job_id", mgr.GetMetadataFolderCompareJob)
}

// GetDirectorySize godoc
//
// @Summary Get directory size in CephFS
// @Description Get the size of a directory in CephFS using getfattr command
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param path path string true "Directory path"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/dirsize/{path} [get]
func (mgr *StorageMgr) GetDirectorySize(c *gin.Context) {
	// 1. 获取路径参数
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/storage/dirsize/")
	if path == "" {
		resputil.BadRequestError(c, "路径不能为空")
		return
	}

	// 2. 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 3. 执行 Ceph 命令获取目录大小
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	size, err := ceph.GetCephDirectorySize(mgr.kubeClient, mgr.kubeConfig, "rook-ceph", path, prefixConfig)
	if err != nil {
		klog.Warningf("GetDirectorySize: failed to get size for %q, returning unknown sentinel: %v", path, err)
		size = -1
	}

	// 4. 返回结果
	resputil.Success(c, gin.H{
		"path":      path,
		"size":      size,
		"unit":      "bytes",
		"formatted": formatSize(size),
	})
}

// GetMyQuota godoc
//
// @Summary Get current user's storage quota
// @Description Get the storage quota for the currently authenticated user
// @Tags Storage
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/my-quota [get]
func (mgr *StorageMgr) GetMyQuota(c *gin.Context) {
	token := util.GetToken(c)

	var row struct {
		SpaceQuota int64 `gorm:"column:space_quota"`
	}
	if err := query.GetDB().Raw(
		"SELECT space_quota FROM users WHERE id = ? AND deleted_at IS NULL", token.UserID,
	).Scan(&row).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("获取配额失败: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{
		"space_quota":           row.SpaceQuota,
		"space_quota_formatted": formatSize(row.SpaceQuota),
	})
}

// GetAllUserSpaceSizes godoc
//
// @Summary Get all user space sizes
// @Description Get the size of all user spaces from database
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces [get]
func (mgr *StorageMgr) GetAllUserSpaceSizes(c *gin.Context) {
	// 1. 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	// 2. 从数据库中获取用户空间大小和配额
	type UserSpaceInfo struct {
		model.UserSpaceSize
		Username           string `json:"username"`
		SpaceQuota         int64  `json:"space_quota"`
		OriginalSpaceQuota *int64 `json:"original_space_quota"`
		JobsFrozen         bool   `json:"jobs_frozen"`
		ShrinkStage        string `json:"shrink_stage"`
	}

	var userSpaceInfos []UserSpaceInfo
	var total int64

	db := query.GetDB()

	// 计算总数
	if err := db.Model(&model.UserSpaceSize{}).Count(&total).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("获取用户空间大小总数失败: %v", err), resputil.NotSpecified)
		return
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据，关联 User 表获取 SpaceQuota 和 OriginalSpaceQuota
	if err := db.Table("user_space_sizes").
		Select(
			"user_space_sizes.*, " +
				"users.space_quota as space_quota, " +
				"users.original_space_quota as original_space_quota, " +
				"users.name as username, " +
				"users.jobs_frozen as jobs_frozen, " +
				"users.shrink_stage as shrink_stage",
		).
		Joins("LEFT JOIN users ON user_space_sizes.user_id = users.id").
		Offset(offset).Limit(pageSize).
		Find(&userSpaceInfos).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("获取用户空间大小失败: %v", err), resputil.NotSpecified)
		return
	}

	// 3. 格式化结果
	formattedUserSpaces := make([]map[string]any, 0, len(userSpaceInfos))
	for i := range userSpaceInfos {
		info := userSpaceInfos[i]
		item := map[string]any{
			"user":            info.Username,
			"size":            info.Size,
			"quota":           info.SpaceQuota,
			"unit":            "bytes",
			"formatted":       formatSize(info.Size),
			"quota_formatted": formatSize(info.SpaceQuota),
			"is_expanded":     info.OriginalSpaceQuota != nil,
			"jobs_frozen":     info.JobsFrozen,
			"shrink_stage":    info.ShrinkStage,
		}
		if info.OriginalSpaceQuota != nil {
			item["original_quota"] = *info.OriginalSpaceQuota
			item["original_quota_formatted"] = formatSize(*info.OriginalSpaceQuota)
		}
		formattedUserSpaces = append(formattedUserSpaces, item)
	}

	// 4. 返回结果（包含分页信息）
	resputil.Success(c, gin.H{
		"items":      formattedUserSpaces,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}

// SetUserSpaceQuota godoc
//
// @Summary Set user space quota
// @Description Set the space quota for a user
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Param quota body int64 true "Space quota in bytes, -1 for unlimited"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 404 {object} resputil.Response[any] "User not found"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/quota [put]
func (mgr *StorageMgr) SetUserSpaceQuota(c *gin.Context) {
	// 1. 获取用户名参数
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	// 2. 解析请求体
	type QuotaRequest struct {
		Quota int64 `json:"quota" binding:"required"`
	}
	var req QuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, "请求体格式错误: "+err.Error())
		return
	}

	// 3. 验证配额值
	if req.Quota < -1 {
		resputil.BadRequestError(c, "配额值不能小于 -1")
		return
	}

	// 4. 获取用户信息（含临时扩容状态）
	db := query.GetDB()
	var userRow struct {
		model.User
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
	}
	if err := db.Model(&model.User{}).
		Select("users.*, users.original_space_quota").
		Where("name = ?", user).
		First(&userRow).Error; err != nil {
		resputil.Error(c, "用户不存在", resputil.NotSpecified)
		return
	}
	userInfo := userRow.User

	// 5. 更新理论配额
	// 临时扩容期间：只更新 original_space_quota（理论配额），保持 space_quota（现配额）不变
	// 无临时扩容：更新 space_quota（理论配额即现配额）
	isExpanded := userRow.OriginalSpaceQuota != nil
	if isExpanded {
		if err := db.Exec("UPDATE users SET original_space_quota = ? WHERE name = ? AND deleted_at IS NULL", req.Quota, user).Error; err != nil {
			resputil.Error(c, fmt.Sprintf("更新理论配额失败: %v", err), resputil.NotSpecified)
			return
		}
	} else {
		if err := db.Model(&model.User{}).Where("name = ?", user).Update("space_quota", req.Quota).Error; err != nil {
			resputil.Error(c, fmt.Sprintf("更新理论配额失败: %v", err), resputil.NotSpecified)
			return
		}
	}

	// 6. 同步 CephFS 配额
	// 临时扩容期间 Ceph 配额维持现配额不变；无扩容时才同步新值
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	userPath := fmt.Sprintf("/user/%s", userInfo.Space)

	var err error
	if !isExpanded {
		err = ceph.SetCephDirectoryQuota(mgr.kubeClient, mgr.kubeConfig, "rook-ceph", userPath, prefixConfig, req.Quota)
		if err != nil {
			klog.Errorf("SetUserSpaceQuota: 设置用户 %s Ceph 配额失败: %v", user, err)
		}
	}

	// 7. 返回结果
	resputil.Success(c, gin.H{
		"user":             user,
		"quota":            req.Quota,
		"unit":             "bytes",
		"quota_formatted":  formatSize(req.Quota),
		"ceph_quota_set":   err == nil,
		"ceph_quota_error": err,
	})
}

// AutoScaleUserSpaceQuota godoc
//
// @Summary Auto scale user space quota
// @Description Auto scale the space quota for a user based on current usage
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Param body body AutoScaleRequest true "Auto scale configuration"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 404 {object} resputil.Response[any] "User not found"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/autoscale [post]
func (mgr *StorageMgr) AutoScaleUserSpaceQuota(c *gin.Context) {
	// 1. 获取用户名参数
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	// 2. 解析请求体
	var req AutoScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, "请求体格式错误: "+err.Error())
		return
	}

	// 3. 获取用户信息和当前使用空间大小
	db := query.GetDB()
	var userInfo model.User
	if err := db.Where("name = ?", user).First(&userInfo).Error; err != nil {
		resputil.Error(c, "用户不存在", resputil.NotSpecified)
		return
	}

	var userSpaceSize model.UserSpaceSize
	if err := db.Where("user_id = ?", userInfo.ID).First(&userSpaceSize).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("获取用户空间使用情况失败: %v", err), resputil.NotSpecified)
		return
	}

	// 4. 计算新的配额
	currentUsage := userSpaceSize.Size
	newQuota := int64(float64(currentUsage) * req.ScaleUpRatio)

	// 应用最小和最大配额限制
	if req.MinQuota != -1 && newQuota < req.MinQuota {
		newQuota = req.MinQuota
	}
	if req.MaxQuota != -1 && newQuota > req.MaxQuota {
		newQuota = req.MaxQuota
	}

	// 5. 更新用户配额
	if err := db.Model(&model.User{}).Where("name = ?", user).Update("space_quota", newQuota).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("更新用户配额失败: %v", err), resputil.NotSpecified)
		return
	}

	// 6. 实际设置 CephFS 目录配额
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	// 构建用户空间路径
	userPath := fmt.Sprintf("/user/%s", userInfo.Space)

	// 调用 SetCephDirectoryQuota 设置实际配额
	cephErr := ceph.SetCephDirectoryQuota(mgr.kubeClient, mgr.kubeConfig, "rook-ceph", userPath, prefixConfig, newQuota)
	if cephErr != nil {
		// 记录错误但不影响响应，确保数据库更新成功
		klog.Errorf("AutoScaleUserSpaceQuota: 设置用户 %s Ceph 配额失败: %v", user, cephErr)
	}

	// 7. 返回结果
	resputil.Success(c, gin.H{
		"user":                    user,
		"current_usage":           currentUsage,
		"new_quota":               newQuota,
		"unit":                    "bytes",
		"current_usage_formatted": formatSize(currentUsage),
		"new_quota_formatted":     formatSize(newQuota),
		"ceph_quota_set":          cephErr == nil,
		"ceph_quota_error":        cephErr,
	})
}

// RunAutoShrink triggers one manual scan that shrinks users currently in temporary
// expansion state back to their original quota when it is safe to do so.
func (mgr *StorageMgr) RunAutoShrink(c *gin.Context) {
	result, err := patrol.RunAutoShrinkStorageExpansions(c.Request.Context(), &patrol.Clients{
		KubeClient: mgr.kubeClient,
		KubeConfig: mgr.kubeConfig,
		PromClient: mgr.promClient,
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("自动缩容执行失败：%v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{
		"message": result,
	})
}

// ApplyExpansion godoc
//
// @Summary Apply temporary storage expansion for a user
// @Description Save the current quota as original and set an expanded quota
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Param body body object true "expand_bytes: bytes to add on top of current quota"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/apply-expansion [post]
func (mgr *StorageMgr) ApplyExpansion(c *gin.Context) {
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	var req struct {
		ExpandBytes   int64  `json:"expand_bytes" binding:"required,min=1"`
		FreezeNewJobs bool   `json:"freeze_new_jobs"`
		DecisionJobID string `json:"decision_job_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, "请求体格式错误: "+err.Error())
		return
	}

	db := query.GetDB()

	// 查询当前配额和原始配额
	var row struct {
		SpaceQuota         int64  `gorm:"column:space_quota"`
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
	}
	if err := db.Raw(
		"SELECT space_quota, original_space_quota FROM users WHERE name = ? AND deleted_at IS NULL",
		user,
	).Scan(&row).Error; err != nil {
		resputil.Error(c, "用户不存在", resputil.NotSpecified)
		return
	}

	if row.OriginalSpaceQuota != nil {
		resputil.BadRequestError(c, "该用户已存在临时扩容，请先还原后再扩容")
		return
	}

	newQuota := row.SpaceQuota + req.ExpandBytes

	// 保存原始配额，并更新为新配额，同时设置 jobs_frozen
	if err := db.Exec(
		"UPDATE users "+
			"SET original_space_quota = space_quota, space_quota = ?, jobs_frozen = ?, "+
			"shrink_stage = ?, shrink_stage_updated_at = NOW() "+
			"WHERE name = ? AND deleted_at IS NULL",
		newQuota, req.FreezeNewJobs, "expanded", user,
	).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("更新配额失败: %v", err), resputil.NotSpecified)
		return
	}

	// 同步到 CephFS
	var userInfo model.User
	if err := db.Where("name = ?", user).First(&userInfo).Error; err == nil {
		cfg := config.GetConfig()
		prefixConfig := ceph.StoragePrefixConfig{
			User:    cfg.Storage.Prefix.User,
			Account: cfg.Storage.Prefix.Account,
			Public:  cfg.Storage.Prefix.Public,
		}
		if cephErr := ceph.SetCephDirectoryQuota(
			mgr.kubeClient,
			mgr.kubeConfig,
			"rook-ceph",
			fmt.Sprintf("/user/%s", userInfo.Space),
			prefixConfig,
			newQuota,
		); cephErr != nil {
			klog.Errorf("ApplyExpansion: 设置用户 %s Ceph 配额失败: %v", user, cephErr)
		}
	}
	if req.DecisionJobID != "" {
		action := "manual_expand"
		if req.FreezeNewJobs {
			action = "manual_expand_and_freeze"
		}
		_ = storagegovernance.MarkDecisionExecution(c.Request.Context(), req.DecisionJobID, action, nil)
	}

	resputil.Success(c, gin.H{
		"user":                     user,
		"original_quota":           row.SpaceQuota,
		"new_quota":                newQuota,
		"original_quota_formatted": formatSize(row.SpaceQuota),
		"new_quota_formatted":      formatSize(newQuota),
		"jobs_frozen":              req.FreezeNewJobs,
	})
}

// RevertExpansion godoc
//
// @Summary Revert temporary storage expansion for a user
// @Description Restore the user's quota to the original value before expansion
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/revert-expansion [post]
func (mgr *StorageMgr) RevertExpansion(c *gin.Context) {
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	db := query.GetDB()

	var row struct {
		SpaceQuota         int64  `gorm:"column:space_quota"`
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
	}
	if err := db.Raw(
		"SELECT space_quota, original_space_quota FROM users WHERE name = ? AND deleted_at IS NULL",
		user,
	).Scan(&row).Error; err != nil {
		resputil.Error(c, "用户不存在", resputil.NotSpecified)
		return
	}

	if row.OriginalSpaceQuota == nil {
		resputil.BadRequestError(c, "该用户当前没有临时扩容，无需还原")
		return
	}

	originalQuota := *row.OriginalSpaceQuota

	// 查询用户 ID 和当前实际用量，决定是否同时解冻
	var userIDRow struct {
		ID uint `gorm:"column:id"`
	}
	db.Raw("SELECT id FROM users WHERE name = ? AND deleted_at IS NULL", user).Scan(&userIDRow)

	var currentSize int64
	var spaceSize model.UserSpaceSize
	if err := db.Where("user_id = ?", userIDRow.ID).First(&spaceSize).Error; err == nil {
		currentSize = spaceSize.Size
	}

	// 只有还原后的理论配额大于当前用量时才自动解冻；否则保持冻结状态
	shouldUnfreeze := originalQuota <= 0 || currentSize < originalQuota
	if shouldUnfreeze {
		if err := db.Exec(
			"UPDATE users "+
				"SET space_quota = ?, original_space_quota = NULL, jobs_frozen = false, "+
				"shrink_stage = NULL, shrink_stage_updated_at = NULL "+
				"WHERE name = ? AND deleted_at IS NULL",
			originalQuota,
			user,
		).Error; err != nil {
			resputil.Error(c, fmt.Sprintf("还原配额失败: %v", err), resputil.NotSpecified)
			return
		}
	} else {
		// 仅还原配额，不解冻（用量仍超出理论配额）
		if err := db.Exec(
			"UPDATE users "+
				"SET space_quota = ?, original_space_quota = NULL, "+
				"shrink_stage = NULL, shrink_stage_updated_at = NULL "+
				"WHERE name = ? AND deleted_at IS NULL",
			originalQuota,
			user,
		).Error; err != nil {
			resputil.Error(c, fmt.Sprintf("还原配额失败: %v", err), resputil.NotSpecified)
			return
		}
	}

	// 同步到 CephFS
	var userInfo model.User
	if err := db.Where("name = ?", user).First(&userInfo).Error; err == nil {
		cfg := config.GetConfig()
		prefixConfig := ceph.StoragePrefixConfig{
			User:    cfg.Storage.Prefix.User,
			Account: cfg.Storage.Prefix.Account,
			Public:  cfg.Storage.Prefix.Public,
		}
		if cephErr := ceph.SetCephDirectoryQuota(
			mgr.kubeClient,
			mgr.kubeConfig,
			"rook-ceph",
			fmt.Sprintf("/user/%s", userInfo.Space),
			prefixConfig,
			originalQuota,
		); cephErr != nil {
			klog.Errorf("RevertExpansion: 设置用户 %s Ceph 配额失败: %v", user, cephErr)
		}
	}

	resputil.Success(c, gin.H{
		"user":                     user,
		"reverted_quota":           originalQuota,
		"reverted_quota_formatted": formatSize(originalQuota),
		"jobs_unfrozen":            shouldUnfreeze,
	})
}

// UnfreezeJobs godoc
//
// @Summary Manually unfreeze job creation for a user
// @Description Clear the jobs_frozen flag, allowing the user to create new jobs again
// @Tags Storage
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/unfreeze-jobs [post]
func (mgr *StorageMgr) UnfreezeJobs(c *gin.Context) {
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	db := query.GetDB()
	if err := db.Exec("UPDATE users SET jobs_frozen = false WHERE name = ? AND deleted_at IS NULL", user).Error; err != nil {
		resputil.Error(c, fmt.Sprintf("解冻失败: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"user": user, "jobs_frozen": false})
}

// FreezeJobs manually freezes job creation for a user and optionally binds the action to a decision record.
func (mgr *StorageMgr) FreezeJobs(c *gin.Context) {
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	var req struct {
		DecisionJobID string `json:"decision_job_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		resputil.BadRequestError(c, "请求体格式错误: "+err.Error())
		return
	}

	db := query.GetDB()
	if err := db.Exec("UPDATE users SET jobs_frozen = true WHERE name = ? AND deleted_at IS NULL", user).Error; err != nil {
		if req.DecisionJobID != "" {
			_ = storagegovernance.MarkDecisionExecution(c.Request.Context(), req.DecisionJobID, "manual_freeze_failed", err)
		}
		resputil.Error(c, fmt.Sprintf("冻结失败: %v", err), resputil.NotSpecified)
		return
	}

	if req.DecisionJobID != "" {
		_ = storagegovernance.MarkDecisionExecution(c.Request.Context(), req.DecisionJobID, "manual_freeze", nil)
	}

	resputil.Success(c, gin.H{"user": user, "jobs_frozen": true})
}

// TriggerLLMDecision godoc
//
// @Summary Trigger LLM storage expansion decision for a user
// @Description Calls Claude agent to analyze whether a user needs temporary storage expansion
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/storage/admin/user-spaces/{user}/llm-decision [post]
// TriggerLLMDecision 异步启动 LLM 分析，立即返回 job_id
func (mgr *StorageMgr) TriggerLLMDecision(c *gin.Context) {
	user := c.Param("user")
	if user == "" {
		resputil.BadRequestError(c, "用户名不能为空")
		return
	}

	engine := storagegovernance.NewEngine(
		mgr.kubeClient,
		mgr.kubeConfig,
		mgr.promClient,
		storagegovernance.DefaultConstraintConfig(),
	)
	jobID, err := engine.StartAsyncDecision(context.Background(), storagegovernance.DecisionRequest{
		Username:      user,
		Source:        model.StorageDecisionSourceManual,
		TriggerReason: "manual llm decision request",
	})
	if err != nil {
		klog.Errorf("TriggerLLMDecision: user=%s err=%v", user, err)
		resputil.Error(c, fmt.Sprintf("鍚姩 LLM 鍒嗘瀽澶辫触: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"job_id": jobID})
}

// GetLLMDecisionStatus 查询 LLM 分析任务状态
func (mgr *StorageMgr) GetLLMDecisionStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	job, err := storagegovernance.GetDecisionStatus(c.Request.Context(), jobID)

	if err != nil {
		resputil.Error(c, "任务不存在", resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

// formatSize 格式化大小为人类可读格式
// ListStorageDecisions returns paginated persisted storage decision records.
func (mgr *StorageMgr) ListStorageDecisions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	result, err := storagegovernance.ListDecisionRecords(
		c.Request.Context(),
		page,
		pageSize,
		c.Query("user"),
		c.Query("status"),
		c.Query("source"),
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list storage decisions: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, result)
}

// GetStorageDecision returns one persisted storage decision record with full details.
func (mgr *StorageMgr) GetStorageDecision(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		resputil.BadRequestError(c, "job_id cannot be empty")
		return
	}

	result, err := storagegovernance.GetDecisionRecord(c.Request.Context(), jobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to get storage decision: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, result)
}

// ReplayStorageDecisions re-evaluates stored decisions under the current or overridden safety policy.
func (mgr *StorageMgr) ReplayStorageDecisions(c *gin.Context) {
	var req struct {
		Limit                    int      `json:"limit"`
		MaxExpandRatio           *float64 `json:"max_expand_ratio"`
		MaxExpandBytes           *int64   `json:"max_expand_bytes"`
		MinPlatformReservedRatio *float64 `json:"min_platform_reserved_ratio"`
		MinPlatformReservedBytes *int64   `json:"min_platform_reserved_bytes"`
		ExpansionCooldownHours   *int     `json:"expansion_cooldown_hours"`
		ForceFreezeWhenOverQuota *bool    `json:"force_freeze_when_over_quota"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		resputil.BadRequestError(c, "invalid replay request: "+err.Error())
		return
	}

	cfg := storagegovernance.DefaultConstraintConfig()
	if req.MaxExpandRatio != nil {
		cfg.MaxExpandRatio = *req.MaxExpandRatio
	}
	if req.MaxExpandBytes != nil {
		cfg.MaxExpandBytes = *req.MaxExpandBytes
	}
	if req.MinPlatformReservedRatio != nil {
		cfg.MinPlatformReservedRatio = *req.MinPlatformReservedRatio
	}
	if req.MinPlatformReservedBytes != nil {
		cfg.MinPlatformReservedBytes = *req.MinPlatformReservedBytes
	}
	if req.ExpansionCooldownHours != nil {
		cfg.ExpansionCooldown = time.Duration(*req.ExpansionCooldownHours) * time.Hour
	}
	if req.ForceFreezeWhenOverQuota != nil {
		cfg.ForceFreezeWhenOverQuota = *req.ForceFreezeWhenOverQuota
	}

	summary, err := storagegovernance.ReplayStoredDecisions(c.Request.Context(), cfg, req.Limit)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to replay storage decisions: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, summary)
}

func (mgr *StorageMgr) metadataIndexService() *storageindex.Service {
	return storageindex.NewService(mgr.kubeClient, mgr.kubeConfig)
}

// TriggerMetadataIndexScan starts an asynchronous metadata indexing run for one workspace.
func (mgr *StorageMgr) TriggerMetadataIndexScan(c *gin.Context) {
	var req struct {
		WorkspaceType string `json:"workspace_type" binding:"required"`
		WorkspaceName string `json:"workspace_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, "invalid metadata scan request: "+err.Error())
		return
	}

	scanID, err := mgr.metadataIndexService().StartFullScan(c.Request.Context(), storageindex.StartScanRequest{
		WorkspaceType: model.StorageIndexWorkspaceType(strings.TrimSpace(req.WorkspaceType)),
		WorkspaceName: strings.TrimSpace(req.WorkspaceName),
		TriggerSource: "manual",
		ScanMode:      model.StorageIndexScanModeDailyRefresh,
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to start metadata scan: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{
		"scan_id":        scanID,
		"workspace_type": req.WorkspaceType,
		"workspace_name": req.WorkspaceName,
	})
}

// GetMetadataIndexScan returns the current status of a metadata indexing job.
func (mgr *StorageMgr) GetMetadataIndexScan(c *gin.Context) {
	scanID := strings.TrimSpace(c.Param("scan_id"))
	if scanID == "" {
		resputil.BadRequestError(c, "scan_id cannot be empty")
		return
	}

	job, err := mgr.metadataIndexService().GetScanJob(c.Request.Context(), scanID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query metadata scan: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

// GetMetadataWorkspaceOverview returns the latest indexed overview for a workspace.
func (mgr *StorageMgr) GetMetadataWorkspaceOverview(c *gin.Context) {
	workspaceType := model.StorageIndexWorkspaceType(strings.TrimSpace(c.Param("workspace_type")))
	workspaceName := strings.TrimSpace(c.Param("workspace_name"))

	overview, err := mgr.metadataIndexService().GetWorkspaceOverview(c.Request.Context(), workspaceType, workspaceName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query metadata overview: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, overview)
}

// ListMetadataWorkspaceRedundancyHits returns redundancy hits detected against the public baseline.
func (mgr *StorageMgr) ListMetadataWorkspaceRedundancyHits(c *gin.Context) {
	mgr.listStorageIndexPage(c, "redundancy hits", func(
		ctx context.Context,
		workspaceType model.StorageIndexWorkspaceType,
		workspaceName string,
		page int,
		pageSize int,
	) (any, int64, error) {
		return mgr.metadataIndexService().ListRedundancyHits(ctx, workspaceType, workspaceName, page, pageSize)
	})
}

func (mgr *StorageMgr) ListMetadataWorkspaceCandidates(c *gin.Context) {
	mgr.listStorageIndexPage(c, "candidates", func(
		ctx context.Context,
		workspaceType model.StorageIndexWorkspaceType,
		workspaceName string,
		page int,
		pageSize int,
	) (any, int64, error) {
		return mgr.metadataIndexService().ListCandidates(ctx, workspaceType, workspaceName, page, pageSize)
	})
}

func (mgr *StorageMgr) listStorageIndexPage(
	c *gin.Context,
	resourceName string,
	queryPage func(context.Context, model.StorageIndexWorkspaceType, string, int, int) (any, int64, error),
) {
	workspaceType := model.StorageIndexWorkspaceType(strings.TrimSpace(c.Param("workspace_type")))
	workspaceName := strings.TrimSpace(c.Param("workspace_name"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	items, total, err := queryPage(c.Request.Context(), workspaceType, workspaceName, page, pageSize)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query %s: %v", resourceName, err), resputil.NotSpecified)
		return
	}

	successStorageIndexPage(c, items, total, page, pageSize)
}

func (mgr *StorageMgr) ListMetadataWorkspaceCandidateFiles(c *gin.Context) {
	workspaceType := model.StorageIndexWorkspaceType(strings.TrimSpace(c.Param("workspace_type")))
	workspaceName := strings.TrimSpace(c.Param("workspace_name"))
	candidatePath := strings.TrimSpace(c.Query("candidate_path"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "200"))

	items, total, err := mgr.metadataIndexService().ListCandidateFiles(
		c.Request.Context(),
		workspaceType,
		workspaceName,
		candidatePath,
		page,
		pageSize,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query candidate files: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}

func (mgr *StorageMgr) CompareMetadataFolders(c *gin.Context) {
	var req struct {
		LeftPath    string `json:"left_path" binding:"required"`
		RightPath   string `json:"right_path" binding:"required"`
		CompareType string `json:"compare_type"`
		CompareMode string `json:"compare_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, "invalid compare folder request: "+err.Error())
		return
	}

	jobID, err := mgr.metadataIndexService().StartCompareDirectories(
		c.Request.Context(),
		strings.TrimSpace(req.LeftPath),
		strings.TrimSpace(req.RightPath),
		strings.TrimSpace(req.CompareType),
		strings.TrimSpace(req.CompareMode),
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to start compare job: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"job_id": jobID})
}

func (mgr *StorageMgr) GetMetadataFolderCompareJob(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("job_id"))
	if jobID == "" {
		resputil.BadRequestError(c, "job_id cannot be empty")
		return
	}

	job, err := mgr.metadataIndexService().GetCompareDirectoryJob(jobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query compare job: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes <= 0 {
		return "0 B"
	}
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func successStorageIndexPage(c *gin.Context, items any, total int64, page, pageSize int) {
	resputil.Success(c, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}
