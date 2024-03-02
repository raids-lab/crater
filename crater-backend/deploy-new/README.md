# Crater Deploy

## MySQL

- https://stackoverflow.com/questions/76041385/trying-to-configure-mysql-innodb-cluster-but-have-pods-pending-error
- https://blog.csdn.net/weixin_46510209/article/details/131782499

### 部署到同一节点问题

参考 https://blog.csdn.net/duoyasong5907/article/details/121139327，手动设置反亲和性。

```yaml
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/component: database
      app.kubernetes.io/created-by: mysql-operator
      app.kubernetes.io/instance: mysql-innodbcluster-mysql-cluster-ceph-mysql-server
      app.kubernetes.io/managed-by: mysql-operator
      app.kubernetes.io/name: mysql-innodbcluster-mysql-server
      component: mysqld
      mysql.oracle.com/cluster: mysql-cluster-ceph
      tier: mysql
  template:
    metadata:
      creationTimestamp: null
      labels:
        app.kubernetes.io/component: database
        app.kubernetes.io/created-by: mysql-operator
        app.kubernetes.io/instance: mysql-innodbcluster-mysql-cluster-ceph-mysql-server
        app.kubernetes.io/managed-by: mysql-operator
        app.kubernetes.io/name: mysql-innodbcluster-mysql-server
        component: mysqld
        mysql.oracle.com/cluster: mysql-cluster-ceph
        tier: mysql
    spec:
      # 添加反亲和性，尽量让 Pod 调度到不同的节点上
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    # 使用来自 metadata 的标签
                    app.kubernetes.io/component: database
                    app.kubernetes.io/created-by: mysql-operator
                    app.kubernetes.io/instance: mysql-innodbcluster-mysql-cluster-ceph-mysql-server
                    app.kubernetes.io/managed-by: mysql-operator
                    app.kubernetes.io/name: mysql-innodbcluster-mysql-server
                    component: mysqld
                    mysql.oracle.com/cluster: mysql-cluster-ceph
                    tier: mysql
                topologyKey: "kubernetes.io/hostname"
```
