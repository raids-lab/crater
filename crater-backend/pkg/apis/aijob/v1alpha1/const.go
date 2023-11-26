package v1alpha1

const (
	// OperatorNameLabel represents the label key for the operator name, e.g. tf-operator, mpi-operator, etc.
	OperatorNameLabel = "aisystem.github.com/operator-name"

	// JobNameLabel represents the label key for the job name, the value is the job name.
	JobNameLabel = "aisystem.github.com/job-name"

	// JobRoleLabel represents the label key for the job role, e.g. master.
	JobRoleLabel = "aisystem.github.com/job-role"

	// Labels for job
	LabelKeyTaskType          = "aisystem.github.com/task-type"
	LabelKeyTaskSLO           = "aisystem.github.com/task-slo"
	LabeKeyTaskID             = "aisystem.github.com/taskid"
	LabelKeyTaskUser          = "aisystem.github.com/task-user"
	LabelKeyEstimatedTime     = "aisystem.github.com/estimated-time"
	AnnotationKeyProfileStat  = "aisystem.github.com/profile-stat"
	AnnotationKeyPreemptInfo  = "aisystem.github.com/preempt-info"
	AnnotationKeyPreemptCost  = "aisystem.github.com/preempt-cost"
	AnnotationKeyPreemptCount = "aisystem.github.com/preempt-count"
)
