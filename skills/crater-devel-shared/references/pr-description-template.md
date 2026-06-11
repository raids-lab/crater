# PR Description Template

Use this template when generating a Crater PR description. Replace every `<...>` placeholder with real content before showing it to the developer or using it in a PR.

```markdown
<One sentence summarizing the PR and motivation if any.>

### Changes

- <Grouped summary of what changed, mentioning affected areas or files where useful>

### Testing

AI:
- `<command-or-check>`: <result observed by the Agent>

Developer:
- <Manual check with page / role / action / observed result, or document / language / link / terminology / placeholder checked by the developer>

### Screenshots

- <Required for frontend / UI changes: attach or link screenshots of affected interface state(s)>

### Other (optional)

<Special risks, migration notes, rollout notes, or compatibility notes.>

---

<一句话中文概述本 PR，必要时包含动机。>

### 修改

- <按「做了什么」归类概述改动，必要时说明涉及区域或文件>

### 测试

AI：
- `<命令或检查>`：<Agent 观察到的结果>

开发者：
- <开发者亲自执行的人工检查，写清页面 / 角色 / 操作 / 观察结果，或文档 / 语言版本 / 链接 / 术语 / 占位符检查>

### 截图

- <涉及前端 / UI 改动时必须附受影响界面状态的截图或链接>

### 其他（可选）

<特殊风险、迁移说明、上线说明或兼容性说明。>

---

Resolve #<issue-number>
```

If there is no related issue, omit the final `Resolve` block. Keep simple changes short; do not expand the description just to fill every section.

## Example

This example is documentation-only, so it omits the Screenshots section and the related issue block. The Testing section separates AI checks from developer manual reading.

```markdown
Improve Crater CLI user docs, contribution docs, and review guidance, adding a clear review entrypoint for completed CLI development stages and routing Copilot reviews through the CLI review guide.

### Changes

- **User docs**: Rewrite `cli/README.md` and add `cli/README.zh-CN.md`, keeping only the CLI overview, minimal usage, Agent Skills installation entrypoint, and license information
- **Contribution docs**: Add `cli/CONTRIBUTING.md` and `cli/CONTRIBUTING.zh-CN.md`, describing the CLI documentation-driven workflow and when to use related Make targets during development and testing
- **Review guide**: Add `cli/docs/REVIEW.md` to guide developers and AI reviewers when checking code, docs, tests, golden files, and Skills after a development stage
- **Copilot review instruction**: Add `.github/instructions/cli.instructions.md` so changes under `cli/**` are reviewed according to `cli/docs/REVIEW.md`
- **License and doc responsibilities**: Add Apache License 2.0 under `cli/`, and link the new review guide from `SPEC`, `COMMANDS`, and `ARCHITECTURE`

### Testing

AI:
- `git diff --check`: passed

Developer:
- Manually read the added and updated CLI docs, including `cli/README.md`, `cli/README.zh-CN.md`, `cli/CONTRIBUTING.md`, `cli/CONTRIBUTING.zh-CN.md`, and `cli/docs/REVIEW.md`, confirming the responsibility boundaries, bilingual content, and Skills installation guidance are as expected

---

完善 Crater CLI 的用户文档、贡献文档与 review 指南，补齐 CLI 阶段性开发完成后的审查入口和 Copilot 审查路由。

### 修改

- **用户文档**：重写 `cli/README.md`，新增 `cli/README.zh-CN.md`，保留 CLI 简介、最小用法、Agent Skills 安装入口和许可证说明
- **贡献文档**：新增 `cli/CONTRIBUTING.md` 与 `cli/CONTRIBUTING.zh-CN.md`，说明 CLI 文档驱动开发流程，并在开发与测试过程中指明相关 Make 目标的使用场景
- **Review 指南**：新增 `cli/docs/REVIEW.md`，用于指导开发者和 AI reviewer 在阶段性开发完成后检查代码、文档、测试、golden 文件和 Skills 的一致性
- **Copilot 审查指令**：新增 `.github/instructions/cli.instructions.md`，让 `cli/**` 变更按 `cli/docs/REVIEW.md` 进行审查
- **许可证与文档职责**：为 `cli/` 添加 Apache License 2.0，并在 `SPEC`、`COMMANDS`、`ARCHITECTURE` 中补充 `REVIEW.md` 的职责入口

### 测试

AI：
- `git diff --check`：通过

开发者：
- 人工阅读检查新增和修改的 CLI 文档，包括 `cli/README.md`、`cli/README.zh-CN.md`、`cli/CONTRIBUTING.md`、`cli/CONTRIBUTING.zh-CN.md` 和 `cli/docs/REVIEW.md`，确认职责边界、双语内容和 Skills 安装说明符合预期
```
