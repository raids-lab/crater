# GPU Portal Frontend

[![Pipeline Status](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/badges/main/pipeline.svg)](https://gitlab.act.buaa.edu.cn/gpu-portal/gpu-portal-frontend/-/commits/main)
[![Release Version](https://img.shields.io/badge/Release-0.0.0-blue)](https://codec.sensetime.com/)
[![Develop Version](https://img.shields.io/badge/Develop-0.0.0-orange)](http://10.151.166.71:8000/)

> [GPU 集群管理与作业调度 Portal 设计和任务分解](https://docs.qq.com/doc/DWENFVWpzSW16TGFV)

Portal of GPU Cluster Management and Job Scheduling based on K8S.

## Installation

This project uses node and pnpm. Go check them out if you don't have them locally installed.

- Install Node.js: [Win / Mac](https://nodejs.org/en/download) | [Linux](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions)
- Pnpm: `npm install -g pnpm@latest`

Check the version of Node and Pnpm:

```bash
node -v
# v18.17.0
pnpm -v
# 8.6.10
```

Now you can clone this project, and deploy it:

```bash
git clone git@gitlab.act.buaa.edu.cn:gpu-portal/gpu-portal-frontend.git
cd gpu-portal-frontend
pnpm install
pnpm run dev
# pnpm run build
```

The app will automatically reload if you change any of the source files.

## Development

If you are using _Visual Studio Code_, follow [Profiles in Visual Studio Code](https://code.visualstudio.com/docs/editor/profiles#_import) to import configure from `.vscode/React.code-profile`. Else if you are using any other IDE, you should install required plugins manually.

- Lint: [Pritter](https://prettier.io/docs/en/install)
- CSS: [tailwindcss](https://tailwindcss.com/docs/guides/vite)
- UI Kit: [shadcn](https://ui.shadcn.com/examples/dashboard)

## What's included

Within the download you'll see something like this:

```bash
gpu-portal-frontend
├── public/          # static files
│
├── src/             # project root
│   ├── assets/      # images, icons, etc.
│   ├── compoments/  # common components
│   ├── pages/       # application pages
│   ├── routes/      # routes config
│   ├── services/    # application apis
│   ├── utils/       # logger, store, etc.
│   ├── ...
│   ├── index.css    # tailwind styles
│   └── main.tsx     # react root
├── ...
├── index.html       # html template
└── package.json     # dependencies
```
