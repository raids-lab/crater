package taskqueue

import (
	"strconv"

	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/util/queue"
)

// 每个用户自己的作业队列排序
type queueBase struct {
	queue.Queue // lessFunc func(a, b interface{}) bool
}

type userQueue struct {
	username        string
	gauranteedQueue *queueBase
	bestEffortQueue *queueBase
}

// NewUserQueue: 新建一个空的用户队列
func NewUserQueue(username string) *userQueue {
	return &userQueue{
		username: username,
		gauranteedQueue: &queueBase{
			queue.New(keyFunc, fifoOrdering),
		},
		bestEffortQueue: &queueBase{
			queue.New(keyFunc, fifoOrdering),
		},
	}
}

// 默认key函数是taskID
func keyFunc(obj interface{}) string {
	t := obj.(*models.TaskAttr)
	return strconv.FormatUint(uint64(t.ID), 10)
}

// 按照提交时间顺序排队
func fifoOrdering(a, b interface{}) bool {
	tA := a.(*models.TaskAttr)
	tB := b.(*models.TaskAttr)
	return tA.CreatedAt.After(tB.CreatedAt)
	// return !tB.CreatedAt.Before(tA.CreatedAt)
}
