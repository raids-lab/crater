package controller

import (
	"sync"

	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"
	quotaapi "k8s.io/ai-task-controller/pkg/apis/tenantquota/v1alpha1"
	constants "k8s.io/ai-task-controller/pkg/constants"
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

func (m *QuotaInfosMap) AddOrUpdateQuotaInfo(key string, dq *quotaapi.TenantQuota) (added bool, quotaInfo *QuotaInfo) {
	// key := dq.Namespace + "/" + dq.Name
	// key, _ := cache.MetaNamespaceKeyFunc(dq)
	if _, ok := m.m.Load(key); !ok {
		quotaInfo := &QuotaInfo{
			Name:      dq.Name,
			Namespace: dq.Namespace,
			Hard:      dq.Spec.Hard.DeepCopy(),
			Soft:      dq.Spec.Soft.DeepCopy(),
			HardUsed:  v1.ResourceList{},
			SoftUsed:  v1.ResourceList{},
			UsedJobs:  map[string]*aijobapi.AIJob{},
		}
		m.m.Store(key, quotaInfo)
		added = true
	} else {
		m.UpdateQuotaInfo(key, dq)
		added = false
	}
	return
}

// UpdateQuotaInfo updates QuotaInfo Hard and Soft
func (m *QuotaInfosMap) UpdateQuotaInfo(key string, dq *quotaapi.TenantQuota) {
	// key := dq.Namespace + "/" + dq.Name
	// key, _ := cache.MetaNamespaceKeyFunc(dq)
	if value, ok := m.m.Load(key); ok {
		info := value.(*QuotaInfo)
		info.Lock()
		defer info.Unlock()
		if !CmpQuotaInfoAndDQSame(info, dq) {
			info.Hard = dq.Spec.Hard.DeepCopy()
			info.Soft = dq.Spec.Soft.DeepCopy()
		}
	}
}

// DeleteQuotaInfo deletes QuotaInfo
func (m *QuotaInfosMap) DeleteQuotaInfo(key string) {
	m.m.Delete(key)
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

// CmpQuotaInfoAndDQSame compares the hard and soft resource list of QuotaInfo and TenantQuota
func CmpQuotaInfoAndDQSame(info *QuotaInfo, dq *quotaapi.TenantQuota) bool {
	if !CmpResourceListSame(info.Hard, dq.Spec.Hard) {
		return false
	}
	if !CmpResourceListSame(info.Soft, dq.Spec.Soft) {
		return false
	}
	return true
}

func GetQuotaInfo(namespace string, name string) *QuotaInfo {
	key := namespace + "/" + name
	if value, ok := QuotaInfosData.m.Load(key); ok {
		return value.(*QuotaInfo)
	}
	return nil
}

func CheckJobQuotaExceed(info *QuotaInfo, job *aijobapi.AIJob) bool {
	info.RLock()
	defer info.RUnlock()
	if job.Labels[constants.TaskSLOLabelKey] == constants.TaskHighSLOLabelValue {
		return CheckResourceListExceed(info.Hard, info.HardUsed, job.Spec.ResourceRequest)
	} else if job.Labels[constants.TaskSLOLabelKey] == constants.TaskLowSLOLabelValue {
		return CheckResourceListExceed(info.Soft, info.SoftUsed, job.Spec.ResourceRequest)
	}
	return false
}


