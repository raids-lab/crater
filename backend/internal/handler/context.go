package handler

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

type ContextMgr struct {
	name string
	client.Client
}

func NewContextMgr(cl client.Client) Manager {
	return &ContextMgr{
		name:   "context",
		Client: cl,
	}
}

func (mgr *ContextMgr) GetName() string {
	return mgr.name
}

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("queues", mgr.ListUserQueue)
	g.GET("quota", mgr.GetQuota)
	g.GET("info", mgr.GetUserInfo)
}

func (mgr *ContextMgr) RegisterAdmin(_ *gin.RouterGroup) {}

// GetQuota godoc
// @Summary Get the queue information
// @Description query the queue information by client-go
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Volcano Queue Quota"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "other errors"
// @Router /v1/context/queue [get]
//
//nolint:gocyclo // TODO: refactor
func (mgr *ContextMgr) GetQuota(c *gin.Context) {
	token := util.GetToken(c)
	if token.QueueName == util.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.TokenExpired)
		return
	}

	queue := scheduling.Queue{}
	err := mgr.Get(c, types.NamespacedName{Name: token.QueueName, Namespace: config.GetConfig().Workspace.Namespace}, &queue)
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.TokenExpired)
		return
	}

	allocated := queue.Status.Allocated
	guarantee := queue.Spec.Guarantee.Resource
	deserved := queue.Spec.Deserved
	capability := queue.Spec.Capability

	// resources is a map, key is the resource name, value is the resource amount
	resources := make(map[v1.ResourceName]payload.ResourceResp)

	for name, quantity := range allocated {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/") {
			resources[name] = payload.ResourceResp{
				Label: string(name),
				Allocated: lo.ToPtr(payload.ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}
	for name, quantity := range guarantee {
		if v, ok := resources[name]; ok {
			v.Guarantee = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range deserved {
		if v, ok := resources[name]; ok {
			v.Deserved = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range capability {
		if v, ok := resources[name]; ok {
			v.Capability = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}

	// if capability is not set, read max from db
	r := query.Resource
	for name, resource := range resources {
		if resource.Capability == nil {
			resouece, err := r.WithContext(c).Where(r.ResourceName.Eq(string(name))).First()
			if err != nil {
				continue
			}
			resource.Capability = &payload.ResourceBase{
				Amount: resouece.Amount,
				Format: resouece.Format,
			}
			resources[name] = resource
		}
	}

	// map contains cpu, memory, gpus, get them from the map
	cpu := resources[v1.ResourceCPU]
	cpu.Label = "cpu"
	memory := resources[v1.ResourceMemory]
	memory.Label = "mem"
	var gpus []payload.ResourceResp
	for name, resource := range resources {
		if strings.Contains(string(name), "/") {
			// convert nvidia.com/v100 to v100
			split := strings.Split(string(name), "/")
			if len(split) == 2 {
				resourceType := split[1]
				label := resourceType
				resource.Label = label
			}
			gpus = append(gpus, resource)
		}
	}
	sort.Slice(gpus, func(i, j int) bool {
		return gpus[i].Label < gpus[j].Label
	})

	resputil.Success(c, payload.QuotaResp{
		CPU:    cpu,
		Memory: memory,
		GPUs:   gpus,
	})
}

type (
	UserInfoResp struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
		Phone    string `json:"phone"`
	}
)

// GetUserInfo godoc
// @Summary Get user information
// @Description Get user information from the database
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[UserInfoResp] "user information"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/context/info [get]
func (mgr *ContextMgr) GetUserInfo(c *gin.Context) {
	token := util.GetToken(c)
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.UserNotFound)
		return
	}

	userAttr := user.Attributes.Data()
	info := UserInfoResp{
		Nickname: user.Nickname,
		Email:    userAttr.Email,
		Avatar:   userAttr.Avatar,
		Phone:    userAttr.Phone,
	}

	resputil.Success(c, info)
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
func (mgr *ContextMgr) ListUserQueue(c *gin.Context) {
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
