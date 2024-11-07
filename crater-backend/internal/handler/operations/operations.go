package operations

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/config"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/monitor"
	mysmtp "github.com/raids-lab/crater/pkg/smtp"
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

func NewOperationsMgr(conf handler.RegisterConfig) handler.Manager {
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
	g.POST("/whitelist", mgr.AddJobWhiteList)
	g.GET("/whitelist", mgr.GetWhiteList)
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

type JobFreRequest struct {
	TimeRange int `form:"timeRange" binding:"required"`
	Util      int `form:"util"`
}

func (mgr *OperationsMgr) getJobWhiteList(c *gin.Context) ([]string, error) {
	var cleanList []string
	wlDB := query.Whitelist
	data, err := wlDB.WithContext(c).Find()
	if err != nil {
		return nil, err
	}
	for _, item := range data {
		cleanList = append(cleanList, item.Name)
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

var newEntries struct {
	Entries []string `json:"white_list"`
}

// AddJobWhiteList godoc
// @Summary Add job white list
// @Description add job white list
// @Tags Operations
// @Accept json
// @Produce json
// @param newEntries body []string true "white list"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/whitelist [post]
func (mgr *OperationsMgr) AddJobWhiteList(c *gin.Context) {
	if err := c.ShouldBindJSON(&newEntries); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.InvalidRequest)
		return
	}
	wlDB := query.Whitelist
	lists := []*model.Whitelist{}
	for _, job := range newEntries.Entries {
		lists = append(lists, &model.Whitelist{Name: job})
	}
	err := wlDB.WithContext(c).CreateInBatches(lists, 2)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "White list updated successfully")
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
	deleteJobList := []string{}

	for _, job := range unUsedJobs {
		if lo.Contains(whiteList, job.jobName) {
			fmt.Printf("Job %s is in the white list\n", job)
			continue
		}
		if err := mgr.SendGPUAlarm(c, job.jobName); err != nil {
			fmt.Println("Send Alarm Email failed:", err)
		}
		if err := mgr.deleteJobByName(c, job.jobAPIVersion, job.jobType, job.jobName); err != nil {
			fmt.Printf("Delete job %s failed\n", job)
			fmt.Println(err)
		}
		fmt.Printf("Delete job %s successfully\n", job)
		deleteJobList = append(deleteJobList, job.jobName)
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

func (mgr *OperationsMgr) SendGPUAlarm(c *gin.Context, jobname string) error {
	j := query.Job
	u := query.User
	job, err := j.WithContext(c).Where(j.JobName.Eq(jobname)).First()
	if err != nil {
		return err
	}
	user, err := u.WithContext(c).Where(u.ID.Eq(job.UserID)).First()
	if err != nil {
		return err
	}
	subject := "Crater平台Job删除告警"
	body := fmt.Sprintf("用户：%s您好,您的job:%s由于占用了gpu资源,但gpu资源利用率太低,平台即将删除该job", user.Name, jobname)
	err = mysmtp.SendEmail(*user.Attributes.Data().Email, subject, body)
	if err != nil {
		return err
	}
	return nil
}
