#!/bin/bash

# 检查输入参数
if [ $# -ne 3 ]; then
    echo "Usage: $0 <namespace> <service_account_name> <secret_name>"
    exit 1
fi

NAMESPACE=$1
SA_NAME=$2

# 获取 Secret 名称
SECRET_NAME=$3

# 获取 CA 证书
CA_CERT=$(kubectl get secret $SECRET_NAME -n $NAMESPACE -o jsonpath='{.data.ca\.crt}')

# 获取令牌
TOKEN=$(kubectl get secret $SECRET_NAME -n $NAMESPACE -o jsonpath='{.data.token}' | base64 --decode)

# 生成 Kubeconfig
cat <<EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: $CA_CERT
    server: <your-kubernetes-api-server-url>
  name: my-cluster
contexts:
- context:
    cluster: my-cluster
    user: $SA_NAME-user
    namespace: $NAMESPACE
  name: $SA_NAME-context
current-context: $SA_NAME-context
users:
- name: $SA_NAME-user
  user:
    token: $TOKEN
EOF