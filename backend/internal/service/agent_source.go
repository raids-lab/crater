package service

import "strings"

const (
	agentSessionSourceChat     = "chat"
	agentSessionSourceOpsAudit = "ops_audit"
	agentSessionSourceSystem   = "system"
	agentToolCallSourceBackend = "backend"
)

func normalizeAgentSessionSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case agentSessionSourceOpsAudit:
		return agentSessionSourceOpsAudit
	case agentSessionSourceSystem:
		return agentSessionSourceSystem
	default:
		return agentSessionSourceChat
	}
}

func normalizeAgentToolCallSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "local":
		return "local"
	default:
		return agentToolCallSourceBackend
	}
}
