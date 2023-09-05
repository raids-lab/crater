package taskqueue

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
func (tq *TaskQueue) InitUserQueue(username string, taskList []models.TaskAttr) {
	tq.Lock()
	defer tq.Unlock()
	q := NewUserQueue(username)
	for _, task := range taskList {
		if task.SLO == models.HighSLO {
			q.gauranteedQueue.PushIfNotPresent(task)
		} else if task.SLO == models.LowSLO {
			q.bestEffortQueue.PushIfNotPresent(task)
		}
	}
	tq.userQueues[username] = q

}

// AddTask 有新的task提交的时候，添加到队列中
func (tq *TaskQueue) AddTask(task *models.TaskAttr) {
	tq.Lock()
	defer tq.Unlock()
	q, ok := tq.userQueues[task.UserName]
	if ok {
		return
	}
	if task.SLO == models.HighSLO {
		q.gauranteedQueue.PushIfNotPresent(task)
	} else if task.SLO == models.LowSLO {
		q.bestEffortQueue.PushIfNotPresent(task)
	}
}

func (tq *TaskQueue) DeleteTask(task *models.TaskAttr) {
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
