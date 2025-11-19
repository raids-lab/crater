# 🌋 Crater 前端开发文档（中文）

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，本仓库为其前端项目，主要用于提供：算力编排、作业管理、监控可视化、模型与数据集管理等一体化 Web 控制台。

## 🛠️ 环境准备

> [!NOTE]
> 请先安装 **Node.js** 和 **pnpm**：
>
> - Node.js 官方下载：[https://nodejs.org/en/download](https://nodejs.org/en/download)

推荐使用 [nvm](https://github.com/nvm-sh/nvm) 管理 Node.js 版本，便于在不同项目间切换。

安装完成后请确认版本：

```bash
node -v  # 推荐 v22.x 或更高
pnpm -v  # 推荐 v10.x 或更高
```

---

## 💻 本地开发指南

### 1. 编辑器配置

**VS Code 用户：**

1. 打开 VS Code
2. 通过 `Profiles > Import Profile` 导入仓库内 `.vscode/React.code-profile`
3. 安装推荐插件（会自动提示）

**其他编辑器用户：**

请手动配置（建议）：

- Prettier：统一代码风格
- ESLint：静态代码检查
- Tailwind CSS IntelliSense：Tailwind 类名智能提示

---

### 2. 克隆与初始化项目

```bash
git clone https://github.com/YOUR_USERNAME/crater.git
cd crater/frontend
pnpm install
```

---

### 3. 启动开发服务器

```bash
make run
```

执行成功后，即可在浏览器访问：

```text
http://localhost:5180
```

说明：

- `VITE_SERVER_PROXY_BACKEND`：后端 API 网关地址
- `VITE_SERVER_PROXY_STORAGE`：对象存储 / 文件服务地址
- 将 `VITE_USE_MSW` 设为 `true` 可启用本地 Mock（见后文）
- DevTools 配置用于开发调试，可按需关闭

---

## 🚀 核心技术栈

Crater Frontend 基于现代 React 技术栈构建，主要包括：

- **语言**：TypeScript
- **框架**：React 19
- **状态管理**：Jotai
- **数据请求**：TanStack Query v5
- **样式**：Tailwind CSS
- **UI 组件库**：
  - `shadcn/ui`：Headless + Tailwind 风格组件
  - `Flowbite`：Tailwind 模板与组件
  - `TanStack Table`：Headless 表格组件（适配复杂表格场景）

整体设计遵循：

- 组件化、可重用
- 类型安全
- 与后端 API 解耦
- 易于扩展、适配不同集群与业务场景

---

## 🧪 API Mock（MSW）

当后端服务尚未就绪或本地联调不便时，可通过 [MSW](https://mswjs.io/) 进行接口模拟，方便开发调试。

使用步骤：

1. 在 `.env.development` 中设置：

   ```env
   VITE_USE_MSW=true
   ```

2. 在 `src/mocks/handlers.ts` 中编写或扩展接口 Mock 逻辑：
   - 可根据后端接口路径定义对应的 `rest.get/post/put/delete` 等处理函数
   - 建议按模块拆分，保持 Mock 代码可维护

---

## 📦 依赖管理

查看依赖更新情况：

```bash
pnpm outdated
```

升级依赖：

```bash
pnpm update             # 安全的小版本更新
pnpm update --latest    # 包含大版本更新（请谨慎使用，注意破坏性变更）
```

### 更新 shadcn 组件

当需要同步最新 shadcn/ui 实现时，可批量更新现有组件：

```bash
for file in src/components/ui/*.tsx; do
  pnpm dlx shadcn@latest add -yo $(basename "$file" .tsx)
done
```

> 建议更新前确认组件是否有破坏性改动，并进行回归测试。

---

## 🚀 部署说明

生产环境推荐搭配 Crater 后端与 Kubernetes 集群，以 Helm 方式部署完整系统。

Crater 的 Helm Chart 仓库：

- `https://github.com/raids-lab/crater`

部署步骤（简要）：

1. 在目标集群中安装并配置 Helm
2. 按 Crater 主项目文档配置后端、存储及相关依赖
3. 构建前端产物并挂载至 Nginx/Ingress 等组件中对外提供访问

> 具体部署流程请参考 Crater 主项目（后端与 Helm Chart）文档。

---

## 📁 项目结构概览

```bash
src/
├── components/           # 通用组件
│   ├── custom/           # 业务定制组件
│   ├── layout/           # 布局组件（导航、侧边栏等）
│   └── ui/               # 基于 shadcn 的 UI 组件
├── hooks/                # 自定义 Hooks
├── lib/                  # 工具函数、通用库
├── pages/                # 路由页面
│   ├── Admin/            # 管理员后台页面
│   ├── Portal/           # 作业管理入口（如 Job、Notebook 等）
│   └── ...               # 其他功能模块页面
├── services/             # API 封装与请求逻辑
├── stores/               # 全局/模块状态（Jotai atoms 等）
├── types/                # 全局与模块 TypeScript 类型定义
└── ...
```

约定建议：

- 页面逻辑尽量保持轻量，将通用逻辑抽离到 `hooks/` 与 `lib/`
- API 调用统一经由 `services/` 管理，便于切换后端实现或添加鉴权逻辑
- 组件拆分遵循「UI 与业务解耦」原则，提升复用性

---

## 🐛 已知问题

1. **暗色模式下浏览器自动填充样式异常**
   - 表单在深色主题下，浏览器自动填充可能会出现白色背景。
   - 参考：TailwindCSS 讨论帖 [TailwindCSS#8679](https://github.com/tailwindlabs/tailwindcss/discussions/8679)
   - 如遇类似问题，可在全局样式中对 `:-webkit-autofill` 进行覆盖。

---

---

### ✍️ Commit Message 规范

格式：

```text
type(scope): subject
```

示例：

```text
feat(portal): add job submission form
fix(admin): resolve user role validation issue
docs(readme): update contribution guidelines
```

可用 `type`：

- `feat`：新功能
- `fix`：缺陷修复
- `docs`：文档更新
- `style`：代码风格调整（不影响逻辑）
- `refactor`：重构（无新功能或修复）
- `test`：测试相关
- `chore`：构建、依赖、脚本等非业务代码调整

`scope`（可选）用于标记影响范围：

- 示例：`portal`、`admin`、`ui`、`api` 等

---

## 🚨 问题反馈

提交 Issue 时请尽量提供完整信息，便于快速定位问题：

- 复现步骤（Step by Step）
- 期望行为与实际行为
- 截图或录屏（如果适用）
- 浏览器信息（如：Chrome 版本）、操作系统、分辨率等环境信息
- 如与后端交互相关，可附上接口返回示例（脱敏后）
