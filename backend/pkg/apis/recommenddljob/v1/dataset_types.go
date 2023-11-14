/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DataSetPhase string

const (
	DataSetPhaseInitial     = "Initial"
	DataSetPhasePVCPending  = "Pending"
	DataSetPhasePVCBound    = "Bound"
	DataSetPhaseDownloading = "Downloading"
	DataSetPhaseReady       = "Ready"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataSetSpec defines the desired state of DataSet
type DataSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// bind pvc name
	PVC string `json:"pvc,omitempty"`

	// donwnloadUrl
	DownloadURL string `json:"download_url,omitempty"`

	// size GB
	Size int `json:"size,omitempty"`
}

// DataSetStatus defines the observed state of DataSet
type DataSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// phase
	Phase DataSetPhase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DataSet is the Schema for the datasets API
type DataSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataSetSpec   `json:"spec,omitempty"`
	Status DataSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataSetList contains a list of DataSet
type DataSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataSet{}, &DataSetList{})
}
