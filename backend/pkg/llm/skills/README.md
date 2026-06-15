# LLM Skills

This directory stores domain-specific prompt skills for the Crater backend.

## Storage Governance Agent Skill

- Default skill file:
  [storage-governance-agent/SKILL.md](D:/crater/backend/pkg/llm/skills/storage-governance-agent/SKILL.md)
- Loaded only by DeepSeek/OpenAI-compatible agent mode:
  [llm_tools.go](D:/crater/backend/pkg/llm/llm_tools.go)

## Runtime Controls

- `CRATER_STORAGE_AGENT_SKILL_ENABLED`
  - Optional, default `true`
  - Set to `false` to disable loading the skill
- `CRATER_STORAGE_AGENT_SKILL_PATH`
  - Optional
  - When set, the backend loads the skill text from this external file path instead of the embedded default skill
