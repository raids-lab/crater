# Crater Web Frontend

[![Pipeline Status](https://gitlab.act.buaa.edu.cn/raids/resource-scheduling/crater/web-frontend/badges/main/pipeline.svg)](https://gitlab.act.buaa.edu.cn/raids/resource-scheduling/crater/web-frontend/-/commits/main)
[![Release Version](https://img.shields.io/badge/Release-0.1-blue)](https://crater.act.buaa.edu.cn/)

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，提供了一站式的 GPU 集群管理解决方案。要了解更多信息，请访问 [GPU 集群管理与作业调度 Portal 设计和任务分解](https://docs.qq.com/doc/DWENFVWpzSW16TGFV)。

## 1. 环境准备

在开始之前，请确保您的开发环境中已安装 Node.js 和 pnpm。如果您参与的前端项目较多，建议您使用 [nvm](https://github.com/nvm-sh/nvm) 来管理 Node.js 版本。nvm 的安装与升级方法请参考 [nvm 官方文档](https://github.com/nvm-sh/nvm?tab=readme-ov-file#installing-and-updating)。否则，您可以直接安装 Node.js 和 pnpm：

- Node.js: [Win / Mac](https://nodejs.org/en/download) | [Linux](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions)
- Pnpm: `npm install -g pnpm@latest`

检查 Node 和 pnpm 是否安装成功：

```bash
node -v
# v20.x.x

pnpm -v
# 8.x.x
```

## 2. 开发

### 2.1 项目配置

> 这一步非常重要！请您配置后再进行开发工作！

如果您使用 Visual Studio Code 开发本项目，请参考 [Profiles in Visual Studio Code](https://code.visualstudio.com/docs/editor/profiles#_import)，导入 `.vscode/React.code-profile` 路径下的配置文件。配置文件中包含项目所需的基本插件和配置，导入后即可直接开发。

如果您使用其他 IDE，则需要手动配置开发环境。您可以查看 `.vscode/React.code-profile` 文件内容，看看有哪些需要安装的插件或设置（如 Prettier, Eslint 等）。

现在，您可以克隆本项目并在本地运行：

```bash
git clone git@gitlab.act.buaa.edu.cn:raids/resource-scheduling/crater/web-frontend.git
cd gpu-portal-frontend

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

### 2.2 主要技术栈

本项目使用的主要技术栈如下：

- 语言：[TypeScript](https://www.typescriptlang.org/docs)
- 前端框架：[React](https://react.dev/learn)
  - 状态管理：[Recoil](https://recoiljs.org/zh-hans/)
  - 查询管理：[Tanstack Query](https://tanstack.com/query/latest)
- CSS 框架：[Tailwind CSS](https://tailwindcss.com/docs/guides/vite)

本项目使用了以下 UI 库，您可以阅读其文档了解更多信息：

- [shadcn/ui](https://ui.shadcn.com/examples/dashboard)：项目中使用的主要无头部件（headless component）库。
  - [Oxidus](https://oxidus.vercel.app/)：_shadcn/ui_ 的主题颜色生成器。
  - [v0 by Vercel](https://v0.dev/)：使用生成式 AI 基于 shadcn/ui 和 Tailwind CSS 创建可复制粘贴的 React 代码。
- [Flowbite](https://flowbite.com/docs/getting-started/react/)：当你需要借用一些 Tailwind CSS 代码时使用。
- [Tanstack Table](https://tanstack.com/table/v8)：项目中使用的无头表格（headless table）组件。

为了规范代码风格与提交风格，本项目使用了以下工具：

- [ESLint](https://eslint.org/docs/user-guide/getting-started)：用于 JavaScript 和 TypeScript 的代码检查工具。
- [Prettier](https://prettier.io/docs/en/index.html)：代码格式化工具。
- [Commitlint](https://commitlint.js.org/#/)：用于提交信息的代码检查工具。
- [Commitizen](https://github.com/commitizen/cz-cli)：commitizen 命令行工具。
- [Husky](https://typicode.github.io/husky/#/)：Git 钩子。

### 2.3 Mock 接口

当前端需要开发新功能，但后端相应接口还没有开发完成时，需要在前端模拟接口。项目使用了 MSW 工具。

- [MSW](https://mswjs.io/)：an API mocking library that allows you to write client-agnostic mocks and reuse them across any frameworks, tools, and environments.

要开启此功能，可以修改 `.env.development` ，设置 `VITE_USE_MSW=true`，请避免将此改动同时提交到代码中。

```bash
VITE_API_BASE_URL="http://localhost:8099/"
VITE_USE_MSW=false
```

之后，在 `src/mocks/handlers.ts` 中添加新的处理函数。

### 2.4 依赖管理

本项目使用 pnpm 作为依赖管理工具，定期更新依赖是一个好习惯。

- 可以使用 `pnpm outdated` 命令查看是否有新版本的依赖包。
- 通过 `pnpm update` 命令，可以更新有小版本更新的依赖包。
- 通过 `pnpm update --latest` 命令，将会更新所有依赖项到它们的最新版本（包括大版本更新）。这可能会导致依赖项之间的不兼容性问题，需要仔细评估。

## 3. 部署

要部署到集群中时，只需打上相应的标签。

```bash
git tag v0.x.x
git push origin --tag
```

GitLab CI/CD 会根据标签进行部署。

## 4. 项目结构

```bash
gpu-portal-frontend
├── public/          # 静态文件
│
├── src/             # 项目根目录
│   ├── assets/      # 静态资源
│   ├── compoments/  # 组件
│   │   ├── custom/  # 自定义组件
│   │   ├── icon/    # 图标组件
│   │   ├── layout/  # 整体布局
│   │   └── ui/      # Shadcn 生成组件
│   ├── hooks/       # 通用 Hook
│   ├── lib/         # 通用 Library
│   ├── mocks/       # 后端接口模拟
│   ├── pages/       # 页面
│   │   ├── Admin/   # 管理员
│   │   ├── Portal/  # AI Job
│   │   ├── PortalR/ # Recommend DL Job
│   │   └── ...
│   ├── services/    # 后端 API
│   ├── utils/       # 日志记录器、存储等
│   ├── ...
│   ├── index.css    # Tailwind 根样式
│   └── main.tsx     # 应用程序入口
├── ...
├── index.html       # HTML 模板
└── package.json     # 依赖
```

## 5. 未解决的问题

1. 在深色模式下，使用浏览器的自动填充功能时，INPUT 背景变为白色。相关讨论：[Tailwind autofill: prefix on form inputs not working](https://github.com/tailwindlabs/tailwindcss/discussions/8679)。
