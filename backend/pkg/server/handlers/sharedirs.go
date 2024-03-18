package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/config"
	resputil "github.com/raids-lab/crater/pkg/server/response"
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
