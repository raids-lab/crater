package profiler

import (
	"strconv"

	"github.com/raids-lab/crater/pkg/models"
)

// 默认key函数是taskID
func keyFunc(obj any) string {
	t := obj.(*models.AITask)
	return strconv.FormatUint(uint64(t.ID), 10)
}

// 按照提交时间顺序排队
// todo: 可能需要按照任务的大小？
func fifoOrdering(a, b any) bool {
	tA := a.(*models.AITask)
	tB := b.(*models.AITask)
	return tA.CreatedAt.After(tB.CreatedAt)
}
