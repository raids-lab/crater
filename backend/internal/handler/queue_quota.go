package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/prequeuewatcher"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewQueueQuotaMgr)
}

type QueueQuotaMgr struct {
	name    string
	service *service.PrequeueService
	watcher *prequeuewatcher.PrequeueWatcher
}

func NewQueueQuotaMgr(conf *RegisterConfig) Manager {
	return &QueueQuotaMgr{
		name:    "queue-quotas",
		service: conf.PrequeueService,
		watcher: conf.PrequeueWatcher,
	}
}

func (mgr *QueueQuotaMgr) GetName() string                      { return mgr.name }
func (mgr *QueueQuotaMgr) RegisterPublic(_ *gin.RouterGroup)    {}
func (mgr *QueueQuotaMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *QueueQuotaMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.GetQueueQuotas)
	g.POST("", mgr.CreateQueueQuota)
	g.PUT("/:id", mgr.UpdateQueueQuota)
	g.DELETE("/:id", mgr.DeleteQueueQuota)
}

type QueueQuotaResp struct {
	Quotas []QueueQuotaConfigItemResp `json:"quotas"`
}

type QueueQuotaConfigItemResp struct {
	ID                    uint              `json:"id"`
	Name                  string            `json:"name"`
	Enabled               bool              `json:"enabled"`
	PrequeueCandidateSize int               `json:"prequeueCandidateSize"`
	Quota                 map[string]string `json:"quota"`
}

type QueueQuotaReq struct {
	Name                  string            `json:"name"`
	Enabled               bool              `json:"enabled"`
	PrequeueCandidateSize int               `json:"prequeueCandidateSize"`
	Quota                 map[string]string `json:"quota"`
}

type QueueQuotaIDReq struct {
	ID uint `uri:"id" binding:"required"`
}

// GetQueueQuotas godoc
// @Summary		获取队列内资源限制
// @Description	查询当前系统的各队列资源限制配置
// @Tags			QueueQuota
// @Produce		json
// @Security		Bearer
// @Success		200		{object}	resputil.Response[QueueQuotaResp]	"配置信息"
// @Failure		500		{object}	resputil.Response[any]				"服务器错误"
// @Router			/v1/admin/queue-quotas [get]
func (mgr *QueueQuotaMgr) GetQueueQuotas(c *gin.Context) {
	config, err := mgr.service.GetConfig(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, QueueQuotaResp{
		Quotas: toQueueQuotaConfigItemRespList(config.Quotas),
	})
}

// CreateQueueQuota godoc
// @Summary		创建队列内资源限制
// @Description	创建单个队列的资源限制配置
// @Tags			QueueQuota
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			data	body		QueueQuotaReq							true	"配置信息"
// @Success		200		{object}	resputil.Response[QueueQuotaConfigItemResp]	"创建成功"
// @Failure		400		{object}	resputil.Response[any]		"参数错误"
// @Failure		409		{object}	resputil.Response[any]		"名称冲突"
// @Router			/v1/admin/queue-quotas [post]
func (mgr *QueueQuotaMgr) CreateQueueQuota(c *gin.Context) {
	var req QueueQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	item := &service.QueueQuotaConfigItem{
		Name:                  req.Name,
		Enabled:               req.Enabled,
		PrequeueCandidateSize: req.PrequeueCandidateSize,
		Quota:                 req.Quota,
	}

	quota, err := mgr.service.CreateConfig(c, item)
	if err != nil {
		mgr.writeQueueQuotaError(c, err)
		return
	}

	resputil.Success(c, toQueueQuotaConfigItemResp(quota))
	if mgr.watcher != nil {
		mgr.watcher.RequestFullScan()
	}
}

// UpdateQueueQuota godoc
// @Summary		更新队列内资源限制
// @Description	更新单个队列的资源限制配置
// @Tags			QueueQuota
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		uint									true	"队列配置 ID"
// @Param			data	body		QueueQuotaReq							true	"配置信息"
// @Success		200		{object}	resputil.Response[QueueQuotaConfigItemResp]	"更新成功"
// @Failure		400		{object}	resputil.Response[any]		"参数错误"
// @Failure		404		{object}	resputil.Response[any]		"配置不存在"
// @Failure		409		{object}	resputil.Response[any]		"名称冲突"
// @Router			/v1/admin/queue-quotas/{id} [put]
func (mgr *QueueQuotaMgr) UpdateQueueQuota(c *gin.Context) {
	var uri QueueQuotaIDReq
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req QueueQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	item := &service.QueueQuotaConfigItem{
		ID:                    uri.ID,
		Name:                  req.Name,
		Enabled:               req.Enabled,
		PrequeueCandidateSize: req.PrequeueCandidateSize,
		Quota:                 req.Quota,
	}

	quota, err := mgr.service.UpdateConfig(c, item)
	if err != nil {
		mgr.writeQueueQuotaError(c, err)
		return
	}

	resputil.Success(c, toQueueQuotaConfigItemResp(quota))
	if mgr.watcher != nil {
		mgr.watcher.RequestFullScan()
	}
}

// DeleteQueueQuota godoc
// @Summary		删除队列内资源限制
// @Description	删除单个队列的资源限制配置
// @Tags			QueueQuota
// @Produce		json
// @Security		Bearer
// @Param			id	path		uint						true	"队列配置 ID"
// @Success		200	{object}	resputil.Response[string]	"删除成功"
// @Failure		400	{object}	resputil.Response[any]		"参数错误"
// @Failure		404	{object}	resputil.Response[any]		"配置不存在"
// @Router			/v1/admin/queue-quotas/{id} [delete]
func (mgr *QueueQuotaMgr) DeleteQueueQuota(c *gin.Context) {
	var uri QueueQuotaIDReq
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := mgr.service.DeleteQueueQuota(c, uri.ID); err != nil {
		mgr.writeQueueQuotaError(c, err)
		return
	}

	resputil.Success(c, "Queue quota deleted successfully")
	if mgr.watcher != nil {
		mgr.watcher.RequestFullScan()
	}
}

func (mgr *QueueQuotaMgr) writeQueueQuotaError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrQueueQuotaNotFound) {
		resputil.HTTPError(c, http.StatusNotFound, err.Error(), resputil.InvalidRequest)
		return
	}
	if errors.Is(err, service.ErrQueueQuotaNameConflict) {
		resputil.HTTPError(c, http.StatusConflict, err.Error(), resputil.InvalidRequest)
		return
	}
	if errors.Is(err, service.ErrQueueQuotaNameRequired) {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if errors.Is(err, service.ErrQueueQuotaInvalidQuota) {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
	}
}

func toQueueQuotaConfigItemResp(item *service.QueueQuotaConfigItem) QueueQuotaConfigItemResp {
	if item == nil {
		return QueueQuotaConfigItemResp{}
	}

	return QueueQuotaConfigItemResp{
		ID:                    item.ID,
		Name:                  item.Name,
		Enabled:               item.Enabled,
		PrequeueCandidateSize: item.PrequeueCandidateSize,
		Quota:                 item.Quota,
	}
}

func toQueueQuotaConfigItemRespList(items []service.QueueQuotaConfigItem) []QueueQuotaConfigItemResp {
	resp := make([]QueueQuotaConfigItemResp, 0, len(items))
	for i := range items {
		resp = append(resp, toQueueQuotaConfigItemResp(&items[i]))
	}
	return resp
}
