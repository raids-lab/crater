FROM crater-harbor.act.buaa.edu.cn/docker.io/library/alpine:3.21.0

# 从 ./bin/nerdctl 拷贝到镜像内
COPY ./bin/nerdctl /usr/local/bin/nerdctl
COPY ./deployments/nerdctl/snapshot.sh /snapshot.sh

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