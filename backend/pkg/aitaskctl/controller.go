package aitaskctl

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aisystem/ai-protal/pkg/crclient"
	quotadb "github.com/aisystem/ai-protal/pkg/db/quota"
	taskdb "github.com/aisystem/ai-protal/pkg/db/task"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/profiler"
	"github.com/aisystem/ai-protal/pkg/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TaskController 调度task
type TaskController struct {
	jobControl     *crclient.JobControl      // 创建AIJob的接口
	quotaDB        quotadb.DBService         // 获取db中的quota
	taskDB         taskdb.DBService          // 获取db中的task
	taskQueue      *TaskQueue                // 队列信息
	quotaInfos     sync.Map                  // quota缓存
	jobStatusChan  <-chan util.JobStatusChan // 获取job的状态变更信息，同步到数据库
	taskUpdateChan <-chan util.TaskUpdateChan
	profiler       *profiler.Profiler
}

// NewTaskController returns a new *TaskController
func NewTaskController(client client.Client, statusChan <-chan util.JobStatusChan) *TaskController {
	return &TaskController{
		jobControl:    &crclient.JobControl{Client: client},
		quotaDB:       quotadb.NewDBService(),
		taskDB:        taskdb.NewDBService(),
		taskQueue:     NewTaskQueue(),
		quotaInfos:    sync.Map{},
		jobStatusChan: statusChan,
		// taskUpdateChan: taskChan,
	}
}

func (c *TaskController) SetProfiler(profiler *profiler.Profiler) {
	c.profiler = profiler
}

// Init init taskQueue And quotaInfos
func (c *TaskController) Init() error {

	// init quotas
	quotas, err := c.quotaDB.ListAllQuotas()
	// logrus.Info(quotas)
	if err != nil {
		logrus.Errorf("list all quotas failed, err: %v", err)
	}
	logrus.Infof("list all quotas success, len: %v", len(quotas))
	for _, quota := range quotas {
		// 添加quota
		c.AddOrUpdateQuotaInfo(quota.UserName, quota)
		queueingList, err := c.taskDB.ListByUserAndStatuses(quota.UserName, models.TaskQueueingStatuses)
		if err != nil {
			logrus.Errorf("list user:%v queueing tasks failed, err: %v", quota.UserName, err)
			continue
		}
		c.taskQueue.InitUserQueue(quota.UserName, queueingList)
	}
	return nil
}

// Start method
func (c *TaskController) Start(ctx context.Context) error {
	// init share dirs
	// err := c.jobControl.InitShareDir()
	// if err != nil {
	// 	logrus.Errorf("get share dirs failed, err: %v", err)
	// 	return err
	// }
	// 1. init 初始化task队列和quota信息，存在缓存里
	// c.Init()
	// 2. 接受AIJOB的crd状态变更信息，通过jobStatusChan
	go c.watchJobStatus(ctx)
	// 3. 接收task变更信息
	// go c.watchTaskUpdate(ctx)
	// 4. schedule线程
	go wait.UntilWithContext(ctx, c.schedule, time.Second*5)
	return nil
}

// Init 初始化队列信息, quota 信息

// WatchJobStatus, 根据AIJob的状态更新task的状态
func (c *TaskController) watchJobStatus(ctx context.Context) {
	for {
		select {
		case status := <-c.jobStatusChan:
			// logrus.Infof("get job status event, taskID: %v, newStatus: %v", status.TaskID, status.NewStatus)
			// 更新task在db中的状态
			task, err := c.updateTaskStatus(status.TaskID, status.NewStatus, status.Reason)
			if err != nil {
				logrus.Errorf("update task status failed, err: %v", err)
				continue
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
	task, err := c.taskDB.GetByID(event.TaskID)
	if err != nil {
		logrus.Errorf("get task update event failed, err: %v", err)
		return
	}
	tidStr := strconv.FormatUint(uint64(event.TaskID), 10)
	logrus.Infof("get task update event, taskID: %v, operation: %v", tidStr, event.Operation)
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
			logrus.Errorf("delete job from task failed, err: %v", err)
		}
		// delete from profiler
		if c.profiler != nil && task.ProfileStatus == models.Profiling {
			c.profiler.DeleteProfilePodFromTask(task.ID)
		}
		logrus.Infof("delete task in task controller, %d", event.TaskID)
	}
}

// deprecated
func (c *TaskController) watchTaskUpdate(ctx context.Context) {
	for {
		select {
		case t := <-c.taskUpdateChan:
			// 更新task在队列的状态
			task, err := c.taskDB.GetByID(t.TaskID)
			tidStr := strconv.FormatUint(uint64(t.TaskID), 10)
			logrus.Infof("get task update event, taskID: %v, operation: %v", tidStr, t.Operation)
			// 1. delete的情况
			if t.Operation == util.DeleteTask {
				c.taskQueue.DeleteTaskByUserNameAndTaskID(t.UserName, tidStr)
				// delete in cluster
				err = c.jobControl.DeleteJobFromTask(task)
				if err != nil {
					logrus.Errorf("delete job from task failed, err: %v", err)
				}
				logrus.Infof("delete task in task controller, %d", t.TaskID)
				continue
			} else if t.Operation == util.CreateTask {
				// 2. create
				c.taskQueue.AddTask(task)
			} else if t.Operation == util.UpdateTask {
				// 3. update slo
				c.taskQueue.UpdateTask(task)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *TaskController) updateTaskStatus(taskID string, status string, reason string) (*models.AITask, error) {
	//convert taskID to uint
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
	t.Status = status
	return t, nil
}

// schedule 简单功能：从用户队列中
func (c *TaskController) schedule(ctx context.Context) {
	// log := ctrl.LoggerFrom(ctx)
	// 等待调度的队列
	candiates := make([]*models.AITask, 0)

	// 1. gauranteed job schedule
	for username, q := range c.taskQueue.userQueues {
		// 1. 复制一份quota
		// logrus.Infof(username, q.gauranteedQueue)
		quotaCopy := c.GetQuotaInfoSnapshotByUsername(username)
		if quotaCopy == nil {
			logrus.Errorf("quota not found, username: %v", username)
			continue
		}
		// 2. 从gauranteedQueue队列选出不超过quota的作业
		for _, t := range q.gauranteedQueue.List() {
			task := t.(*models.AITask)

			resourcelist, _ := models.JSONToResourceList(task.ResourceRequest)
			if !quotaCopy.CheckHardQuotaExceed(resourcelist) {
				candiates = append(candiates, task)
				quotaCopy.AddTask(task)
				// logrus.Infof("task quota check succeed, %v/%v", task.UserName, task.TaskName)
			} else {
				// logrus.Infof("task quota exceed, %v/%v, request:%v, used:%v, hard:%v", task.UserName, task.TaskName, task.ResourceRequest, quotaCopy.HardUsed, quotaCopy.Hard)
				// break // bug: 如果先检查资源多的，可能后面的都调度不了？？
			}
		}
	}

	// 2. best effort queue的作业调度
	for _, q := range c.taskQueue.userQueues {

		for _, t := range q.bestEffortQueue.List() {
			task := t.(*models.AITask)
			// update profile status
			if c.profiler != nil {
				// todo: udpate profile status???
				if task.Status == models.TaskQueueingStatus && task.ProfileStatus == models.UnProfiled {
					c.profiler.SubmitProfileTask(task.ID)
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
	for _, candidate := range candiates {
		task, err := c.taskDB.GetByID(candidate.ID)
		if err != nil {
			logrus.Errorf("get task from db failed, err: %v", err)
			continue
		}
		// check profiling status
		if task.ProfileStatus == models.Profiling {
			continue
		} else if task.ProfileStatus == models.ProfileFailed {
			c.taskDB.UpdateStatus(task.ID, models.TaskFailedStatus, "task profile failed")
			c.taskQueue.DeleteTask(task)
			continue
		}
		// submit AIJob
		err = c.admitTask(task)
		if err != nil {
			logrus.Errorf("create job from task failed, err: %v", err)
		} else {
			logrus.Infof("create job from task success, taskID: %v", task.ID)
		}

	}

}

// admitTask 创建对应的aijob到集群中，更新task状态，更新quota
func (c *TaskController) admitTask(task *models.AITask) error {

	err := c.jobControl.CreateJobFromTask(task)
	if err != nil {
		c.taskDB.UpdateStatus(task.ID, models.TaskFailedStatus, err.Error())
		c.taskQueue.DeleteTask(task)
		return err
	}
	// 更新task状态
	if err = c.taskDB.UpdateStatus(task.ID, models.TaskCreatedStatus, "AIJob created"); err != nil {
		return err
	}
	// 更新quota
	quotaInfo := c.GetQuotaInfo(task.UserName)
	if quotaInfo == nil {
		// todo:
		return fmt.Errorf("quota not found")
	}
	quotaInfo.AddTask(task)
	c.taskQueue.DeleteTask(task)
	return nil
}
