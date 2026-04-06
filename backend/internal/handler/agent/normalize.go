package agent

import (
	"encoding/json"
	"strconv"
	"strings"
)

func normalizePageContext(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var page map[string]any
	if err := json.Unmarshal(raw, &page); err != nil {
		return map[string]any{}
	}
	if jobName, ok := page["jobName"]; ok {
		page["job_name"] = jobName
	}
	if jobStatus, ok := page["jobStatus"]; ok {
		page["job_status"] = jobStatus
	}
	if nodeName, ok := page["nodeName"]; ok {
		page["node_name"] = nodeName
	}
	if entryPoint, ok := page["entryPoint"]; ok {
		page["entrypoint"] = entryPoint
	}
	return page
}

func normalizeClientContext(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var clientContext map[string]any
	if err := json.Unmarshal(raw, &clientContext); err != nil {
		return map[string]any{}
	}
	return clientContext
}

func normalizeOrchestrationMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "multi_agent":
		return "multi_agent"
	default:
		return "single_agent"
	}
}

func normalizeJobStatuses(statuses []string) []string {
	normalized := make([]string, 0, len(statuses)*2)
	for _, status := range statuses {
		if trimmed := strings.TrimSpace(status); trimmed != "" {
			normalized = append(normalized, trimmed)
			normalized = append(normalized, strings.ToUpper(trimmed[:1])+strings.ToLower(trimmed[1:]))
		}
	}
	return normalized
}

func parseToolArgsMap(rawArgs json.RawMessage) map[string]any {
	args := map[string]any{}
	_ = json.Unmarshal(rawArgs, &args)
	return args
}

func getToolArgString(args map[string]any, key, fallback string) string {
	value, _ := args[key].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func getToolArgInt(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func messageContainsAny(target string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(target, part) {
			return true
		}
	}
	return false
}
