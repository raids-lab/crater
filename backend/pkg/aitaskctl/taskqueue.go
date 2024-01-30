package aitaskctl

import (
	"sync"

	"github.com/aisystem/ai-protal/pkg/models"
)

// TaskQueue 保存每个用户的队列
type TaskQueue struct {
	sync.RWMutex

	userQueues map[string]*userQueue
}

// NewTaskQueue 返回一个空的TaskQueue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		userQueues: make(map[string]*userQueue),
	}
}

// InitUserQueue 有新quota/user添加的时候，初始化，传入的是用户的taskList
func (tq *TaskQueue) InitUserQueue(username string, taskList []models.AITask) {
	tq.Lock()
	defer tq.Unlock()
	q := NewUserQueue(username)
	for _, task := range taskList {
		if task.SLO == models.HighSLO {
			q.gauranteedQueue.PushIfNotPresent(&task)
		} else if task.SLO == models.LowSLO {
			q.bestEffortQueue.PushIfNotPresent(&task)
		}
	}
	tq.userQueues[username] = q
}

// AddTask 有新的task提交的时候，添加到队列中
func (tq *TaskQueue) AddTask(task *models.AITask) {
	tq.Lock()
	defer tq.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	if task.SLO == models.HighSLO {
		q.gauranteedQueue.PushOrUpdate(task)
	} else if task.SLO == models.LowSLO {
		q.bestEffortQueue.PushOrUpdate(task)
	}
}

// DeleteTask deletes task in queue when task is scheduled
func (tq *TaskQueue) DeleteTask(task *models.AITask) {
	tq.Lock()
	defer tq.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	if task.SLO == models.HighSLO {
		q.gauranteedQueue.Delete(task)
	} else if task.SLO == models.LowSLO {
		q.bestEffortQueue.Delete(task)
	}
}

// DeleteTaskByUserNameAndTaskID deletes task that is deleted
func (tq *TaskQueue) DeleteTaskByUserNameAndTaskID(username string, taskid string) {
	tq.Lock()
	defer tq.Unlock()
	q, ok := tq.userQueues[username]
	if !ok {
		return
	}
	q.gauranteedQueue.DeleteByKey(taskid)
	q.bestEffortQueue.DeleteByKey(taskid)
}

// UpdateTask updates task
func (tq *TaskQueue) UpdateTask(task *models.AITask) {
	tq.Lock()
	defer tq.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	if task.SLO == models.HighSLO {
		q.gauranteedQueue.PushOrUpdate(task)
		q.bestEffortQueue.Delete(task)
	} else if task.SLO == models.LowSLO {
		q.bestEffortQueue.PushOrUpdate(task)
		q.gauranteedQueue.Delete(task)
	}
}
