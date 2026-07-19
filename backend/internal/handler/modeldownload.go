package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"k8s.io/client-go/kubernetes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/governance/modeldataset"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	defaultDownloadLogTailLines int64 = 1000
	maxStoredDownloadLogBytes         = 64 * 1024
	maxCapturedReadmeBytes            = modeldataset.MaxStoredReadmeBytes
	readmeLogChunkCharacters          = 4096
	maxDownloadRevisionLength         = 128
	CategoryModel                     = "model"
	CategoryDataset                   = "dataset"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewModelDownloadMgr)
}

type ModelDownloadMgr struct {
	name      string
	crClient  kubernetes.Interface
	namespace string
}

func NewModelDownloadMgr(conf *RegisterConfig) Manager {
	return &ModelDownloadMgr{
		name:      "model-download",
		crClient:  conf.KubeClient,
		namespace: config.GetConfig().Namespaces.Job,
	}
}

func (mgr *ModelDownloadMgr) GetName() string { return mgr.name }

func (mgr *ModelDownloadMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ModelDownloadMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/models/download", mgr.CreateDownload)
	g.GET("/models/downloads", mgr.ListDownloads)
	g.GET("/models/downloads/:id", mgr.GetDownload)
	g.GET("/models/downloads/:id/logs", mgr.GetDownloadLogs)
	g.POST("/models/downloads/:id/retry", mgr.RetryDownload)
	g.POST("/models/downloads/:id/pause", mgr.PauseDownload)
	g.POST("/models/downloads/:id/resume", mgr.ResumeDownload)
	g.DELETE("/models/downloads/:id", mgr.DeleteDownload)
}

func (mgr *ModelDownloadMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/models/downloads", mgr.ListAllDownloads)
	g.DELETE("/models/downloads/:id", mgr.AdminDeleteDownload)
}

// Request/Response types
type CreateDownloadReq struct {
	Name     string `json:"name" binding:"required"`
	Revision string `json:"revision"`
	Source   string `json:"source"`
	Category string `json:"category" binding:"required,oneof=model dataset"`
	// Token is an optional access token for gated/private repositories on the
	// source site. It is only forwarded to the download Job as an env var and is
	// never persisted on the (shared, deduplicated) download record.
	Token string `json:"token"`
}

type DownloadActionReq struct {
	Token string `json:"token"`
	// Revision is optional and only used by retry. A non-nil empty value means
	// "use the source default branch" while preserving the failed record's path.
	Revision *string `json:"revision"`
}

type ModelDownloadResp struct {
	ID              uint           `json:"id"`
	Name            string         `json:"name"`
	Source          string         `json:"source"`
	Category        string         `json:"category"`
	Revision        string         `json:"revision"`
	Path            string         `json:"path"`
	SizeBytes       int64          `json:"sizeBytes"`
	DownloadedBytes int64          `json:"downloadedBytes"`
	DownloadSpeed   string         `json:"downloadSpeed"`
	Status          string         `json:"status"`
	Message         string         `json:"message"`
	JobName         string         `json:"jobName"`
	CreatorID       uint           `json:"creatorId"`
	ReferenceCount  int            `json:"referenceCount"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	SourceUpdatedAt *time.Time     `json:"sourceUpdatedAt"`
	UserInfo        model.UserInfo `json:"userInfo"`
	CanManage       bool           `json:"canManage"`
	CanDelete       bool           `json:"canDelete"`
	CanViewLogs     bool           `json:"canViewLogs"`
	SourceURL       string         `json:"sourceUrl"`
	DisplayName     string         `json:"displayName"`
	License         string         `json:"license"`
	Task            string         `json:"task"`
	Library         string         `json:"library"`
	ModelType       string         `json:"modelType"`
	ParameterCount  int64          `json:"parameterCount"`
	SourceCreatedAt *time.Time     `json:"sourceCreatedAt"`
}

type ListDownloadsReq struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize,default=20"`
	Category string `form:"category"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

type ModelDownloadListResp struct {
	Total   int64               `json:"total"`
	Items   []ModelDownloadResp `json:"items"`
	Summary map[string]int64    `json:"summary"`
}

func bindDownloadAction(c *gin.Context) (DownloadActionReq, error) {
	var req DownloadActionReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		return req, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid download action request")
	}
	return req, nil
}

func (mgr *ModelDownloadMgr) canViewDownloadLogs(
	c *gin.Context, downloadID uint, token util.JWTMessage,
) (bool, error) {
	q := query.ModelDownload
	download, err := q.WithContext(c).Where(q.ID.Eq(downloadID)).First()
	if err != nil {
		return false, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found")
	}
	if download.CreatorID == token.UserID || token.RolePlatform == model.RoleAdmin {
		return true, nil
	}

	qUserDownload := query.UserModelDownload
	_, err = qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(downloadID)).
		First()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, bizerr.Internal.DatabaseError.Wrap(err, "check download log permission failed")
}

func (mgr *ModelDownloadMgr) applyLogViewPermissions(
	c *gin.Context, downloads []*model.ModelDownload, responses []ModelDownloadResp, token util.JWTMessage,
) error {
	if token.RolePlatform == model.RoleAdmin {
		for i := range responses {
			responses[i].CanViewLogs = true
		}
		return nil
	}

	ids := make([]uint, 0, len(downloads))
	for i, download := range downloads {
		if download.CreatorID == token.UserID {
			responses[i].CanViewLogs = true
			continue
		}
		ids = append(ids, download.ID)
	}
	if len(ids) == 0 {
		return nil
	}

	q := query.UserModelDownload
	associations, err := q.WithContext(c).
		Where(q.UserID.Eq(token.UserID), q.ModelDownloadID.In(ids...)).
		Find()
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "list download log permissions failed")
	}
	allowed := make(map[uint]struct{}, len(associations))
	for _, association := range associations {
		allowed[association.ModelDownloadID] = struct{}{}
	}
	for i, download := range downloads {
		if responses[i].CanViewLogs {
			continue
		}
		_, responses[i].CanViewLogs = allowed[download.ID]
	}
	return nil
}

// CreateDownload godoc
//
// associateUserWithDownload 关联用户与下载记录
func (mgr *ModelDownloadMgr) associateUserWithDownload(
	c *gin.Context, txQ *query.Query, userID uint, downloadID uint,
) error {
	txQUserDownload := txQ.UserModelDownload
	txQModelDownload := txQ.ModelDownload

	userDownload, _ := txQUserDownload.WithContext(c).
		Where(txQUserDownload.UserID.Eq(userID), txQUserDownload.ModelDownloadID.Eq(downloadID)).First()
	if userDownload == nil {
		if err := txQUserDownload.WithContext(c).Create(&model.UserModelDownload{
			UserID: userID, ModelDownloadID: downloadID}); err != nil {
			return fmt.Errorf("create association failed: %w", err)
		}
		_, _ = txQModelDownload.WithContext(c).Where(txQModelDownload.ID.Eq(downloadID)).UpdateSimple(txQModelDownload.ReferenceCount.Add(1))
	}
	return nil
}

// findReadyOrOngoingDownload only reuses a download with the exact requested
// upstream identity. The canonical storage path is shared by all variants, so
// a different source or revision must be reported as a conflict instead of
// silently satisfying the request with unrelated files.
func (mgr *ModelDownloadMgr) findReadyOrOngoingDownload(
	c *gin.Context, txQ *query.Query,
	name string, source model.ModelSource, category model.DownloadCategory, revision string,
) (*model.ModelDownload, error) {
	q := txQ.ModelDownload

	readyDownload, err := q.WithContext(c).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.Name.Eq(name), q.Source.Eq(string(source)), q.Category.Eq(string(category)),
			q.Revision.Eq(revision),
			q.Status.Eq(string(model.ModelDownloadStatusReady))).
		First()
	if err == nil {
		return readyDownload, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "find ready logical download")
	}

	ongoingDownload, err := q.WithContext(c).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.Name.Eq(name), q.Source.Eq(string(source)), q.Category.Eq(string(category)),
			q.Revision.Eq(revision),
			q.Status.In(string(model.ModelDownloadStatusPending), string(model.ModelDownloadStatusDownloading),
				string(model.ModelDownloadStatusPaused))).
		First()
	if err == nil {
		return ongoingDownload, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "find ongoing logical download")
	}
	return nil, nil
}

// lockModelDownloadIdentity serializes first-time downloads whose current
// database uniqueness still differs by source and revision. Existing
// installations may contain more than one historical record, so changing the
// unique index in place would make upgrades fail; a PostgreSQL transaction
// advisory lock prevents two new variants from racing into the same short path.
func lockModelDownloadIdentity(
	c *gin.Context, txQ *query.Query, name string, category model.DownloadCategory,
) error {
	db := txQ.ModelDownload.WithContext(c).UnderlyingDB()
	if db.Name() != "postgres" {
		return nil
	}
	identity := string(category) + ":" + name
	if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", identity).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "lock logical download identity")
	}
	return nil
}

// checkLogicalDownloadConflict blocks a new source/revision when a historical
// failed or soft-deleted record already owns storage for the same public model.
// Ready and ongoing records are handled earlier and reused instead.
func checkLogicalDownloadConflict(
	c *gin.Context, txQ *query.Query,
	name string, source model.ModelSource, category model.DownloadCategory, revision string,
) error {
	var conflict model.ModelDownload
	db := txQ.ModelDownload.WithContext(c).Unscoped().UnderlyingDB()
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("name = ? AND category = ? AND NOT (source = ? AND revision = ?)",
			name, category, source, revision).
		Order("id ASC").
		First(&conflict).Error
	if err == nil {
		return bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf(
			"model already has storage at %s from %s revision %q; reuse or resolve that record before downloading another source or revision",
			conflict.Path, conflict.Source, conflict.Revision,
		))
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Internal.DatabaseError.Wrap(err, "check logical download conflict")
	}
	return nil
}

// restoreAndResetSoftDeletedDownload 恢复并重置软删除的下载记录
func (mgr *ModelDownloadMgr) restoreAndResetSoftDeletedDownload(
	c *gin.Context, txQ *query.Query,
	name string, source model.ModelSource, category model.DownloadCategory, revision string, username string,
) (*model.ModelDownload, error) {
	q := txQ.ModelDownload

	softDeletedDownload, _ := q.WithContext(c).Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.Name.Eq(name), q.Source.Eq(string(source)),
			q.Category.Eq(string(category)), q.Revision.Eq(revision),
			q.DeletedAt.IsNotNull()).First()

	if softDeletedDownload == nil {
		return nil, nil
	}

	// 恢复软删除的记录
	_, err := q.WithContext(c).Unscoped().
		Where(q.ID.Eq(softDeletedDownload.ID)).
		Update(q.DeletedAt, nil)
	if err != nil {
		return nil, fmt.Errorf("restore soft-deleted record failed: %w", err)
	}

	// 如果是失败状态，重置为待下载
	if softDeletedDownload.Status == model.ModelDownloadStatusFailed {
		newJobName := fmt.Sprintf("model-dl-%s-%s", username, uuid.New().String()[:8])
		updates := map[string]any{
			"status":   model.ModelDownloadStatusPending,
			"message":  "",
			"job_name": newJobName,
		}
		_, err := q.WithContext(c).Where(q.ID.Eq(softDeletedDownload.ID)).Updates(updates)
		if err != nil {
			return nil, fmt.Errorf("update soft-deleted record failed: %w", err)
		}
		softDeletedDownload.Status = model.ModelDownloadStatusPending
		softDeletedDownload.JobName = newJobName
	}

	return softDeletedDownload, nil
}

// resetFailedDownload 重置失败的下载记录
func (mgr *ModelDownloadMgr) resetFailedDownload(
	c *gin.Context, txQ *query.Query,
	name string, source model.ModelSource, category model.DownloadCategory, revision string, username string,
) (*model.ModelDownload, error) {
	q := txQ.ModelDownload

	failedDownload, _ := q.WithContext(c).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.Name.Eq(name), q.Source.Eq(string(source)),
			q.Category.Eq(string(category)), q.Revision.Eq(revision),
			q.Status.Eq(string(model.ModelDownloadStatusFailed))).First()

	if failedDownload == nil {
		return nil, nil
	}

	// 重置失败的记录
	newJobName := fmt.Sprintf("model-dl-%s-%s", username, uuid.New().String()[:8])
	updates := map[string]any{
		"status":   model.ModelDownloadStatusPending,
		"message":  "",
		"job_name": newJobName,
	}
	_, err := q.WithContext(c).Where(q.ID.Eq(failedDownload.ID)).Updates(updates)
	if err != nil {
		return nil, fmt.Errorf("update failed download failed: %w", err)
	}
	failedDownload.Status = model.ModelDownloadStatusPending
	failedDownload.JobName = newJobName

	return failedDownload, nil
}

// getOrCreateDownload 在事务中获取或创建下载任务
func (mgr *ModelDownloadMgr) getOrCreateDownload(
	c *gin.Context, req *CreateDownloadReq, token util.JWTMessage,
	source model.ModelSource, category model.DownloadCategory, downloadPath, revision string,
) (*model.ModelDownload, bool, error) {
	db := query.Use(query.GetDB())
	var download *model.ModelDownload
	var isNewDownload bool

	err := db.Transaction(func(tx *query.Query) error {
		if err := lockModelDownloadIdentity(c, tx, req.Name, category); err != nil {
			return err
		}

		// 1. 查找已完成或正在进行的下载
		existing, err := mgr.findReadyOrOngoingDownload(
			c, tx, req.Name, source, category, revision,
		)
		if err != nil {
			return err
		}
		if existing != nil {
			if err := mgr.associateUserWithDownload(c, tx, token.UserID, existing.ID); err != nil {
				return err
			}
			download, isNewDownload = existing, false
			return nil
		}

		if err := checkLogicalDownloadConflict(c, tx, req.Name, source, category, revision); err != nil {
			return err
		}

		// 2. 查找并恢复软删除的记录
		restored, err := mgr.restoreAndResetSoftDeletedDownload(c, tx, req.Name, source, category, revision, token.Username)
		if err != nil {
			return err
		}
		if restored != nil {
			if err := mgr.associateUserWithDownload(c, tx, token.UserID, restored.ID); err != nil {
				return err
			}
			download, isNewDownload = restored, shouldSubmitRestoredDownload(restored)
			return nil
		}

		// 3. 重置失败的记录
		failed, err := mgr.resetFailedDownload(c, tx, req.Name, source, category, revision, token.Username)
		if err != nil {
			return err
		}
		if failed != nil {
			if err := mgr.associateUserWithDownload(c, tx, token.UserID, failed.ID); err != nil {
				return err
			}
			download, isNewDownload = failed, true
			return nil
		}

		// 4. 创建新的下载任务
		txQ := tx.ModelDownload
		txQUserDownload := tx.UserModelDownload
		newDownload := &model.ModelDownload{
			Name: req.Name, Source: source, Category: category, Revision: revision, Path: downloadPath,
			Status: model.ModelDownloadStatusPending, JobName: fmt.Sprintf("model-dl-%s-%s", token.Username, uuid.New().String()[:8]),
			CreatorID: token.UserID, ReferenceCount: 1,
		}
		if err := txQ.WithContext(c).Create(newDownload); err != nil {
			return fmt.Errorf("create download failed: %w", err)
		}
		if err := txQUserDownload.WithContext(c).Create(&model.UserModelDownload{
			UserID: token.UserID, ModelDownloadID: newDownload.ID}); err != nil {
			return fmt.Errorf("create association failed: %w", err)
		}
		download, isNewDownload = newDownload, true
		return nil
	})

	return download, isNewDownload, err
}

func shouldSubmitRestoredDownload(download *model.ModelDownload) bool {
	return download.Status != model.ModelDownloadStatusReady
}

// @Summary		创建模型下载任务
// @Description	创建一个新的模型下载任务
// @Tags			ModelDownload
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		CreateDownloadReq		true	"下载请求"
// @Success		200		{object}	resputil.Response[ModelDownloadResp]
// @Router			/v1/models/download [POST]
func (mgr *ModelDownloadMgr) CreateDownload(c *gin.Context) {
	var req CreateDownloadReq
	token := util.GetToken(c)

	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}

	if !isValidModelName(req.Name) {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid model name format, expected: owner/model-name"))
		return
	}

	// 设置默认来源
	source := model.ModelSourceModelScope
	if req.Source != "" {
		source = model.ModelSource(req.Source)
	}
	if source != model.ModelSourceModelScope && source != model.ModelSourceHuggingFace {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("source must be modelscope or huggingface"))
		return
	}
	if len(req.Revision) > maxDownloadRevisionLength {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("revision must not exceed 128 characters"))
		return
	}

	// 设置分类
	category := model.DownloadCategory(req.Category)

	// 生成安全的路径名
	safeName := sanitizeModelName(req.Name)

	// 根据category自动确定下载路径: public/Models/ 或 public/Datasets/
	var basePath string
	if category == model.DownloadCategoryModel {
		basePath = "public/Models"
	} else {
		basePath = "public/Datasets"
	}
	downloadPath := modelDownloadStoragePath(basePath, safeName)

	// 在事务中获取或创建下载任务
	download, isNewDownload, err := mgr.getOrCreateDownload(c, &req, token, source, category, downloadPath, req.Revision)
	if err != nil {
		klog.Errorf("get or create download failed: %v", err)
		resputil.HandleError(c, err)
		return
	}

	// 如果是已存在的下载（Ready 或 Ongoing）
	if !isNewDownload {
		resp := convertDownloadToResp(download, token)
		resp.CanViewLogs = true
		if download.Status == model.ModelDownloadStatusReady {
			c.JSON(http.StatusOK, gin.H{
				"code": resputil.OK,
				"data": resp,
				"msg":  fmt.Sprintf("该资源已下载完成，位置: %s", download.Path),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code": resputil.OK,
				"data": resp,
				"msg":  "该资源正在下载中，已将您加入共享列表",
			})
		}
		return
	}

	// 提交 K8s Job
	if err := mgr.submitDownloadJob(c, download, token.Username, req.Token); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		q := query.ModelDownload
		updates := map[string]any{
			"status":  model.ModelDownloadStatusFailed,
			"message": fmt.Sprintf("submit job failed: %v", err),
		}
		_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(updates)

		resputil.Error(c, "submit download job failed", resputil.NotSpecified)
		return
	}

	// 更新状态为 Downloading
	q := query.ModelDownload
	_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Update(q.Status, model.ModelDownloadStatusDownloading)

	resputil.Success(c, convertDownloadToResp(download, token))
}

// ListDownloads godoc
//
//	@Summary		获取模型下载任务列表
//	@Description	下载记录对全平台用户可见。带 page 参数时返回分页结构(含状态汇总),否则返回全量数组(兼容旧客户端)
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			category	query		string	false	"过滤类别: model 或 dataset"
//	@Param			page		query		int		false	"页码(从1开始);不传则返回全量数组"
//	@Param			pageSize	query		int		false	"每页数量,默认20,最大100"
//	@Param			status		query		string	false	"过滤状态: Pending/Downloading/Paused/Ready/Failed"
//	@Param			search		query		string	false	"按名称模糊搜索"
//	@Success		200	{object}	resputil.Response[ModelDownloadListResp]
//	@Router			/v1/models/downloads [GET]
//
//nolint:gocyclo // Pagination, compatibility mode, filters, and permission enrichment share one endpoint contract.
func (mgr *ModelDownloadMgr) ListDownloads(c *gin.Context) {
	token := util.GetToken(c)

	var req ListDownloadsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request parameter"))
		return
	}

	q := query.ModelDownload
	builder := q.WithContext(c).Preload(q.Creator)
	if req.Category == CategoryModel || req.Category == CategoryDataset {
		builder = builder.Where(q.Category.Eq(req.Category))
	}

	// 兼容模式(无 page 参数, 旧版 CLI/前端): 返回全量数组
	if req.Page <= 0 {
		downloads, err := builder.Order(q.CreatedAt.Desc()).Find()
		if err != nil {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list downloads failed"))
			return
		}
		resp := make([]ModelDownloadResp, len(downloads))
		for i, d := range downloads {
			resp[i] = convertDownloadToResp(d, token)
		}
		if err := mgr.applyLogViewPermissions(c, downloads, resp, token); err != nil {
			resputil.HandleError(c, err)
			return
		}
		resputil.Success(c, resp)
		return
	}

	if req.Status != "" {
		builder = builder.Where(q.Status.Eq(req.Status))
	}
	if req.Search != "" {
		builder = builder.Where(q.Name.Like("%" + req.Search + "%"))
	}

	const maxPageSize = 100
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > maxPageSize {
		req.PageSize = maxPageSize
	}

	// 按更新时间倒序:刚重试/正在下载的任务排在最前
	downloads, total, err := builder.Order(q.UpdatedAt.Desc()).FindByPage((req.Page-1)*req.PageSize, req.PageSize)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list downloads failed"))
		return
	}

	// 状态汇总只按 category 过滤,反映整体情况而不受搜索/状态筛选影响
	summary, err := mgr.downloadStatusSummary(c, req.Category)
	if err != nil {
		klog.Warningf("count download status summary failed: %v", err)
		summary = map[string]int64{}
	}

	items := make([]ModelDownloadResp, len(downloads))
	for i, d := range downloads {
		items[i] = convertDownloadToResp(d, token)
	}
	if err := mgr.applyLogViewPermissions(c, downloads, items, token); err != nil {
		resputil.HandleError(c, err)
		return
	}

	resputil.Success(c, ModelDownloadListResp{Total: total, Items: items, Summary: summary})
}

// downloadStatusSummary returns record counts grouped by status, optionally
// scoped to a category.
func (mgr *ModelDownloadMgr) downloadStatusSummary(c *gin.Context, category string) (map[string]int64, error) {
	q := query.ModelDownload
	builder := q.WithContext(c)
	if category == CategoryModel || category == CategoryDataset {
		builder = builder.Where(q.Category.Eq(category))
	}

	var rows []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := builder.Select(q.Status, q.ID.Count().As("count")).Group(q.Status).Scan(&rows); err != nil {
		return nil, err
	}

	summary := make(map[string]int64, len(rows))
	for _, row := range rows {
		summary[row.Status] = row.Count
	}
	return summary, nil
}

// GetDownload godoc
//
//	@Summary		获取单个模型下载任务详情
//	@Description	根据 ID 获取模型下载任务详情
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id} [GET]
func (mgr *ModelDownloadMgr) GetDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}
	// 下载记录对所有登录用户可见
	q := query.ModelDownload
	download, err := q.WithContext(c).
		Preload(q.Creator).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found"))
		return
	}

	resp := convertDownloadToResp(download, token)
	canViewLogs, err := mgr.canViewDownloadLogs(c, download.ID, token)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resp.CanViewLogs = canViewLogs

	resputil.Success(c, resp)
}

// requireManagePermission allows only the creator or a platform administrator
// to mutate a download task.
func (mgr *ModelDownloadMgr) requireManagePermission(c *gin.Context, downloadID uint) (*model.ModelDownload, error) {
	q := query.ModelDownload
	download, err := q.WithContext(c).Where(q.ID.Eq(downloadID)).First()
	if err != nil {
		return nil, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found")
	}

	token := util.GetToken(c)
	if download.CreatorID != token.UserID && token.RolePlatform != model.RoleAdmin {
		return nil, bizerr.Forbidden.PermissionDenied.New("only the creator or a platform administrator can manage this download")
	}
	return download, nil
}

// RetryDownload godoc
//
//	@Summary		重试失败的下载任务
//	@Description	重新提交失败的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Param			data	body		DownloadActionReq	false	"可选的临时访问令牌和重试版本"
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id}/retry [POST]
func (mgr *ModelDownloadMgr) RetryDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}
	action, err := bindDownloadAction(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := normalizeRetryRevision(&action); err != nil {
		resputil.HandleError(c, err)
		return
	}

	if _, err := mgr.requireManagePermission(c, req.ID); err != nil {
		resputil.HandleError(c, err)
		return
	}

	download, err := prepareRetryDownload(c, req.ID, token.Username, action.Revision)

	if err != nil {
		klog.Errorf("retry download transaction failed: %v", err)
		resputil.HandleError(c, err)
		return
	}
	q := query.ModelDownload

	// 事务成功后提交 Job
	if err := mgr.submitDownloadJob(c, download, token.Username, action.Token); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		// 回滚状态
		updates := map[string]any{
			"status":  model.ModelDownloadStatusFailed,
			"message": fmt.Sprintf("submit job failed: %v", err),
		}
		_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(updates)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "submit download job failed"))
		return
	}

	download.Status = model.ModelDownloadStatusDownloading
	download.Message = ""

	resputil.Success(c, convertDownloadToResp(download, token))
}

func normalizeRetryRevision(action *DownloadActionReq) error {
	if action.Revision == nil {
		return nil
	}
	revision := strings.TrimSpace(*action.Revision)
	if len(revision) > maxDownloadRevisionLength {
		return bizerr.BadRequest.ParameterError.New("revision must not exceed 128 characters")
	}
	action.Revision = &revision
	return nil
}

// prepareRetryDownload updates the failed record under a row lock. Correcting
// the revision deliberately does not rewrite Path: the source SDK receives the
// same local directory and can validate/reuse files from the failed attempt.
func prepareRetryDownload(
	c *gin.Context, downloadID uint, username string, revision *string,
) (*model.ModelDownload, error) {
	db := query.Use(query.GetDB())
	var download *model.ModelDownload
	err := db.Transaction(func(tx *query.Query) error {
		txQ := tx.ModelDownload
		d, err := txQ.WithContext(c).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(txQ.ID.Eq(downloadID)).
			First()
		if err != nil {
			return bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found")
		}
		if d.Status != model.ModelDownloadStatusFailed {
			return bizerr.Conflict.ResourceStatusError.New(
				fmt.Sprintf("only failed downloads can be retried, current status: %s", d.Status),
			)
		}
		if err := checkRetryRevisionConflict(c, tx, d, revision); err != nil {
			return err
		}

		newJobName := fmt.Sprintf("model-dl-%s-%s", username, uuid.New().String()[:8])
		updates := map[string]any{
			"status": model.ModelDownloadStatusDownloading, "message": "", "job_name": newJobName,
		}
		if revision != nil {
			updates["revision"] = *revision
		}
		if _, err = txQ.WithContext(c).Where(txQ.ID.Eq(d.ID)).Updates(updates); err != nil {
			return retryUpdateError(err)
		}

		d.Status, d.JobName, d.Message = model.ModelDownloadStatusDownloading, newJobName, ""
		if revision != nil {
			d.Revision = *revision
		}
		download = d
		return nil
	})
	return download, err
}

func checkRetryRevisionConflict(
	c *gin.Context, txQ *query.Query, download *model.ModelDownload, revision *string,
) error {
	if revision == nil || *revision == download.Revision {
		return nil
	}
	conflict, err := txQ.ModelDownload.WithContext(c).Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(txQ.ModelDownload.ID.Neq(download.ID), txQ.ModelDownload.Name.Eq(download.Name),
			txQ.ModelDownload.Source.Eq(string(download.Source)),
			txQ.ModelDownload.Category.Eq(string(download.Category)),
			txQ.ModelDownload.Revision.Eq(*revision)).
		First()
	if err == nil && conflict != nil {
		return bizerr.Conflict.ResourceStatusError.New("download with requested revision already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Internal.DatabaseError.Wrap(err, "check retry revision conflict")
	}
	return nil
}

func retryUpdateError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return bizerr.Conflict.ResourceStatusError.New("download with requested revision already exists")
	}
	var sqlStateError interface{ SQLState() string }
	if errors.As(err, &sqlStateError) && sqlStateError.SQLState() == "23505" {
		return bizerr.Conflict.ResourceStatusError.New("download with requested revision already exists")
	}
	return bizerr.Internal.DatabaseError.Wrap(err, "update download record failed")
}

// DeleteDownload godoc
//
//	@Summary		删除模型下载任务
//	@Description	删除下载任务记录(仅平台管理员),已下载的文件保留在存储中
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Router			/v1/models/downloads/{id} [DELETE]
func (mgr *ModelDownloadMgr) DeleteDownload(c *gin.Context) {
	token := util.GetToken(c)
	if !canDeleteDownload(token) {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("only a platform administrator can delete download records"))
		return
	}
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}
	q := query.ModelDownload
	download, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found"))
		return
	}

	// Capture the latest logs and stop an active Job before deleting its record.
	if download.JobName != "" &&
		(download.Status == model.ModelDownloadStatusDownloading || download.Status == model.ModelDownloadStatusPending) {
		mgr.captureJobLogsToRecord(c, download)
		if err := mgr.deleteDownloadJob(c, download.JobName); err != nil {
			resputil.HandleError(c, err)
			return
		}
	}

	if err := mgr.deleteDownloadRecord(c, req.ID); err != nil {
		resputil.HandleError(c, err)
		return
	}

	klog.Infof("Download record %d deleted, files preserved at: %s", req.ID, download.Path)
	resputil.Success(c, "deleted successfully")
}

func canDeleteDownload(token util.JWTMessage) bool {
	return token.RolePlatform == model.RoleAdmin
}

// PauseDownload godoc
//
//	@Summary		暂停下载任务
//	@Description	暂停正在进行的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id}/pause [POST]
func (mgr *ModelDownloadMgr) PauseDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}
	download, err := mgr.requireManagePermission(c, req.ID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	q := query.ModelDownload

	if download.Status != model.ModelDownloadStatusDownloading {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only downloading tasks can be paused"))
		return
	}

	// Pausing is implemented by deleting the Job. Persist logs before removal.
	mgr.captureJobLogsToRecord(c, download)
	if err := mgr.deleteDownloadJob(c, download.JobName); err != nil {
		resputil.HandleError(c, err)
		return
	}

	updates := map[string]any{
		"status":  model.ModelDownloadStatusPaused,
		"message": "Download paused by user",
	}
	if _, err := q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(updates); err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "update paused download failed"))
		return
	}

	download.Status = model.ModelDownloadStatusPaused
	download.Message = "Download paused by user"

	resputil.Success(c, convertDownloadToResp(download, token))
}

// ResumeDownload godoc
//
//	@Summary		恢复下载任务
//	@Description	恢复已暂停的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Param			data	body		DownloadActionReq	false	"可选的临时访问令牌"
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id}/resume [POST]
func (mgr *ModelDownloadMgr) ResumeDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}
	action, err := bindDownloadAction(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	download, err := mgr.requireManagePermission(c, req.ID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	q := query.ModelDownload
	db := query.Use(query.GetDB())
	err = db.Transaction(func(tx *query.Query) error {
		txQ := tx.ModelDownload
		locked, lockErr := txQ.WithContext(c).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(txQ.ID.Eq(download.ID)).
			First()
		if lockErr != nil {
			return lockErr
		}
		if locked.Status != model.ModelDownloadStatusPaused {
			return bizerr.Conflict.ResourceStatusError.New("only paused tasks can be resumed")
		}

		newJobName := fmt.Sprintf("model-dl-%s-%s", token.Username, uuid.New().String()[:8])
		if _, updateErr := txQ.WithContext(c).Where(txQ.ID.Eq(locked.ID)).Updates(map[string]any{
			"status":   model.ModelDownloadStatusDownloading,
			"message":  "",
			"job_name": newJobName,
		}); updateErr != nil {
			return updateErr
		}
		locked.Status = model.ModelDownloadStatusDownloading
		locked.JobName = newJobName
		locked.Message = ""
		download = locked
		return nil
	})
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	// 提交新 Job (会从已下载的部分继续)
	if err := mgr.submitDownloadJob(c, download, token.Username, action.Token); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		// 回滚状态
		rollbackUpdates := map[string]any{
			"status":  model.ModelDownloadStatusPaused,
			"message": fmt.Sprintf("resume failed: %v", err),
		}
		_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(rollbackUpdates)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "submit download job failed"))
		return
	}

	download.Status = model.ModelDownloadStatusDownloading
	download.Message = ""

	resputil.Success(c, convertDownloadToResp(download, token))
}

// ListAllDownloads godoc
//
//	@Summary		管理员获取所有模型下载任务
//	@Description	管理员查看所有用户的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]ModelDownloadResp]
//	@Router			/v1/admin/models/downloads [GET]
func (mgr *ModelDownloadMgr) ListAllDownloads(c *gin.Context) {
	token := util.GetToken(c)
	q := query.ModelDownload

	downloads, err := q.WithContext(c).
		Preload(q.Creator).
		Order(q.CreatedAt.Desc()).
		Find()
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list downloads failed"))
		return
	}

	resp := make([]ModelDownloadResp, len(downloads))
	for i, d := range downloads {
		resp[i] = convertDownloadToResp(d, token)
	}

	resputil.Success(c, resp)
}

// AdminDeleteDownload godoc
//
//	@Summary		管理员删除模型下载任务
//	@Description	管理员删除任意用户的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Router			/v1/admin/models/downloads/{id} [DELETE]
func (mgr *ModelDownloadMgr) AdminDeleteDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}

	q := query.ModelDownload
	download, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found"))
		return
	}
	if download.JobName != "" &&
		(download.Status == model.ModelDownloadStatusDownloading || download.Status == model.ModelDownloadStatusPending) {
		mgr.captureJobLogsToRecord(c, download)
		if err := mgr.deleteDownloadJob(c, download.JobName); err != nil {
			resputil.HandleError(c, err)
			return
		}
	}
	if err := mgr.deleteDownloadRecord(c, req.ID); err != nil {
		resputil.HandleError(c, err)
		return
	}

	resputil.Success(c, "deleted successfully")
}

// downloadJobBackoffLimit lets the Job retry transient failures (network blips)
// a couple of times before being marked Failed.
const downloadJobBackoffLimit int32 = 2

const (
	defaultModelDownloaderImage = "ghcr.io/raids-lab/crater-model-downloader:v1.0.0"
	huggingFaceHubVersion       = "1.23.0"
	modelScopeVersion           = "1.38.1"
	modelScopeHubVersion        = "0.1.7"
)

func (mgr *ModelDownloadMgr) submitDownloadJob(c *gin.Context, download *model.ModelDownload, username, accessToken string) error {
	physicalPath := mgr.convertToPhysicalPath(download.Path)
	subPath := filepath.Dir(physicalPath)
	modelDirName := filepath.Base(physicalPath)

	downloadCmd := mgr.buildDownloadCommand(download, modelDirName)
	memRequest, memLimit := mgr.memoryForModel(download.Name)
	backoffLimit := downloadJobBackoffLimit

	// 创建 Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      download.JobName,
			Namespace: mgr.namespace,
			Labels: map[string]string{
				"app":               "model-download",
				"model-download-id": fmt.Sprintf("%d", download.ID),
				"user":              username,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr.To(utils.SevenDaySeconds),
			BackoffLimit:            &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":               "model-download",
						"model-download-id": fmt.Sprintf("%d", download.ID),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:    corev1.RestartPolicyNever,
					ImagePullSecrets: downloadImagePullSecrets(config.GetConfig().Secrets.ImagePullSecretName),
					Containers: []corev1.Container{
						{
							Name:    "downloader",
							Image:   mgr.getDownloadImage(download.Source),
							Command: []string{"/bin/bash", "-c"},
							Args:    []string{downloadCmd},
							Env:     downloadTokenEnv(download.Source, accessToken),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: memRequest,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("3"),
									corev1.ResourceMemory: memLimit,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "crater-storage",
									MountPath: "/data",
									SubPath:   subPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "crater-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: config.GetConfig().Storage.PVC.ReadWriteMany,
								},
							},
						},
					},
				},
			},
		},
	}

	// 提交 Job 到集群
	_, err := mgr.crClient.BatchV1().Jobs(mgr.namespace).Create(c, job, metav1.CreateOptions{})
	return err
}

// downloadImagePullSecrets keeps the public-image path credential-free while
// allowing deployments to use a private, internally mirrored downloader image.
// The secret itself remains an administrator-owned deployment setting; download
// requests never accept or persist registry credentials.
func downloadImagePullSecrets(secretName string) []corev1.LocalObjectReference {
	if secretName == "" {
		return nil
	}
	return []corev1.LocalObjectReference{{Name: secretName}}
}

// downloadTokenEnv injects the access token for the given source as the env var
// the corresponding SDK expects, when a token is provided.
func downloadTokenEnv(source model.ModelSource, accessToken string) []corev1.EnvVar {
	if accessToken == "" {
		return nil
	}
	switch source {
	case model.ModelSourceHuggingFace:
		return []corev1.EnvVar{
			{Name: "HF_TOKEN", Value: accessToken},
			{Name: "HUGGING_FACE_HUB_TOKEN", Value: accessToken},
		}
	case model.ModelSourceModelScope:
		return []corev1.EnvVar{{Name: "MODELSCOPE_API_TOKEN", Value: accessToken}}
	default:
		return nil
	}
}

// Parameter-count thresholds (in billions) used to size download job memory.
const (
	paramThresholdHuge   = 70
	paramThresholdLarge  = 30
	paramThresholdMedium = 13
)

// memoryForModel returns request/limit memory sized by hints in the model name.
// Larger parameter counts need more RAM to stage shards before flushing to disk.
func (mgr *ModelDownloadMgr) memoryForModel(name string) (request, limit resource.Quantity) {
	lower := strings.ToLower(name)
	billions := parseParamBillions(lower)
	switch {
	case billions >= paramThresholdHuge:
		return resource.MustParse("4Gi"), resource.MustParse("24Gi")
	case billions >= paramThresholdLarge:
		return resource.MustParse("4Gi"), resource.MustParse("16Gi")
	case billions >= paramThresholdMedium:
		return resource.MustParse("2Gi"), resource.MustParse("12Gi")
	default:
		return resource.MustParse("2Gi"), resource.MustParse("6Gi")
	}
}

// parseParamBillions extracts a parameter count in billions from a model name
// such as "Qwen/Qwen2.5-7B" or "meta-llama/Llama-3.2-70b". Returns 0 if unknown.
func parseParamBillions(lowerName string) float64 {
	matches := paramSizePattern.FindAllStringSubmatch(lowerName, -1)
	var maxB float64
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err == nil && v > maxB {
			maxB = v
		}
	}
	return maxB
}

var paramSizePattern = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*b\b`)

func (mgr *ModelDownloadMgr) getDownloadImage(_ model.ModelSource) string {
	// 优先使用配置文件中的镜像
	if img := config.GetConfig().ModelDownload.Image; img != "" {
		return img
	}
	// The public default is reproducible and can be mirrored or overridden by each deployment.
	return defaultModelDownloaderImage
}

func (mgr *ModelDownloadMgr) buildDownloadCommand(download *model.ModelDownload, modelDirName string) string {
	var installCmd, preflightCmd, downloadCommand, totalProbeCmd, metadataCmd string
	huggingFaceEndpoint := config.GetConfig().HuggingFaceDownloadEndpoint()
	disableXetCmd := ""
	if shouldDisableHuggingFaceXet(download.Source, huggingFaceEndpoint) {
		disableXetCmd = "export HF_HUB_DISABLE_XET=1"
	}
	if download.Source == model.ModelSourceHuggingFace {
		// The project image already contains this dependency. The fallback preserves
		// compatibility with existing custom Python images used by open-source deployments.
		installCmd = fmt.Sprintf(
			`if ! python -c 'import huggingface_hub' >/dev/null 2>&1; then
    echo "huggingface_hub is missing; installing the tested fallback version"
    pip install --no-cache-dir 'huggingface_hub==%s'
fi
python -c 'import huggingface_hub; print("[TOOLS] huggingface_hub=" + huggingface_hub.__version__)'`,
			huggingFaceHubVersion,
		)

		// 2. 用 Python API snapshot_download，而不是 huggingface-cli
		// 根据类别设置 repo_type
		var repoType string
		if download.Category == model.DownloadCategoryDataset {
			repoType = "dataset"
		} else {
			repoType = "model"
		}

		downloadCommand = fmt.Sprintf(`
python - << 'PY'
import os
from huggingface_hub import snapshot_download

repo_id = %q
revision = %q
repo_type = %q

kwargs = {
    "repo_id": repo_id,
    "repo_type": repo_type,
    "local_dir": os.environ["OUT_DIR"],
    "local_dir_use_symlinks": False,
    "resume_download": True,
}
if revision:
    kwargs["revision"] = revision

snapshot_download(**kwargs)
PY
`, download.Name, download.Revision, repoType)

		// Best-effort total-size probe so the UI can render a real progress bar.
		// Runs inside `|| true` so a probe failure never fails the download job.
		totalProbeCmd = fmt.Sprintf(`
echo "Probing repository total size..."
python - << 'PY' || true
try:
    from huggingface_hub import HfApi
    info = HfApi().repo_info(%q, repo_type=%q, revision=%q or None, files_metadata=True)
    total = sum((s.size or 0) for s in (info.siblings or []))
    if total > 0:
        print(f"[TOTAL] size_bytes={total}", flush=True)
except Exception:
    pass
PY
`, download.Name, repoType, download.Revision)

		metadataCmd = fmt.Sprintf(`
echo "Reading repository metadata..."
python - << 'PY' || true
import json
import os
import urllib.error
import urllib.parse
import urllib.request
from huggingface_hub import HfApi

info = HfApi().repo_info(%q, repo_type=%q, revision=%q or None)
updated_at = getattr(info, "last_modified", None) or getattr(info, "lastModified", None)
created_at = getattr(info, "created_at", None) or getattr(info, "createdAt", None)
card_data = getattr(info, "card_data", None) or getattr(info, "cardData", None) or {}
config = getattr(info, "config", None) or {}
safetensors = getattr(info, "safetensors", None) or {}
license_name = card_data.get("license", "") if isinstance(card_data, dict) else getattr(card_data, "license", "")
model_type = config.get("model_type", "") if isinstance(config, dict) else getattr(config, "model_type", "")
parameter_count = safetensors.get("total", 0) if isinstance(safetensors, dict) else getattr(safetensors, "total", 0)
owner = %q.split("/", 1)[0]
avatar_url = ""
for owner_type in ("organizations", "users"):
    endpoint = "{}/api/{}/{}/overview".format(
        os.environ.get("HF_ENDPOINT", "https://huggingface.co").rstrip("/"),
        owner_type,
        urllib.parse.quote(owner, safe=""),
    )
    try:
        with urllib.request.urlopen(endpoint, timeout=10) as response:
            avatar_url = json.load(response).get("avatarUrl", "")
        break
    except urllib.error.HTTPError as error:
        if error.code != 404:
            break
    except Exception:
        break
print("[META] " + json.dumps({
	"display_name": %q.split("/", 1)[-1],
	"license": str(license_name or ""),
	"task": str(getattr(info, "pipeline_tag", "") or ""),
	"library": str(getattr(info, "library_name", "") or ""),
	"model_type": str(model_type or ""),
	"parameter_count": int(parameter_count or 0),
	"private": bool(getattr(info, "private", False)),
	"gated": bool(getattr(info, "gated", False)),
    "downloads": int(getattr(info, "downloads", 0) or 0),
    "likes": int(getattr(info, "likes", 0) or 0),
    "logo_url": avatar_url,
	"created_at": created_at.isoformat() if created_at else "",
    "updated_at": updated_at.isoformat() if updated_at else "",
    "tags": list(getattr(info, "tags", None) or [])[:4],
}, separators=(",", ":")), flush=True)
PY
`, download.Name, repoType, download.Revision, download.Name, download.Name)
	} else {
		// The project image already contains these dependencies. The fallback keeps
		// user-supplied Python images working while avoiding an unpinned installation.
		installCmd = fmt.Sprintf(
			`if ! python -c 'import modelscope, modelscope_hub' >/dev/null 2>&1; then
    echo "ModelScope clients are missing; installing the tested fallback versions"
    pip install --no-cache-dir 'modelscope==%s' 'modelscope-hub==%s'
fi
python - << 'PY'
import modelscope
import modelscope_hub
print("[TOOLS] modelscope={} modelscope_hub={}".format(modelscope.__version__, modelscope_hub.__version__))
PY`,
			modelScopeVersion, modelScopeHubVersion,
		)

		// Validate an explicit revision before downloading. ModelScope's legacy file
		// API currently returns Files=null for an unknown branch, which otherwise
		// surfaces as an opaque "NoneType is not iterable" client error.
		resourcePath := "models"
		if download.Category == model.DownloadCategoryDataset {
			resourcePath = "datasets"
		}
		preflightCmd = fmt.Sprintf(`
echo "Validating ModelScope revision..."
python - << 'PY'
import json
import os
import sys
import urllib.parse
import urllib.request

repo_id = %q
revision = %q
resource_path = %q
if revision:
    endpoint = os.environ.get("MODELSCOPE_ENDPOINT", "https://modelscope.cn").rstrip("/")
    url = endpoint + "/api/v1/{}/{}/revisions".format(
        resource_path, urllib.parse.quote(repo_id, safe="/"))
    request = urllib.request.Request(url)
    token = os.environ.get("MODELSCOPE_API_TOKEN", "")
    if token:
        request.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            payload = json.load(response)
        revision_map = (payload.get("Data") or payload.get("data") or {}).get("RevisionMap") or {}
        entries = (revision_map.get("Branches") or []) + (revision_map.get("Tags") or [])
        available = sorted({entry.get("Revision") for entry in entries if entry.get("Revision")})
    except Exception as error:
        print("[WARN] revision validation unavailable: {}".format(error), flush=True)
    else:
        if available and revision not in available:
            print(
                "[ERROR] revision_not_found: {!r} is not available for {}; available revisions: {}".format(
                    revision, repo_id, ", ".join(available)
                ),
                file=sys.stderr,
                flush=True,
            )
            raise SystemExit(22)
PY
`, download.Name, download.Revision, resourcePath)

		// Invoke the CLI through an argument array so
		// user-provided revisions never pass through shell parsing.
		var resourceFlag string
		if download.Category == model.DownloadCategoryDataset {
			resourceFlag = "--dataset"
		} else {
			resourceFlag = "--model"
		}

		downloadCommand = fmt.Sprintf(`
python - << 'PY'
import os
import subprocess

resource_flag = %q
repo_id = %q
revision = %q
args = ["modelscope", "download", resource_flag, repo_id]
if revision:
    args.extend(["--revision", revision])
args.extend(["--local_dir", os.environ["OUT_DIR"]])
subprocess.run(args, check=True)
PY
`, resourceFlag, download.Name, download.Revision)

		// Best-effort total-size probe so the UI can render a real progress bar.
		// Runs inside `|| true` so a probe failure never fails the download job.
		totalProbeCmd = fmt.Sprintf(`
echo "Probing repository total size..."
python - << 'PY' || true
try:
    from modelscope.hub.api import HubApi
    api = HubApi()
    name, revision, category = %q, %q, %q
    if category == "dataset":
        files = api.get_dataset_files(repo_id=name, revision=revision or "master", root_path="/", recursive=True)
    else:
        files = api.get_model_files(name, revision=revision or None, recursive=True)
    total = sum(int(f.get("Size") or 0) for f in (files or []) if f.get("Type") == "blob")
    if total > 0:
        print(f"[TOTAL] size_bytes={total}", flush=True)
except Exception:
    pass
PY
`, download.Name, download.Revision, string(download.Category))

		metadataCmd = fmt.Sprintf(`
echo "Reading repository metadata..."
python - << 'PY' || true
import json
import os
import urllib.request

url = os.environ.get("MODELSCOPE_ENDPOINT", "https://modelscope.cn").rstrip("/") + "/openapi/v1/%s/%s"
request = urllib.request.Request(url)
token = os.environ.get("MODELSCOPE_API_TOKEN", "")
if token:
    request.add_header("Authorization", "Bearer " + token)
with urllib.request.urlopen(request, timeout=20) as response:
    payload = json.load(response).get("data", {})
print("[META] " + json.dumps({
	"display_name": payload.get("display_name") or "",
	"description": (payload.get("description") or "")[:500],
	"license": payload.get("license") or "",
	"task": (payload.get("tasks") or [""])[0],
	"library": next((tag.split(":", 1)[1] for tag in payload.get("tags", []) if tag.startswith("library:")), ""),
	"model_type": next((tag.split(":", 1)[1] for tag in payload.get("tags", []) if tag.startswith("model_type:")), ""),
	"parameter_count": int(payload.get("params") or 0),
	"private": bool(payload.get("private", False)),
	"gated": bool(payload.get("gated", False)),
	"login_required": bool(payload.get("login_required", False)),
    "downloads": int(payload.get("downloads") or 0),
    "likes": int(payload.get("likes") or 0),
	"created_at": payload.get("created_at") or "",
    "updated_at": payload.get("last_modified") or "",
	"tags": list(dict.fromkeys((payload.get("tasks") or []) + (payload.get("tags") or [])))[:8],
}, separators=(",", ":")), flush=True)
PY
`, resourcePath, download.Name)
	}

	// Capture the README from the exact downloaded revision. It is compressed and
	// split across bounded log lines so the reconciler can persist it immediately
	// without relying on the periodic source metadata refresh job.
	descScript := fmt.Sprintf(`
echo "Extracting summary from README..."
python - << 'PY' || true
import base64
import os
import zlib

max_readme_bytes = %d
chunk_characters = %d
raw_readme = b""
for name in ("README.md", "readme.md", "README.MD", "README"):
    p = os.path.join(os.environ["OUT_DIR"], name)
    if os.path.isfile(p):
        try:
            with open(p, "rb") as f:
                raw_readme = f.read(max_readme_bytes)
        except Exception:
            pass
        break
text = raw_readme.decode("utf-8", errors="ignore")
if text:
    payload = base64.b64encode(zlib.compress(text.encode("utf-8"), level=9)).decode("ascii")
    print("[README] begin zlib+base64", flush=True)
    for offset in range(0, len(payload), chunk_characters):
        print("[README] chunk " + payload[offset:offset + chunk_characters], flush=True)
    print("[README] end", flush=True)
if text.startswith("---"):
    end = text.find("\n---", 3)
    if end != -1:
        text = text[end + 4:]
para = ""
code_fence = chr(96) * 3
for block in text.split("\n\n"):
    line = " ".join(block.split())
    if not line or line.startswith(("#", "<", "|", "[!", "![", "---", code_fence)):
        continue
    if len(line) < 20:
        continue
    para = line
    break
if para:
    print("[DESC] " + para[:300], flush=True)
PY
`, maxCapturedReadmeBytes, readmeLogChunkCharacters)

	// 进度监控脚本（保持你原来的逻辑）
	progressScript := `
monitor_progress() {
    while true; do
        if [ -d "$OUT_DIR" ]; then
            CURRENT_SIZE=$(du -sb "$OUT_DIR" 2>/dev/null | cut -f1 || echo 0)
            echo "[PROGRESS] downloaded_bytes=$CURRENT_SIZE"
        fi
        sleep 5
    done
}
monitor_progress &
MONITOR_PID=$!
trap "kill $MONITOR_PID 2>/dev/null || true" EXIT
`

	return fmt.Sprintf(`
set -euo pipefail
export HF_ENDPOINT=%q
%s
export MODELSCOPE_ENDPOINT=%q
OUT_DIR="/data/%s"
export OUT_DIR
mkdir -p "$OUT_DIR"
echo "Downloading model: %s from %s to $OUT_DIR"

# Surface the failing step clearly so the controller can classify the reason.
on_error() {
    code=$?
    echo "[ERROR] download failed at line $1 (exit code $code)" >&2
    exit $code
}
trap 'on_error $LINENO' ERR

# Verify preinstalled tools, with a pinned fallback for custom legacy images.
%s

# 确保 Python 包路径可用
export PYTHONPATH="${PYTHONPATH:-}:/usr/local/lib/python3.11/site-packages"
# 确保 PATH 包含 pip 安装的二进制目录（尽管我们现在不用 CLI 了，保留无妨）
export PATH="/usr/local/bin:/root/.local/bin:$PATH"

%s

%s

%s

# 执行下载
START_TIME=$(date +%%s)
%s
END_TIME=$(date +%%s)

kill $MONITOR_PID 2>/dev/null || true

echo "Download completed successfully"

%s

%s

# 修改权限，使得所有人都可以读取 (目录755, 文件644)
echo "Changing permissions for $OUT_DIR..."
chmod -R 755 "$OUT_DIR" || echo "Warning: Failed to change permissions"

SIZE=$(du -sb "$OUT_DIR" | cut -f1)
DURATION=$((END_TIME - START_TIME))
if [ $DURATION -gt 0 ]; then
    SPEED=$(( SIZE / DURATION ))
    echo "[RESULT] size_bytes=$SIZE duration_seconds=$DURATION speed_bytes_per_sec=$SPEED"
else
    echo "[RESULT] size_bytes=$SIZE"
fi
`,
		huggingFaceEndpoint,
		disableXetCmd,
		config.GetConfig().ModelScopeDownloadEndpoint(),
		modelDirName,
		download.Name,
		download.Source,
		installCmd,
		totalProbeCmd,
		preflightCmd,
		progressScript,
		downloadCommand,
		metadataCmd,
		descScript,
	)
}

func shouldDisableHuggingFaceXet(source model.ModelSource, endpoint string) bool {
	return source == model.ModelSourceHuggingFace &&
		strings.TrimRight(strings.TrimSpace(endpoint), "/") != "https://huggingface.co"
}

// convertToPhysicalPath 将前端路径转换为物理存储路径
func (mgr *ModelDownloadMgr) convertToPhysicalPath(frontendPath string) string {
	// public -> sugon-gpu-incoming
	if strings.HasPrefix(frontendPath, "public/") || frontendPath == "public" {
		return strings.Replace(frontendPath, "public", config.GetConfig().Storage.Prefix.Public, 1)
	}
	// user -> sugon-gpu-home-lab (if needed in future)
	if strings.HasPrefix(frontendPath, "user/") || frontendPath == "user" {
		return strings.Replace(frontendPath, "user", config.GetConfig().Storage.Prefix.User, 1)
	}
	return frontendPath
}

func isValidModelName(name string) bool {
	// 验证格式: owner/model-name
	pattern := `^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`
	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

func sanitizeModelName(name string) string {
	// 保留 / 以支持 owner/model-name 目录结构
	// 只移除其他特殊字符
	pattern := regexp.MustCompile(`[^A-Za-z0-9_./-]`)
	return pattern.ReplaceAllString(name, "")
}

func modelDownloadStoragePath(basePath, safeName string) string {
	return filepath.Join(basePath, safeName)
}

func convertDownloadToResp(d *model.ModelDownload, token util.JWTMessage) ModelDownloadResp {
	username := d.Creator.Name
	nickname := d.Creator.Nickname
	// Creator may not be preloaded on create/mutation paths; the actor is the
	// creator there, so fall back to the token identity.
	if username == "" && d.CreatorID == token.UserID {
		username = token.Username
	}
	if nickname == "" {
		nickname = username
	}

	return ModelDownloadResp{
		ID:              d.ID,
		Name:            d.Name,
		Source:          string(d.Source),
		Category:        string(d.Category),
		Revision:        d.Revision,
		Path:            d.Path,
		SizeBytes:       d.SizeBytes,
		DownloadedBytes: d.DownloadedBytes,
		DownloadSpeed:   d.DownloadSpeed,
		Status:          string(d.Status),
		Message:         d.Message,
		JobName:         d.JobName,
		CreatorID:       d.CreatorID,
		ReferenceCount:  d.ReferenceCount,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		SourceUpdatedAt: d.SourceUpdatedAt,
		UserInfo:        model.UserInfo{Username: username, Nickname: nickname},
		CanManage:       d.CreatorID == token.UserID || token.RolePlatform == model.RoleAdmin,
		CanDelete:       canDeleteDownload(token),
		CanViewLogs:     d.CreatorID == token.UserID || token.RolePlatform == model.RoleAdmin,
		SourceURL:       sourceURLForDownload(d),
		DisplayName:     d.DisplayName,
		License:         d.License,
		Task:            d.Task,
		Library:         d.Library,
		ModelType:       d.ModelType,
		ParameterCount:  d.ParameterCount,
		SourceCreatedAt: d.SourceCreatedAt,
	}
}

func sourceURLForDownload(download *model.ModelDownload) string {
	if download.SourceURL != "" {
		return download.SourceURL
	}
	if download.Source == model.ModelSourceHuggingFace {
		endpoint := config.GetConfig().HuggingFaceDownloadEndpoint()
		if download.Category == model.DownloadCategoryDataset {
			return endpoint + "/datasets/" + download.Name
		}
		return endpoint + "/" + download.Name
	}
	endpoint := config.GetConfig().ModelScopeDownloadEndpoint()
	if download.Category == model.DownloadCategoryDataset {
		return endpoint + "/datasets/" + download.Name
	}
	return endpoint + "/models/" + download.Name
}

func (mgr *ModelDownloadMgr) deleteDownloadJob(c *gin.Context, jobName string) error {
	if jobName == "" {
		return nil
	}
	propagation := metav1.DeletePropagationForeground
	err := mgr.crClient.BatchV1().Jobs(mgr.namespace).Delete(c, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err == nil || k8serrors.IsNotFound(err) {
		return nil
	}
	return bizerr.Internal.K8sServiceError.Wrap(err, "stop download job failed")
}

func (mgr *ModelDownloadMgr) deleteDownloadRecord(c *gin.Context, downloadID uint) error {
	db := query.Use(query.GetDB())
	err := db.Transaction(func(tx *query.Query) error {
		qUserDownload := tx.UserModelDownload
		if _, deleteErr := qUserDownload.WithContext(c).
			Where(qUserDownload.ModelDownloadID.Eq(downloadID)).
			Delete(); deleteErr != nil {
			return deleteErr
		}

		qDownload := tx.ModelDownload
		result, deleteErr := qDownload.WithContext(c).Where(qDownload.ID.Eq(downloadID)).Delete()
		if deleteErr != nil {
			return deleteErr
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "delete download record failed")
	}
	return nil
}

// captureJobLogsToRecord best-effort persists the current pod log tail on the
// download record before its Job is deleted (pause/delete), so the history
// stays inspectable after the pod is gone.
func (mgr *ModelDownloadMgr) captureJobLogsToRecord(c *gin.Context, download *model.ModelDownload) {
	pod := mgr.findLatestDownloadPod(c, download.JobName)
	if pod == nil {
		return
	}

	logOptions := &corev1.PodLogOptions{
		Container: "downloader",
		TailLines: ptr.To(defaultDownloadLogTailLines),
	}
	raw, err := mgr.crClient.CoreV1().Pods(mgr.namespace).GetLogs(pod.Name, logOptions).DoRaw(c)
	if err != nil || len(raw) == 0 {
		return
	}

	now := time.Now()
	q := query.ModelDownload
	if _, err := q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(map[string]any{
		"logs":          truncateDownloadLogTail(string(raw), maxStoredDownloadLogBytes),
		"logs_saved_at": now,
	}); err != nil {
		klog.Warningf("failed to capture logs for download %d: %v", download.ID, err)
	}
}

func truncateDownloadLogTail(logs string, maxBytes int) string {
	if len(logs) <= maxBytes {
		return logs
	}
	tail := logs[len(logs)-maxBytes:]
	if idx := strings.IndexByte(tail, '\n'); idx >= 0 && idx+1 < len(tail) {
		return tail[idx+1:]
	}
	return tail
}

// findLatestDownloadPod returns the newest pod of the download job, or nil.
func (mgr *ModelDownloadMgr) findLatestDownloadPod(c *gin.Context, jobName string) *corev1.Pod {
	if jobName == "" {
		return nil
	}
	pods, err := mgr.crClient.CoreV1().Pods(mgr.namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil || len(pods.Items) == 0 {
		return nil
	}
	latestPod := &pods.Items[0]
	for i := range pods.Items {
		if pods.Items[i].CreationTimestamp.After(latestPod.CreationTimestamp.Time) {
			latestPod = &pods.Items[i]
		}
	}
	return latestPod
}

// GetDownloadLogs godoc
//
//	@Summary		获取模型下载任务日志
//	@Description	返回定时持久化到下载记录中的日志
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Router			/v1/models/downloads/{id}/logs [GET]
func (mgr *ModelDownloadMgr) GetDownloadLogs(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid download id"))
		return
	}

	token := util.GetToken(c)
	canViewLogs, err := mgr.canViewDownloadLogs(c, req.ID, token)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if !canViewLogs {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("you do not have permission to view this download log"))
		return
	}

	q := query.ModelDownload
	download, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "download not found"))
		return
	}

	if download.Logs == "" {
		resputil.Success(c, "Waiting for download logs to be synchronized...")
		return
	}
	resputil.Success(c, download.Logs)
}
