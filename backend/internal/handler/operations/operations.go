package operations

import (
	"fmt"

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
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

func (mgr *OperationsMgr) RegisterProtected(_ *gin.RouterGroup) {
}

func (mgr *OperationsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/whitelist", mgr.GetWhiteList)
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
	g.PUT("/keep/:name", mgr.SetKeepWhenLowResourceUsage)
}

type JobFreRequest struct {
	TimeRange int `form:"timeRange" binding:"required"`
	Util      int `form:"util"`
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

// DeleteUnUsedJobList godoc
// @Summary Delete not using gpu job list
// @Description check job list and delete not using gpu job
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query JobFreRequest true "timeRange util"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/job [delete]
func (mgr *OperationsMgr) DeleteUnUsedJobList(c *gin.Context) {
	var req JobFreRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.TimeRange <= 0 {
		resputil.BadRequestError(c, "timeRange must be greater than 0")
		return
	}

	unUsedJobs := mgr.getLeastUsedGPUJobs(c, req.TimeRange, req.Util)
	whiteList, _ := mgr.getJobWhiteList(c)

	jobDB := query.Job
	remindedJobs, _ := jobDB.WithContext(c).Where(jobDB.Reminded).Find()
	remindMap := make(map[string]bool)
	for _, item := range remindedJobs {
		remindMap[item.JobName] = item.Reminded
	}
	// todo: including other job types

	deleteJobList := []string{}

	for _, job := range unUsedJobs {
		if lo.Contains(whiteList, job.jobName) {
			fmt.Printf("Job %s is in the white list\n", job)
			continue
		}

		if remind, ok := remindMap[job.jobName]; ok && remind {
			// if remind is true, then delete job
			if err := mgr.deleteJobByName(c, job.jobAPIVersion, job.jobType, job.jobName); err != nil {
				fmt.Printf("Delete job %s failed\n", job)
			}
			deleteJobList = append(deleteJobList, job.jobName)
			// remove from remindMap
			delete(remindMap, job.jobName)
		} else {
			// if remind is false, then send alarm email, and set remind to true
			if err := alert.GetAlertMgr().JobFreed(c, job.jobName, nil); err != nil {
				fmt.Println("Send Alarm Email failed:", err)
			}
			// set remind to true
			if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(job.jobName)).Update(jobDB.Reminded, true); err != nil {
				fmt.Printf("Update job %s remind failed\n", job)
			}
		}
	}

	// for the jobs remaining in remindMap, set remind to false
	for jobName := range remindMap {
		if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(jobName)).Update(jobDB.Reminded, false); err != nil {
			fmt.Printf("Update job %s remind failed\n", jobName)
		}
	}
	response := map[string][]string{
		"delete_job_list": deleteJobList,
	}
	resputil.Success(c, response)
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
