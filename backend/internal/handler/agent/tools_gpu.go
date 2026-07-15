package agent

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

func detectGPUResourceName(resources v1.ResourceList) v1.ResourceName {
	for name := range resources {
		if isGPUResourceName(string(name)) {
			return name
		}
	}
	return ""
}

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

func moveResourceQuantity(resources v1.ResourceList, oldName, newName v1.ResourceName) {
	if resources == nil || oldName == "" || oldName == newName {
		return
	}
	if quantity, ok := resources[oldName]; ok {
		resources[newName] = quantity
		delete(resources, oldName)
	}
}

func removeGPUResourcesExcept(resources v1.ResourceList, keep v1.ResourceName) bool {
	if resources == nil {
		return false
	}
	changed := false
	for name := range resources {
		if name == keep || !isGPUResourceName(string(name)) {
			continue
		}
		delete(resources, name)
		changed = true
	}
	return changed
}

func isGPUResourceName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "nvidia.com/") || strings.Contains(normalized, "/gpu") || strings.Contains(normalized, "gpu") {
		return true
	}
	for _, model := range []string{"v100", "a100", "h100", "l40s", "rtx4090"} {
		if normalized == model || strings.HasSuffix(normalized, "/"+model) {
			return true
		}
	}
	return false
}

func extractGPUModelFromResourceName(name string) string {
	parts := strings.SplitN(strings.TrimSpace(name), "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return name
}
