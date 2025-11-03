package utils

import v1 "k8s.io/api/core/v1"

func CalculateRequsetsByContainers(containers []v1.Container) (resources v1.ResourceList) {
	resources = make(v1.ResourceList, 0)
	for j := range containers {
		container := &containers[j]
		resources = SumResources(resources, container.Resources.Requests)
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
