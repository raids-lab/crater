---
applyTo: "cli/**"
---

# CLI 审查指令

此文档用于让 Copilot 在审查 `cli/` 目录变更时加载 CLI 专用 review 指南。

凡审查 `cli/` 目录变更，`code-review` 应先阅读并遵循 `cli/docs/REVIEW.md`。该文档是 CLI 代码、文档、测试与 Agent Skills 变更的审查入口。

同时参考 `cli/CONTRIBUTING.md` 中的开发流程与 Makefile 入口；CLI 构建 / 测试前应确认本地 `go version` 符合 `cli/go.mod`。

本文件不定义问题严重程度，也不重复 `cli/docs/REVIEW.md` 的检查项。审查时应根据实际影响判断重要程度；若使用的 review 模式已有严重程度体系，按该体系表达即可。
