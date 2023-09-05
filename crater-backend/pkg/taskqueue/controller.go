package taskqueue

import (
	"fmt"
	"sync"

	quotadb "github.com/aisystem/ai-protal/pkg/db/quota"
	taskdb "github.com/aisystem/ai-protal/pkg/db/task"
	"github.com/aisystem/ai-protal/pkg/models"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
type TaskController struct {
	jobControl *control.JobControl
	quotaDB    quotadb.DBService
	taskDB     taskdb.DBService
	taskQueue  *TaskQueue
	quotaInfos sync.Map
}

// todo:
func NewTaskController(client client.Client) *TaskController {
	return &TaskController{
		jobControl: &control.JobControl{Client: client},
		quotaDB:    quotadb.NewDBService(),
		taskDB:     taskdb.NewDBService(),
		taskQueue:  NewTaskQueue(),
		quotaInfos: sync.Map{},
	}
}

// Init 初始化队列信息
func (c *TaskController) Init() {
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
}

// Schedule
// 简单功能：从用户队列中
func (c *TaskController) schedule() {
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
		if err != nil{
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

