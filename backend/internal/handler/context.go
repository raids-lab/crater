package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
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
	QuotaResp struct {
		Capability v1.ResourceList `json:"capability"`
		Allocated  v1.ResourceList `json:"allocated"`
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

	resputil.Success(c, QuotaResp{Capability: queue.Spec.Capability, Allocated: queue.Status.Allocated})
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
