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

CLI 所用 API 变化时，须使用根贡献文档的 [CLI / 后端 API 兼容版本](../docs/zh-CN/CONTRIBUTING.md#cli--后端-api-兼容版本) 决策表。只有 API 契约本身发生变化时，后端和 CLI 的 `APIVersion` 才同步更新。CLI 开始强依赖契约中已有能力时，`APIVersion` 不变；仅在没有回退路径时，才将 `MinSupportedBackendAPIVersion` 提升至首次提供该能力的版本。PR 描述或验证记录必须写明这两项决策；不调整时也要说明原因。

需要手动试用 CLI 时，先构建本地二进制：

```bash
make build
```

该命令会执行 `go mod tidy` 并构建本地 `./crater` 二进制。

正式发布或打包构建应传入 CLI 产品版本，例如 `make build APP_VERSION=0.4.0`；该值会写入标准 `User-Agent` Header。本地构建默认为 `dev`。

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

npm 发布辅助脚本有独立的单元测试：

```bash
make npm-test
```

`make pre-commit-check` 是本地聚合检查，会同时运行 `make test` 和 `make npm-test`；CI 则将两类职责拆分到不同 job：

```bash
make pre-commit-check
```

## 4. 发布维护

全仓库发布触发见根文档 [发布 Workflow](../docs/zh-CN/CONTRIBUTING.md#发布-workflow)。CLI 遵守同一划分：`main` 更新不发布 CLI 产物；只有精确 `vX.Y.Z` tag 才发布 npm 包。不要用 GitHub Release 挂 CLI 资产，也不要用它启动其它 workflow。

CLI 发布自动化包含两个入口：

- `cli-pr.yml` 先运行 `Check CLI`（`make test`），再运行 `Check npm packaging`（npm 打包脚本测试、六目标交叉编译、`npm pack`，以及在 Linux 上安装入口包）。
- `cli-release.yml` 只接受精确的 `vX.Y.Z` tag，并按“平台包优先、入口包最后”的顺序发布 npm。它不会创建或更新 GitHub Release。

精确的正式发布 tag 是唯一发布入口，同时还会启动现有的前端、后端、Storage 和 Helm workflow。只推送一次 tag，等待该次正式发布结束后再推送下一个；不要移动已经用于正式发布的 tag。若需要 GitHub Release，只用于人工撰写更新说明，不会触发任何 workflow，也不再挂 CLI 二进制。

npm 分发包含入口包 `@raids-lab/crater-cli` 和以下可选原生平台包：

- `@raids-lab/crater-cli-darwin-arm64`
- `@raids-lab/crater-cli-darwin-x64`
- `@raids-lab/crater-cli-linux-arm64`
- `@raids-lab/crater-cli-linux-x64`
- `@raids-lab/crater-cli-win32-arm64`
- `@raids-lab/crater-cli-win32-x64`

### npm 首次发布

只有包已经存在时，才能配置 npm Trusted Publishing 或 staged publishing。因此，`cli-release.yml` 的首次发布版本采用直接发布，不会暂停等待批准。创建第一个正式 tag 前：

1. 为发布所用 npm 账号启用 2FA，并确认它可以在 `@raids-lab` 下发布公开包。
2. 创建一个短期有效、仅限 `@raids-lab` scope 和上述包发布权限的 granular npm access token。CI 直接发布所用 token 必须能够在无人交互输入 OTP 的情况下完成发布。
3. 将它保存为仓库 Actions secret `NPM_TOKEN`。不得把 token 写入文件、命令输出、Issue、PR 或 workflow input。
4. 人工确认目标 tag 和已经通过的 CLI PR 检查。推送匹配 tag 后会立即启动前端、后端、Storage、Helm、CLI 以及不可逆的 npm 发布 workflow。

若首次发布中途只完成部分包，发布器可以安全重跑：它会查询 registry，跳过已经公开的相同 package/version，继续发布剩余平台包，并最后发布入口包。npm 一旦接受某个 package/version，该组合永远不能再次使用。

### 迁移到带暂存审批的 Trusted Publishing

七个包都完成首次发布后，在下一个版本前通过单独的、经过审查的改动完成迁移：

1. 使用 npm CLI 11.15 或更新版本以及受 2FA 保护的 npm 登录会话，为每个包配置 GitHub Trusted Publisher。仓库填写 `raids-lab/crater`，workflow 文件填写 `cli-release.yml`，只授予 `npm stage publish`，不要授予直接 `npm publish`。
2. 将发布器从直接 `npm publish` 调整为 `npm stage publish`，移除首发 token 检查，并把“等待 live registry 可见”的逻辑改成适用于 staged submission 的检查。使用 2FA 审批时先处理六个平台包，最后处理入口包。
3. 完整验证一次暂存发布后，删除 GitHub secret `NPM_TOKEN`、吊销临时 npm token，并把每个包设为要求 2FA 且禁止传统 token 发布。

不要只修改 npm 侧权限而继续保留 workflow 的直接发布逻辑；否则下一次发布会在构建完成后失败。

## 5. 提交前检查

确认以下事项：

- `cli/docs/` 中相关文档与实现一致。
- 测试覆盖了本次变更引入的风险。
- 如果更新了 golden 文件，它们由 `make snapshot-update` 生成，并且已经人工审查。
- README 面向普通用户，不包含内部开发指引。
- 如果修改了 Agent Skills，它们遵守 `docs/SPEC.md`，只说明如何使用已有 CLI 契约，不单独定义新的命令行为。
- CLI 所用 API 的变更已明确记录当前版本和最低后端版本的决策。
