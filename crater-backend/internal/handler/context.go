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
		resputil.Error(c, fmt.Sprintf("find quota of user in project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	q := query.Quota
	var quotaInProject payload.Quota
	err = q.WithContext(c).Where(q.ProjectID.Eq(token.ProjectID)).Select(q.ALL).Scan(&quotaInProject)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("find quota of project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	newQuota := QuotaLimitOrNot(&quota, &quotaInProject)

	resputil.Success(c, *newQuota)

	// 获取当前用户当前项目的Quota
}
func QuotaLimitOrNot(quota, quotaInProject *payload.Quota) *payload.Quota {
	const nolimit = -1
	if quota.CPU == nolimit {
		quota.CPU = quotaInProject.CPU
	}
	if quota.CPUReq == nolimit {
		quota.CPUReq = quotaInProject.CPUReq
	}
	if quota.GPU == nolimit {
		quota.GPU = quotaInProject.GPU
	}
	if quota.GPUReq == nolimit {
		quota.GPUReq = quotaInProject.GPUReq
	}
	if quota.GPUMemReq == nolimit {
		quota.GPUMemReq = quotaInProject.GPUMemReq
	}
	if quota.GPUMem == nolimit {
		quota.GPUMem = quotaInProject.GPUMem
	}
	if quota.Node == nolimit {
		quota.Node = quotaInProject.Node
	}
	if quota.NodeReq == nolimit {
		quota.NodeReq = quotaInProject.NodeReq
	}
	if quota.Job == nolimit {
		quota.Job = quotaInProject.Job
	}
	if quota.JobReq == nolimit {
		quota.JobReq = quotaInProject.JobReq
	}
	if quota.Node == nolimit {
		quota.Node = quotaInProject.Node
	}
	if quota.NodeReq == nolimit {
		quota.NodeReq = quotaInProject.NodeReq
	}
	return quota
}
