package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const (
	LabelKeyQueueCreatedBy = "crater.raids.io/queue-created-by"
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

type (
	GetQueueReq struct {
		QueueName string `json:"name"`
	}
	GetQueueResp struct {
		QueueName string `json:"name"`
		// division index of overall quota
		Weight string `json:"weight"`
		// Reclaimable indicate whether the queue can be reclaimed by other queue
		Reclaimable *bool `json:"reclaimable"`
		// hard quota
		Capability string `json:"capability"`
		// least quota
		Guarantee string `json:"guarantee"`
		// preempt quota
		Deserved string `json:"deserved"`
		// status
		Type string `json:"type"`

		Affinity []string `json:"affinity"`

		State string `json:"state"`

		// The number of 'Unknown' PodGroup in this queue.
		Unknown int32 `json:"unknown"`
		// The number of 'Pending' PodGroup in this queue.
		Pending int32 `json:"pending"`
		// The number of 'Running' PodGroup in this queue.
		Running int32 `json:"runningy"`
		// The number of `Inqueue` PodGroup in this queue.
		Inqueue int32 `json:"inqueue"`
		// The number of `Completed` PodGroup in this queue.
		Completed int32 `json:"completed"`

		// Reservation is the profile of resource reservation for queue
		Allocated string `json:"allocated"`

		CreatedAt string `json:"created_at"`

		CreatedBy string `json:"created_by"`
	}
)

// / GetQueue godoc
// @Summary 获取vc的queue
// @Description crd client
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Param GetQueueReq query string true "queue名称"
// @Success 200 {object} resputil.Response[GetQueueResp] "整理的queue字段"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/queues/{name} [get]
func (mgr *QueueMgr) GetQueue(c *gin.Context) {
	var req GetQueueReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Get(c, client.ObjectKey{Name: req.QueueName, Namespace: namespace}, queue)
	if err != nil {
		resputil.Error(c, "List queue failed", resputil.NotSpecified)
		return
	}
	resp := GetQueueResp{
		QueueName:   queue.Name,
		Weight:      fmt.Sprintf("%d", queue.Spec.Weight),
		Reclaimable: queue.Spec.Reclaimable,
		Capability:  model.ResourceListToJSON(queue.Spec.Capability),
		Guarantee:   model.ResourceListToJSON(queue.Spec.Guarantee.Resource),
		Affinity:    queue.Spec.Affinity.NodeGroupAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		Type:        queue.Spec.Type,
		State:       string(queue.Status.State),
		Unknown:     queue.Status.Unknown,
		Pending:     queue.Status.Pending,
		Running:     queue.Status.Running,
		Inqueue:     queue.Status.Inqueue,
		Completed:   queue.Status.Completed,
		Allocated:   model.ResourceListToJSON(queue.Status.Allocated),
		CreatedAt:   queue.ObjectMeta.CreationTimestamp.String(),
		CreatedBy:   queue.ObjectMeta.Labels[LabelKeyQueueCreatedBy],
	}
	resputil.Success(c, resp)
}

type CreateQueueReq struct {
	QueueName string `json:"name"`
	// division index of overall quota
	Weight string `json:"weight"`
	// Reclaimable indicate whether the queue can be reclaimed by other queue
	Reclaimable *bool `json:"reclaimable"`
	// hard quota
	Capability string `json:"capability"`
	// least quota
	Guarantee string `json:"guarantee"`
	// preempt quota
	Deserved string `json:"deserved"`
	// If specified, the pod owned by the queue will be scheduled with constraint
	Affinity []string `json:"affinity"`
	// Type define the type of queue
	Type string `json:"type"`
}

// / CreateQueue godoc
// @Summary 创建vc队列
// @Description crd cilent
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Param CreateQueueReq body CreateQueueReq true "创建queue结构体"
// @Success 200 {object} resputil.Response[any] "返回创建的queue"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/queues [post]
func (mgr *QueueMgr) CreateQueue(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateQueueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	namespace := config.GetConfig().Workspace.Namespace

	weight, err := strconv.ParseInt(req.Weight, 10, 32)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	capability, err := model.JSONToResourceList(req.Capability)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	guarantee, err := model.JSONToResourceList(req.Guarantee)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var affinity *scheduling.Affinity
	if len(req.Affinity) > 0 {
		affinity = lo.ToPtr(scheduling.Affinity{
			NodeGroupAffinity: lo.ToPtr(scheduling.NodeGroupAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: req.Affinity,
			}),
		})
	}
	labels := map[string]string{
		LabelKeyQueueCreatedBy: token.Username,
	}
	queue := &scheduling.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.QueueName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: scheduling.QueueSpec{
			Weight:      int32(weight),
			Reclaimable: req.Reclaimable,
			Capability:  capability,
			Guarantee:   scheduling.Guarantee{Resource: guarantee},
			Affinity:    affinity,
			Type:        req.Type,
		},
	}

	if err := mgr.Create(c, queue); err != nil {
		resputil.Error(c, "Created queue failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, queue)
}

type UpdateQueueReq struct {
	// division index of overall quota
	Weight *string `json:"weight,omitempty"`
	// Reclaimable indicate whether the queue can be reclaimed by other queue
	Reclaimable *bool `json:"reclaimable,omitempty"`
	// hard quota
	Capability *string `json:"capability,omitempty"`
	// least quota
	Guarantee *string `json:"guarantee,omitempty"`
	// preempt quota
	Deserved string `json:"deserved,omitempty"`
	// If specified, the pod owned by the queue will be scheduled with constraint
	Affinity *[]string `json:"affinity,omitempty"`
	// Type define the type of queue
	Type *string `json:"type,omitempty"`
}

// / UpdateQueue godoc
// @Summary 更新queue字段
// @Description crd client
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Param GetQueueReq query string true "队列名字"
// @Param UpdateQueueReq body UpdateQueueReq false "可选更新内容"
// @Success 200 {object} resputil.Response[any] "返回修改的队列"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/queues/{name} [post]
func (mgr *QueueMgr) UpdateQueue(c *gin.Context) {
	var getReq GetQueueReq
	if err := c.ShouldBindUri(&getReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var req UpdateQueueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Get(c, client.ObjectKey{Name: getReq.QueueName, Namespace: namespace}, queue)
	if err != nil {
		resputil.Error(c, "Update queue failed", resputil.NotSpecified)
		return
	}

	if req.Weight != nil {
		weight, err := strconv.ParseInt(*req.Weight, 10, 32)
		if err != nil {
			resputil.Error(c, "Update queue failed", resputil.NotSpecified)
			return
		}
		queue.Spec.Weight = int32(weight)
	}

	if req.Reclaimable != nil {
		queue.Spec.Reclaimable = req.Reclaimable
	}

	if req.Capability != nil {
		capability, err := model.JSONToResourceList(*req.Capability)
		if err != nil {
			resputil.Error(c, "Update queue failed", resputil.NotSpecified)
			return
		}
		queue.Spec.Capability = capability
	}

	if req.Guarantee != nil {
		guarantee, err := model.JSONToResourceList(*req.Guarantee)
		if err != nil {
			resputil.Error(c, "Update queue failed", resputil.NotSpecified)
			return
		}
		queue.Spec.Capability = guarantee
	}

	if req.Affinity != nil {
		queue.Spec.Affinity.NodeGroupAffinity.RequiredDuringSchedulingIgnoredDuringExecution = *req.Affinity
	}

	if req.Type != nil {
		queue.Spec.Type = *req.Type
	}

	if err := mgr.Update(c, queue); err != nil {
		resputil.Error(c, "Update queue failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, queue)
}

// / DeleteQueue godoc
// @Summary 删除queue
// @Description crd client
// @Tags Queue
// @Accept json
// @Produce json
// @Security Bearer
// @Param GetQueueReq query string true "队列名字"
// @Success 200 {object} resputil.Response[any] "返回删除的队列"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/queues/{name} [delete]
func (mgr *QueueMgr) DeleteQueue(c *gin.Context) {
	var getReq GetQueueReq
	if err := c.ShouldBindUri(&getReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Get(c, client.ObjectKey{Name: getReq.QueueName, Namespace: namespace}, queue)
	if err != nil {
		resputil.Error(c, "Delete queue failed", resputil.NotSpecified)
		return
	}

	if queue.Status.Running != 0 {
		resputil.Error(c, "Delete queue failed, Queue has non-empty Running PodGroup", resputil.NotSpecified)
		return
	}

	if err := mgr.Delete(c, queue); err != nil {
		resputil.Error(c, "Delete queue failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, queue)
}
