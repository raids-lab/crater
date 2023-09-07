package taskqueue

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/aisystem/ai-protal/pkg/crclient"
	quotadb "github.com/aisystem/ai-protal/pkg/db/quota"
	taskdb "github.com/aisystem/ai-protal/pkg/db/task"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/util"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
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
}

// NewTaskController returns a new *TaskController
func NewTaskController(client client.Client, statusChan <-chan util.JobStatusChan, taskChan <-chan util.TaskUpdateChan) *TaskController {
	return &TaskController{
		jobControl:     &crclient.JobControl{Client: client},
		quotaDB:        quotadb.NewDBService(),
		taskDB:         taskdb.NewDBService(),
		taskQueue:      NewTaskQueue(),
		quotaInfos:     sync.Map{},
		jobStatusChan:  statusChan,
		taskUpdateChan: taskChan,
	}
}

// Start method
func (c *TaskController) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("task-controller")
	ctx = ctrl.LoggerInto(ctx, log)
	// 1. init 初始化task队列和quota信息，存在缓存里
	// c.Init()
	// 2. 接受job状态变更信息
	go c.WatchJobStatus(ctx)
	// 3. 接收task变更信息
	go c.WatchTaskUpdate(ctx)
	// 4. schedule线程
	go wait.UntilWithContext(ctx, c.schedule, 0)
	return nil
}

// Init 初始化队列信息, quota 信息
func (c *TaskController) Init() error {
	quotas, err := c.quotaDB.ListAllQuotas()
	if err != nil {
		// todo:
	}

	for _, quota := range quotas {
		// 添加quota
		c.AddOrUpdateQuotaInfo(quota.UserName, quota)
		queueingList, err := c.taskDB.ListByUserAndStatus(quota.UserName, models.QueueingStatus)
		if err != nil {
			// todo:
		}
		c.taskQueue.InitUserQueue(quota.UserName, queueingList)
	}
	return nil
}

// WatchJobStatus
func (c *TaskController) WatchJobStatus(ctx context.Context) {
	for {
		select {
		case status := <-c.jobStatusChan:
			// 更新task在db中的状态
			task, err := c.updateTaskStatus(status.TaskID, status.NewStatus)
			if err != nil {
				// todo:
			}
			// 更新quota，减去已经完成的作业的资源
			if util.IsCompletedStatus(status.NewStatus) {
				quotaInfo := c.GetQuotaInfo(task.UserName)
				if quotaInfo != nil {
					quotaInfo.DeleteTask(task)
				}
			} else if status.NewStatus == models.QueueingStatus {
				// 更新task队列
				c.taskQueue.AddTask(task)
			}
		case <-ctx.Done():
			return
		}
	}
}

// WatchTaskUpdate
func (c *TaskController) WatchTaskUpdate(ctx context.Context) {
	for {
		select {
		case t := <-c.taskUpdateChan:
			// 更新task在队列的状态
			task, err := c.taskDB.GetByID(t.TaskID)
			tidStr := strconv.FormatUint(uint64(t.TaskID), 10)
			// 1. delete的情况
			if err != nil && t.Operation == util.DeleteTask {
				c.taskQueue.DeleteTaskByUserNameAndTaskID(t.UserName, tidStr)
				continue
			} else if t.Operation == util.CreateTask {
				// 2. create
				c.taskQueue.AddTask(models.FormatTaskModelToAttr(task))
			} else if t.Operation == util.UpdateTask {
				// 3. update slo
				c.taskQueue.UpdateTask(models.FormatTaskModelToAttr(task))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *TaskController) updateTaskStatus(taskID string, status string) (*models.TaskAttr, error) {
	//convert taskID to uint
	tid, _ := strconv.ParseUint(taskID, 10, 64)

	t, err := c.taskDB.GetByID(uint(tid))
	if err != nil {
		return nil, err
	}
	if t.Status != status {
		err = c.taskDB.UpdateStatus(uint(tid), status)
		if err != nil {
			return nil, err
		}
	}
	t.Status = status
	return models.FormatTaskModelToAttr(t), nil
}

// Schedule
// 简单功能：从用户队列中
func (c *TaskController) schedule(ctx context.Context) {
	// log := ctrl.LoggerFrom(ctx)
	// 等待调度的队列
	candiates := make([]*models.TaskAttr, 0)

	for username, q := range c.taskQueue.userQueues {
		// 1. 复制一份quota
		quotaCopy := c.GetQuotaInfoSnapshotByUsername(username)
		if quotaCopy == nil {
			// todo: err
			continue
		}
		// 2. 从gauranteedQueue队列选出不超过quota的作业
		for _, t := range q.gauranteedQueue.List() {
			task := t.(*models.TaskAttr)
			if !quotaCopy.CheckHardQuotaExceed(task.ResourceRequest) {
				candiates = append(candiates, task)
				quotaCopy.AddTask(task)
			} else {
				break
			}
		}
	}

	// todo: best effort queue的作业调度

	// todo: candidates队列的调度策略
	for _, task := range candiates {
		err := c.admitTask(task)
		if err != nil {
			// todo:
		}

	}

}

func (c *TaskController) admitTask(task *models.TaskAttr) error {
	//
	err := c.jobControl.CreateJobFromTask(task)
	if err != nil {
		// todo:
		return err
	}
	// 更新task状态
	if err = c.taskDB.UpdateStatus(task.ID, models.PendingStatus); err != nil {
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
