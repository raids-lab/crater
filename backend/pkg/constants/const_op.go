package constants

const (
	// Operation Types
	OpTypeSetExclusive        = "SetExclusive"
	OpTypeCancelExclusive     = "CancelExclusive"
	OpTypeSetUnschedulable    = "SetUnschedulable"
	OpTypeCancelUnschedulable = "CancelUnschedulable"
	OpTypeDrainNode           = "DrainNode"
	OpTypeUpdateVPA           = "UpdateVPA"
	OpTypeDeleteJob           = "DeleteJob"
	OpTypeSetStorageQuota     = "SetStorageQuota"

	// Execution Status
	OpStatusSuccess = "Success"
	OpStatusFailed  = "Failed"
)
