/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

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

package payload

// TensorboardSourceJobReq identifies one source job and optionally overrides
// the TensorBoard event directory declared by that job.
type TensorboardSourceJobReq struct {
	JobName string `json:"jobName"`
	LogDir  string `json:"logDir,omitempty"`
}

// CreateTensorboardReq describes the optional source jobs and log directory.
// Without a source job, LogDir must point into the current user's personal workspace.
// With source jobs, the backend resolves trusted data mounts from jobs owned by the current user.
// SourceJobName, SourceJobNames, and LogDir remain available for legacy API clients.
type CreateTensorboardReq struct {
	Name           string                    `json:"name" binding:"required" example:"Training comparison"`
	SourceJobName  string                    `json:"sourceJobName,omitempty" example:"job-old-xxxx"`
	SourceJobNames []string                  `json:"sourceJobNames,omitempty"`
	SourceJobs     []TensorboardSourceJobReq `json:"sourceJobs,omitempty"`
	LogDir         string                    `json:"logDir,omitempty" example:"/mnt/vol0/logs"`
}

// TensorboardSourceConfigResp contains TensorBoard settings declared by a source job.
type TensorboardSourceConfigResp struct {
	LogDir string `json:"logDir"`
}

// CreateTensorboardResp describes a newly created TensorBoard panel.
type CreateTensorboardResp struct {
	Name          string `json:"name"`          // User-defined display name.
	TensorboardID string `json:"tensorboardId"` // Unique panel identifier.
	AccessPath    string `json:"accessPath"`    // Relative route prefix used to access the panel.
}

type TensorboardStatus string

type TensorboardStatusReason string

const (
	TensorboardStatusPending   TensorboardStatus = "pending"
	TensorboardStatusStarting  TensorboardStatus = "starting"
	TensorboardStatusReady     TensorboardStatus = "ready"
	TensorboardStatusFailed    TensorboardStatus = "failed"
	TensorboardStatusCompleted TensorboardStatus = "completed"
	TensorboardStatusExpired   TensorboardStatus = "expired"

	TensorboardStatusReasonWaitingForSchedule TensorboardStatusReason = "waiting_for_schedule"
	TensorboardStatusReasonPodStarting        TensorboardStatusReason = "pod_starting"
	TensorboardStatusReasonJobFailed          TensorboardStatusReason = "job_failed"
	TensorboardStatusReasonRuntimeExpired     TensorboardStatusReason = "runtime_expired"
	TensorboardStatusReasonIdleExpired        TensorboardStatusReason = "idle_expired"
	TensorboardStatusReasonCompleted          TensorboardStatusReason = "completed"
	TensorboardStatusReasonReady              TensorboardStatusReason = "ready"
)

// TensorboardInfo describes a panel and its current availability.
type TensorboardInfo struct {
	Name          string                  `json:"name"`
	ID            string                  `json:"id"`
	Expiration    string                  `json:"expiration"`
	CreatedAt     string                  `json:"createdAt"`
	AccessPath    string                  `json:"accessPath"`
	Status        TensorboardStatus       `json:"status"`
	StatusReason  TensorboardStatusReason `json:"statusReason"`
	StatusMessage string                  `json:"statusMessage"`
}
