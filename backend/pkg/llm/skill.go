package llm

import (
	"embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	StorageAgentSkillEnabledEnv = "CRATER_STORAGE_AGENT_SKILL_ENABLED"
	StorageAgentSkillPathEnv    = "CRATER_STORAGE_AGENT_SKILL_PATH"
	defaultStorageAgentSkill    = "skills/storage-governance-agent/SKILL.md"
)

//go:embed skills/storage-governance-agent/SKILL.md
var embeddedSkills embed.FS

func loadStorageAgentSkill() (content, source string, err error) {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv(StorageAgentSkillEnabledEnv)); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return "", "", fmt.Errorf("invalid %s value %q: %w", StorageAgentSkillEnabledEnv, raw, err)
		}
		enabled = parsed
	}
	if !enabled {
		return "", "", nil
	}

	if customPath := strings.TrimSpace(os.Getenv(StorageAgentSkillPathEnv)); customPath != "" {
		data, err := os.ReadFile(customPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to read storage agent skill from %s: %w", customPath, err)
		}
		return strings.TrimSpace(string(data)), customPath, nil
	}

	data, err := embeddedSkills.ReadFile(defaultStorageAgentSkill)
	if err != nil {
		return "", "", fmt.Errorf("failed to read embedded storage agent skill: %w", err)
	}
	return strings.TrimSpace(string(data)), defaultStorageAgentSkill, nil
}
