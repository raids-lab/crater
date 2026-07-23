package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (mgr *AgentMgr) buildToolOutcomeMessage(toolName, status string, result any, fallback string) string {
	if strings.TrimSpace(fallback) != "" && status != agentToolStatusSuccess {
		return fallback
	}

	target := extractToolOutcomeTarget(result)
	switch status {
	case agentToolStatusSuccess:
		if target != "" {
			return fmt.Sprintf("%s 已完成，目标：%s。", toolName, target)
		}
		return fmt.Sprintf("%s 已完成。", toolName)
	case agentToolStatusRejected:
		return fmt.Sprintf("已取消 %s。", toolName)
	case agentTurnStatusCancelled:
		return fmt.Sprintf("%s 已取消。", toolName)
	default:
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return fmt.Sprintf("%s 执行失败。", toolName)
	}
}

func extractToolOutcomeTarget(result any) string {
	resultMap, ok := result.(map[string]any)
	if !ok {
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return ""
		}
		if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
			return ""
		}
	}
	for _, key := range []string{"job_name", "jobName", "name", "node", "node_name", "image"} {
		if value, ok := resultMap[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
