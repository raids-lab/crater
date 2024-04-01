package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/config"
	gpusvc "github.com/raids-lab/crater/pkg/db/gpu"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

type ClusterMgr struct {
	gpuClient gpusvc.DBService
}

func NewClusterMgr() *ClusterMgr {
	return &ClusterMgr{
		gpuClient: gpusvc.NewDBService(),
	}
}

func (mgr *ClusterMgr) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/list", mgr.List)
	nodes := g.Group("/nodes")
	nodes.GET("/list", mgr.ListNodes)
}

func (mgr *ClusterMgr) List(c *gin.Context) {
	dirs := config.GetShareDirNames()
	resputil.Success(c, dirs)
}

func (mgr *ClusterMgr) ListNodes(c *gin.Context) {
	nodes, err := mgr.gpuClient.List()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list GPU: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nodes)
}
