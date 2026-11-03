package vcjob

import "strconv"

const AnnotationKeyWaitingToleranceSeconds = "crater.raids.io/waiting-tolerance-seconds"

// ApplyScheduleMetadataAnnotations stamps the tolerance the extender reads when deciding whether a
// waiting job has timed out. Absent annotation means the job never times out.
func ApplyScheduleMetadataAnnotations(annotations map[string]string, waitingToleranceSeconds *int64) {
	if waitingToleranceSeconds == nil {
		delete(annotations, AnnotationKeyWaitingToleranceSeconds)
		return
	}
	annotations[AnnotationKeyWaitingToleranceSeconds] = strconv.FormatInt(*waitingToleranceSeconds, 10)
}
