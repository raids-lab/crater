package agent

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

func normalizeGPUModelName(input string) string {
	model := strings.TrimSpace(strings.ToLower(input))
	model = strings.ReplaceAll(model, " ", "-")
	return model
}

func normalizeGPUResourceName(current v1.ResourceName, gpuModel string) v1.ResourceName {
	model := normalizeGPUModelName(gpuModel)
	if model == "" {
		return current
	}
	if strings.Contains(model, "/") {
		return v1.ResourceName(model)
	}

	vendor := "nvidia.com"
	if current != "" {
		parts := strings.SplitN(string(current), "/", 2)
		if len(parts) == 2 && parts[0] != "" {
			vendor = parts[0]
		}
	}
	return v1.ResourceName(fmt.Sprintf("%s/%s", vendor, model))
}
