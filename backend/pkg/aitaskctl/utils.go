package aitaskctl

import (
	v1 "k8s.io/api/core/v1"
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
