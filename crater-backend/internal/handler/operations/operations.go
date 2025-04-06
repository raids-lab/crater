package operations

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/alert"
	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/config"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/monitor"
)

const (
	VCJOBAPIVERSION   = "batch.volcano.sh/v1alpha1"
	VCJOBKIND         = "Job"
	AIJOBAPIVERSION   = "aisystem.github.com/v1alpha1"
	AIJOBKIND         = "AIJob"
	CRONJOBNAMESPACE  = "crater"
	CRAONJOBLABELKEY  = "crater.raids-lab.io/component"
	CRONJOBLABELVALUE = "cronjob"
)

type JobInfo struct {
	jobName       string
	jobType       string
	jobAPIVersion string
}

type SetKeepRequest struct {
	Name string `uri:"name" binding:"required"`
}

type SetLockTimeRequest struct {
	Name        string `json:"name" binding:"required"`
	IsPermanent bool   `json:"isPermanent"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Minutes     int    `json:"minutes"`
}

type ClearLockTimeRequest struct {
	Name string `json:"name" binding:"required"`
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
	g.DELETE("/cleanup", mgr.CleanupJobs)
	g.DELETE("/waiting/jupyter", mgr.CancelWaitingJupyterJobs)
	g.GET("/cronjob", mgr.GetCronjobConfigs)
	g.PUT("/cronjob", mgr.UpdateCronjobConfig)
	g.PUT("/add/locktime", mgr.AddLockTime)
	g.PUT("/clear/locktime", mgr.ClearLockTime)
}

type FreeJobRequest struct {
	TimeRange int  `form:"timeRange" binding:"required"`
	WaitTime  int  `form:"waitTime"`
	Util      int  `form:"util"`
	Confirm   bool `form:"confirm"`
}

type CleanupJobsRequest struct {
	BatchDays       *int `form:"batchDays"`
	InteractiveDays *int `form:"interactiveDays"`
}

type CancelWaitingJupyterJobsRequest struct {
	WaitMinitues int `form:"waitMinitues" binding:"required"`
}

type CronjobConfigs struct {
	Name     string            `json:"name"`
	Schedule string            `json:"schedule"`
	Suspend  bool              `json:"suspend"`
	Configs  map[string]string `json:"configs"`
}

func (mgr *OperationsMgr) getJobWhiteList(c *gin.Context) ([]string, error) {
	var cleanList []string
	jobDB := query.Job
	curTime := util.GetLocalTime()

	data, err := jobDB.WithContext(c).Where(jobDB.LockedTimestamp.Gt(curTime)).Find()

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

// SetLockTime godoc
// @Summary set LockTime of the job
// @Description set LockTime of the job
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/add/locktime [put]
func (mgr *OperationsMgr) AddLockTime(c *gin.Context) {
	var req SetLockTimeRequest

	// JSON 参数
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobDB := query.Job

	// 永久锁定
	lockTime := util.GetPermanentTime()

	// 非永久锁定
	if !req.IsPermanent {
		lockTime = util.GetLocalTime().Add(
			time.Duration(req.Days)*24*time.Hour + time.Duration(req.Hours)*time.Hour + time.Duration(req.Minutes)*time.Minute,
		)
	}

	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.LockedTimestamp, lockTime); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	message := fmt.Sprintf("Parmanently lock %s", req.Name)
	if !req.IsPermanent {
		message = fmt.Sprintf("Set %s lockTime to %s", req.Name, lockTime.Format("2006-01-02 15:04:05"))
	}
	resputil.Success(c, message)
}

// ClearLockTime godoc
// @Summary clear LockTime of the job
// @Description clear LockTime of the job
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/clear/locktime [put]
func (mgr *OperationsMgr) ClearLockTime(c *gin.Context) {
	var req ClearLockTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	jobDB := query.Job
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.LockedTimestamp, nil); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Clear %s lockTime", req.Name)
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

	// delete time: 即当前时间 + waitTime
	deleteTime := util.GetLocalTime().Add(time.Duration(req.WaitTime) * time.Minute)

	// remind
	reminded := mgr.remindLeastUsedGPUJobs(c, req.TimeRange, req.Util, deleteTime)

	// delete
	deleted := mgr.deleteLeastUsedGPUJobs(c, req.TimeRange+req.WaitTime, req.Util, true)

	resputil.Success(c, map[string][]string{
		"reminded": reminded,
		"deleted":  deleted,
	})
}

// getTimeouts calculates the timeout durations based on request parameters
func getTimeouts(req *CleanupJobsRequest) (batchJobTimeout, interactiveJobTimeout time.Duration) {
	batchJobTimeout = 4 * 24 * time.Hour
	interactiveJobTimeout = 24 * time.Hour
	if req.BatchDays != nil {
		batchJobTimeout = time.Duration(*req.BatchDays) * 24 * time.Hour
	}
	if req.InteractiveDays != nil {
		interactiveJobTimeout = time.Duration(*req.InteractiveDays) * 24 * time.Hour
	}
	return
}

// getJobsToCheck identifies jobs that need to be checked for cleanup
func (mgr *OperationsMgr) getJobsToCheck(c *gin.Context, batchTimeout, interactiveTimeout time.Duration) ([]JobInfo, error) {
	jobDB := query.Job
	runningJobs, err := jobDB.WithContext(c).Where(jobDB.Status.Eq(string(batch.Running))).Find()
	if err != nil {
		logutils.Log.Errorf("Failed to get running jobs: %v", err)
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return nil, err
	}
	jobsToCheck := make([]JobInfo, 0, len(runningJobs))
	now := time.Now()
	for _, job := range runningJobs {
		jobAge := now.Sub(job.RunningTimestamp)
		shouldCheck := false
		if job.JobType == model.JobTypeJupyter || job.JobType == model.JobTypeWebIDE {
			shouldCheck = jobAge > interactiveTimeout
		} else {
			shouldCheck = jobAge > batchTimeout
		}
		if !shouldCheck {
			continue
		}
		jobInfo, ok := mgr.getJobInfo(c, job)
		if !ok {
			continue
		}
		jobsToCheck = append(jobsToCheck, jobInfo)
	}
	return jobsToCheck, nil
}

// getJobInfo retrieves job information from Kubernetes
func (mgr *OperationsMgr) getJobInfo(c *gin.Context, job *model.Job) (JobInfo, bool) {
	jobPodsList := mgr.promClient.GetJobPodsList()
	namespace := config.GetConfig().Workspace.Namespace
	pods, ok := jobPodsList[job.JobName]
	if !ok || len(pods) == 0 {
		return JobInfo{}, false
	}
	pod, err := mgr.kubeClient.CoreV1().Pods(namespace).Get(c, pods[0], metav1.GetOptions{})
	if err != nil || len(pod.OwnerReferences) == 0 {
		return JobInfo{}, false
	}
	return JobInfo{
		jobName:       job.JobName,
		jobType:       pod.OwnerReferences[0].Kind,
		jobAPIVersion: pod.OwnerReferences[0].APIVersion,
	}, true
}

// CleanupJobs godoc
// @Summary Cleanup jobs based on type and duration
// @Description Delete batch jobs older than 4 days and interactive jobs older than 1 day
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query CleanupJobsRequest true "batchDays interactiveDays"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/operations/cleanup [delete]
// validateCleanupRequest validates the cleanup request parameters
func (mgr *OperationsMgr) CleanupJobs(c *gin.Context) {
	var req CleanupJobsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	batchTimeout, interactiveTimeout := getTimeouts(&req)
	jobDB := query.Job
	jobsToCheck, err := mgr.getJobsToCheck(c, batchTimeout, interactiveTimeout)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobsToDelete := mgr.filterJobsToKeep(c, jobsToCheck)
	deletedJobList := []string{}
	for _, job := range jobsToDelete {
		if err := mgr.deleteJobByName(c, job.jobAPIVersion, job.jobType, job.jobName); err != nil {
			logutils.Log.Infof("Delete job %s failed: %v", job.jobName, err)
			continue
		}
		deletedJobList = append(deletedJobList, job.jobName)
		dbJob, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).First()
		if err != nil {
			continue
		}
		if dbJob.AlertEnabled {
			alertMgr := alert.GetAlertMgr()
			if err := alertMgr.CleanJob(c, job.jobName, nil); err != nil {
				logutils.Log.Infof("Send Alarm Email failed for job %s", job.jobName)
			}
		}
	}

	defaultRemindTime := 24 * time.Hour
	remindedJobList := mgr.remindLongTimeJobs(c, batchTimeout, interactiveTimeout, defaultRemindTime)

	resputil.Success(c, map[string][]string{
		"deleted":  deletedJobList,
		"reminded": remindedJobList,
	})
}

func (mgr *OperationsMgr) remindLongTimeJobs(c *gin.Context, batchTimeout, interactiveTimeout, timeRange time.Duration) []string {
	jobDB := query.Job
	jobsToCheck, err := mgr.getJobsToCheck(c, batchTimeout-timeRange, interactiveTimeout-timeRange)
	if err != nil {
		return nil
	}

	remindedJobList := []string{}
	deleteTime := time.Now().Add(-timeRange)
	for _, job := range jobsToCheck {
		dbJob, dbErr := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).First()
		if dbErr != nil {
			continue
		}
		if !dbJob.AlertEnabled {
			continue
		}
		alertMgr := alert.GetAlertMgr()
		if err := alertMgr.RemindLongTimeRunningJob(c, job.jobName, deleteTime, nil); err != nil {
			logutils.Log.Infof("Send Alarm Email failed for job %s", job.jobName)
		}
		remindedJobList = append(remindedJobList, job.jobName)
	}
	return remindedJobList
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

func (mgr *OperationsMgr) deleteLeastUsedGPUJobs(c *gin.Context, duration, gpuUtil int, confirm bool) []string {
	toDeleteJobs := mgr.filterJobsToKeep(c, mgr.getLeastUsedGPUJobs(c, duration, gpuUtil))
	deletedJobList := []string{}
	jobDB := query.Job
	for _, job := range toDeleteJobs {
		dbJob, dbErr := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).First()

		if dbErr != nil {
			continue
		}

		deletedJobList = append(deletedJobList, job.jobName)

		if !confirm {
			// just get to delete job list
			continue
		}

		if err := mgr.deleteJobByName(c, job.jobAPIVersion, job.jobType, job.jobName); err != nil {
			logutils.Log.Infof("Delete job %s failed\n", job)
		}

		if !dbJob.AlertEnabled {
			// no need to send alert
			continue
		}

		alertMgr := alert.GetAlertMgr()
		if err := alertMgr.DeleteJob(c, job.jobName, nil); err != nil {
			logutils.Log.Infof("Send Alarm Email failed for job %s", job)
		}
	}
	return deletedJobList
}

func (mgr *OperationsMgr) remindLeastUsedGPUJobs(c *gin.Context, duration, gpuUtil int, deleteTime time.Time) []string {
	jobDB := query.Job

	toRemindJobs := mgr.filterJobsToKeep(c, mgr.getLeastUsedGPUJobs(c, duration, gpuUtil))

	remindedJobList := []string{}

	// check all the running jobs
	remindedJobs, _ := jobDB.WithContext(c).Where(jobDB.Status.Eq(string(batch.Running))).Find()

	remindMap := make(map[string]bool)
	for _, item := range remindedJobs {
		remindMap[item.JobName] = item.Reminded
	}

	for _, job := range toRemindJobs {
		dbJob, dbErr := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).First()
		if dbErr != nil {
			continue
		}

		if !dbJob.AlertEnabled {
			continue
		}

		if remind, ok := remindMap[job.jobName]; ok && !remind {
			// if remind is false, then send alarm email, and set remind to true
			alertMgr := alert.GetAlertMgr()
			if err := alertMgr.RemindLowUsageJob(c, job.jobName, deleteTime, nil); err != nil {
				logutils.Log.Infof("Send Alarm Email failed for job %s", job.jobName)
			}
			// set remind to true
			if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).Update(jobDB.Reminded, true); err != nil {
				logutils.Log.Infof("Update job %s remind failed\n", job)
			}
			remindedJobList = append(remindedJobList, job.jobName)
		}
		delete(remindMap, job.jobName)
	}

	// for the jobs remaining in remindMap, set remind to false
	for jobName := range remindMap {
		if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(jobName)).Update(jobDB.Reminded, false); err != nil {
			logutils.Log.Infof("Update job %s remind failed\n", jobName)
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

func (mgr *OperationsMgr) getLeastUsedGPUJobs(c *gin.Context, duration, gpuUtil int) []JobInfo {
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
		_util := fmt.Sprintf("%d", gpuUtil)
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

// DeleteUnscheduledJupyterJobs godoc
// @Summary Delete unscheduled jupyter jobs
// @Description check pending jupyter jobs, delete if not scheduled
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query CancelWaitingJupyterJobsRequest true "waitMinitues"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/waiting/jupyter [delete]
func (mgr *OperationsMgr) CancelWaitingJupyterJobs(c *gin.Context) {
	var req CancelWaitingJupyterJobsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	deletedJobs := mgr.deleteUnscheduledJupyterJobs(c, req.WaitMinitues)
	resputil.Success(c, deletedJobs)
}

func (mgr *OperationsMgr) deleteUnscheduledJupyterJobs(c *gin.Context, waitMinitues int) []string {
	jobDB := query.Job
	// status = batch.Pending, jobType = model.JobTypeJupyter, now - CreationTimestamp > waitMinitues
	jobs, err := jobDB.WithContext(c).Where(
		jobDB.Status.Eq(string(batch.Pending)),
		jobDB.JobType.Eq(string(model.JobTypeJupyter)),
		jobDB.CreationTimestamp.Lt(time.Now().Add(-time.Duration(waitMinitues)*time.Minute)),
	).Find()

	if err != nil {
		logutils.Log.Errorf("Failed to get unscheduled jupyter jobs: %v", err)
		return nil
	}

	deletedJobs := []string{}
	for _, job := range jobs {
		if mgr.isJobscheduled(c, job.JobName) {
			continue
		}

		// delete job
		vcjob := &batch.Job{}
		namespace := config.GetConfig().Workspace.Namespace
		if err := mgr.client.Get(c, client.ObjectKey{Name: job.JobName, Namespace: namespace}, vcjob); err != nil {
			logutils.Log.Errorf("Failed to get job %s: %v", job.JobName, err)
			continue
		}

		if err := mgr.client.Delete(c, vcjob); err != nil {
			logutils.Log.Errorf("Failed to delete job %s: %v", job.JobName, err)
			continue
		}

		deletedJobs = append(deletedJobs, job.JobName)
	}

	return deletedJobs
}

// 所有Pod被schedule，返回true；否则返回false
func (mgr *OperationsMgr) isJobscheduled(c *gin.Context, jobName string) bool {
	// get pods: namespace=config.GetConfig().Workspace.Namespace, label['volcano.sh/job-name']=jobName
	namespace := config.GetConfig().Workspace.Namespace
	pods, err := mgr.kubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("volcano.sh/job-name=%s", jobName),
	})
	if err != nil {
		logutils.Log.Errorf("Failed to get pods: %v", err)
		return false
	}

	result := true
	for i := range pods.Items {
		pod := &pods.Items[i]
		// check pod's conditions: type==PodScheduled && status==True
		thisPodScheduled := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type != "PodScheduled" {
				continue
			}
			if condition.Status == "True" {
				thisPodScheduled = true
				break
			}
			if condition.Status == "False" {
				return false
			}
		}
		result = result && thisPodScheduled
	}
	return result
}

// UpdateCronjobConfig godoc
// @Summary Update cronjob config
// @Description Update one cronjob config
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use body CronjobConfigs true "CronjobConfigs"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/cronjob [put]
func (mgr *OperationsMgr) UpdateCronjobConfig(c *gin.Context) {
	var req CronjobConfigs
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	fmt.Println(req)
	if err := mgr.updateCronjobConfig(c, req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Successfully update cronjob config")
}

func (mgr *OperationsMgr) updateCronjobConfig(c *gin.Context, cronjobConfigs CronjobConfigs) error {
	namespace := CRONJOBNAMESPACE
	cronjob, err := mgr.kubeClient.BatchV1().CronJobs(namespace).Get(c, cronjobConfigs.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cronjob %s: %w", cronjobConfigs.Name, err)
	}

	// 更新 schedule
	cronjob.Spec.Schedule = cronjobConfigs.Schedule

	// 更新 suspend 字段
	*cronjob.Spec.Suspend = cronjobConfigs.Suspend

	// CronJob 确保只有一个 container
	if len(cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("cronjob %s has no container", cronjobConfigs.Name)
	}
	container := &cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	// 检查并更新 env 变量，确保传入的 env 必须已经存在
	for key, newVal := range cronjobConfigs.Configs {
		found := false
		for idx, env := range container.Env {
			if env.Name == key {
				container.Env[idx].Value = newVal
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %s missing env key: %s", container.Name, key)
		}
	}

	// 更新 CronJob 对象
	_, err = mgr.kubeClient.BatchV1().CronJobs(namespace).Update(c, cronjob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cronjob %s: %w", cronjobConfigs.Name, err)
	}
	return nil
}

// GetCronjobConfigs godoc
// @Summary Get all cronjob configs
// @Description Get all cronjob configs
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/cronjob [get]
func (mgr *OperationsMgr) GetCronjobConfigs(c *gin.Context) {
	configs, err := mgr.getCronjobConfigs(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, configs)
}

func (mgr *OperationsMgr) getCronjobConfigs(c *gin.Context) ([]CronjobConfigs, error) {
	// get all cronjobs in namespace crater, label crater.raids-lab.io/component=cronjob
	// apiVersion: batch/v1, kind: CronJob
	cronjobConfigList := make([]CronjobConfigs, 0)
	namespace := CRONJOBNAMESPACE
	labelSelector := fmt.Sprintf("%s=%s", CRAONJOBLABELKEY, CRONJOBLABELVALUE)
	cronjobs, err := mgr.kubeClient.BatchV1().CronJobs(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logutils.Log.Errorf("Failed to get cronjobs: %v", err)
		return nil, err
	}
	for i := range cronjobs.Items {
		cronjob := &cronjobs.Items[i]
		configs := make(map[string]string)

		// cronjob 确保只有一个 container
		containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
		if len(containers) != 1 {
			continue
		}
		container := containers[0]
		for _, env := range container.Env {
			configs[env.Name] = env.Value
		}
		cronjobConfigList = append(cronjobConfigList, CronjobConfigs{
			Name:     cronjob.Name,
			Configs:  configs,
			Suspend:  *cronjob.Spec.Suspend,
			Schedule: cronjob.Spec.Schedule,
		})
	}
	return cronjobConfigList, nil
}
