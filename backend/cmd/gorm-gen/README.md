# Crater Gorm Gen

在生成代码前，首先通过 Kubectl 将集群中的数据库转发到本地。

`generate.go` 默认将连接本地的 PostgreSQL 数据库，端口和密码将从环境变量中读取。

```bash
export PGPASSWORD=$(kubectl get secret -n crater postgres.postgres-cluster.credentials.postgresql.acid.zalan.do -o 'jsonpath={.data.password}' | base64 -d)
export PGPORT=6432

cd cmd/gorm-gen
go run generate.go
```
