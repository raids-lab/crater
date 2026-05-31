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
	OpTypeScanCheckpoint      = "ScanCheckpoint"
	OpTypeRestoreCheckpoint   = "RestoreCheckpoint"
	OpTypeDeleteCheckpoint    = "DeleteCheckpoint"
	OpTypeCleanupCheckpoint   = "CleanupCheckpoint"

	// Execution Status
	OpStatusSuccess = "Success"
	OpStatusFailed  = "Failed"
)
