package operations

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
)

// HandleTriggerPatrolJob manually triggers a patrol-type cron job.
func (mgr *OperationsMgr) HandleTriggerPatrolJob(c *gin.Context) {
	jobName := c.Param("jobName")
	if jobName == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("missing job name"))
		return
	}

	if mgr.cronJobManager == nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.New("cron job manager not available"))
		return
	}

	result, err := mgr.cronJobManager.ExecutePatrolJobNow(context.Background(), jobName)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "failed to execute patrol job"))
		return
	}

	resputil.Success(c, result)
}
