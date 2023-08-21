package models

import (
	v1 "k8s.io/api/core/v1"
)

// Quota model
type Quota struct {
	UserName  string
	Namespace string
	Hard      v1.ResourceList
	Soft      v1.ResourceList
}
