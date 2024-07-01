# Crater Document

该网站使用 [Docusaurus](https://docusaurus.io/)，一个现代的静态网站生成器构建。

### 安装

```bash
$ pnpm install
```

### 本地开发

```bash
$ pnpm start
```

该命令将启动一个本地开发服务器，并自动打开浏览器窗口。大多数更改会实时反映，无需重新启动服务器。

### 构建

```bash
$ pnpm build
```

该命令将生成的静态内容放入 `build` 目录，可以使用任何静态内容托管服务来进行托管。

### 部署

这部分代码还没来得及整理，目前部署方式是重启 K8S Deployment，会重新拉取代码并渲染。

TODO:

- [ ] 为部署镜像添加就绪探针，从而支持滚动更新
- [ ] 添加自动化部署流程文档
