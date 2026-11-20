import type { AgentSkill } from "../skills/types.js";

function listItems(items: readonly string[]): string {
  return items.map((item) => `- ${item}`).join("\n");
}

export function buildCapabilityContract(skill: AgentSkill): string {
  const tools =
    skill.activeTools.length === 0
      ? "- 当前没有可调用的业务工具。"
      : listItems(
          skill.activeTools.map((toolName) => `可调用工具：${toolName}`),
        );

  return `# 当前能力契约

## 已启用能力
${listItems(skill.capabilities)}

## 尚未启用或禁止的能力
${listItems(skill.limitations)}

## 当前工具
${tools}

必须严格遵守以上能力边界：
- 这些条目用于判断你能否完成请求，不需要逐项展示给用户；仅在能力缺口直接影响用户请求时说明相关的一项。
- 只能根据“已启用能力”和成功返回的工具结果说明自己可以做什么。
- 解释用户已经提供的日志或错误信息不属于实时查询，可以直接依据材料分析，无需先声明没有工具。
- 请求确实需要尚未启用的查询或操作时，简要说明相关限制；只有存在匹配且有帮助的入口时才提供替代路径，不要附加无关功能介绍，也不得暗示稍后可以查询或执行。
- 没有对应工具时，不得使用“我帮你查一下”“我帮你看看具体情况”“我帮你选择可用资源”等暗示已具备实时访问能力的表述。
- 不得把通用 Kubernetes 或机器学习经验表述成 Crater 平台的既有功能、配置或当前状态。`;
}
