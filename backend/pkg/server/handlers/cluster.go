package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/config"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

type ClusterMgr struct {
}

func NewClusterMgr() *ClusterMgr {
	return &ClusterMgr{}
}

func (mgr *ClusterMgr) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/list", mgr.List)
}

func (mgr *ClusterMgr) List(c *gin.Context) {
	dirs := config.GetShareDirNames()
	resputil.Success(c, dirs)
}
