package v1alpha1

const (
	// OperatorNameLabel represents the label key for the operator name, e.g. tf-operator, mpi-operator, etc.
	OperatorNameLabel = "aisystem.github.com/operator-name"

	// JobNameLabel represents the label key for the job name, the value is the job name.
	JobNameLabel = "aisystem.github.com/job-name"

	// JobRoleLabel represents the label key for the job role, e.g. master.
	JobRoleLabel = "aisystem.github.com/job-role"

	// Labels for job
	TaskTypeLabelKey = "aisystem.github.com/task-type"
	TaskSLOLabelKey  = "aisystem.github.com/task-slo"
	TaskIDKey        = "aisystem.github.com/taskid"
)
