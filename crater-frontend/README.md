# GPU Portal Frontend

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

If you are using _Visual Studio Code_, follow [Profiles in Visual Studio Code](https://code.visualstudio.com/docs/editor/profiles#_import) to import configure from `.vscode/React.code-profile`.

If you are using _WebStorm_, you should install required plugins manually.

## [WIP] What's included

Within the download you'll find the following directories and files, logically grouping common assets and providing both compiled and minified variations. You'll see something like this:

```
coreui-free-react-admin-template
├── public/          # static files
│   └── index.html   # html template
│
├── src/             # project root
│   ├── assets/      # images, icons, etc.
│   ├── components/  # common components - header, footer, sidebar, etc.
│   ├── layouts/     # layout containers
│   ├── scss/        # scss styles
│   ├── views/       # application views
│   ├── _nav.js      # sidebar navigation config
│   ├── App.js
│   ├── ...
│   ├── index.js
│   ├── routes.js    # routes config
│   └── store.js     # template state example
│
└── package.json
```
