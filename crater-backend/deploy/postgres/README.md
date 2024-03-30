# Postgres 数据库部署

使用开源库 [Postgres Operator](https://github.com/zalando/postgres-operator) 安装 Postgres 集群。

## Operator

> [Manual deployment setup on Kubernetes](https://postgres-operator.readthedocs.io/en/latest/quickstart/#manual-deployment-setup-on-kubernetes)

相比官方代码的改动：

1. 在 `postgres-operator` 命名空间安装 Operator
2. 更换 `postgres-operator.yaml` 镜像地址，从 `registry.opensource.zalan.do/acid/postgres-operator:v1.11.0` 替换为 `***REMOVED***/crater/postgres-operator:v1.11.0`

```bash
cd deploy/postgres

kubectl create namespace postgres-operator
kubectl create -f operator/configmap.yaml  # configuration
kubectl create -f operator/operator-service-account-rbac.yaml  # identity and permissions
kubectl create -f operator/postgres-operator.yaml  # deployment
```

## Cluster

> [Create a Postgres cluster](https://postgres-operator.readthedocs.io/en/latest/quickstart/#create-a-postgres-cluster)

相比官方代码的改动：

1. 在 `crater` 命名空间安装 Cluster
2. 更换镜像地址，从 `ghcr.io/zalando/spilo-16:3.2-p2` 替换为 `***REMOVED***/crater/spilo-16:3.2-p2`
3. 修改 Team ID 为 `crater`，数据库用户为 `backend`
4. 使用 `rook-ceph-block` 作为存储类

```bash
cd deploy/postgres

kubectl apply -f cluster/minimal-postgres-manifest.yaml
```

完成后，为 Master Pod 的 Service 添加选择器（这部分还需要读一下官方代码，为什么没有默认选择）：

```yaml
  selector:
    application: spilo
    cluster-name: postgres-cluster
    spilo-role: master
```

## Connect to DB

- [Connect to PostgreSQL](https://postgres-operator.readthedocs.io/en/latest/user/#connect-to-postgresql)

通过其中一个数据库 pod（例如 master）上的 `port-forward`，您可以从您的计算机连接到 PostgreSQL 数据库。

```bash
# get name of master pod of postgres-cluster (crater)
export PGMASTER=$(kubectl get pods -o jsonpath={.items..metadata.name} -l application=spilo,cluster-name=postgres-cluster,spilo-role=master -n crater)

# set up port forward
kubectl port-forward $PGMASTER 6432:5432 -n crater
```

打开另一个 CLI 并使用例如连接到数据库psql 客户端。当与 foo_user 用户等清单角色连接时，从创建 postgres-cluster 时生成的 K8s 密钥中读取其密码。由于默认情况下拒绝非加密连接，因此将 SSL 模式设置为 `require`：

```bash
export PGPASSWORD=$(kubectl get secret -n crater postgres.postgres-cluster.credentials.postgresql.acid.zalan.do -o 'jsonpath={.data.password}' | base64 -d)
export PGSSLMODE=require
psql -U postgres -h localhost -p 6432
```