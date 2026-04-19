package utils

import (
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
