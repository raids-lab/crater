package quota

import (
	"fmt"
	"sync"

	aijobapi "github.com/aisys/ai-task-controller/pkg/apis/aijob/v1alpha1"
	constants "github.com/aisys/ai-task-controller/pkg/constants"
	"github.com/aisys/ai-task-controller/pkg/models"
	v1 "k8s.io/api/core/v1"
)

var QuotaInfosData = &QuotaInfosMap{}

type QuotaInfosMap struct {
	m sync.Map
}

type QuotaInfo struct {
	sync.RWMutex
	Name      string
	Namespace string
	Hard      v1.ResourceList
	Soft      v1.ResourceList
	HardUsed  v1.ResourceList
	SoftUsed  v1.ResourceList
	UsedJobs  map[string]*aijobapi.AIJob
}

func QuotaNamespaceFunc(userName string) string {
	return fmt.Sprintf("user-%s", userName)
}

func UserNameFromNamespaceFunc(namespace string) string {
	return namespace[len("user-"):]
}

func (m *QuotaInfosMap) AddOrUpdateQuotaInfo(name string, quota models.Quota) (added bool, quotaInfo *QuotaInfo) {
	namespace := QuotaNamespaceFunc(quota.UserName)
	// key, _ := cache.MetaNamespaceKeyFunc(dq)
	hardQuota, _ := models.JSONToResourceList(quota.HardQuota)
	if _, ok := m.m.Load(namespace); !ok {
		quotaInfo := &QuotaInfo{
			Name:      quota.UserName,
			Namespace: namespace,
			Hard:      hardQuota,
			// Soft:      dq.Spec.Soft.DeepCopy(),
			HardUsed: v1.ResourceList{},
			SoftUsed: v1.ResourceList{},
			UsedJobs: map[string]*aijobapi.AIJob{},
		}
		m.m.Store(namespace, quotaInfo)
		added = true
	} else {
		m.UpdateQuotaInfoHard(namespace, hardQuota)
		added = false
	}
	return
}

// UpdateQuotaInfoHard updates QuotaInfo Hard
func (m *QuotaInfosMap) UpdateQuotaInfoHard(namespace string, hard v1.ResourceList) {
	// key := dq.Namespace + "/" + dq.Name
	// key, _ := cache.MetaNamespaceKeyFunc(dq)
	if value, ok := m.m.Load(namespace); ok {
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
func (m *QuotaInfosMap) DeleteQuotaInfo(namespace string) {
	m.m.Delete(namespace)
}

// AddJob adds Running Job Quota
func (info *QuotaInfo) AddJob(job *aijobapi.AIJob) {
	info.Lock()
	defer info.Unlock()
	key := job.Namespace + "/" + job.Name
	if job, ok := info.UsedJobs[key]; !ok {
		info.UsedJobs[key] = job
		if job.Labels[constants.TaskSLOLabelKey] == constants.TaskHighSLOLabelValue {
			AddResourceList(info.HardUsed, job.Spec.ResourceRequest)
		} else if job.Labels[constants.TaskSLOLabelKey] == constants.TaskLowSLOLabelValue {
			AddResourceList(info.SoftUsed, job.Spec.ResourceRequest)
		}
	}
}

// DeleteJob deletes Completed or Deleted Job Quota
func (info *QuotaInfo) DeleteJob(job *aijobapi.AIJob) {
	info.Lock()
	defer info.Unlock()
	key := job.Namespace + "/" + job.Name
	if job, ok := info.UsedJobs[key]; ok {
		delete(info.UsedJobs, key)
		if job.Labels[constants.TaskSLOLabelKey] == constants.TaskHighSLOLabelValue {
			SubResourceList(info.HardUsed, job.Spec.ResourceRequest)
		} else if job.Labels[constants.TaskSLOLabelKey] == constants.TaskLowSLOLabelValue {
			SubResourceList(info.SoftUsed, job.Spec.ResourceRequest)
		}
	}
}

// CheckJobQuotaExceed 判断作业的hard quota是否超出限制
func (info *QuotaInfo) CheckJobQuotaExceed(job *aijobapi.AIJob) bool {
	info.RLock()
	defer info.RUnlock()
	if job.Labels[constants.TaskSLOLabelKey] == constants.TaskHighSLOLabelValue {
		return CheckResourceListExceed(info.Hard, info.HardUsed, job.Spec.ResourceRequest)
	} else if job.Labels[constants.TaskSLOLabelKey] == constants.TaskLowSLOLabelValue {
		// return CheckResourceListExceed(info.Soft, info.SoftUsed, job.Spec.ResourceRequest)
		// todo:
	}
	return false
}

// GetQuotaInfo 通过namespace获取quota信息
func GetQuotaInfo(namespace string) *QuotaInfo {
	if value, ok := QuotaInfosData.m.Load(namespace); ok {
		return value.(*QuotaInfo)
	} else {
		// 从数据库拉数据
		username := UserNameFromNamespaceFunc(namespace)
		quotadb, err := quotaDB.GetByUserName(username)
		if err != nil {
			// todo: handler err
			return nil
		}
		_, info := QuotaInfosData.AddOrUpdateQuotaInfo(username, *quotadb)
		return info
	}
}
