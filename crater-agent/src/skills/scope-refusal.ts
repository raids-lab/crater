import type { AgentSkill } from "./types.js";
import type { AgentLocaleConfig } from "../runtime-config.js";

export function createScopeRefusalSkill(
  localeConfig: AgentLocaleConfig,
): AgentSkill {
  return {
    id: "scope-refusal",
    instructions: "",
    capabilities: ["识别请求是否属于 Crater 平台用户侧使用范围。"],
    limitations: ["不回答与 Crater 平台无关的知识、创作、娱乐或角色扮演请求。"],
    activeTools: [],
    staticResponse: localeConfig.scopeRefusalMessage,
  };
}
