package handlers

import (
	"github.com/aisystem/ai-protal/pkg/config"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/gin-gonic/gin"
)

type ShareDirMgr struct {
}

func NewShareDirMgr() *ShareDirMgr {
	return &ShareDirMgr{}
}

func (mgr *ShareDirMgr) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/list", mgr.List)
}

func (mgr *ShareDirMgr) List(c *gin.Context) {
	dirs := config.GetShareDirNames()
	resputil.Success(c, dirs)
}
