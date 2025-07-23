FROM docker.io/library/alpine:3.21.0

# 安装 wget 和 tar，下载并安装 nerdctl 二进制文件
RUN apk add --no-cache wget tar && \
    NERDCTL_VERSION=1.7.7 && \
    TARGETARCH=$(uname -m) && \
    case ${TARGETARCH} in \
        x86_64) TARGETARCH="amd64" ;; \
        aarch64) TARGETARCH="arm64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    wget -O /tmp/nerdctl.tar.gz "https://github.com/containerd/nerdctl/releases/download/v${NERDCTL_VERSION}/nerdctl-${NERDCTL_VERSION}-linux-${TARGETARCH}.tar.gz" && \
    tar -xzf /tmp/nerdctl.tar.gz -C /tmp && \
    cp /tmp/nerdctl /usr/local/bin/nerdctl && \
    rm -rf /tmp/nerdctl.tar.gz /tmp/nerdctl && \
    apk del wget tar
COPY ./snapshot.sh /snapshot.sh

# 将脚本设置为可执行
RUN chmod +x /usr/local/bin/nerdctl /snapshot.sh

# 设置工作目录为根目录
WORKDIR /

# Usage: snapshot.sh \
# [--namespace NAMESPACE] \
# [--pod-name PODNAME] \
# [--container-name CONTAINERNAME] \
# [--image-url IMAGE_URL] \
# [--size-limit SIZE_LIMIT_IN_GiB] \
# [--dry-run]
CMD ["/snapshot.sh"]