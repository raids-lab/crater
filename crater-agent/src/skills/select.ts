import { platformGuideSkill } from "./platform-guide.js";
import type { AgentSkill } from "./types.js";

export function selectSkill(): AgentSkill {
  return platformGuideSkill;
}
