[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# 为 Crater Frontend 贡献

本文档是 `frontend/` 开发的完整规范。先阅读仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md) 的全局守则与贡献流程，再按本文档开发。

修改 React 路由、组件、hooks、前端 API 调用、i18n 文案、UI 行为或前端构建 / lint 流程时，使用本文档。

## 技术栈

- **语言**：TypeScript
- **框架**：React 19
- **路由**：TanStack Router
- **数据请求**：TanStack Query v5
- **状态管理**：Jotai
- **样式**：Tailwind CSS
- **UI 库**：shadcn/ui（无头组件）、Flowbite（Tailwind 模板）、TanStack Table（无头表格）

## 本地调试

- 安装 Node.js 与 pnpm（推荐用 [nvm](https://github.com/nvm-sh/nvm) 管理 Node 版本）：`node -v` 应为 v22+，`pnpm -v` 应为 v10+。
- VS Code 用户可通过 `Profiles > Import Profile` 导入 `.vscode/React.code-profile` 并安装推荐扩展；其他 IDE 手动配置 Prettier、ESLint、Tailwind CSS IntelliSense。

多数前端开发应一并启动前端和后端进行调试。后端可以通过自身配置连接测试集群已有依赖，因此普通 UI 开发不要尝试启动完整 Crater 集群或本地数据库。只有任务适合接口模拟时才使用 MSW。

前端开发服务默认通过 `VITE_SERVER_PROXY_BACKEND` 将主后端 API 代理到 `http://localhost:8088/`，并通过 `VITE_SERVER_PROXY_STORAGE` 将存储 API 代理到 `http://localhost:7320/`。调试文件管理、数据集、模型、上传或下载等存储相关页面和流程时，需要先在后端目录运行 `make run-storage` 启动 storage server。

`make run` 只会从 `.env.development` 中显式的 `PORT=...` 配置读取开发服务端口。如果没有该配置，会回退到 `5180`。`VITE_SERVER_PROXY_BACKEND` 和 `VITE_SERVER_PROXY_STORAGE` 等代理变量不会被当作端口配置。

```bash
cd frontend
pnpm install
make run
```

按全局守则，构建 / lint 走 `make`，**由维护者本机执行**；需要验证时输出命令与原因即可。可自动修复的 lint / 格式问题优先用 `make lint-fix` 或开发者常用的 `pnpm lint --fix`；全量 Prettier 格式化用 `make format`，翻译文件格式化用 `make format-translation`。

### API Mock（MSW）

开发期可用 MSW 模拟接口：在 `.env.development` 设 `VITE_USE_MSW=true`，在 `src/mocks/handlers.ts` 添加 handler。推荐通过仓库根统一配置管理维护 `.env.development`（见根 CONTRIBUTING）。

### 依赖管理

```bash
pnpm outdated            # 查看可更新
pnpm update              # 小版本更新
pnpm update --latest     # 大版本更新（谨慎）
```

更新 shadcn 组件：

```bash
for file in src/components/ui/*.tsx; do
  pnpm dlx shadcn@latest add -yo $(basename "$file" .tsx)
done
```

## 目录结构

```
src/
├── components/           # 可复用组件
│   ├── ui/               # shadcn 组件
│   ├── ui-custom/        # 自定义样式层组件
│   ├── custom/           # 自定义业务组件
│   └── layout/           # 布局
├── hooks/                # 自定义 hooks
├── services/             # API 服务
├── routes/               # 基于路由的页面
└── ...
```

## 组件复用（核心规范）

- 新建前优先复用已有 UI、业务、表单与 hook 基础能力。先检查 `src/components/ui-custom/`（样式层）、`src/components/form/`（表单控件与 metadata form）、`src/components/`（业务组件）和 `src/hooks/`。
- 只有现有组件、表单控件或 hook 无法满足行为、布局或领域模型时，才新建；新抽象优先贴近当前功能，除非已经有明确复用需求。
- **修改被广泛引用的公共组件必须非常谨慎**：先做风险评估、检查所有引用、向维护者说明充分理由后再改。涉及高复用基础组件、表单控件、metadata form、hook 或 `ui-custom/` 的变更，需明确确认对现有引用页面的影响范围与兼容性，并要求开发者人工抽查代表性受影响页面。

## Hooks（核心规范）

- 判断当前使用的身份用 `src/hooks/use-admin.tsx` 的 `useIsAdmin()`，不要自行实现。
- 实现功能前先检查 `src/hooks/` 是否已有可直接复用的 hook。

## 接口与错误

- 管理员视图调带 `admin` 前缀的管理员接口，普通用户调不带前缀的用户接口（与后端路由对应）。
- API 错误使用后端响应信封 `code`、`data`、`msg`。默认 UI 应保留有助于用户理解失败原因的后端 `msg`；现有错误组件支持时，同时保留 HTTP 状态和业务码等稳定排查事实。
- 将 `src/services/error_code.ts` 作为前端业务错误码和 group 的生成参考。它由 `src/services/generator.py` 解析后端定义生成，主要来源是 `backend/internal/bizerr/groups.go`。不要在 UI 代码或文档中复制数字错误码表；代码里导入生成常量，文档叙述中只把具体 code 当示例使用。
- 前端**默认不识别业务错误码**。无逻辑上特殊处理的必要时，使用共享错误处理路径（`showErrorToast`、`handleApiErrorByCode`、`markApiErrorHandled`），不要用页面局部的泛化 toast 覆盖后端信息。
- 只有页面需要改变交互行为时，才按业务码做特殊处理，例如高亮字段、展示冲突解决流程、静默认证重试提示，或引导管理员检查依赖。此时应处理 `src/services/error_code.ts` 中最窄的生成错误码；错误被页面消费后调用 `markApiErrorHandled`，并保留能继续展示后端 `msg` 的兜底。
- 面向用户的错误文案应回答“什么失败了、用户下一步能做什么”。面向管理员的流程还应保留足够排查事实，例如受影响资源、HTTP 状态、业务码、后端消息，以及页面已有且可安全展示的关联 ID 或对象标识。
- 不要在 `catch` / `onError` 中吞掉错误、只写 console，或用“操作失败”这类泛化文案替代可行动的后端消息。如果后端消息不安全或不可行动，应修正后端错误契约，而不是在前端隐藏。
- 非幂等操作（创建、更新、删除、停止、锁定 / 解锁、配额变更等）执行前必须展示确认弹出框。弹窗应清楚说明操作对象和后果，并复用项目现有 Dialog / AlertDialog 模式。
- 对耗时可能较长的非幂等请求，还必须增加 Loading 状态并禁用相关按钮，避免重复提交。
- 作业创建、克隆作业与作业模板流程依赖 `src/components/form/types.ts` 中 `MetadataForm*` 生成的带版本模板 JSON（`version`、`type`、`data`），并由 `src/utils/form.ts` 中的模板导入 helper 解析。修改持久化作业配置字段或表单数据结构时，必须判断是否提升对应 `MetadataForm*` version。如果旧模板仍需可用，必须注册“上一版本 -> 新版本”的迁移函数；之后更旧版本通过相邻版本迁移链逐步升级。不支持的更早版本必须清晰失败，不能静默当作当前结构解析。任何新增的作业模板数据、克隆 source 或 template source 加载入口都必须复用共享模板导入 / 迁移 helper；不要为单个入口手写兼容，也不要把作业模板迁移套到非作业模板 payload 的任意 JSON 上。

## 多语言（核心规范）

- **禁止硬编码文本**，必须接入项目多语言方案；标签先只写中文。
- 翻译 key 必须使用英文语义 key，不能使用中文原文作为 key。遵循当前 dotted-key 风格，并按功能、页面或组件放入合适 domain，例如 `navigation.*`、`jobs.*`、`accountDetail.*`、`accountForm.*`、`adminJobOverview.*`。
- 不要新增中文翻译 key。当前 locale 文件中仍有部分中文 key 属于历史债务；除非当前任务明确涉及这些文案迁移，否则不要顺手大范围修改。
- 新增或修改可翻译文案时，必须在同一次改动中同步更新所有语言的 `translation.json`。翻译要准确，并保持项目专有名词在各语言间一致。

## 体验与一致性

- 新页面保持与现有页面一致的布局、风格与配色（参考现有页面布局）。
- 新页面**不要忘记设置面包屑导航**。
- 对不易理解的输入项、开关或配置项，在标签 / 标题旁添加小问号帮助图标，并通过鼠标悬浮 tooltip 解释它的作用、适用场景、关键机制或影响。不要假设平台用户或管理员天然理解云计算、Kubernetes、调度、存储、网络或系统领域的概念和专有名词。
- 关注人文关怀细节（如按钮顺序），必要时针对窄屏做响应式适配。

## 提交 Frontend 改动前

遵循根 CONTRIBUTING 的 Commit 规范（`type(scope): subject`，scope 如 `portal`、`admin`、`ui`、`api`）。报告缺陷时请附复现步骤、期望与实际行为、截图（如有）、浏览器 / OS 版本。

改动涉及前端界面时，PR 中必须附受影响界面状态的截图。截图属于开发者人工验证的一部分，应与 PR 测试部分说明的页面、角色和操作对应。

创建或更新 PR 前：

- 运行相关 `make` 检查，通常为 `make pre-commit-check`。
- 对变更过的 API 错误路径，验证通用错误展示和所有按业务码特殊处理的分支，包括用户看到的消息，以及管理员可用于排查的事实。
- 确认所有可见文案接入 i18n，且所有语言 `translation.json` 已同步。
- 确认新增或修改的翻译 key 是英文语义 key，并放在合适 domain。
- 确认受影响页面已经由开发者人工检查，并记录角色、页面、操作和观察结果，供 PR 描述使用。
- 前端 / UI 改动附界面截图。

## 已知问题

- 暗色模式输入框样式：浏览器自动填充会在暗色模式下产生白色背景（[TailwindCSS#8679](https://github.com/tailwindlabs/tailwindcss/discussions/8679)）。
