import { platformUserGuideKnowledge } from "../knowledge/platform-user-guide.js";
import { baseInstructions } from "../prompts/base.js";
import { platformGuideInstructions } from "../prompts/platform-guide.js";
import type { AgentSkill } from "./types.js";

export const platformGuideSkill = {
  id: "platform-guide",
  instructions: [
    baseInstructions,
    platformGuideInstructions,
    platformUserGuideKnowledge,
  ].join("\n\n"),
  capabilities: [
    "依据内置的 Crater 官方用户文档知识解释平台用途和用户侧流程。",
    "根据用户目标说明 Jupyter、Web IDE、单机批处理和分布式作业类型的适用场景。",
    "说明镜像、文件、数据集、模型、挂载和持久化的稳定使用规则。",
    "提供当前已登记的用户端菜单名称、页面入口和新用户操作路径。",
  ],
  limitations: [
    "不能查询当前作业、Pod、节点、资源容量、配额、镜像列表、日志、监控或下载记录。",
    "不能确认用户当前可访问的具体镜像、数据集、模型、GPU 型号或其他资源。",
    "不能创建、修改、停止、删除、复制或重新提交任何平台资源。",
    "不能执行管理员运维操作，也不处理与 Crater 平台无关的请求。",
  ],
  activeTools: [],
} as const satisfies AgentSkill;
