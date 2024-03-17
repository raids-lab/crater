package handlers

import (
	"github.com/aisystem/ai-protal/pkg/config"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/gin-gonic/gin"
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
