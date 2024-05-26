package handler

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
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
	client.Client
}

func NewContextMgr(cl client.Client) Manager {
	return &ContextMgr{
		Client: cl,
	}
}

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/queue", mgr.GetQueue)
	g.GET("/info", mgr.GetUserInfo)
}

func (mgr *ContextMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type (
	ResourceBase struct {
		Amount int64  `json:"amount"`
		Format string `json:"format"`
	}

	ResourceResp struct {
		Label      string        `json:"label"`
		Allocated  *ResourceBase `json:"allocated"`
		Guarantee  *ResourceBase `json:"guarantee"`
		Deserved   *ResourceBase `json:"deserved"`
		Capability *ResourceBase `json:"capability"`
	}

	QuotaResp struct {
		CPU    ResourceResp   `json:"cpu"`
		Memory ResourceResp   `json:"memory"`
		GPUs   []ResourceResp `json:"gpus"`
	}
)

// GetQueue godoc
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
func (mgr *ContextMgr) GetQueue(c *gin.Context) {
	token := util.GetToken(c)
	if token.QueueName == util.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.QueueNotFound)
		return
	}

	queue := scheduling.Queue{}
	err := mgr.Get(c, types.NamespacedName{Name: token.QueueName, Namespace: config.GetConfig().Workspace.Namespace}, &queue)
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.QueueNotFound)
		return
	}

	allocated := queue.Status.Allocated
	guarantee := queue.Spec.Guarantee.Resource
	deserved := queue.Spec.Deserved
	capability := queue.Spec.Capability

	// resources is a map, key is the resource name, value is the resource amount
	resources := make(map[v1.ResourceName]ResourceResp)

	for name, quantity := range allocated {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.HasPrefix(string(name), "nvidia.com/") {
			resources[name] = ResourceResp{
				Label: string(name),
				Allocated: lo.ToPtr(ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}
	for name, quantity := range guarantee {
		if v, ok := resources[name]; ok {
			v.Guarantee = lo.ToPtr(ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range deserved {
		if v, ok := resources[name]; ok {
			v.Deserved = lo.ToPtr(ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range capability {
		if v, ok := resources[name]; ok {
			v.Capability = lo.ToPtr(ResourceBase{
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
			resource.Capability = &ResourceBase{
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
	var gpus []ResourceResp
	for name, resource := range resources {
		if strings.HasPrefix(string(name), "nvidia.com/") {
			// convert nvidia.com/v100 to v100
			resource.Label = strings.TrimPrefix(string(name), "nvidia.com/")
			gpus = append(gpus, resource)
		}
	}
	sort.Slice(gpus, func(i, j int) bool {
		return gpus[i].Label < gpus[j].Label
	})

	resputil.Success(c, QuotaResp{
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
