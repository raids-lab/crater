package handler

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

// 管理当前的上下文（用户+项目）
type ContextMgr struct {
}

func NewContextMgr() Manager {
	return &ContextMgr{}
}

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/quota", mgr.GetQuota)
}

func (mgr *ContextMgr) RegisterAdmin(_ *gin.RouterGroup) {}

// GetQuota godoc
// @Summary 获取当前用户当前项目的Quota
// @Description 如果 UserProject 表没有对 Quota 进行限制，则返回项目的 Quota
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[model.EmbeddedQuota] "配额信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/context/quota [get]
func (mgr *ContextMgr) GetQuota(c *gin.Context) {
	token, err := util.GetToken(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get token failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	up := query.UserProject
	var quota model.EmbeddedQuota
	err = up.WithContext(c).Where(up.ProjectID.Eq(token.ProjectID), up.UserID.Eq(token.UserID)).Select(up.ALL).Scan(&quota)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("find quota of user in project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	q := query.Quota
	var quotaInProject model.EmbeddedQuota
	err = q.WithContext(c).Where(q.ProjectID.Eq(token.ProjectID)).Select(q.ALL).Scan(&quotaInProject)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("find quota of project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	newQuota := getUserQuota(&quota, &quotaInProject)

	resputil.Success(c, *newQuota)
}

// 获取用户的配额
// 如果用户在项目中有配额限制，则返回用户在项目中的配额
// 否则返回项目的配额
func getUserQuota(quota, quotaInProject *model.EmbeddedQuota) *model.EmbeddedQuota {
	quotaValue := reflect.ValueOf(quota).Elem()
	projectValue := reflect.ValueOf(quotaInProject).Elem()

	for i := 0; i < quotaValue.NumField(); i++ {
		field := quotaValue.Field(i)
		if field.Interface() == model.Unlimited {
			// 对于 Unlimited 的字段，使用项目的对应字段
			field.Set(projectValue.Field(i))
		} else if field.Type() == reflect.TypeOf((*string)(nil)) {
			// 对于指针类型的字段，如果为 nil，使用项目的对应字段
			field.Set(projectValue.Field(i))
		}
	}

	return quota
}
