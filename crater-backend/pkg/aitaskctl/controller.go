package aitaskctl

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/crclient"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/util"
)

// TaskController 调度task
type TaskControllerInterface interface {
	AddOrUpdateQuotaInfo(queue *model.Account) (added bool, quotaInfo *QuotaInfo)
	DeleteQuotaInfo(namespace string)
	GetQuotaInfo(username string) *QuotaInfo
	GetQuotaInfoSnapshotByUsername(username string) *QuotaInfoSnapshot
	Init() error
	ListQuotaInfoSnapshot() []QuotaInfoSnapshot
	SetProfiler(srcProfiler *Profiler)
	Start(ctx context.Context) error
	TaskUpdated(event util.TaskUpdateChan)
	UpdateQuotaInfoHard(username string, hard v1.ResourceList)
}

type TaskController struct {
	jobControl    *crclient.JobControl      // 创建AIJob的接口
	taskDB        DBService                 // 获取db中的task
	taskQueue     *TaskQueue                // 队列信息
	quotaInfos    sync.Map                  // quota缓存
	jobStatusChan <-chan util.JobStatusChan // 获取job的状态变更信息，同步到数据库
	// taskUpdateChan <-chan util.TaskUpdateChan
	profiler *Profiler
}

const (
	scheduleInterval = 5 * time.Second
)

// NewTaskController returns a new *TaskController
func NewTaskController(
	serviceManager crclient.ServiceManagerInterface,
	crClient client.Client,
	cs kubernetes.Interface,
	statusChan <-chan util.JobStatusChan,
) TaskControllerInterface {
	return &TaskController{
		jobControl:    &crclient.JobControl{Client: crClient, KubeClient: cs, ServiceManager: serviceManager},
		taskDB:        NewDBService(),
		taskQueue:     NewTaskQueue(),
		quotaInfos:    sync.Map{},
		jobStatusChan: statusChan,
		// taskUpdateChan: taskChan,
	}
}

func (c *TaskController) SetProfiler(srcProfiler *Profiler) {
	c.profiler = srcProfiler
}

// Init init taskQueue And quotaInfos
func (c *TaskController) Init() error {
	// init quotas
	q := query.Account
	quotas, err := q.WithContext(context.Background()).Find()
	if err != nil {
		klog.Errorf("list all quotas failed, err: %v", err)
	}
	for i := range quotas {
		// 添加quota
		c.AddOrUpdateQuotaInfo(quotas[i])
		queueingList, err := c.taskDB.ListByUserAndStatuses(quotas[i].Name, model.EmiasTaskQueueingStatuses)
		if err != nil {
			klog.Errorf("list user:%v queueing tasks failed, err: %v", quotas[i].Name, err)
			continue
		}
		c.taskQueue.InitUserQueue(quotas[i].Name, queueingList)
	}
	return nil
}

// Start method
func (c *TaskController) Start(ctx context.Context) error {
	// init share dirs
	// err := c.jobControl.InitShareDir()
	// if err != nil {
	// 	klog.Errorf("get share dirs failed, err: %v", err)
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
			task, err := c.updateTaskStatus(ctx, status.TaskID, status.NewStatus, status.Reason)
			if err != nil {
				klog.Infof("get job status event, taskID: %v, newStatus: %v", status.TaskID, status.NewStatus)
				continue
			}
			// 更新quota，减去已经完成的作业的资源
			if util.IsCompletedStatus(status.NewStatus) {
				quotaInfo := c.GetQuotaInfo(task.UserName)
				if quotaInfo != nil {
					quotaInfo.DeleteTask(task)
				}
			} else if status.NewStatus == model.EmiasTaskQueueingStatus { // todo: fixme??
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
	task, err := c.taskDB.GetByID(event.TaskID)
	if err != nil {
		klog.Errorf("get task update event failed, err: %v", err)
		return
	}
	tidStr := strconv.FormatUint(uint64(event.TaskID), 10)
	klog.Infof("get task update event, taskID: %v, operation: %v", tidStr, event.Operation)
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
			klog.Errorf("delete job from task failed, err: %v", err)
		}
		// delete from profiler
		if c.profiler != nil && (task.ProfileStatus == model.EmiasProfileQueued || task.ProfileStatus == model.EmiasProfiling) {
			c.profiler.DeleteProfilePodFromTask(task.ID)
		}
		klog.Infof("delete task in task controller, %d", event.TaskID)
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
// 			klog.Infof("get task update event, taskID: %v, operation: %v", tidStr, t.Operation)
// 			// 1. delete的情况
// 			if t.Operation == util.DeleteTask {
// 				c.taskQueue.DeleteTaskByUserNameAndTaskID(t.UserName, tidStr)
// 				// delete in cluster
// 				err = c.jobControl.DeleteJobFromTask(task)
// 				if err != nil {
// 					klog.Errorf("delete job from task failed, err: %v", err)
// 				}
// 				klog.Infof("delete task in task controller, %d", t.TaskID)
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

func (c *TaskController) updateTaskStatus(ctx context.Context, taskID, status, reason string) (*model.AITask, error) {
	// convert taskID to uint
	tid, _ := strconv.ParseUint(taskID, 10, 64)

	t, err := c.taskDB.GetByID(uint(tid))
	if err != nil {
		return nil, err
	}
	if t.Status != status {
		err = c.taskDB.UpdateStatus(uint(tid), status, reason)
		if err != nil {
			return nil, err
		}
	}
	if t.Node == "" && status == string(aijobapi.Running) {
		nodeName, err := c.jobControl.GetNodeNameFromTask(ctx, t)
		if err != nil {
			return t, nil
		}
		t.Node = nodeName
		err = c.taskDB.UpdateNodeName(t.ID, nodeName)
		if err != nil {
			return nil, err
		}
	}
	t.Status = status
	return t, nil
}

// schedule 简单功能：从用户队列中
//
//nolint:gocyclo // TODO: figure out a better way to handle this
func (c *TaskController) schedule(_ context.Context) {
	// 等待调度的队列
	candiates := make([]*model.AITask, 0)

	// 1. guaranteed job schedule
	for username, q := range c.taskQueue.userQueues {
		// 1. 复制一份quota
		// klog.Infof(username, q.gauranteedQueue)
		quotaCopy := c.GetQuotaInfoSnapshotByUsername(username)
		if quotaCopy == nil {
			klog.Errorf("quota not found, username: %v", username)
			continue
		}
		// 2. 从gauranteedQueue队列选出不超过quota的作业
		for _, t := range q.gauranteedQueue.List() {
			task := t.(*model.AITask)

			if !quotaCopy.CheckHardQuotaExceed(task) {
				candiates = append(candiates, task)
				quotaCopy.AddTask(task)
				klog.Infof("task quota check succeed,user %v task %v taskid %v", task.UserName, task.TaskName, task.ID)
			} else {
				klog.Infof("task quota exceed, user %v task %v taskid %v, request:%v, used:%v, hard:%v",
					task.UserName, task.TaskName, task.ID, task.ResourceRequest, quotaCopy.HardUsed, quotaCopy.Hard)
				// bug: 如果先检查资源多的，可能后面的都调度不了？？
			}
		}
	}

	// 2. best effort queue的作业调度
	for _, q := range c.taskQueue.userQueues {
		for _, t := range q.bestEffortQueue.List() {
			task := t.(*model.AITask)
			// klog.Infof("user:%v, task: %v, task status:%v, profile status: %v", task.UserName, task.ID, task.Status, task.ProfileStatus)
			// update profile status
			if c.profiler != nil {
				// todo: udpate profile status???
				if task.Status == model.EmiasTaskQueueingStatus && task.ProfileStatus == model.EmiasUnProfiled {
					c.profiler.SubmitProfileTask(task.ID)
					klog.Infof("submit profile task, user:%v taskID:%v, taskName:%v", task.UserName, task.ID, task.TaskName)
					task.ProfileStatus = model.EmiasProfileQueued
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
		task, err := c.taskDB.GetByID(candidate.ID)
		if err != nil {
			klog.Errorf("get task from db failed, err: %v", err)
			continue
		}
		// check profiling status
		if task.ProfileStatus == model.EmiasProfiling || task.ProfileStatus == model.EmiasProfileQueued {
			continue
		} else if task.ProfileStatus == model.EmiasProfileFailed {
			//nolint:gocritic // TODO: figure out a better way to handle this
			// c.taskDB.UpdateStatus(task.ID, model.EmiasTaskFailedStatus, "task profile failed")
			c.taskQueue.DeleteTask(task)
			continue
		}
		// submit AIJob
		err = c.admitTask(task)
		if err != nil {
			klog.Errorf("create job from task: %v", err)
		} else {
			klog.Infof("create job from task success, taskID: %v", task.ID)
		}
	}
}

// admitTask 创建对应的aijob到集群中，更新task状态，更新quota
func (c *TaskController) admitTask(task *model.AITask) error {
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
		err = c.taskDB.UpdateStatus(task.ID, model.EmiasTaskFailedStatus, err.Error())
		if err != nil {
			return err
		}
		c.taskQueue.DeleteTask(task)
		return err
	}
	// 更新task状态
	err = c.taskDB.UpdateStatus(task.ID, model.EmiasTaskCreatedStatus, "AIJob created")
	if err != nil {
		return err
	}
	// 更新jobname
	err = c.taskDB.UpdateJobName(task.ID, jobname)
	if err != nil {
		return err
	}

	quotaInfo.AddTask(task)
	c.taskQueue.DeleteTask(task)
	return nil
}
