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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type TrainingJobStatus string

const (
	TrainingJobInitial  = "Initial"
	TrainingJobPending  = "Pending"
	TrainingJobRunning  = "Running"
	TrainingJobFinished = "Finished"
	TrainingJobFailed   = "Failed"
)

type RunningType string

const (
	LongRunning = "long-running"
	OneShot     = "one-shot"
)

type DataRelationShipType string

const (
	InputRelation  = "input"
	OutputRelation = "output"
	BothWay        = "bothway"
)

type DataRelationShip struct {
	// releationship type
	// +kubebuilder:validation:Enum=input;output;bothway
	Type DataRelationShipType `json:"type"` // input, output, bothyway

	// releationship job
	JobName string `json:"job_name"`

	// relationship job namespace
	JobNamespace string `json:"job_namespace"`
}

type DataSetRef struct {
	// dataset name
	Name string `json:"name"`

	// datasets ref
	DataSet DataSetSpec `json:"dataset"`
}

// RecommendDLJobSpec defines the desired state of RecommendDLJob
type RecommendDLJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// pod replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	// running type
	// +kubebuilder:validation:Enum=long-running;one-shot
	RunningType RunningType `json:"running_type"`

	// dataset
	DataSets []DataSetRef `json:"datasets,omitempty"`

	// relationship
	RelationShips []DataRelationShip `json:"releation_ships,omitempty"`

	// pod template
	Template v1.PodTemplateSpec `json:"template,omitempty"`

	// userinfo
	Username string `json:"username"`

	// taskinfo macs
	Macs int64 `json:"macs"`

	// taskinfo params
	Params int64 `json:"params"`

	// taskinfo batch_size
	BatchSize int `json:"batch_size"`

	// taskinfo embedding size total
	// +kubebuilder:validation:Optional
	EmbeddingSizeTotal int64 `json:"embedding_size_total"`

	// taskinfo emebdding dim total
	// +kubebuilder:validation:Optional
	EmbeddingDimTotal int `json:"embedding_dim_total"`

	// taskinfo embedding tables
	// +kubebuilder:validation:Optional
	EmbeddingTableCount int `json:"embedding_table_count"`

	// vocabulary_size
	// +kubebuilder:validation:Optional
	VocabularySize []int `json:"vocabulary_size"`

	// embeddig_dim
	// +kubebuilder:validation:Optional
	EmbeddingDim []int `json:"embedding_dim"`

	// input_tensor
	// +kubebuilder:validation:Optional
	InputTensor []int `json:"input_tensor"`
}

// RecommendDLJobStatus defines the observed state of RecommendDLJob
type RecommendDLJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// job uuid
	UUID string `json:"uuid"`

	// status phase
	// +kubebuilder:validation:Enum=Initial;Pending;Running;Finished;Failed
	Phase TrainingJobStatus `json:"phase"`

	// pods names
	PodNames []string `json:"pod_names"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The Job Status Phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status

// RecommendDLJob is the Schema for the recommenddljobs API
type RecommendDLJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecommendDLJobSpec   `json:"spec,omitempty"`
	Status RecommendDLJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RecommendDLJobList contains a list of RecommendDLJob
type RecommendDLJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RecommendDLJob `json:"items"`
}

//nolint:gochecknoinits // This is required by kubebuilder.
func init() {
	SchemeBuilder.Register(&RecommendDLJob{}, &RecommendDLJobList{})
}
