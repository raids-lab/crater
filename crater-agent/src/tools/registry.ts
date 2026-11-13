import type { ToolSet } from "ai";

// Business tools are intentionally empty in the foundation milestone. Future
// skills must explicitly select tools from this registry.
export const toolRegistry = {} satisfies ToolSet;

export function selectSkillTools(activeTools: readonly string[]): ToolSet {
  const registry: ToolSet = toolRegistry;
  const selectedTools: ToolSet = {};
  for (const toolName of activeTools) {
    const selectedTool = registry[toolName];
    if (!selectedTool) {
      throw new Error(`Skill requested an unknown tool: ${toolName}`);
    }
    selectedTools[toolName] = selectedTool;
  }
  return selectedTools;
}
