package vcjob

import (
	"fmt"
	"strconv"

	"github.com/raids-lab/crater/dao/model"
)

const (
	AnnotationKeyScheduleType            = "crater.raids.io/schedule-type"
	AnnotationKeyWaitingToleranceSeconds = "crater.raids.io/waiting-tolerance-seconds"
)

func ParseJobScheduleMetadata(annotations map[string]string) (model.ScheduleType, *int64, error) {
	scheduleType, err := model.ParseScheduleType(annotations[AnnotationKeyScheduleType])
	if err != nil {
		return model.ScheduleTypeNormal, nil, err
	}
	waitingToleranceSeconds, err := model.ParseWaitingToleranceSeconds(
		annotations[AnnotationKeyWaitingToleranceSeconds],
	)
	if err != nil {
		return model.ScheduleTypeNormal, nil, err
	}
	if waitingToleranceSeconds != nil && scheduleType == model.ScheduleTypeBackfill {
		return model.ScheduleTypeNormal, nil, fmt.Errorf(
			"waitingToleranceSeconds is only supported for normal jobs",
		)
	}
	return scheduleType, waitingToleranceSeconds, nil
}

func ApplyScheduleMetadataAnnotations(
	annotations map[string]string,
	scheduleType model.ScheduleType,
	waitingToleranceSeconds *int64,
) {
	annotations[AnnotationKeyScheduleType] = strconv.Itoa(int(scheduleType))
	if waitingToleranceSeconds == nil {
		delete(annotations, AnnotationKeyWaitingToleranceSeconds)
		return
	}
	annotations[AnnotationKeyWaitingToleranceSeconds] = strconv.FormatInt(
		*waitingToleranceSeconds,
		10,
	)
}
