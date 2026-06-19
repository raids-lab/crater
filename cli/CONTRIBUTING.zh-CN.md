[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# 参与 Crater CLI 开发

Crater CLI 采用文档驱动开发。修改代码前，请先确认你触及的行为或规则由哪份文档定义。这样可以让实现、测试、Agent Skills 与用户可见的命令契约保持一致。

## 1. 理解契约

先找到负责本次改动的文档：

- 新增或修改命令、flag、位置参数、stdout/stderr 行为、JSON 字段、错误或退出码预期时，先看 [docs/COMMANDS.md](docs/COMMANDS.md)。
- 修改跨命令规则、共享输出契约、错误分类、快照要求、i18n 规则、补全规则、沙箱行为或开发流程时，先看 [docs/SPEC.md](docs/SPEC.md)。
- 修改包职责、模块边界、请求流程、状态/凭据访问或测试基础设施时，先看 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)。
- 阶段性开发完成前，阅读 [docs/REVIEW.md](docs/REVIEW.md)，按其中内容检查本次变更。

如果计划中的代码改动没有对应契约，请先更新合适的文档。如果文档是正确的而代码不一致，应让代码和测试回到文档约定。

你可以随时在 `cli/` 目录下运行 `make help` 查看本地工作流命令。

执行 CLI 构建或测试 target 前，先运行 `go version`，确认它与 `cli/go.mod` 一致（当前为 `1.25.4`）。

## 2. 实现改动

改动应尽量限制在对应命令域或共享模块内。

跨命令规则和共享实现约束见 [docs/SPEC.md](docs/SPEC.md)，用户可见命令契约见 [docs/COMMANDS.md](docs/COMMANDS.md)，包边界和请求流程见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)。如果修改 Agent Skills，遵守 `docs/SPEC.md` 中的 Skills 规则。

需要手动试用 CLI 时，先构建本地二进制：

```bash
make build
```

该命令会执行 `go mod tidy` 并构建本地 `./crater` 二进制。

涉及用户可见 CLI 行为时，要求开发者按 `docs/COMMANDS.md` 和 `docs/SPEC.md` 手动执行关键命令路径。Agent 运行的测试可以降低风险，但不能替代开发者对平台契约的人工验证。

## 3. 测试改动

根据改动内容选择测试。

如果修改了解析、映射、补全、输出 helper、状态 helper 或测试工具等纯逻辑，运行单元测试目标。它运行包级单元测试，并排除快照测试：

```bash
make unit-test
```

如果用户可见 CLI 输出不应变化，运行：

```bash
make snapshot-check
```

如果命令契约有意变化，重新生成快照：

```bash
make snapshot-update
```

然后按 `docs/SPEC.md` 和 `docs/REVIEW.md` 人工审查 `cli/testdata/snapshots/` 的 diff。Golden 文件必须通过这种方式生成，不要手工编辑。

除纯文档改动且不影响生成文件或代码外，创建或更新 PR 前应运行完整 CLI 测试目标。该目标会同时运行单元测试与快照校验：

```bash
make test
```

为与其他子项目保持一致，`make pre-commit-check` 也可用，目前等价于 `make test`：

```bash
make pre-commit-check
```

## 4. 提交前检查

确认以下事项：

- `cli/docs/` 中相关文档与实现一致。
- 测试覆盖了本次变更引入的风险。
- 如果更新了 golden 文件，它们由 `make snapshot-update` 生成，并且已经人工审查。
- README 面向普通用户，不包含内部开发指引。
- 如果修改了 Agent Skills，它们遵守 `docs/SPEC.md`，只说明如何使用已有 CLI 契约，不单独定义新的命令行为。
