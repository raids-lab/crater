package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"k8s.io/client-go/kubernetes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

const defaultDownloadLogTailLines int64 = 1000

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
}

type ModelDownloadResp struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	Source         string    `json:"source"`
	Category       string    `json:"category"`
	Revision       string    `json:"revision"`
	Path           string    `json:"path"`
	SizeBytes      int64     `json:"sizeBytes"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	JobName        string    `json:"jobName"`
	CreatorID      uint      `json:"creatorId"`
	ReferenceCount int       `json:"referenceCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CreateDownload godoc
//
//	@Summary		创建模型下载任务
//	@Description	创建一个新的模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body		CreateDownloadReq		true	"下载请求"
//	@Success		200		{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/download [POST]
func (mgr *ModelDownloadMgr) CreateDownload(c *gin.Context) {
	var req CreateDownloadReq
	token := util.GetToken(c)

	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if !isValidModelName(req.Name) {
		resputil.BadRequestError(c, "invalid model name format, expected: owner/model-name")
		return
	}

	// 设置默认来源
	source := model.ModelSourceModelScope
	if req.Source != "" {
		source = model.ModelSource(req.Source)
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
	downloadPath := filepath.Join(basePath, safeName)

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	revision := req.Revision
	existingDownload, _ := q.WithContext(c).
		Where(q.Name.Eq(req.Name), q.Source.Eq(string(source)), q.Category.Eq(string(category)), q.Revision.Eq(revision)).
		First()

	if existingDownload != nil {
		// 检查当前用户是否已关联此下载
		userDownload, _ := qUserDownload.WithContext(c).
			Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(existingDownload.ID)).
			First()

		if userDownload != nil {
			// 用户已有此下载,返回成功并提示位置
			resp := convertDownloadToResp(existingDownload)
			// 在响应中添加友好提示（前端可以通过特殊字段识别）
			c.JSON(http.StatusOK, gin.H{
				"code": resputil.OK,
				"data": resp,
				"msg":  fmt.Sprintf("该资源已存在于您的下载列表中，位置: %s", existingDownload.Path),
			})
			return
		}

		// 用户没有此下载,创建关联并增加引用计数
		newUserDownload := &model.UserModelDownload{
			UserID:          token.UserID,
			ModelDownloadID: existingDownload.ID,
		}
		if err := qUserDownload.WithContext(c).Create(newUserDownload); err != nil {
			klog.Errorf("create user-download association failed: %v", err)
			resputil.Error(c, "create download association failed", resputil.NotSpecified)
			return
		}

		// 增加引用计数
		_, _ = q.WithContext(c).Where(q.ID.Eq(existingDownload.ID)).UpdateSimple(q.ReferenceCount.Add(1))

		resputil.Success(c, convertDownloadToResp(existingDownload))
		return
	}

	// 不存在,创建新的下载任务
	jobName := fmt.Sprintf("model-dl-%s-%s", token.Username, uuid.New().String()[:8])

	download := &model.ModelDownload{
		Name:           req.Name,
		Source:         source,
		Category:       category,
		Revision:       revision,
		Path:           downloadPath,
		Status:         model.ModelDownloadStatusPending,
		JobName:        jobName,
		CreatorID:      token.UserID,
		ReferenceCount: 1,
	}

	if err := q.WithContext(c).Create(download); err != nil {
		klog.Errorf("create model download record failed: %v", err)
		resputil.Error(c, "create download record failed", resputil.NotSpecified)
		return
	}

	// 创建用户关联
	userDownload := &model.UserModelDownload{
		UserID:          token.UserID,
		ModelDownloadID: download.ID,
	}
	if err := qUserDownload.WithContext(c).Create(userDownload); err != nil {
		klog.Errorf("create user-download association failed: %v", err)
		// 删除刚创建的download记录
		_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Delete()
		resputil.Error(c, "create download association failed", resputil.NotSpecified)
		return
	}

	// 提交 K8s Job
	if err := mgr.submitDownloadJob(c, download, token.Username); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		//nolint:gofmt // gofmt incorrectly formats this map literal
		updates := map[string]interface{}{
			"status":  model.ModelDownloadStatusFailed,
			"message": fmt.Sprintf("submit job failed: %v", err),
		}
		_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(updates)

		resputil.Error(c, "submit download job failed", resputil.NotSpecified)
		return
	}

	// 更新状态为 Downloading
	_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Update(q.Status, model.ModelDownloadStatusDownloading)

	resputil.Success(c, convertDownloadToResp(download))
}

// ListDownloads godoc
//
//	@Summary		获取用户的模型下载任务列表
//	@Description	获取当前用户的所有模型下载任务
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]ModelDownloadResp]
//	@Router			/v1/models/downloads [GET]
func (mgr *ModelDownloadMgr) ListDownloads(c *gin.Context) {
	token := util.GetToken(c)
	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 通过关联表查询用户的下载
	userDownloads, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID)).
		Find()
	if err != nil {
		klog.Errorf("list user downloads failed: %v", err)
		resputil.Error(c, "list downloads failed", resputil.NotSpecified)
		return
	}

	// 获取所有下载的ID
	downloadIDs := make([]uint, len(userDownloads))
	for i, ud := range userDownloads {
		downloadIDs[i] = ud.ModelDownloadID
	}

	if len(downloadIDs) == 0 {
		resputil.Success(c, []ModelDownloadResp{})
		return
	}

	// 查询所有下载详情
	downloads, err := q.WithContext(c).
		Where(q.ID.In(downloadIDs...)).
		Order(q.CreatedAt.Desc()).
		Find()
	if err != nil {
		klog.Errorf("list downloads failed: %v", err)
		resputil.Error(c, "list downloads failed", resputil.NotSpecified)
		return
	}

	resp := make([]ModelDownloadResp, len(downloads))
	for i, d := range downloads {
		resp[i] = convertDownloadToResp(d)
	}

	resputil.Success(c, resp)
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
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 检查用户是否有此下载的权限
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(req.ID)).
		First()
	if err != nil || userDownload == nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 获取下载详情
	download, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	resputil.Success(c, convertDownloadToResp(download))
}

// checkUserDownloadPermission 检查用户是否有权限操作指定的下载任务
func (mgr *ModelDownloadMgr) checkUserDownloadPermission(c *gin.Context, downloadID, userID uint) (*model.ModelDownload, error) {
	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 检查用户是否有此下载的权限
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(userID), qUserDownload.ModelDownloadID.Eq(downloadID)).
		First()
	if err != nil || userDownload == nil {
		return nil, fmt.Errorf("download not found or permission denied")
	}

	// 获取下载详情
	download, err := q.WithContext(c).Where(q.ID.Eq(downloadID)).First()
	if err != nil {
		return nil, fmt.Errorf("download not found")
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
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id}/retry [POST]
func (mgr *ModelDownloadMgr) RetryDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 检查用户是否有此下载的权限
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(req.ID)).
		First()
	if err != nil || userDownload == nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 获取下载详情
	download, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	if download.Status != model.ModelDownloadStatusFailed {
		resputil.BadRequestError(c, "only failed downloads can be retried")
		return
	}

	// 生成新的 Job 名称
	newJobName := fmt.Sprintf("model-dl-%s-%s", token.Username, uuid.New().String()[:8])
	download.JobName = newJobName

	// 提交新 Job
	if err := mgr.submitDownloadJob(c, download, token.Username); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		resputil.Error(c, "submit download job failed", resputil.NotSpecified)
		return
	}

	// 更新状态
	//nolint:gofmt // gofmt incorrectly formats this map literal
	_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(map[string]interface{}{
		"status":   model.ModelDownloadStatusDownloading,
		"message":  "",
		"job_name": newJobName,
	})

	download.Status = model.ModelDownloadStatusDownloading
	download.Message = ""

	resputil.Success(c, convertDownloadToResp(download))
}

// DeleteDownload godoc
//
//	@Summary		删除模型下载任务
//	@Description	删除指定的模型下载任务记录
//	@Tags			ModelDownload
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"下载任务ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Router			/v1/models/downloads/{id} [DELETE]
func (mgr *ModelDownloadMgr) DeleteDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 查找用户和下载的关联
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(req.ID)).
		First()
	if err != nil || userDownload == nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 删除用户关联
	result, err := qUserDownload.WithContext(c).
		Where(qUserDownload.ID.Eq(userDownload.ID)).
		Delete()
	if err != nil || result.RowsAffected == 0 {
		resputil.Error(c, "delete association failed", resputil.NotSpecified)
		return
	}

	remainingCount, _ := qUserDownload.WithContext(c).
		Where(qUserDownload.ModelDownloadID.Eq(req.ID)).
		Count()

	// 更新引用计数
	download, _ := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if download != nil {
		_, _ = q.WithContext(c).Where(q.ID.Eq(req.ID)).UpdateSimple(q.ReferenceCount.Value(int(remainingCount)))

		// 如果没有任何用户关联了,软删除下载记录(保留文件)
		if remainingCount == 0 {
			// 停止Job(如果还在运行)
			if download.JobName != "" && download.Status == model.ModelDownloadStatusDownloading {
				_ = mgr.crClient.BatchV1().Jobs(mgr.namespace).Delete(c, download.JobName, metav1.DeleteOptions{})
			}

			// 软删除下载记录(GORM自动设置DeletedAt，文件保留在存储中)
			_, _ = q.WithContext(c).Where(q.ID.Eq(req.ID)).Delete()

			klog.Infof("Soft deleted download record %d (refCount=0), files preserved at: %s", req.ID, download.Path)
		}
	}

	resputil.Success(c, "deleted successfully")
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
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 检查用户是否有此下载的权限
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(req.ID)).
		First()
	if err != nil || userDownload == nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 获取下载详情
	download, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	if download.Status != model.ModelDownloadStatusDownloading {
		resputil.BadRequestError(c, "only downloading tasks can be paused")
		return
	}

	// 删除 Job 来暂停下载
	err = mgr.crClient.BatchV1().Jobs(mgr.namespace).Delete(c, download.JobName, metav1.DeleteOptions{})
	if err != nil {
		klog.Warningf("delete job for pause failed: %v", err)
	}

	// 更新状态为 Paused
	//nolint:gofmt // gofmt incorrectly formats this map literal
	_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(map[string]interface{}{
		"status":  model.ModelDownloadStatusPaused,
		"message": "Download paused by user",
	})

	download.Status = model.ModelDownloadStatusPaused
	download.Message = "Download paused by user"

	resputil.Success(c, convertDownloadToResp(download))
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
//	@Success		200	{object}	resputil.Response[ModelDownloadResp]
//	@Router			/v1/models/downloads/{id}/resume [POST]
func (mgr *ModelDownloadMgr) ResumeDownload(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, "invalid id")
		return
	}

	download, err := mgr.checkUserDownloadPermission(c, req.ID, token.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	q := query.ModelDownload

	if download.Status != model.ModelDownloadStatusPaused {
		resputil.BadRequestError(c, "only paused tasks can be resumed")
		return
	}

	// 生成新的 Job 名称
	newJobName := fmt.Sprintf("model-dl-%s-%s", token.Username, uuid.New().String()[:8])
	download.JobName = newJobName

	// 提交新 Job (会从已下载的部分继续)
	if err := mgr.submitDownloadJob(c, download, token.Username); err != nil {
		klog.Errorf("submit download job failed: %v", err)
		resputil.Error(c, "submit download job failed", resputil.NotSpecified)
		return
	}

	// 更新状态
	//nolint:gofmt // gofmt incorrectly formats this map literal
	_, _ = q.WithContext(c).Where(q.ID.Eq(download.ID)).Updates(map[string]interface{}{
		"status":   model.ModelDownloadStatusDownloading,
		"message":  "",
		"job_name": newJobName,
	})

	download.Status = model.ModelDownloadStatusDownloading
	download.Message = ""

	resputil.Success(c, convertDownloadToResp(download))
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
	q := query.ModelDownload

	downloads, err := q.WithContext(c).
		Order(q.CreatedAt.Desc()).
		Find()
	if err != nil {
		klog.Errorf("list all downloads failed: %v", err)
		resputil.Error(c, "list downloads failed", resputil.NotSpecified)
		return
	}

	resp := make([]ModelDownloadResp, len(downloads))
	for i, d := range downloads {
		resp[i] = convertDownloadToResp(d)
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
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	result, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		Delete()
	if err != nil || result.RowsAffected == 0 {
		resputil.Error(c, "download not found or delete failed", resputil.InvalidRequest)
		return
	}

	resputil.Success(c, "deleted successfully")
}

func (mgr *ModelDownloadMgr) submitDownloadJob(c *gin.Context, download *model.ModelDownload, username string) error {
	physicalPath := mgr.convertToPhysicalPath(download.Path)
	subPath := filepath.Dir(physicalPath)
	modelDirName := filepath.Base(physicalPath)

	downloadCmd := mgr.buildDownloadCommand(download, modelDirName)

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
			BackoffLimit: func() *int32 { i := int32(2); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":               "model-download",
						"model-download-id": fmt.Sprintf("%d", download.ID),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "downloader",
							Image:   mgr.getDownloadImage(download.Source),
							Command: []string{"/bin/bash", "-c"},
							Args:    []string{downloadCmd},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
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

func (mgr *ModelDownloadMgr) getDownloadImage(_ model.ModelSource) string {
	// 使用官方 Python 镜像,在运行时安装所需的包
	// 这样可以避免架构不匹配的问题
	return "python:3.11-slim"
}

func (mgr *ModelDownloadMgr) buildDownloadCommand(download *model.ModelDownload, modelDirName string) string {
	// 清华源配置
	pypiMirror := "https://pypi.tuna.tsinghua.edu.cn/simple"
	trustedHost := "--trusted-host pypi.tuna.tsinghua.edu.cn"

	var installCmd, downloadCommand string
	if download.Source == model.ModelSourceHuggingFace {
		// 1. 安装 huggingface_hub
		installCmd = fmt.Sprintf(
			"pip install -U 'huggingface_hub>=0.23.0' -i %s %s",
			pypiMirror, trustedHost,
		)

		// 2. 用 Python API snapshot_download，而不是 huggingface-cli
		downloadCommand = fmt.Sprintf(`
python - << 'PY'
import os
from huggingface_hub import snapshot_download

repo_id = %q
revision = %q

kwargs = {
    "repo_id": repo_id,
    "local_dir": os.environ["OUT_DIR"],
    "local_dir_use_symlinks": False,
    "resume_download": True,
}
if revision:
    kwargs["revision"] = revision

snapshot_download(**kwargs)
PY
`, download.Name, download.Revision)
	} else {
		// ModelScope 这块可以沿用原来的 CLI 方式
		installCmd = fmt.Sprintf(
			"pip install -q modelscope -i %s %s",
			pypiMirror, trustedHost,
		)

		modelName := download.Name
		if download.Revision != "" {
			downloadCommand = fmt.Sprintf(
				`modelscope download --model %s --revision %s --local_dir "$OUT_DIR"`,
				modelName, download.Revision,
			)
		} else {
			downloadCommand = fmt.Sprintf(
				`modelscope download --model %s --local_dir "$OUT_DIR"`,
				modelName,
			)
		}
	}

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
export HF_ENDPOINT=https://hf-mirror.com
OUT_DIR="/data/%s"
export OUT_DIR
mkdir -p "$OUT_DIR"
echo "Downloading model: %s from %s to $OUT_DIR"

# 安装依赖
echo "Installing dependencies from Tsinghua mirror..."
%s

# 确保 Python 包路径可用
export PYTHONPATH="${PYTHONPATH:-}:/usr/local/lib/python3.11/site-packages"
# 确保 PATH 包含 pip 安装的二进制目录（尽管我们现在不用 CLI 了，保留无妨）
export PATH="/usr/local/bin:/root/.local/bin:$PATH"

%s

# 执行下载
START_TIME=$(date +%%s)
%s
END_TIME=$(date +%%s)

kill $MONITOR_PID 2>/dev/null || true

echo "Download completed successfully"
SIZE=$(du -sb "$OUT_DIR" | cut -f1)
DURATION=$((END_TIME - START_TIME))
if [ $DURATION -gt 0 ]; then
    SPEED=$(( SIZE / DURATION ))
    echo "[RESULT] size_bytes=$SIZE duration_seconds=$DURATION speed_bytes_per_sec=$SPEED"
else
    echo "[RESULT] size_bytes=$SIZE"
fi
`,
		modelDirName,
		download.Name,
		download.Source,
		installCmd,
		progressScript,
		downloadCommand,
	)
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
	// 将 owner/model-name 转换为 owner-model-name
	safe := strings.ReplaceAll(name, "/", "-")
	// 移除其他特殊字符
	pattern := regexp.MustCompile(`[^A-Za-z0-9_.-]`)
	return pattern.ReplaceAllString(safe, "")
}

func convertDownloadToResp(d *model.ModelDownload) ModelDownloadResp {
	return ModelDownloadResp{
		ID:             d.ID,
		Name:           d.Name,
		Source:         string(d.Source),
		Category:       string(d.Category),
		Revision:       d.Revision,
		Path:           d.Path,
		SizeBytes:      d.SizeBytes,
		Status:         string(d.Status),
		Message:        d.Message,
		JobName:        d.JobName,
		CreatorID:      d.CreatorID,
		ReferenceCount: d.ReferenceCount,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// GetDownloadLogs godoc
//
//	@Summary		获取模型下载任务的 Pod 日志
//	@Description	获取模型下载任务的实时日志
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
	token := util.GetToken(c)

	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, "invalid id")
		return
	}

	q := query.ModelDownload
	qUserDownload := query.UserModelDownload

	// 检查用户是否有此下载的权限
	userDownload, err := qUserDownload.WithContext(c).
		Where(qUserDownload.UserID.Eq(token.UserID), qUserDownload.ModelDownloadID.Eq(req.ID)).
		First()
	if err != nil || userDownload == nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 获取下载详情
	download, err := q.WithContext(c).
		Where(q.ID.Eq(req.ID)).
		First()
	if err != nil {
		resputil.Error(c, "download not found", resputil.InvalidRequest)
		return
	}

	// 获取 Job 对应的 Pod
	pods, err := mgr.crClient.CoreV1().Pods(mgr.namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", download.JobName),
	})
	if err != nil || len(pods.Items) == 0 {
		resputil.Success(c, "等待 Pod 启动...")
		return
	}

	// 获取最新的 Pod
	latestPod := &pods.Items[0]
	for i := range pods.Items {
		if pods.Items[i].CreationTimestamp.After(latestPod.CreationTimestamp.Time) {
			latestPod = &pods.Items[i]
		}
	}

	// 检查 Pod 状态,如果还没有准备好,返回友好提示
	if latestPod.Status.Phase == corev1.PodPending {
		resputil.Success(c, fmt.Sprintf("Pod 正在启动中... (状态: %s)", latestPod.Status.Phase))
		return
	}

	// 获取日志
	logOptions := &corev1.PodLogOptions{
		Container: "downloader",
		TailLines: func() *int64 { i := defaultDownloadLogTailLines; return &i }(),
	}

	req2 := mgr.crClient.CoreV1().Pods(mgr.namespace).GetLogs(latestPod.Name, logOptions)
	logs, err := req2.DoRaw(c)
	if err != nil {
		// 如果是容器还未准备好的错误,返回友好提示
		errMsg := err.Error()
		if latestPod.Status.Phase == corev1.PodPending || latestPod.Status.Phase == corev1.PodUnknown {
			resputil.Success(c, fmt.Sprintf("Pod 正在启动中,请稍后... (状态: %s)", latestPod.Status.Phase))
			return
		}
		klog.Warningf("get pod logs failed (pod=%s, phase=%s): %v", latestPod.Name, latestPod.Status.Phase, err)
		resputil.Success(c, fmt.Sprintf("暂时无法获取日志: %s", errMsg))
		return
	}

	resputil.Success(c, string(logs))
}
