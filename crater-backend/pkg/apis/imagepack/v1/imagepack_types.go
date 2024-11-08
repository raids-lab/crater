/*
Copyright 2024.

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

type PackStatus string

const (
	PackJobInitial  PackStatus = "Initial"
	PackJobPending  PackStatus = "Pending"
	PackJobRunning  PackStatus = "Running"
	PackJobFinished PackStatus = "Finished"
	PackJobFailed   PackStatus = "Failed"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImagePackSpec defines the desired state of ImagePack
type ImagePackSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ImagePack. Edit imagepack_types.go to remove/update
	UserName string `json:"userName"`

	GitRepository   string `json:"gitRepository"`
	AccessToken     string `json:"accessToken"`
	Dockerfile      string `json:"dockerfile,omitempty"`
	RegistryServer  string `json:"registryServer"`
	RegistryUser    string `json:"registryUser"`
	RegistryPass    string `json:"registryPass"`
	RegistryProject string `json:"registryProject"`
	ImageName       string `json:"imageName"`
	ImageTag        string `json:"imageTag"`
	// Template v1.PodTemplateSpec `json:"template,omitempty"`
}

// ImagePackStatus defines the observed state of ImagePack
type ImagePackStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=Initial;Pending;Running;Finished;Failed
	Stage     PackStatus `json:"status"`
	PodName   string     `json:"podName"`
	UUID      string     `json:"uuid"`
	ImageLink string     `json:"imageLink"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImagePack is the Schema for the imagepacks API
type ImagePack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImagePackSpec   `json:"spec,omitempty"`
	Status ImagePackStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImagePackList contains a list of ImagePack
type ImagePackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImagePack `json:"items"`
}

//nolint:gochecknoinits // This is required by kubebuilder.
func init() {
	SchemeBuilder.Register(&ImagePack{}, &ImagePackList{})
}
