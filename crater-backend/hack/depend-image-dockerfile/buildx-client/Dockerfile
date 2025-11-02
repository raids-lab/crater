# 使用 Ubuntu 22.04 基础镜像
FROM docker.io/library/ubuntu:22.04

# 设置环境变量避免交互式安装
ENV DEBIAN_FRONTEND=noninteractive
ARG userNumber=1000
ARG groupNumber=1000
ARG fsGroupNumber=1000

# 创建用户和组
RUN groupadd --gid $groupNumber builder && \
    useradd --uid $userNumber --gid $groupNumber --create-home builder && \
    usermod -aG sudo builder

# 创建 docker 组并将用户加入
RUN groupadd --gid 1001 docker && \
    usermod -aG docker builder

# 安装依赖包
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    software-properties-common

# 添加 Docker 官方 GPG 密钥
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# 设置 Docker 仓库
RUN echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker CLI（不包含守护进程）
RUN apt-get update && apt-get install -y \
    docker-ce-cli \
    docker-buildx-plugin \
    git

# 保证/.docker文件夹持久化，保存buildx的配置信息
ENV DOCKER_CONFIG=/.docker
RUN mkdir -p /.docker/buildx && \
    chown -R $userNumber:$groupNumber /.docker && \
    chmod 770 /.docker && \
    chmod 770 /.docker/buildx

# 切换到非root用户
USER builder