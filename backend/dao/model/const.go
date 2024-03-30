package model

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	RoleGuest = "guest"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

var (
	DefaultQuota = v1.ResourceList{
		v1.ResourceCPU:                    resource.MustParse("2"),
		v1.ResourceMemory:                 resource.MustParse("4Gi"),
		v1.ResourceName("nvidia.com/gpu"): resource.MustParse("0"),
	}
)
