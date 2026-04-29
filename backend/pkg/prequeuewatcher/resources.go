package prequeuewatcher

import (
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/utils"
)

func selectMinimalPreemptionSubset(candidates []*model.Job, deficit v1.ResourceList) []*model.Job {
	if len(candidates) == 0 || len(deficit) == 0 {
		return nil
	}

	for size := 1; size <= len(candidates); size++ {
		current := make([]*model.Job, 0, size)
		if result, ok := searchPreemptionSubset(candidates, deficit, 0, size, current, make(v1.ResourceList)); ok {
			return result
		}
	}

	return nil
}

func searchPreemptionSubset(
	candidates []*model.Job,
	deficit v1.ResourceList,
	start int,
	remaining int,
	current []*model.Job,
	sum v1.ResourceList,
) ([]*model.Job, bool) {
	if remaining == 0 {
		if utils.ResourceListCovers(sum, deficit) {
			result := make([]*model.Job, len(current))
			copy(result, current)
			return result, true
		}
		return nil, false
	}
	if len(candidates)-start < remaining {
		return nil, false
	}

	nextCurrent := make([]*model.Job, len(current), len(current)+1)
	copy(nextCurrent, current)
	for i := start; i <= len(candidates)-remaining; i++ {
		nextCurrent = append(nextCurrent, candidates[i])
		nextSum := utils.SumResources(sum, candidates[i].Resources.Data())
		if result, ok := searchPreemptionSubset(candidates, deficit, i+1, remaining-1, nextCurrent, nextSum); ok {
			return result, true
		}
	}

	return nil, false
}
