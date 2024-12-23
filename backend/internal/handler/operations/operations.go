package operations

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/alert"
	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/config"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

const (
	VCJOBAPIVERSION = "batch.volcano.sh/v1alpha1"
	VCJOBKIND       = "Job"
	AIJOBAPIVERSION = "aisystem.github.com/v1alpha1"
	AIJOBKIND       = "AIJob"
)

type JobInfo struct {
	jobName       string
	jobType       string
	jobAPIVersion string
}

type SetKeepRequest struct {
	Name string `uri:"name" binding:"required"`
}

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewOperationsMgr)
}

type OperationsMgr struct {
	name           string
	client         client.Client
	kubeClient     kubernetes.Interface
	promClient     monitor.PrometheusInterface
	taskService    tasksvc.DBService
	taskController aitaskctl.TaskControllerInterface
}

func NewOperationsMgr(conf *handler.RegisterConfig) handler.Manager {
	return &OperationsMgr{
		name:           "operations",
		client:         conf.Client,
		kubeClient:     conf.KubeClient,
		promClient:     conf.PrometheusClient,
		taskService:    tasksvc.NewDBService(),
		taskController: conf.AITaskCtrl,
	}
}

func (mgr *OperationsMgr) GetName() string { return mgr.name }

func (mgr *OperationsMgr) RegisterPublic(g *gin.RouterGroup) {
	g.DELETE("/auto", mgr.AutoDelete)
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

func (mgr *OperationsMgr) RegisterProtected(_ *gin.RouterGroup) {
}

func (mgr *OperationsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/whitelist", mgr.GetWhiteList)
	g.DELETE("/auto", mgr.AutoDelete)
	g.PUT("/keep/:name", mgr.SetKeepWhenLowResourceUsage)
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

type FreeJobRequest struct {
	TimeRange int  `form:"timeRange" binding:"required"`
	WaitTime  int  `form:"waitTime"`
	Util      int  `form:"util"`
	Confirm   bool `form:"confirm"`
}

func (mgr *OperationsMgr) getJobWhiteList(c *gin.Context) ([]string, error) {
	var cleanList []string
	jobDB := query.Job
	data, err := jobDB.WithContext(c).Where(jobDB.KeepWhenLowResourceUsage).Find()
	if err != nil {
		return nil, err
	}
	for _, item := range data {
		cleanList = append(cleanList, item.JobName)
	}
	return cleanList, nil
}

func (mgr *OperationsMgr) deleteJobByName(c *gin.Context, jobAPIVersion, jobType, jobName string) error {
	if jobType == VCJOBKIND && jobAPIVersion == VCJOBAPIVERSION {
		return mgr.deleteVCJobByName(c, jobName)
	}
	if jobType == AIJOBKIND && jobAPIVersion == AIJOBAPIVERSION {
		return mgr.deleteAIJobByName(c, jobName)
	}
	return nil
}

func (mgr *OperationsMgr) deleteAIJobByName(c *gin.Context, jobName string) error {
	aijob := &aijobapi.AIJob{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.client.Get(c, client.ObjectKey{Name: jobName, Namespace: namespace}, aijob); err != nil {
		return err
	}

	if err := mgr.client.Delete(c, aijob); err != nil {
		return err
	}
	return nil
}

func (mgr *OperationsMgr) deleteVCJobByName(c *gin.Context, jobName string) error {
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.client.Get(c, client.ObjectKey{Name: jobName, Namespace: namespace}, job); err != nil {
		return err
	}

	if err := mgr.client.Delete(c, job); err != nil {
		return err
	}

	return nil
}

// GetWhiteList godoc
// @Summary Get job white list
// @Description get job white list
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/whitelist [get]
func (mgr *OperationsMgr) GetWhiteList(c *gin.Context) {
	whiteList, err := mgr.getJobWhiteList(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, whiteList)
}

// SetKeepWhenLowResourceUsage godoc
// @Summary set KeepWhenLowResourceUsage of the job to the opposite value
// @Description set KeepWhenLowResourceUsage of the job to the opposite value
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/keep/{name} [put]
func (mgr *OperationsMgr) SetKeepWhenLowResourceUsage(c *gin.Context) {
	var req SetKeepRequest
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobDB := query.Job
	j, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	pre := j.KeepWhenLowResourceUsage
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.KeepWhenLowResourceUsage, !pre); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Set %s keepWhenLowResourceUsage to %t", req.Name, !j.KeepWhenLowResourceUsage)
	resputil.Success(c, message)
}

// AutoDelete godoc
// @Summary Auto delete not using gpu job list
// @Description check job list and delete not using gpu job
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query FreeJobRequest true "timeRange util"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/auto [delete]
func (mgr *OperationsMgr) AutoDelete(c *gin.Context) {
	var req FreeJobRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.TimeRange <= 0 || req.WaitTime <= 0 {
		resputil.BadRequestError(c, "timeRange and waitTime must be greater than 0")
		return
	}

	// delete time: 即当前时间+waitTime
	deleteTime := time.Now().Add(time.Duration(req.WaitTime) * time.Minute)

	// remind
	reminded := mgr.remindLeastUsedGPUJobs(c, req.TimeRange, req.Util, deleteTime)

	// delete
	deleted := mgr.deleteLeastUsedGPUJobs(c, req.TimeRange+req.WaitTime, req.Util, true)

	resputil.Success(c, map[string][]string{
		"reminded": reminded,
		"deleted":  deleted,
	})
}

// DeleteUnUsedJobList godoc
// @Summary Delete not using gpu job list
// @Description check job list and delete not using gpu job
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query FreeJobRequest true "timeRange util"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/job [delete]
func (mgr *OperationsMgr) DeleteUnUsedJobList(c *gin.Context) {
	var req FreeJobRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.TimeRange <= 0 {
		resputil.BadRequestError(c, "timeRange must be greater than 0")
		return
	}

	deleted := mgr.deleteLeastUsedGPUJobs(c, req.TimeRange, req.Util, req.Confirm)

	resputil.Success(c, deleted)
}

func (mgr *OperationsMgr) deleteLeastUsedGPUJobs(c *gin.Context, duration, util int, confirm bool) []string {
	toDeleteJobs := mgr.filterJobsToKeep(c, mgr.getLeastUsedGPUJobs(c, duration, util))
	deletedJobList := []string{}
	for _, job := range toDeleteJobs {
		if confirm {
			if err := mgr.deleteJobByName(c, job.jobAPIVersion, job.jobType, job.jobName); err != nil {
				fmt.Printf("Delete job %s failed\n", job)
			}
			alertMgr := alert.GetAlertMgr()
			if alertMgr != nil {
				if err := alertMgr.DeleteJob(c, job.jobName, nil); err != nil {
					fmt.Println("Send Alarm Email failed:", err)
				}
			}
		}
		deletedJobList = append(deletedJobList, job.jobName)
	}
	return deletedJobList
}

func (mgr *OperationsMgr) remindLeastUsedGPUJobs(c *gin.Context, duration, util int, deleteTime time.Time) []string {
	jobDB := query.Job

	toRemindJobs := mgr.filterJobsToKeep(c, mgr.getLeastUsedGPUJobs(c, duration, util))

	remindedJobList := []string{}

	remindedJobs, _ := jobDB.WithContext(c).Where(jobDB.Reminded).Find()
	remindMap := make(map[string]bool)
	for _, item := range remindedJobs {
		remindMap[item.JobName] = item.Reminded
	}

	for _, job := range toRemindJobs {
		if remind, ok := remindMap[job.jobName]; ok && !remind {
			// if remind is false, then send alarm email, and set remind to true
			alertMgr := alert.GetAlertMgr()
			if alertMgr != nil {
				if err := alertMgr.RemindLowUsageJob(c, job.jobName, deleteTime, nil); err != nil {
					fmt.Println("Send Alarm Email failed:", err)
				}
			}
			// set remind to true
			if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).Update(jobDB.Reminded, true); err != nil {
				fmt.Printf("Update job %s remind failed\n", job)
			}
			remindedJobList = append(remindedJobList, job.jobName)
		}
	}

	// for the jobs remaining in remindMap, set remind to false
	for jobName := range remindMap {
		if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(jobName)).Update(jobDB.Reminded, false); err != nil {
			fmt.Printf("Update job %s remind failed\n", jobName)
		}
	}
	return remindedJobList
}

// 过滤掉在whitelist中的job
func (mgr *OperationsMgr) filterJobsToKeep(c *gin.Context, jobs []JobInfo) []JobInfo {
	whiteList, _ := mgr.getJobWhiteList(c)
	var filteredJobs []JobInfo
	for _, job := range jobs {
		if !lo.Contains(whiteList, job.jobName) {
			filteredJobs = append(filteredJobs, job)
		}
	}
	return filteredJobs
}

func (mgr *OperationsMgr) getLeastUsedGPUJobs(c *gin.Context, duration, util int) []JobInfo {
	var gpuJobPodsList map[string]string
	namespace := config.GetConfig().Workspace.Namespace
	gpuUtilMap := mgr.promClient.QueryNodeGPUUtil()
	jobPodsList := mgr.promClient.GetJobPodsList()
	gpuJobPodsList = make(map[string]string)
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		curPod := gpuUtil.Pod
		for job, pods := range jobPodsList {
			for _, pod := range pods {
				if curPod == pod {
					gpuJobPodsList[curPod] = job
					break
				}
			}
		}
	}

	leastUsedJobs := make([]JobInfo, 0)
	for podName, job := range gpuJobPodsList {
		// 将time和util转换为string类型
		_time := fmt.Sprintf("%d", duration)
		_util := fmt.Sprintf("%d", util)
		if mgr.promClient.GetLeastUsedGPUJobList(podName, _time, _util) > 0 {
			pod, _ := mgr.kubeClient.CoreV1().Pods(namespace).Get(c, podName, metav1.GetOptions{})
			if len(pod.OwnerReferences) == 0 {
				continue
			}
			leastUsedJobs = append(leastUsedJobs, JobInfo{
				jobName:       job,
				jobType:       pod.OwnerReferences[0].Kind,
				jobAPIVersion: pod.OwnerReferences[0].APIVersion,
			})
		}
	}
	return leastUsedJobs
}
