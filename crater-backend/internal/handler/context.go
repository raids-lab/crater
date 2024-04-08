package handler

import "github.com/gin-gonic/gin"

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

func (mgr *ContextMgr) GetQuota(_ *gin.Context) {
	// 获取当前用户当前项目的Quota
}
