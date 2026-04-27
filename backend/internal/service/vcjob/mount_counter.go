package vcjob

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func increaseDatasetMountCount(ctx context.Context, job *batch.Job) error {
	if job == nil || len(job.Annotations) == 0 {
		return nil
	}

	rawDatasetIDs := strings.TrimSpace(job.Annotations[AnnotationKeyMountedDatasetIDs])
	if rawDatasetIDs == "" {
		return nil
	}

	datasetIDs, err := parseMountedDatasetIDs(rawDatasetIDs)
	if err != nil {
		return err
	}
	if len(datasetIDs) == 0 {
		return nil
	}

	return query.GetDB().WithContext(ctx).
		Model(&model.Dataset{}).
		Where("id IN ?", datasetIDs).
		UpdateColumn("mount_count", gorm.Expr("mount_count + ?", 1)).
		Error
}

func parseMountedDatasetIDs(raw string) ([]uint, error) {
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("invalid mounted dataset ids annotation: %w", err)
	}

	idSet := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		idSet[id] = struct{}{}
	}

	deduplicatedIDs := make([]uint, 0, len(idSet))
	for id := range idSet {
		deduplicatedIDs = append(deduplicatedIDs, id)
	}
	sort.Slice(deduplicatedIDs, func(i, j int) bool {
		return deduplicatedIDs[i] < deduplicatedIDs[j]
	})
	return deduplicatedIDs, nil
}
