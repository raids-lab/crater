package aitaskctl

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	"github.com/raids-lab/crater/pkg/profiler"
	"github.com/raids-lab/crater/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TaskController 调度task
type (
	TaskController struct {
		jobControl    *crclient.JobControl      // 创建AIJob的接口
		taskQueue     *TaskQueue                // 队列信息
		quotaInfos    sync.Map                  // quota缓存
		jobStatusChan <-chan util.JobStatusChan // 获取job的状态变更信息，同步到数据库
		// taskUpdateChan <-chan util.TaskUpdateChan
		profiler *profiler.Profiler
	}
)

const (
	scheduleInterval = 5 * time.Second
)

var (
	taskNameSpace = config.GetConfig().Workspace.Namespace
)

// NewTaskController returns a new *TaskController
func NewTaskController(crClient client.Client, cs kubernetes.Interface, statusChan <-chan util.JobStatusChan) *TaskController {
	return &TaskController{
		jobControl:    &crclient.JobControl{Client: crClient, KubeClient: cs},
		taskQueue:     NewTaskQueue(),
		quotaInfos:    sync.Map{},
		jobStatusChan: statusChan,
		// taskUpdateChan: taskChan,
	}
}

func (c *TaskController) SetProfiler(srcProfiler *profiler.Profiler) {
	c.profiler = srcProfiler
}

func (c *TaskController) ParseUserquotas(userquotas []*model.UserProject) ([]*models.Quota, error) {
	var quotas []*models.Quota
	for i := range userquotas {
		quota := &models.Quota{}
		quota.UserName = fmt.Sprintf("%d-%d", userquotas[i].UserID, userquotas[i].ProjectID)
		resourceList := v1.ResourceList{
			v1.ResourceCPU:                    resource.MustParse(fmt.Sprint(userquotas[i].CPUReq)),
			v1.ResourceMemory:                 resource.MustParse(fmt.Sprintf("%dGi", userquotas[i].MemReq)),
			v1.ResourceName("nvidia.com/gpu"): resource.MustParse(fmt.Sprint(userquotas[i].GPUReq)),
		}
		quota.HardQuota = model.ResourceListToJSON(resourceList)
		quotas = append(quotas, quota)
	}
	return quotas, nil
}

func (c *TaskController) ConvertJobStatusToTaskStatus(jobStatus model.JobStatus) string {
	switch jobStatus {
	case model.JobInitial, model.JobCreated:
		return models.TaskQueueingStatus
	case model.JobRunning:
		return models.TaskRunningStatus
	case model.JobSucceeded:
		return models.TaskSucceededStatus
	case model.JobFailed:
		return models.TaskFailedStatus
	case model.JobPreempted:
		return models.TaskPreemptedStatus
	default:
		return "" // Unknown or not applicable status
	}
}

func (c *TaskController) ConvertTaskStatusToJobStatus(taskStatus string) model.JobStatus {
	switch taskStatus {
	case models.TaskQueueingStatus:
		return model.JobCreated // Assuming JobCreated as the reverse for TaskQueueingStatus
	case models.TaskRunningStatus:
		return model.JobRunning
	case models.TaskSucceededStatus:
		return model.JobSucceeded
	case models.TaskFailedStatus:
		return model.JobFailed
	case models.TaskPreemptedStatus:
		return model.JobPreempted
	default:
		return 0 // Assuming 0 as the default unknown JobStatus
	}
}

func (c *TaskController) ParseTask(aijobs []*model.AIJob) ([]models.AITask, error) {
	var tasks []models.AITask
	// not for request
	for i := range aijobs {
		task := &models.TaskAttr{}
		task.ID = aijobs[i].ID
		task.Namespace = taskNameSpace
		task.TaskName = aijobs[i].Name
		task.Status = c.ConvertJobStatusToTaskStatus(aijobs[i].Status)
		task.UserName = fmt.Sprintf("%d-%d", aijobs[i].UserID, aijobs[i].ProjectID)
		task.TaskType = aijobs[i].TaskType
		resourceRequst, err := model.JSONToResourceList(aijobs[i].ResourceRequest)
		if err != nil {
			return nil, err
		}
		task.ResourceRequest = resourceRequst
		extraMap := model.JSONStringToMap(*aijobs[i].Extra)
		slo, err := strconv.ParseUint(extraMap["SLO"], 10, 64)
		if err != nil {
			return nil, err
		}
		task.SLO = uint(slo)
		task.Image = extraMap["Image"]
		task.WorkingDir = extraMap["WorkingDir"]
		task.ShareDirs = models.JSONStringToVolumes(extraMap["ShareDirs"])
		task.Command = extraMap["Command"]
		task.GPUModel = extraMap["GPUModel"]
		task.SchedulerName = extraMap["SchedulerName"]
		tasks = append(tasks, *models.FormatTaskAttrToModel(task))
	}
	return tasks, nil
}

func (c *TaskController) ListByUserAndStatuses(userName string, statuses []string) ([]models.AITask, error) {
	var tasks []models.AITask
	var aijobs []*model.AIJob
	var err error
	var uid, pid uint
	fmt.Sscanf(userName, "%d-%d", uid, pid)
	jobQueryModel := query.AIJob
	exec := jobQueryModel.WithContext(context.Background())
	exec = exec.Where(jobQueryModel.UserID.Eq(uid), jobQueryModel.ProjectID.Eq(pid))
	if len(statuses) == 0 {
		aijobs, err = exec.Find()
		if err != nil {
			logutils.Log.Errorf("list user:%v tasks failed, err: %v", userName, err)
			return nil, err
		}
		tasks, err = c.ParseTask(aijobs)
	} else {
		for i := range statuses {
			exec.Where(jobQueryModel.Status.In(uint8(c.ConvertTaskStatusToJobStatus(statuses[i]))))
		}
		aijobs, err = exec.Find()
		if err != nil {
			logutils.Log.Errorf("list user:%v tasks failed, err: %v", userName, err)
			return nil, err
		}
		tasks, err = c.ParseTask(aijobs)
	}
	return tasks, err
}

func (c *TaskController) UpdateStatus(taskID uint, status string) error {
	jobQueryModel := query.AIJob
	aijob, err := jobQueryModel.WithContext(context.Background()).Where(jobQueryModel.ID.Eq(taskID)).First()
	if err != nil {
		logutils.Log.Errorf("get task update failed, err: %v", err)
		return err
	}

	jobStat := c.ConvertTaskStatusToJobStatus(status)
	if aijob.Status == jobStat {
		return nil
	}

	updateMap := make(map[string]any)
	updateMap["status"] = jobStat

	t := time.Now()
	switch status {
	case models.TaskCreatedStatus:
		updateMap["admitted_at"] = &t
	case models.TaskRunningStatus:
		updateMap["started_at"] = &t
	case models.TaskSucceededStatus, models.TaskFailedStatus:
		updateMap["finish_at"] = &t
	}
	_, err = jobQueryModel.WithContext(context.Background()).Where(jobQueryModel.ID.Eq(taskID)).Updates(updateMap)
	return err
}

func (c *TaskController) GetByID(taskID uint) (*models.AITask, error) {
	jobQueryModel := query.AIJob
	aijob, err := jobQueryModel.WithContext(context.Background()).Where(jobQueryModel.ID.Eq(taskID)).First()
	if err != nil {
		logutils.Log.Errorf("get task update event failed, err: %v", err)
		return nil, err
	}
	tasks, err := c.ParseTask([]*model.AIJob{aijob})
	if err != nil {
		logutils.Log.Errorf("get task update event failed, err: %v", err)
		return nil, err
	}
	task := &tasks[0]
	return task, err
}

func getUserQuota(quota, quotaInProject *model.EmbeddedQuota) {
	quotaValue := reflect.ValueOf(quota).Elem()
	projectValue := reflect.ValueOf(quotaInProject).Elem()

	for i := 0; i < quotaValue.NumField(); i++ {
		field := quotaValue.Field(i)
		if field.Interface() == model.Unlimited {
			// 对于 Unlimited 的字段，使用项目的对应字段
			field.Set(projectValue.Field(i))
		} else if field.Type() == reflect.TypeOf((*string)(nil)) {
			// 对于指针类型的字段，如果为 nil，使用项目的对应字段
			field.Set(projectValue.Field(i))
		}
	}
}

func (c *TaskController) ListAllQuotas() ([]*models.Quota, error) {
	var ret []*models.Quota
	projectQueryModel := query.Project
	upQueryModel := query.UserProject
	projects, err := projectQueryModel.WithContext(context.Background()).Find()
	if err != nil {
		logutils.Log.Errorf("list all quotas failed, err: %v", err)
		return nil, err
	}
	for i := range projects {
		userQuotas, err := upQueryModel.WithContext(context.Background()).Where(upQueryModel.ProjectID.Eq(projects[i].ID)).Find()
		if err != nil {
			logutils.Log.Errorf("list all quotas failed, err: %v", err)
			return nil, err
		}
		for j := range userQuotas {
			getUserQuota(&userQuotas[j].EmbeddedQuota, &projects[i].EmbeddedQuota)
		}
		quotas, err := c.ParseUserquotas(userQuotas)
		if err != nil {
			logutils.Log.Errorf("list all quotas failed, err: %v", err)
			return nil, err
		}
		ret = append(ret, quotas...)
	}

	return ret, nil
}

// Init init taskQueue And quotaInfos
func (c *TaskController) Init() error {
	// init quotas
	quotas, err := c.ListAllQuotas()
	if err != nil {
		logutils.Log.Errorf("list all quotas failed, err: %v", err)
	}
	logutils.Log.Infof("list all quotas success, len: %v", len(quotas))
	for i := range quotas {
		// 添加quota
		c.AddOrUpdateQuotaInfo(quotas[i].UserName, quotas[i])
		queueingList, err := c.ListByUserAndStatuses(quotas[i].UserName, models.TaskQueueingStatuses)
		if err != nil {
			logutils.Log.Errorf("list user:%v queueing tasks failed, err: %v", quotas[i].UserName, err)
			continue
		}
		c.taskQueue.InitUserQueue(quotas[i].UserName, queueingList)
	}
	return nil
}

// AddUser adds a user with the specified username and quota to the TaskController.
// It also initializes the user's task queue based on their quota.
func (c *TaskController) AddUser(_ string, quota *models.Quota) {
	c.AddOrUpdateQuotaInfo(quota.UserName, quota)
	queueingList, err := c.ListByUserAndStatuses(quota.UserName, models.TaskQueueingStatuses)
	if err != nil {
		logutils.Log.Errorf("list user:%v queueing tasks failed, err: %v", quota.UserName, err)
	}
	c.taskQueue.InitUserQueue(quota.UserName, queueingList)
}

// Start method
func (c *TaskController) Start(ctx context.Context) error {
	// init share dirs
	// err := c.jobControl.InitShareDir()
	// if err != nil {
	// 	logutils.Log.Errorf("get share dirs failed, err: %v", err)
	// 	return err
	// }
	// 1. init 初始化task队列和quota信息，存在缓存里
	// c.Init()
	// 2. 接受AIJOB的crd状态变更信息，通过jobStatusChan
	go c.watchJobStatus(ctx)
	// 3. 接收task变更信息
	// go c.watchTaskUpdate(ctx)
	// 4. schedule线程
	go wait.UntilWithContext(ctx, c.schedule, scheduleInterval)
	return nil
}

// Init 初始化队列信息, quota 信息

// WatchJobStatus, 根据AIJob的状态更新task的状态
func (c *TaskController) watchJobStatus(ctx context.Context) {
	for {
		select {
		case status := <-c.jobStatusChan:
			// 更新task在db中的状态
			task, err := c.updateTaskStatus(status.TaskID, status.NewStatus)
			if err != nil {
				logutils.Log.Errorf("update task status failed, err: %v", err)
				logutils.Log.Infof("get job status event, taskID: %v, newStatus: %v", status.TaskID, status.NewStatus)
				break
			}
			// 更新quota，减去已经完成的作业的资源
			if util.IsCompletedStatus(status.NewStatus) {
				quotaInfo := c.GetQuotaInfo(task.UserName)
				if quotaInfo != nil {
					quotaInfo.DeleteTask(task)
				}
			} else if status.NewStatus == models.TaskQueueingStatus { // todo: fixme??
				// 更新task队列
				c.taskQueue.AddTask(task)
			}
		case <-ctx.Done():
			return
		}
	}
}

// hook: receive task updated event from server events
func (c *TaskController) TaskUpdated(event util.TaskUpdateChan) {
	task, err := c.GetByID(event.TaskID)
	if err != nil {
		logutils.Log.Errorf("get task update event failed, err: %v", err)
		return
	}
	tidStr := strconv.FormatUint(uint64(event.TaskID), 10)
	logutils.Log.Infof("get task update event, taskID: %v, operation: %v", tidStr, event.Operation)
	switch event.Operation {
	case util.CreateTask:
		//
		c.taskQueue.AddTask(task)
	case util.UpdateTask:
		// mostly update slo
		c.taskQueue.UpdateTask(task)
	case util.DeleteTask:
		c.taskQueue.DeleteTaskByUserNameAndTaskID(event.UserName, tidStr)
		quotaInfo := c.GetQuotaInfo(task.UserName)
		if quotaInfo != nil {
			quotaInfo.DeleteTask(task)
		}
		// delete aijob in cluster? todo:
		err = c.jobControl.DeleteJobFromTask(task)
		if err != nil {
			logutils.Log.Errorf("delete job from task failed, err: %v", err)
		}
		// delete from profiler
		if c.profiler != nil && (task.ProfileStatus == models.ProfileQueued || task.ProfileStatus == models.Profiling) {
			c.profiler.DeleteProfilePodFromTask(task.ID)
		}
		logutils.Log.Infof("delete task in task controller, %d", event.TaskID)
	}
}

// deprecated
// func (c *TaskController) watchTaskUpdate(ctx context.Context) {
// 	for {
// 		select {
// 		case t := <-c.taskUpdateChan:
// 			// 更新task在队列的状态
// 			task, err := c.taskDB.GetByID(t.TaskID)
// 			tidStr := strconv.FormatUint(uint64(t.TaskID), 10)
// 			logutils.Log.Infof("get task update event, taskID: %v, operation: %v", tidStr, t.Operation)
// 			// 1. delete的情况
// 			if t.Operation == util.DeleteTask {
// 				c.taskQueue.DeleteTaskByUserNameAndTaskID(t.UserName, tidStr)
// 				// delete in cluster
// 				err = c.jobControl.DeleteJobFromTask(task)
// 				if err != nil {
// 					logutils.Log.Errorf("delete job from task failed, err: %v", err)
// 				}
// 				logutils.Log.Infof("delete task in task controller, %d", t.TaskID)
// 				continue
// 			} else if t.Operation == util.CreateTask {
// 				// 2. create
// 				c.taskQueue.AddTask(task)
// 			} else if t.Operation == util.UpdateTask {
// 				// 3. update slo
// 				c.taskQueue.UpdateTask(task)
// 			}
// 		case <-ctx.Done():
// 			return
// 		}
// 	}
// }

func (c *TaskController) updateTaskStatus(taskID, status string) (*models.AITask, error) {
	// convert taskID to uint
	tid, _ := strconv.ParseUint(taskID, 10, 64)

	t, err := c.GetByID(uint(tid))
	if err != nil {
		logutils.Log.Errorf("get task update failed, err: %v", err)
		return nil, err
	}
	if t.Status != status {
		err = c.UpdateStatus(uint(tid), status)
		if err != nil {
			logutils.Log.Errorf("get task update failed, err: %v", err)
			return nil, err
		}
		return nil, err
	}
	t.Status = status
	return t, nil
}

// schedule 简单功能：从用户队列中
//
//nolint:gocyclo // TODO: figure out a better way to handle this
func (c *TaskController) schedule(_ context.Context) {
	// 等待调度的队列
	candiates := make([]*models.AITask, 0)

	// 1. guaranteed job schedule
	for username, q := range c.taskQueue.userQueues {
		// 1. 复制一份quota
		// logutils.Log.Infof(username, q.gauranteedQueue)
		quotaCopy := c.GetQuotaInfoSnapshotByUsername(username)
		if quotaCopy == nil {
			logutils.Log.Errorf("quota not found, username: %v", username)
			continue
		}
		// 2. 从gauranteedQueue队列选出不超过quota的作业
		for _, t := range q.gauranteedQueue.List() {
			task := t.(*models.AITask)

			if !quotaCopy.CheckHardQuotaExceed(task) {
				candiates = append(candiates, task)
				quotaCopy.AddTask(task)
				logutils.Log.Infof("task quota check succeed,user %v task %v taskid %v", task.UserName, task.TaskName, task.ID)
			} else {
				logutils.Log.Infof("task quota exceed, user %v task %v taskid %v, request:%v, used:%v, hard:%v",
					task.UserName, task.TaskName, task.ID, task.ResourceRequest, quotaCopy.HardUsed, quotaCopy.Hard)
				// bug: 如果先检查资源多的，可能后面的都调度不了？？
			}
		}
	}

	// 2. best effort queue的作业调度
	for _, q := range c.taskQueue.userQueues {
		for _, t := range q.bestEffortQueue.List() {
			task := t.(*models.AITask)
			// logutils.Log.Infof("user:%v, task: %v, task status:%v, profile status: %v", task.UserName, task.ID, task.Status, task.ProfileStatus)
			// update profile status
			if c.profiler != nil {
				// todo: udpate profile status???
				if task.Status == models.TaskQueueingStatus && task.ProfileStatus == models.UnProfiled {
					c.profiler.SubmitProfileTask(task.ID)
					logutils.Log.Infof("submit profile task, user:%v taskID:%v, taskName:%v", task.UserName, task.ID, task.TaskName)
					task.ProfileStatus = models.ProfileQueued
				} else {
					// todo: 优化 check profile status
					candiates = append(candiates, task)
				}
			} else {
				candiates = append(candiates, task)
			}
		}
	}

	// todo: candidates队列的调度策略
	// sort by submit time
	sort.Slice(candiates, func(i, j int) bool {
		return candiates[i].CreatedAt.Before(candiates[j].CreatedAt)
	})
	for _, candidate := range candiates {
		task, err := c.GetByID(candidate.ID)
		if err != nil {
			logutils.Log.Errorf("get task from db failed, err: %v", err)
			continue
		}
		// check profiling status
		if task.ProfileStatus == models.Profiling || task.ProfileStatus == models.ProfileQueued {
			continue
		} else if task.ProfileStatus == models.ProfileFailed {
			//nolint:gocritic // TODO: figure out a better way to handle this
			// c.taskDB.UpdateStatus(task.ID, models.TaskFailedStatus, "task profile failed")
			c.taskQueue.DeleteTask(task)
			continue
		}
		// submit AIJob
		err = c.admitTask(task)
		if err != nil {
			logutils.Log.Errorf("create job from task: %v", err)
		} else {
			logutils.Log.Infof("create job from task success, taskID: %v", task.ID)
		}
	}
}

// admitTask 创建对应的aijob到集群中，更新task状态，更新quota
func (c *TaskController) admitTask(task *models.AITask) error {
	// 重新check quota
	// 更新quota
	quotaInfo := c.GetQuotaInfo(task.UserName)
	if quotaInfo == nil {
		// todo:
		return fmt.Errorf("quota not found")
	}

	if quotaInfo.CheckHardQuotaExceed(task) {
		return fmt.Errorf("quota exceed")
	}

	jobname, err := c.jobControl.CreateJobFromTask(task)
	if err != nil {
		err = c.UpdateStatus(task.ID, models.TaskFailedStatus)
		if err != nil {
			return err
		}
		c.taskQueue.DeleteTask(task)
		return err
	}
	// 更新task状态
	err = c.UpdateStatus(task.ID, models.TaskCreatedStatus)
	if err != nil {
		return err
	}
	// 更新jobname
	jobQueryModel := query.AIJob
	_, err = jobQueryModel.WithContext(context.Background()).Where(jobQueryModel.ID.Eq(task.ID)).Update(jobQueryModel.Name, jobname)
	if err != nil {
		logutils.Log.Errorf("admit task update failed, err: %v", err)
		return err
	}

	quotaInfo.AddTask(task)
	c.taskQueue.DeleteTask(task)
	return nil
}
