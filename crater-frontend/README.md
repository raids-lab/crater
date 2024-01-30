<p align="center">
<img src="./public/vite.svg" alt="Logo" width="200"/>
</p>
<h1 align="center">Crater</h1>

[![Pipeline Status](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/badges/main/pipeline.svg)](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/-/commits/main)
[![Develop Version](https://img.shields.io/badge/Develop-0.1.0-orange)](http://192.168.5.60:8888/)
[![Release Version](https://img.shields.io/badge/Release-0.1.0-blue)](http://192.168.5.60:32088/)

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
git clone git@gitlab.act.buaa.edu.cn:raids/resource-scheduling/crater/web-frontend.git
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

确保 `make.sh` 具有可执行权限（使用 `chmod +x make.sh`）。

- 使用 `./make.sh build "IMAGE_URL:VERSION"` 来在本地构建应用，然后构建并上传 Docker 镜像。
- 使用 `./make.sh deploy "IMAGE_URL:VERSION"` 来部署到 Kubernetes。

```bash
bash make.sh build harbor.act.buaa.edu.cn/crater/web-frontend:v0.1.3
bash make.sh deploy harbor.act.buaa.edu.cn/crater/web-frontend:v0.1.3
```

确保项目中有可运行的 pnpm 构建脚本，并且 dist 目录是构建输出的位置。脚本假设你有权限执行 Docker、kubectl 和 pnpm 命令。

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
