# Crater Gorm Gen

Before generating code, first use `kubectl` to port-forward the database in the cluster to your local machine. Alternatively, if port forwarding is already configured in the cluster, you can directly connect to the database in the cluster.


在生成代码前，首先通过 Kubectl 将集群中的数据库转发到本地。或者在集群配置了端口转发的情况下，可以直接连接集群中的数据库。

```bash
make migrate
make curd
```
