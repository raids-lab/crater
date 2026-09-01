package payload

// TensorboardSourceJobReq identifies one source job and optionally overrides
// the TensorBoard event directory declared by that job.
type TensorboardSourceJobReq struct {
	JobName string `json:"jobName"`
	LogDir  string `json:"logDir,omitempty"`
}

// CreateTensorboardReq describes the source jobs and lifetime of a TensorBoard panel.
// The backend resolves trusted data mounts from jobs owned by the current user.
// SourceJobName, SourceJobNames, and LogDir remain available for legacy API clients.
type CreateTensorboardReq struct {
	SourceJobName  string                    `json:"sourceJobName,omitempty" example:"job-old-xxxx"`
	SourceJobNames []string                  `json:"sourceJobNames,omitempty"`
	SourceJobs     []TensorboardSourceJobReq `json:"sourceJobs,omitempty"`
	LogDir         string                    `json:"logDir,omitempty" example:"/mnt/vol0/logs"`
	TTLHours       int32                     `json:"ttlHours" binding:"required,min=1,max=168" example:"24"`
}

// TensorboardSourceConfigResp contains TensorBoard settings declared by a source job.
type TensorboardSourceConfigResp struct {
	LogDir string `json:"logDir"`
}

// CreateTensorboardResp describes a newly created TensorBoard panel.
type CreateTensorboardResp struct {
	TensorboardID string `json:"tensorboardId"` // Unique panel identifier.
	AccessPath    string `json:"accessPath"`    // Relative route prefix used to access the panel.
}

type TensorboardStatus string

type TensorboardStatusReason string

const (
	TensorboardStatusStarting TensorboardStatus = "starting"
	TensorboardStatusReady    TensorboardStatus = "ready"
	TensorboardStatusFailed   TensorboardStatus = "failed"

	TensorboardStatusReasonDeploymentFailed        TensorboardStatusReason = "deployment_failed"
	TensorboardStatusReasonStatusCheckPending      TensorboardStatusReason = "status_check_pending"
	TensorboardStatusReasonPodListPending          TensorboardStatusReason = "pod_list_pending"
	TensorboardStatusReasonImagePullFailed         TensorboardStatusReason = "image_pull_failed"
	TensorboardStatusReasonContainerStartFailed    TensorboardStatusReason = "container_start_failed"
	TensorboardStatusReasonContainerExited         TensorboardStatusReason = "container_exited"
	TensorboardStatusReasonDeploymentStarting      TensorboardStatusReason = "deployment_starting"
	TensorboardStatusReasonNetworkConfigIncomplete TensorboardStatusReason = "network_config_incomplete"
	TensorboardStatusReasonServiceMissing          TensorboardStatusReason = "service_missing"
	TensorboardStatusReasonNetworkCheckPending     TensorboardStatusReason = "network_check_pending"
	TensorboardStatusReasonServiceMisconfigured    TensorboardStatusReason = "service_misconfigured"
	TensorboardStatusReasonIngressMissing          TensorboardStatusReason = "ingress_missing"
	TensorboardStatusReasonIngressMisconfigured    TensorboardStatusReason = "ingress_misconfigured"
	TensorboardStatusReasonEndpointPending         TensorboardStatusReason = "endpoint_pending"
	TensorboardStatusReasonReady                   TensorboardStatusReason = "ready"
)

// TensorboardInfo describes a panel and its current availability.
type TensorboardInfo struct {
	ID            string                  `json:"id"`
	Expiration    string                  `json:"expiration"`
	CreatedAt     string                  `json:"createdAt"`
	AccessPath    string                  `json:"accessPath"`
	Status        TensorboardStatus       `json:"status"`
	StatusReason  TensorboardStatusReason `json:"statusReason"`
	StatusMessage string                  `json:"statusMessage"`
}

// ExtendTTLReq resets a TensorBoard panel lifetime from the current time.
type ExtendTTLReq struct {
	TTLHours int32 `json:"ttlHours" binding:"required,min=1,max=168"`
}
