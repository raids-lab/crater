package utils

import v1 "k8s.io/api/core/v1"

func CalculateRequsetsByContainers(containers []v1.Container) (resources v1.ResourceList) {
	resources = make(v1.ResourceList, 0)
	for j := range containers {
		container := &containers[j]
		for name, quantity := range container.Resources.Requests {
			if v, ok := resources[name]; !ok {
				resources[name] = quantity
			} else {
				v.Add(quantity)
				resources[name] = v
			}
		}
	}
	return resources
}
