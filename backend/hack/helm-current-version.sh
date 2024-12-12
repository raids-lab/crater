#!/bin/bash
set -e

# 定义命名空间
NAMESPACE="crater"
WEBDAV_NAMESPACE="crater-workspace"
# values.yaml 文件路径，根据实际情况调整
VALUES_FILE="charts/crater/values.yaml"

# 检查是否安装了 yq
if ! command -v yq &> /dev/null; then
    echo "yq 未安装，请先安装 yq 再运行此脚本。"
    echo "安装命令（Linux AMD64）:"
    echo "  wget https://github.com/mikefarah/yq/releases/download/v4.34.1/yq_linux_amd64 -O yq"
    echo "  chmod +x yq"
    echo "  sudo mv yq /usr/local/bin/"
    exit 1
fi

# 获取 web-backend 当前使用的镜像
backend_image=$(kubectl get deploy crater-web-backend -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "web-backend 镜像：$backend_image"

# 获取 web-frontend 当前使用的镜像
frontend_image=$(kubectl get deploy crater-web-frontend -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "web-frontend 镜像：$frontend_image"

# 获取 webdav（对应 storage 部分）当前使用的镜像
storage_image=$(kubectl get deploy webdav-deployment -n $WEBDAV_NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "webdav 镜像：$storage_image"

# 提取镜像的 tag（假设镜像名格式为 repository:tag）
backend_tag=$(echo $backend_image | awk -F: '{print $2}')
frontend_tag=$(echo $frontend_image | awk -F: '{print $2}')
storage_tag=$(echo $storage_image | awk -F: '{print $2}')

# 使用 yq 更新 values.yaml 中对应的 tag
yq eval ".web.backend.image.tag = \"$backend_tag\"" -i "$VALUES_FILE"
yq eval ".web.frontend.image.tag = \"$frontend_tag\"" -i "$VALUES_FILE"
yq eval ".web.storage.image.tag = \"$storage_tag\"" -i "$VALUES_FILE"

echo "镜像信息已在 $VALUES_FILE 中替换完成。"