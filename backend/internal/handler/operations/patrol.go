package operations

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
)

// HandleTriggerPatrolJob manually triggers a patrol-type cron job.
func (mgr *OperationsMgr) HandleTriggerPatrolJob(c *gin.Context) {
	jobName := c.Param("jobName")
	if jobName == "" {
		resputil.BadRequestError(c, "missing job name")
		return
	}

	if mgr.cronJobManager == nil {
		resputil.Error(c, "cron job manager not available", resputil.NotSpecified)
		return
	}

	result, err := mgr.cronJobManager.ExecutePatrolJobNow(context.Background(), jobName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, result)
}
