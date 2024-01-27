#!/bin/bash

# 检查 docker 和 kubectl 是否安装
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        echo "Error: docker is not installed." >&2
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        echo "Error: kubectl is not installed." >&2
        exit 1
    fi
}

# 构建并上传 Docker 镜像
build_and_push_image() {
    local image_url=$1
    docker build -t "$image_url" .
    docker push "$image_url"
}

# 修改 nginx.yaml 并部署到 Kubernetes
deploy_to_k8s() {
    local image_url=$1
    sed -i "s|image: .*|image: $image_url|" ./deploy/nginx.yaml
    kubectl apply -f ./deploy/nginx.yaml
}

# 主逻辑
main() {
    check_dependencies

    local command=$1
    local image_url=$2

    case $command in
        build)
            build_and_push_image "$image_url"
            ;;
        deploy)
            deploy_to_k8s "$image_url"
            ;;
        *)
            echo "Usage: $0 {build|deploy} <IMAGE_URL:VERSION>"
            exit 1
            ;;
    esac
}

main "$@"