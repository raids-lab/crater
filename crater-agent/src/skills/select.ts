import type { AgentIdentity } from "../page-context.js";
import {
  compilePromptOverridePattern,
  type AgentLocaleConfig,
} from "../runtime-config.js";
import { createPlatformGuideSkill } from "./platform-guide.js";
import { createScopeRefusalSkill } from "./scope-refusal.js";
import type { AgentSkill } from "./types.js";

export function selectSkill(
  userMessages: readonly string[],
  identity: AgentIdentity,
  localeConfig: AgentLocaleConfig,
): AgentSkill {
  const latestMessage = userMessages.at(-1)?.trim() ?? "";
  if (compilePromptOverridePattern(localeConfig).test(latestMessage)) {
    return createScopeRefusalSkill(localeConfig);
  }
  return createPlatformGuideSkill(identity, localeConfig);
}
