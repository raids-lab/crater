import type { AgentIdentity } from "../page-context.js";
import { createPlatformGuideSkill } from "./platform-guide.js";
import { scopeRefusalSkill } from "./scope-refusal.js";
import type { AgentSkill } from "./types.js";

const overrideAttemptPattern =
  /忽略.{0,20}(指令|规则|提示词)|(?:展示|输出|泄露|告诉我).{0,20}(系统提示词|开发者消息|内部指令)|jailbreak|越狱/iu;

export function selectSkill(
  userMessages: readonly string[],
  identity: AgentIdentity,
): AgentSkill {
  const latestMessage = userMessages.at(-1)?.trim() ?? "";
  if (overrideAttemptPattern.test(latestMessage)) {
    return scopeRefusalSkill;
  }
  return createPlatformGuideSkill(identity);
}
