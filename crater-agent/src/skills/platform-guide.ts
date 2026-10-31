import { platformUserGuideKnowledge } from "../knowledge/platform-user-guide.js";
import { baseInstructions } from "../prompts/base.js";
import { platformGuideInstructions } from "../prompts/platform-guide.js";
import type { AgentIdentity } from "../page-context.js";
import { platformReadTools, type AgentSkill } from "./types.js";

export function createPlatformGuideSkill(identity: AgentIdentity): AgentSkill {
  const identityInstructions =
    identity === "admin"
      ? "当前是管理端会话。可读取全平台作业和节点的授权信息；只能推荐 /admin 页面，绝不得推荐 /portal 页面或冒充作业所属用户。"
      : "当前是用户端会话。作业查询只限当前用户在活动账户下的作业；只能推荐 /portal 页面。";
  return {
    id: "platform-guide",
    instructions: [
      baseInstructions,
      identityInstructions,
      platformGuideInstructions,
      platformUserGuideKnowledge,
    ].join("\n\n"),
    capabilities: [
      "解释 Crater 平台的稳定使用流程。",
      "按当前身份查询授权范围内的作业列表与作业诊断信息。",
      "查询节点列表、指定节点容量和集群汇总容量。",
      "根据当前身份提出需要用户确认的页面跳转。",
    ],
    limitations: [
      "不能读取原始作业日志，也不能保证调度估算等于调度器最终决策。",
      "不能创建、修改、停止、删除或重新提交平台资源。",
      "不能越过当前身份和后端对象权限。",
    ],
    activeTools: [
      "navigate_to_page",
      "navigate_to_job_detail",
      ...platformReadTools,
    ],
  };
}
