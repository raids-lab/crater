import type { AgentIdentity, AgentPageContext } from "../page-context.js";
import { createFormAssistantSkill } from "./form-assistant.js";
import { createPlatformGuideSkill } from "./platform-guide.js";
import type { AgentSkill } from "./types.js";

export function selectSkill(
  identity: AgentIdentity,
  context?: AgentPageContext,
): AgentSkill {
  return context
    ? createFormAssistantSkill(context, identity)
    : createPlatformGuideSkill(identity);
}
