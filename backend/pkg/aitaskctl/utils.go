package aitaskctl

import (
	"context"
	"fmt"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

// AddResourceList adds b into a
func AddResourceList(a, b v1.ResourceList) v1.ResourceList {
	for k, v := range b {
		v0 := v.DeepCopy()
		if aVal, ok := a[k]; ok {
			v0.Add(aVal)
		}
		a[k] = v0
	}
	return a
}

// SubResourceList subs b from a
func SubResourceList(a, b v1.ResourceList) v1.ResourceList {
	for k, v := range a {
		va := v.DeepCopy()
		if vb, ok := b[k]; ok {
			va.Sub(vb)
			a[k] = va
		}
	}
	return a
}

// CmpResourceListSame compares if a and b are the same
func CmpResourceListSame(a, b v1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func CheckResourceListExceed(hard, used, requested v1.ResourceList) bool {
	usedCopy := used.DeepCopy()
	AddResourceList(usedCopy, requested)
	for k, usedValue := range usedCopy {
		if hardValue, ok := hard[k]; !ok {
			return true
		} else if usedValue.Cmp(hardValue) > 0 {
			return true
		}
	}
	return false
}

func CheckJupyterLimitBeforeCreateJupyter(
	c context.Context,
	userID, accountID uint,
) error {
	uq := query.UserAccount
	var userQueueQuota datatypes.JSONType[model.QueueQuota]
	err := uq.WithContext(c).
		Where(uq.UserID.Eq(userID)).
		Where(uq.AccountID.Eq(accountID)).
		Select(uq.Quota).
		Scan(&userQueueQuota)
	if err != nil {
		return err
	}

	j := query.Job
	jupyterCount, err := j.WithContext(c).
		Where(j.UserID.Eq(userID)).
		Where(j.AccountID.Eq(accountID)).
		Where(j.JobType.Eq(string(model.JobTypeJupyter))).
		Where(j.Status.In("Running", "Pending")).
		Count()
	if err != nil {
		return err
	}

	maxJupyterCount := -1 // default -1 means no limit

	ac := query.Account
	var userDefaultQuota datatypes.JSONType[model.QueueQuota]
	err = ac.WithContext(c).
		Where(ac.ID.Eq(accountID)).
		Select(ac.UserDefaultQuota).
		Scan(&userDefaultQuota)
	if err != nil {
		return err
	}
	if v, ok := userDefaultQuota.Data().Capability[v1.ResourceName(model.JobTypeJupyter)]; ok {
		maxJupyterCount = int(v.Value())
	}

	uqQuota := userQueueQuota.Data().Capability
	if len(uqQuota) != 0 {
		if v, ok := uqQuota[v1.ResourceName(model.JobTypeJupyter)]; ok {
			maxJupyterCount = int(v.Value())
		}
	}

	if maxJupyterCount == -1 {
		// no limit
		return nil
	}

	if jupyterCount >= int64(maxJupyterCount) {
		return fmt.Errorf("jupyter 数量超过限制: %d", maxJupyterCount)
	}

	return nil
}

func CheckResourcesBeforeCreateJob(
	c context.Context,
	userID, accountID uint,
	createResources v1.ResourceList,
) (exceededResources []v1.ResourceName) {
	uq := query.UserAccount
	var userQueueQuota datatypes.JSONType[model.QueueQuota]
	err := uq.WithContext(c).
		Where(uq.UserID.Eq(userID)).
		Where(uq.AccountID.Eq(accountID)).
		Select(uq.Quota).
		Scan(&userQueueQuota)
	if err != nil {
		return exceededResources
	}

	j := query.Job
	jobResources, err := j.WithContext(c).
		Where(j.UserID.Eq(userID)).
		Where(j.AccountID.Eq(accountID)).
		Where(j.Status.In("Running", "Pending")).
		Select(j.Resources).
		Find()
	if err != nil {
		return exceededResources
	}

	const maxJobResources = 100
	if len(jobResources) >= maxJobResources {
		exceededResources = append(exceededResources, "作业数量超过限制")
		return exceededResources
	}

	uqQuota := userQueueQuota.Data().Capability
	if len(uqQuota) == 0 {
		return exceededResources
	}

	JobQuota := v1.ResourceList{}

	for _, jobResource := range jobResources {
		JobQuota = AddResourceList(JobQuota, jobResource.Resources.Data())
	}

	JobQuota = AddResourceList(JobQuota, createResources)

	for k, usedValue := range JobQuota {
		if hardValue := uqQuota[k]; usedValue.Cmp(hardValue) == 1 {
			exceededResources = append(exceededResources, k)
		}
	}
	return exceededResources
}
