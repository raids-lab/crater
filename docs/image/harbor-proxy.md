# 镜像代理

:::warning

我们的代理服务器经常出现故障，建议您自行下载镜像，并手动上传到 Harbor 仓库中，详情见 [手动上传](./upload.md) 一节。

:::

## 为什么需要镜像仓库代理

#### 科学上网

由于科学上网的缘故，国内无法直接访问到外网的一些镜像仓库（如 gcr.io、docker.io 等），实验室 Harbor 配置了代理，因此用户可以借助 Harbor 的代理来拉取到外网的镜像

#### 避免封禁

实验室可能会经常使用某些镜像，而用户频繁的拉取可能会被某些仓库封禁（如 nvcr.io），因此缓存到本地能避免此问题

#### 加速拉取

对于外网镜像，通过 Harbor 先拉取到本地进行缓存，用户使用时走实验室内网拉取镜像更快

## 提供镜像代理的仓库

目前提供代理的仓库有以下几种，建议大家尽量走代理拉取镜像，若无所需的镜像仓库代理，请联系@黄世一，及时增加新的代理仓库：

| 镜像仓库          | 仓库地址                  |
| ----------------- | ------------------------- |
| nvcr.io           | https://nvcr.io           |
| docker.elastic.co | https://docker.elastic.co |
| registry.k8s.io   | https://registry.k8s.io   |
| ghcr.io           | https://ghcr.io           |
| k8s.gcr.io        | https://k8s.gcr.io        |
| gcr.io            | https://gcr.io            |
| quay.io           | https://quay.io           |
| docker.io         | https://hub.docker.com    |

## 使用方法

使用时，直接在原镜像 URL 前加上：`crater-harbor.act.buaa.edu.cn`即可，例：

原 URL：`docker.io/library/ubuntu:latest`

增加代理后：`crater-harbor.act.buaa.edu.cn/docker.io/library/ubuntu:latest`
