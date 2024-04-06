package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	utils "github.com/raids-lab/crater/pkg/util"
)

type AIJobMgr struct {
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
}

func NewAIJobMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) Manager {
	return &AIJobMgr{
		pvcClient:      pvcClient,
		logClient:      logClient,
		taskController: taskController,
	}
}

func (mgr *AIJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.DELETE("/:id", mgr.Delete)
}

func (mgr *AIJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.DELETE("/:id", mgr.Delete)
}

// Delete godoc
// @Summary Delete an AIJob by ID
// @Description Delete an AI job by its unique identifier.
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "AI job ID"
// @Success 200 {object} resputil.Response[any]
// @Router /v1/aijobs/{id} [delete]
func (mgr *AIJobMgr) Delete(c *gin.Context) {
	id, err := util.GetParamID(c, "id")
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}
	token, _ := util.GetToken(c) // 基本不会出错

	a := query.AIJob
	job, err := a.WithContext(c).Where(a.ID.Eq(id)).Take()
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, err.Error(), resputil.NotSpecified)
		return
	}

	// 检查该请求是否有权限删除任务
	ok := false
	if token.PlatformRole == model.RoleAdmin {
		ok = true
	} else if job.ProjectID == token.ProjectID {
		if token.ProjectRole == model.RoleAdmin {
			ok = true
		} else if token.UserID == job.UserID {
			ok = true
		}
	}
	if !ok {
		resputil.HTTPError(c, http.StatusForbidden, "forbidden", resputil.NotSpecified)
		return
	}

	// 通知任务控制器，删除任务
	mgr.notifyTaskUpdate(id, job.User.Name, utils.DeleteTask)

	// 从数据库中删除任务
	_, err = a.WithContext(c).Delete(job)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete task failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "")
}

func (mgr *AIJobMgr) notifyTaskUpdate(taskID uint, userName string, op utils.TaskOperation) {
	mgr.taskController.TaskUpdated(utils.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}
