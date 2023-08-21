// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AIJobKind is the kind name.
	AIJobKind = "AIJob"
	// AIJobPlural is the plural for AIJob.
	AIJobPlural = "aijobs"
	// AIJobSingular is the singular for AIJob.
	AIJobSingular = "aijob"
)

// AIJob Represents an AIJob resource.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjob
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AIJob struct {
	// Standard Kubernetes type metadata.
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired state of the AIJob.
	Spec JobSpec `json:"spec,omitempty"`

	// Most recently observed status of the AIJob.
	// Read-only (modified by the system).
	Status JobStatus `json:"status,omitempty"`
}

// JobSpec is a desired state description of the AIJob.
type JobSpec struct {
	// Template is the object that describes the pod that
	// will be created for this replica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in ReplicaSpec
	Replicas        int32              `json:"replicas,omitempty"`
	Template        v1.PodTemplateSpec `json:"template,omitempty"`
	ResourceRequest v1.ResourceList    `json:"resourceRequest,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=aijobs
// +kubebuilder:object:root=true

// AIJobList is a list of AIJobs.
type AIJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of AITorchJobs.
	Items []AIJob `json:"items"`
}

// JobStatus represents the current observed state of the training Job.
type JobStatus struct {
	// Phase
	Phase JobPhase `json:"phase,omitempty"`

	// Conditions is an array of current observed job conditions.
	Conditions []JobCondition `json:"conditions"`

	// Represents time when the job was acknowledged by the job controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents last time when the job was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// JobCondition describes the state of the job at a certain point.
type JobCondition struct {
	// Type of job condition.
	Type JobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type JobPhase string

const (
	Pending   JobPhase = "Pending"
	Running   JobPhase = "Running"
	Succeeded JobPhase = "Succeeded"
	Failed    JobPhase = "Failed"
	Suspended JobPhase = "Suspended"
	Unknown   JobPhase = ""
)

// JobConditionType defines all kinds of types of JobStatus.
type JobConditionType string

const (
	// JobCreated means the job has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	JobCreated JobConditionType = "Created"

	// JobRunning means all sub-resources (e.g. services/pods) of this job
	// have been successfully scheduled and launched.
	// The training is running without error.
	JobRunning JobConditionType = "Running"

	// JobSucceeded means all sub-resources (e.g. services/pods) of this job
	// reached phase have terminated in success.
	// The training is complete without error.
	JobSucceeded JobConditionType = "Succeeded"

	// JobFailed means one or more sub-resources (e.g. services/pods) of this job
	// reached phase failed with no restarting.
	// The training has failed its execution.
	JobFailed JobConditionType = "Failed"

	// JobSuspended means one or more sub-resources (e.g. services/pods) of this job
	// reached phase suspended with no restarting.
	// The training has failed its execution.
	JobSuspended JobConditionType = "Suspended"
)
