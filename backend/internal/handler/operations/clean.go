package operations

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/cleaner"
)

func (mgr *OperationsMgr) handleCleanerRequest(
	c *gin.Context,
	req any,
	cleanFunc func(*gin.Context, *cleaner.Clients, any) (any, error),
) {
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	cleanerClients := cleaner.NewCleanerClients(mgr.client, mgr.kubeClient, mgr.promClient)
	res, err := cleanFunc(c, cleanerClients, req)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, res)
}

func (mgr *OperationsMgr) HandleLowGPUUsageJobs(c *gin.Context) {
	mgr.handleCleanerRequest(
		c,
		&cleaner.CleanLowGPUUsageRequest{},
		func(ctx *gin.Context, clients *cleaner.Clients, req any) (any, error) {
			return cleaner.CleanLowGPUUsageJobs(ctx, clients, req.(*cleaner.CleanLowGPUUsageRequest))
		},
	)
}

func (mgr *OperationsMgr) HandleLongTimeRunningJobs(c *gin.Context) {
	mgr.handleCleanerRequest(
		c,
		&cleaner.CleanLongTimeRunningJobsRequest{},
		func(ctx *gin.Context, clients *cleaner.Clients, req any) (any, error) {
			return cleaner.CleanLongTimeRunningJobs(ctx, clients, req.(*cleaner.CleanLongTimeRunningJobsRequest))
		},
	)
}

func (mgr *OperationsMgr) HandleWaitingJupyterJobs(c *gin.Context) {
	mgr.handleCleanerRequest(
		c,
		&cleaner.CancelWaitingJobsRequest{},
		func(ctx *gin.Context, clients *cleaner.Clients, req any) (any, error) {
			return cleaner.CleanWaitingJobs(ctx, clients, req.(*cleaner.CancelWaitingJobsRequest))
		},
	)
}

func (mgr *OperationsMgr) HandleWaitingCustomJobs(c *gin.Context) {
	mgr.handleCleanerRequest(
		c,
		&cleaner.CancelWaitingJobsRequest{},
		func(ctx *gin.Context, clients *cleaner.Clients, req any) (any, error) {
			return cleaner.CleanWaitingJobs(ctx, clients, req.(*cleaner.CancelWaitingJobsRequest))
		},
	)
}
