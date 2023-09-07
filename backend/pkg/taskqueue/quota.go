package taskqueue

import (
	"sync"

	"github.com/aisystem/ai-protal/pkg/models"
	v1 "k8s.io/api/core/v1"
)

type QuotaInfo struct {
	sync.RWMutex
	Name      string
	Hard      v1.ResourceList
	Soft      v1.ResourceList
	HardUsed  v1.ResourceList
	SoftUsed  v1.ResourceList
	UsedTasks map[string]*models.TaskAttr
}

// AddTask adds Running Job Quota
func (info *QuotaInfo) AddTask(task *models.TaskAttr) {
	info.Lock()
	defer info.Unlock()
	key := keyFunc(task)
	// 没有找到task的时候才添加quota
	if task, ok := info.UsedTasks[key]; !ok {
		info.UsedTasks[key] = task
		if task.SLO == models.HighSLO {
			AddResourceList(info.HardUsed, task.ResourceRequest)
		} else if task.SLO == models.LowSLO {
			AddResourceList(info.SoftUsed, task.ResourceRequest)
		}
	}
}

// DeleteTask deletes Completed or Deleted Job Quota
func (info *QuotaInfo) DeleteTask(task *models.TaskAttr) {
	info.Lock()
	defer info.Unlock()
	key := keyFunc(task)
	// 找到quotainfo里的task时才删除quota
	if task, ok := info.UsedTasks[key]; ok {
		delete(info.UsedTasks, key)
		if task.SLO == models.HighSLO {
			SubResourceList(info.HardUsed, task.ResourceRequest)
		} else if task.SLO == models.LowSLO {
			SubResourceList(info.SoftUsed, task.ResourceRequest)
		}
	}
}

// CheckHardQuotaExceed 判断作业的hard quota是否超出限制
func (info *QuotaInfo) CheckHardQuotaExceed(rl v1.ResourceList) bool {
	info.RLock()
	defer info.RUnlock()
	return CheckResourceListExceed(info.Hard, info.HardUsed, rl)
}

func (info *QuotaInfo) Snapshot() *QuotaInfo {
	info.RLock()
	defer info.RUnlock()
	return &QuotaInfo{
		Name:     info.Name,
		Hard:     info.Hard.DeepCopy(),
		Soft:     info.Soft.DeepCopy(),
		HardUsed: info.HardUsed.DeepCopy(),
		SoftUsed: info.SoftUsed.DeepCopy(),
	}
}

func (c *TaskController) GetQuotaInfo(username string) *QuotaInfo {
	if value, ok := c.quotaInfos.Load(username); ok {
		return value.(*QuotaInfo)
	} else {
		quotadb, err := c.quotaDB.GetByUserName(username)
		if err != nil {
			// todo: handler err
			return nil
		}
		_, info := c.AddOrUpdateQuotaInfo(username, *quotadb)
		return info
	}
}

// GetQuotaInfoSnapshotByUsername 获取某个用户的QuotaInfo的clone，对quota的增加减少不改变原数据
func (c *TaskController) GetQuotaInfoSnapshotByUsername(username string) *QuotaInfo {
	if value, ok := c.quotaInfos.Load(username); ok {
		return value.(*QuotaInfo).Snapshot()
	} else {
		return nil
	}
}

func (c *TaskController) AddOrUpdateQuotaInfo(name string, quota models.Quota) (added bool, quotaInfo *QuotaInfo) {

	hardQuota, _ := models.JSONToResourceList(quota.HardQuota)
	if _, ok := c.quotaInfos.Load(name); !ok {
		quotaInfo := &QuotaInfo{
			Name: quota.UserName,
			Hard: hardQuota,
			// Soft:      dq.Spec.Soft.DeepCopy(),
			HardUsed:  v1.ResourceList{},
			SoftUsed:  v1.ResourceList{},
			UsedTasks: make(map[string]*models.TaskAttr),
		}

		// todo: add db tasks
		tasksRunning, err := c.taskDB.ListByUserAndStatus(name, models.RunningStatus)
		if err != nil {
			// todo: handler err
		}
		for _, task := range tasksRunning {
			quotaInfo.AddTask(models.FormatTaskModelToAttr(&task))
		}

		// add db tasks
		tasksPending, err := c.taskDB.ListByUserAndStatus(name, models.PendingStatus)
		if err != nil {
			// todo: handler err
		}
		for _, task := range tasksPending {
			quotaInfo.AddTask(models.FormatTaskModelToAttr(&task))
		}

		c.quotaInfos.Store(name, quotaInfo)
		added = true
	} else {
		c.UpdateQuotaInfoHard(name, hardQuota)
		added = false
	}
	return
}

// UpdateQuotaInfoHard updates QuotaInfo Hard
func (c *TaskController) UpdateQuotaInfoHard(username string, hard v1.ResourceList) {
	if value, ok := c.quotaInfos.Load(username); ok {
		info := value.(*QuotaInfo)
		info.Lock()
		defer info.Unlock()
		if !CmpResourceListSame(info.Hard, hard) {
			info.Hard = hard.DeepCopy()
			// info.Soft = dq.Spec.Soft.DeepCopy()
		}
	}
}

// DeleteQuotaInfo deletes QuotaInfo
func (c *TaskController) DeleteQuotaInfo(namespace string) {
	c.quotaInfos.Delete(namespace)
}
