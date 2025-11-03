package aitaskctl

import (
	"sync"

	"github.com/raids-lab/crater/dao/model"
)

// TaskQueue 保存每个用户的队列
type TaskQueue struct {
	mu sync.RWMutex

	userQueues map[string]*userQueue
}

// NewTaskQueue 返回一个空的TaskQueue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		userQueues: make(map[string]*userQueue),
	}
}

// InitUserQueue 有新quota/user添加的时候，初始化，传入的是用户的taskList
func (tq *TaskQueue) InitUserQueue(username string, taskList []*model.AITask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	q := NewUserQueue(username)
	for i := range taskList {
		task := taskList[i]
		switch task.SLO {
		case model.EmiasHighSLO:
			q.gauranteedQueue.PushIfNotPresent(task)
		case model.EmiasLowSLO:
			q.bestEffortQueue.PushIfNotPresent(task)
		}
	}
	tq.userQueues[username] = q
}

// AddTask 有新的task提交的时候，添加到队列中
func (tq *TaskQueue) AddTask(task *model.AITask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	switch task.SLO {
	case model.EmiasHighSLO:
		q.gauranteedQueue.PushOrUpdate(task)
	case model.EmiasLowSLO:
		q.bestEffortQueue.PushOrUpdate(task)
	}
}

// DeleteTask deletes task in queue when task is scheduled
func (tq *TaskQueue) DeleteTask(task *model.AITask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	switch task.SLO {
	case model.EmiasHighSLO:
		q.gauranteedQueue.Delete(task)
	case model.EmiasLowSLO:
		q.bestEffortQueue.Delete(task)
	}
}

// DeleteTaskByUserNameAndTaskID deletes task that is deleted
func (tq *TaskQueue) DeleteTaskByUserNameAndTaskID(username, taskid string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	q, ok := tq.userQueues[username]
	if !ok {
		return
	}
	q.gauranteedQueue.DeleteByKey(taskid)
	q.bestEffortQueue.DeleteByKey(taskid)
}

// UpdateTask updates task
func (tq *TaskQueue) UpdateTask(task *model.AITask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if !ok {
		return
	}
	switch task.SLO {
	case model.EmiasHighSLO:
		q.gauranteedQueue.PushOrUpdate(task)
		q.bestEffortQueue.Delete(task)
	case model.EmiasLowSLO:
		q.bestEffortQueue.PushOrUpdate(task)
		q.gauranteedQueue.Delete(task)
	}
}
