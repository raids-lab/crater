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
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewContextMgr)
}

type ContextMgr struct {
	name   string
	client client.Client
}

func NewContextMgr(conf RegisterConfig) Manager {
	return &ContextMgr{
		name:   "context",
		client: conf.Client,
	}
}

func (mgr *ContextMgr) GetName() string { return mgr.name }

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("quota", mgr.GetQuota)
	g.PUT("attributes", mgr.UpdateUserAttributes)
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

	queue := scheduling.Queue{}
	err := mgr.client.Get(c, types.NamespacedName{Name: token.QueueName, Namespace: config.GetConfig().Workspace.Namespace}, &queue)
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.TokenInvalid)
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
		ID        uint                                    `json:"id"`
		Name      string                                  `json:"name"`
		Attribute datatypes.JSONType[model.UserAttribute] `json:"attributes"`
	}
)

// UpdateUserAttributes godoc
// @Summary Update user attributes
// @Description Update the attributes of the current user
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Param attributes body model.UserAttribute true "User attributes"
// @Success 200 {object} resputil.Response[any] "User attributes updated"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/context/attributes [put]
func (mgr *ContextMgr) UpdateUserAttributes(c *gin.Context) {
	token := util.GetToken(c)
	u := query.User

	var attributes model.UserAttribute
	if err := c.ShouldBindJSON(&attributes); err != nil {
		resputil.BadRequestError(c, "Invalid request body")
		return
	}

	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}

	user.Attributes = datatypes.NewJSONType(attributes)
	if err := u.WithContext(c).Save(user); err != nil {
		resputil.Error(c, "Failed to update user attributes", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "User attributes updated successfully")
}
