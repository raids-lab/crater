package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
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

func (mgr *ContextMgr) GetQuota(c *gin.Context) {
	token, err := util.GetToken(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get token failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	up := query.UserProject
	var quota payload.Quota
	err = up.WithContext(c).Where(up.ProjectID.Eq(token.ProjectID), up.UserID.Eq(token.UserID)).Select(up.ALL).Scan(&quota)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("find quota failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, quota)

	// 获取当前用户当前项目的Quota
}
