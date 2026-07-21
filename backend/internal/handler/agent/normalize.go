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
	_ = mode
	return agentRoleSingleAgent
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

func getToolArgBool(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
	}
	return fallback
}
