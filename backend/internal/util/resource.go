package util

import v1 "k8s.io/api/core/v1"

const (
	NVIDIARuntimeClass = "nvidia"
)

// receive v1.ResourceList, check if requset nvidia gpu resource more than 0
// return true if has requestnvidia gpu resource, otherwise return false
func HasNVIDIAGPUResource(resources v1.ResourceList) bool {
	v, ok := resources[v1.ResourceName("nvidia.com/gpu")]
	if !ok {
		return false
	}
	return v.Value() > 0
}
