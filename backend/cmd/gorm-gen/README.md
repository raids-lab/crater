# Crater Gorm Gen

在生成代码前，首先通过 Kubectl 将集群中的数据库转发到本地。

```bash
cd cmd/gorm-gen
go run models/migrate.go
go run curd/generate.go
```
