package operations

import "github.com/gin-gonic/gin"

const (
	CRONJOBNAMESPACE  = "crater"
	CRAONJOBLABELKEY  = "crater.raids-lab.io/component"
	CRONJOBLABELVALUE = "cronjob"
)

const (
	CLEAN_LONG_TIME_CRON_JOB_NAME    = "clean-long-time-job"
	CLEAN_LOW_GPU_UTIL_CRON_JOB_NAME = "clean-low-gpu-util-job"
	CLEAN_WAITING_JUPYTER            = "clean-waiting-jupyter"
)

type CronjobHandler interface {
	Execute(c *gin.Context, params map[string]string) (any, error)
}

type LongTimeJobConfig struct {
	BatchDays       int `configmap:"BATCH_DAYS"`
	InteractiveDays int `configmap:"INTERACTIVE_DAYS"`
}

type LowGPUUtilJobConfig struct {
	TimeRange int `configmap:"TIME_RANGE"`
	WaitTime  int `configmap:"WAIT_TIME"`
	Util      int `configmap:"UTIL"`
}

type WaitingJupyterConfig struct {
	WaitMinutes int `configmap:"JUPYTER_WAIT_MINUTES"`
}
