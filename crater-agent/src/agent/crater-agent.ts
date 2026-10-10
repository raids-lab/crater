import { createOpenAICompatible } from "@ai-sdk/openai-compatible";
import { ToolLoopAgent } from "ai";

import type { AgentConfig } from "../config.js";
import { buildCapabilityContract } from "../prompts/capability-contract.js";
import type { AgentSkill } from "../skills/types.js";
import { selectSkillTools } from "../tools/registry.js";

const effectiveModelAlias = "crater-effective-model";

export function createCraterModel(config: AgentConfig, contextToken: string) {
  const provider = createOpenAICompatible({
    name: "crater",
    baseURL: `${config.backendInternalURL}/api/agent/llm`,
    apiKey: contextToken,
    includeUsage: true,
  });

  return provider(effectiveModelAlias);
}

export function createCraterAgent(
  config: AgentConfig,
  contextToken: string,
  skill: AgentSkill,
) {
  return new ToolLoopAgent({
    model: createCraterModel(config, contextToken),
    instructions: [skill.instructions, buildCapabilityContract(skill)].join(
      "\n\n",
    ),
    tools: selectSkillTools(skill.activeTools),
  });
}
