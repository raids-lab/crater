import type { AgentSkill } from "./types.js";

export const scopeRefusalSkill = {
  id: "scope-refusal",
  instructions: "",
  capabilities: ["识别请求是否属于 Crater 平台用户侧使用范围。"],
  limitations: ["不回答与 Crater 平台无关的知识、创作、娱乐或角色扮演请求。"],
  activeTools: [],
  staticResponse:
    "我是 Crater-Agent，只处理 Crater 平台相关问题。你可以询问作业类型与使用流程、镜像、数据集、模型、文件挂载或平台新用户入门。",
} as const satisfies AgentSkill;
