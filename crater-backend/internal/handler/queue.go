package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

type QueueMgr struct {
	client.Client
}

func NewQueueMgr(cl client.Client) Manager {
	return &QueueMgr{
		Client: cl,
	}
}

func (mgr *QueueMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *QueueMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListUserQueue)
}

func (mgr *QueueMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/", mgr.ListQueue)
	g.GET("/:name", mgr.GetQueue)
	g.POST("/", mgr.CreateQueue)
	g.PUT("/:name", mgr.UpdateQueue)
	g.DELETE("/:name", mgr.DeleteQueue)
}

type (
	UserQueueResp struct {
		Name       string           `json:"id"`
		Nickname   string           `json:"name"`
		Role       model.Role       `json:"role"`
		AccessMode model.AccessMode `json:"access"`
	}
)

// ListUserQueue godoc
// @Summary list user queues
// @Description list user queues by user id
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[[]UserQueueResp] "User queue list"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/queues [get]
func (mgr *QueueMgr) ListUserQueue(c *gin.Context) {
	token := util.GetToken(c)

	uq := query.UserQueue
	q := query.Queue

	var userQueues []UserQueueResp
	err := uq.WithContext(c).Where(uq.UserID.Eq(token.UserID)).
		Join(q, uq.QueueID.EqCol(q.ID)).Select(q.Name, q.Nickname, uq.Role, uq.AccessMode).Scan(&userQueues)
	if err != nil {
		resputil.Error(c, "List user queue failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, userQueues)
}

// ListQueue godoc
// @Summary list all queues
// @Description list all queues by client-go
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Volcano queue list"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/queues [get]
func (mgr *QueueMgr) ListQueue(c *gin.Context) {
	var queues scheduling.QueueList
	err := mgr.List(c, &queues)
	if err != nil {
		resputil.Error(c, "List queue failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, queues)
}

func (mgr *QueueMgr) GetQueue(_ *gin.Context) {}

func (mgr *QueueMgr) CreateQueue(_ *gin.Context) {}

func (mgr *QueueMgr) UpdateQueue(_ *gin.Context) {}

func (mgr *QueueMgr) DeleteQueue(_ *gin.Context) {}
