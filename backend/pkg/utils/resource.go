package utils

import (
	"fmt"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func CalculateRequsetsByContainers(containers []v1.Container) (resources v1.ResourceList) {
	resources = make(v1.ResourceList, 0)
	for j := range containers {
		container := &containers[j]
		resources = SumResources(resources, container.Resources.Requests)
	}
	return resources
}

func CalculateLimitsByContainers(containers []v1.Container) (resources v1.ResourceList) {
	resources = make(v1.ResourceList, 0)
	for j := range containers {
		container := &containers[j]
		resources = SumResources(resources, container.Resources.Limits)
	}
	return resources
}

func SumResources(resources ...v1.ResourceList) v1.ResourceList {
	result := make(v1.ResourceList)
	for _, res := range resources {
		for name, quantity := range res {
			if v, ok := result[name]; !ok {
				result[name] = quantity.DeepCopy()
			} else {
				v.Add(quantity)
				result[name] = v
			}
		}
	}
	return result
}

func CalculateReplicatedResources[T any](
	items []T,
	resourceOf func(T) v1.ResourceList,
	replicasOf func(T) int32,
) v1.ResourceList {
	total := v1.ResourceList{}
	for _, item := range items {
		total = SumResources(total, MultiplyResources(resourceOf(item), int64(replicasOf(item))))
	}
	return total
}

func CopyResources(resources v1.ResourceList) v1.ResourceList {
	result := make(v1.ResourceList, len(resources))
	for name, quantity := range resources {
		result[name] = quantity.DeepCopy()
	}
	return result
}

func SubtractResource(base, sub v1.ResourceList) v1.ResourceList {
	result := make(v1.ResourceList, len(base))
	for name, qty := range base {
		result[name] = qty.DeepCopy()
	}
	for name, qty := range sub {
		current := resource.Quantity{}
		if existing, ok := result[name]; ok {
			current = existing.DeepCopy()
		}
		current.Sub(qty)
		result[name] = current
	}
	return result
}

func ResourceDeficit(required, available v1.ResourceList) v1.ResourceList {
	deficit := make(v1.ResourceList)
	for name, req := range required {
		availableQty := resource.Quantity{}
		if qty, ok := available[name]; ok {
			availableQty = qty.DeepCopy()
		}
		if availableQty.Cmp(req) >= 0 {
			continue
		}

		missing := req.DeepCopy()
		missing.Sub(availableQty)
		deficit[name] = missing
	}
	return deficit
}

func ResourceListCovers(available, required v1.ResourceList) bool {
	for name, req := range required {
		availableQty := resource.Quantity{}
		if qty, ok := available[name]; ok {
			availableQty = qty.DeepCopy()
		}
		if availableQty.Cmp(req) < 0 {
			return false
		}
	}
	return true
}

func ResourceDemandScore(resources v1.ResourceList) int64 {
	var score int64
	for _, quantity := range resources {
		score += quantity.MilliValue()
	}
	return score
}

func ResourceListSummary(resources v1.ResourceList) string {
	if len(resources) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(resources))
	for name, quantity := range resources {
		parts = append(parts, fmt.Sprintf("%s=%s", name, quantity.String()))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func MultiplyResources(resources v1.ResourceList, multiplier int64) v1.ResourceList {
	result := make(v1.ResourceList, len(resources))
	for name, quantity := range resources {
		scaled := quantity.DeepCopy()
		scaled.Mul(multiplier)
		result[name] = scaled
	}
	return result
}

func ToStringMap(resources v1.ResourceList) map[string]string {
	if resources == nil {
		return nil
	}
	result := make(map[string]string, len(resources))
	for name, quantity := range resources {
		result[string(name)] = quantity.String()
	}
	return result
}
