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
# v20.8.1
pnpm -v
# 8.9.2
```

Now you can clone this project, and deploy it:

```bash
git clone git@gitlab.act.buaa.edu.cn:act-k8s-portal-system/gpu-portal-frontend.git
cd gpu-portal-frontend
pnpm install
pnpm dev
# pnpm build
```

The app will automatically reload if you change any of the source files.

## Development

If you are using _Visual Studio Code_, follow [Profiles in Visual Studio Code](https://code.visualstudio.com/docs/editor/profiles#_import) to import configure from `.vscode/React.code-profile`. Else if you are using any other IDE, you should install required plugins manually.

- Frame: [React](https://react.dev/learn)
  - State Manager: [recoil](https://recoiljs.org/zh-hans/)
  - Query Manager: [tanstack query](https://tanstack.com/query/latest)
- CSS: [tailwindcss](https://tailwindcss.com/docs/guides/vite)
- UI Kit:
  - [shadcn](https://ui.shadcn.com/examples/dashboard)
  - [flowbite](https://flowbite.com/docs/getting-started/react/)
  - [tanstack table](https://tanstack.com/table/v8)

## Deployment

To deploy the application, you can use Docker with Nginx. The Nginx configuration template can be found at `deploy/frontend/nginx.conf`. Use the following command as a reference:

```bash
pnpm build  # output /dist

docker run -d -p 8888:80 \
-v /home/lyl/workspace/nginx.conf:/etc/nginx/nginx.conf \
-v /home/lyl/workspace/dist:/usr/share/nginx/html \
nginx
```

This command launches a Docker container with Nginx and maps the container's port 80 to the host's port 8888. It also mounts the Nginx configuration file `nginx.conf` and the application's build files located in the dist directory.

Make sure to replace the paths `/home/lyl/workspace/nginx.conf` and `/home/lyl/workspace/dist` with the actual paths on your system where the Nginx configuration file and the application's build files are located.

Once the Docker container is running, you should be able to access the deployed application by navigating to http://192.168.5.60:8888 in your web browser.

## What's included

Within the download you'll see something like this:

```bash
gpu-portal-frontend
├── public/          # static files
│
├── src/             # project root
│   ├── assets/      # images, icons, etc.
│   ├── compoments/  # common components
│   ├── hooks/       # react hooks
│   ├── lib/         # shadcn's lib
│   ├── pages/       # application pages
│   ├── services/    # application apis
│   ├── utils/       # logger, store, etc.
│   ├── ...
│   ├── index.css    # tailwind styles
│   └── main.tsx     # app router
├── ...
├── index.html       # html template
└── package.json     # dependencies
```
