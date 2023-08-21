/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantQuota is a specification for a TenantQuota resource
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName={tq,tqs}
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TenantQuota struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec TenantQuotaSpec `json:"spec"`
	// +optional
	Status TenantQuotaStatus `json:"status"`
}

// TenantQuotaSpec is the spec for a TenantQuota resource
type TenantQuotaSpec struct {
	// +optional
	Hard v1.ResourceList `json:"hard"`
	// +optional
	Soft v1.ResourceList `json:"soft"`
}

// TenantQuotaStatus is the status for a TenantQuota resource
type TenantQuotaStatus struct {
	// +optional
	Hard v1.ResourceList `json:"hard"`
	// +optional
	Soft v1.ResourceList `json:"soft"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantQuotaList is a list of TenantQuota resources
type TenantQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata"`

	Items []TenantQuota `json:"items"`
}
