package vcjob

import (
	"context"
	"sort"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

func increaseDatasetMountCount(ctx context.Context, mounts []util.VolumeMount) error {
	datasetIDs := collectMountedDatasetIDs(mounts)
	if len(datasetIDs) == 0 {
		return nil
	}

	return query.GetDB().WithContext(ctx).
		Model(&model.Dataset{}).
		Where("id IN ?", datasetIDs).
		UpdateColumn("mount_count", gorm.Expr("mount_count + ?", 1)).
		Error
}

func collectMountedDatasetIDs(mounts []util.VolumeMount) []uint {
	idSet := make(map[uint]struct{})
	for _, vm := range mounts {
		if vm.Type != util.DataType || vm.DatasetID == 0 {
			continue
		}
		idSet[vm.DatasetID] = struct{}{}
	}

	ids := make([]uint, 0, len(idSet))
	for datasetID := range idSet {
		ids = append(ids, datasetID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}
