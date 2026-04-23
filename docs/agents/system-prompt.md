# System Prompt 设计文档

> 记录 single_agent 模式下系统提示词的分段模板、动态注入机制和设计意图。
> 最后更新: 2026-04-21

---

## 1. 模板结构

当前模板位于 `crater-agent/crater_agent/agent/prompts.py`，由四段组合而成:

```
_BASE_PROMPT        共享：角色定义 + 核心能力 + 12 条共享原则 + 平台规约 + 资源推荐 + 工具选择指引
  +
_ADMIN_ADDON        管理员专用：6 条补充原则 + 集群诊断指引 + 可观测性/PromQL 参考
  或
_USER_ADDON         用户专用：10 条补充原则（歧义澄清、页面上下文确认、确认卡片优先等）
  +
_CONTEXT_SECTION    动态注入：当前用户信息 + 页面上下文 + capabilities + skills
```

`build_system_prompt()` 根据 `context.capabilities.surface.page_scope`（Go 后端已传递）选择 addon。

---

## 2. 共享原则（_BASE_PROMPT，12 条）

1. 先收集证据再下结论
2. 最少调用原则（2-4 次）
3. 写操作必须确认
4. 结论优先（结论 → 证据 → 建议）
5. 诚实原则
6. 上下文是软线索
7. 工具失败要降级
8. 歧义必须澄清
9. 区分系统名和显示名
10. 最新/最近按时间理解
11. 历史续接要有明确信号
12. 禁止在回复中编造确认卡片（直接调工具触发，不要在文字里伪造卡片内容或说"已生成"）

---

## 3. 管理员补充原则（_ADMIN_ADDON，A1-A6）

A1. 管理员页优先全局工具
A2. 管理员与个人视角分离
A3. 镜像/GPU 推荐必须有证据
A4. 新建作业先补齐素材
A5. 外网检索用途受限
A6. 脚本证据必须受控

另含：管理员集群诊断指引、可观测性与指标查询（PromQL 参考）。

---

## 4. 用户补充原则（_USER_ADDON，U1-U10）

U1. 用户边界优先
U2. 列表先于概览
U3. 确认卡片优先
U4. 确认续接要直接执行
U5. 能用表单就别来回追问
U6. 单作业绑定要谨慎
U7. **页面上下文需确认** — 基于页面推断意图时必须先向用户确认
U8. 镜像/GPU 推荐必须有证据
U9. 新建作业先补齐素材
U10. 平台工具优先

---

## 5. 动态注入机制

`build_system_prompt()` 执行以下注入:

| 占位符 | 数据来源 | 说明 |
|--------|----------|------|
| `{user_name}` | `context.actor.username` | 用户名 |
| `{user_id}` | `context.actor.user_id` | 用户 ID |
| `{role}` | 从 page_scope 推断 | "admin" 或 "user" |
| `{account_name}` | `context.actor.account_name` | 账户名 |
| `{account_id}` | `context.actor.account_id` | 账户 ID |
| `{locale}` | `context.actor.locale` | 语言偏好（默认 zh-CN） |
| `{page_url}` | `context.page.url` 或 `.route` | 当前页面路径 |
| `{page_context_detail}` | 从 page 中提取 | 当前关注的作业/节点/PVC |
| `{capabilities_detail}` | `context.capabilities.confirm_tools` | 需确认的写操作工具列表 |
| `{skills_context}` | `load_all_skills()` | 技能 YAML 知识 |

### 角色推断逻辑

优先使用 Go 后端已计算的 `capabilities.surface.page_scope`；若未设置则从 page route/url 推断。

### Page Context 注入差异

- **admin 场景**: `- 当前关注作业: {job_name}`（直接使用）
- **user 场景**: `- 页面上下文（仅供参考，操作前需用户确认）: 用户正在查看作业 {job_name}`（需用户确认后才能绑定操作）

---

## 6. 代码位置

| 文件 | 内容 |
|------|------|
| `crater-agent/crater_agent/agent/prompts.py` | 模板定义 + build_system_prompt() |
| `crater-agent/crater_agent/skills/loader.py` | 技能 YAML 加载 |
| `crater-agent/crater_agent/orchestrators/single.py` | 调用 build_system_prompt 的时机 |
