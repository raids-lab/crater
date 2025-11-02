package operations

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/monitor"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewOperationsMgr)
}

type OperationsMgr struct {
	name           string
	client         client.Client
	kubeClient     kubernetes.Interface
	promClient     monitor.PrometheusInterface
	taskService    aitaskctl.DBService
	taskController aitaskctl.TaskControllerInterface

	// cron
	cronJobManager *cronjob.CronJobManager
}

func NewOperationsMgr(conf *handler.RegisterConfig) handler.Manager {
	instance := &OperationsMgr{
		name:           "operations",
		client:         conf.Client,
		kubeClient:     conf.KubeClient,
		promClient:     conf.PrometheusClient,
		taskService:    aitaskctl.NewDBService(),
		taskController: conf.AITaskCtrl,
		cronJobManager: conf.CronJobManager,
	}
	return instance
}

func (mgr *OperationsMgr) GetName() string { return mgr.name }

func (mgr *OperationsMgr) RegisterPublic(_ *gin.RouterGroup) {
}

func (mgr *OperationsMgr) RegisterProtected(_ *gin.RouterGroup) {
}

func (mgr *OperationsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/whitelist", mgr.GetWhiteList)
	g.PUT("/keep/:name", mgr.SetKeepWhenLowResourceUsage)
	g.GET("/cronjob", mgr.GetCronjobConfigs)
	g.PUT("/cronjob", mgr.UpdateCronjobConfig)
	g.PUT("/add/locktime", mgr.AddLockTime)
	g.PUT("/clear/locktime", mgr.ClearLockTime)

	g.POST("/cronjob/config/name", mgr.GetCronjobNames)
	g.POST("/cronjob/record/time", mgr.GetCronjobRecordTimeRange)
	g.POST("/cronjob/record/list", mgr.GetCronjobRecords)
	g.POST("/cronjob/record/delete", mgr.DeleteCronjobRecords)
}

func (cm *OperationsMgr) StopCron() {
	cm.cronJobManager.StopCron()
}
