# Crater Gorm Gen

在生成代码前，首先通过 Kubectl 将集群中的数据库转发到本地。

```bash
go run cmd/gorm-gen/models/migrate.go
go run cmd/gorm-gen/curd/generate.go
```
