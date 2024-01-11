![Crater](./src/assets/Logo2.svg)

---

[![Pipeline Status](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/badges/main/pipeline.svg)](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/-/commits/main)
[![Release Version](https://img.shields.io/badge/Release-0.0.0-blue)](http://192.168.5.60:32088/)
[![Develop Version](https://img.shields.io/badge/Develop-0.0.0-orange)](http://192.168.5.60:8888/)

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，提供了一站式的 GPU 集群管理解决方案。要了解更多信息，请访问 [GPU 集群管理与作业调度 Portal 设计和任务分解](https://docs.qq.com/doc/DWENFVWpzSW16TGFV)。

## 1. 安装

在开始之前，请确保您的开发环境中已安装 Node.js 和 pnpm。如果尚未安装，请参考以下步骤：

- Node.js: [Win / Mac](https://nodejs.org/en/download) | [Linux](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions)
- Pnpm: `npm install -g pnpm@latest`

检查 Node 和 pnpm 是否安装成功：

```bash
node -v
# v20.8.1

pnpm -v
# 8.9.2
```

现在，您可以克隆本项目并在本地运行：

```bash
git clone git@gitlab.act.buaa.edu.cn:act-k8s-portal-system/gpu-portal-frontend.git
cd gpu-portal-frontend

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

## 2. 开发

如果您使用 _Visual Studio Code_ 开发本项目，可参考 [Profiles in Visual Studio Code](https://code.visualstudio.com/docs/editor/profiles#_import)，导入 `.vscode/React.code-profile` 路径下的配置文件。配置文件中包含项目所需的基本插件和配置，导入后即可直接开发。

如果您使用其他 IDE，则需要手动配置开发环境。

本项目使用的主要技术栈如下：

- Language: [TypeScript](https://www.typescriptlang.org/docs)
- Frontend Framework: [React](https://react.dev/learn)
  - State Management: [Recoil](https://recoiljs.org/zh-hans/)
  - Query Management: [Tanstack Query](https://tanstack.com/query/latest)
- CSS Framework: [Tailwind CSS](https://tailwindcss.com/docs/guides/vite)

本项目使用了以下 UI 库，您可以阅读其文档了解更多信息：

- [shadcn/ui](https://ui.shadcn.com/examples/dashboard): The primary headless component library used in the project.
  - [Oxidus](https://oxidus.vercel.app/): Theme Color Generator for _shadcn/ui_.
  - [v0 by Vercel](https://v0.dev/): Use generative AI to create copy-and-paste friendly React code based on shadcn/ui and Tailwind CSS.
- [Flowbite](https://flowbite.com/docs/getting-started/react/): When you need to _borrow_ some Tailwind CSS code.
- [Tanstack Table](https://tanstack.com/table/v8): Headless table component used in the project.

为了规范代码风格与提交风格，本项目使用了以下工具：

- [ESLint](https://eslint.org/docs/user-guide/getting-started): Linting tool for JavaScript and TypeScript.
- [Prettier](https://prettier.io/docs/en/index.html): Code formatter.
- [Commitlint](https://commitlint.js.org/#/): Linting tool for commit messages.
- [Commitizen](https://github.com/commitizen/cz-cli): The commitizen command line utility.
- [Husky](https://typicode.github.io/husky/#/): Git hooks.

## 3. 部署

### 3.1 使用 Docker 部署

为快速部署本项目，您可以使用基于 Docker 的 Nginx 托管。Nginx 配置模板位于 `deploy/nginx.conf`。参考以下命令：

```bash
pnpm build  # output /dist

docker run -d -p 8888:80 \
-v /home/lyl/workspace/nginx.conf:/etc/nginx/nginx.conf \
-v /home/lyl/workspace/dist:/usr/share/nginx/html \
nginx
```

此命令启动一个带有 Nginx 的 Docker 容器，并将容器的端口 80 映射到主机的端口 8888。它还挂载了 Nginx 配置文件 nginx.conf 和位于 dist 目录中的应用程序构建文件，以便通过 `docker restart` 命令快速更新。

确保将路径 `/home/lyl/workspace/nginx.conf` 和 `/home/lyl/workspace/dist`dist 替换为您的系统上实际的 Nginx 配置文件和应用程序构建文件所在的路径。

一旦 Docker 容器运行，您应该能够通过在 web 浏览器中访问 `[host_ip]:8888` ，以查看部署的应用程序。

### 3.2 使用 Kubernetes 部署

相关配置文件位于 `deploy/` 目录下。待补充。

## 4. 项目结构

```bash
gpu-portal-frontend
├── public/          # 静态文件
│
├── src/             # 项目根目录
│   ├── assets/      # 图片、图标等
│   ├── compoments/  # 通用组件
│   ├── hooks/       # 通用 Hook
│   ├── lib/         # 通用 Library
│   ├── pages/       # 页面
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

1. 在深色模式下，使用浏览器的自动填充功能时，INPUT 背景变为白色。相关讨论请参见 [Tailwind autofill: prefix on form inputs not working](https://github.com/tailwindlabs/tailwindcss/discussions/8679)。
