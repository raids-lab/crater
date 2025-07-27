# For RAIDS-Lab

## 配置说明

所有密码、密钥、令牌等敏感信息请在 `values.yaml` 的 `global.secrets` 字段下统一填写。  
每个字段旁有注释说明用途，务必根据实际环境填写。

示例：

```yaml
global:
  secrets:
    dbPassword: "<your-db-password>"
    adminPassword: "<your-admin-password>"
    grafanaToken: "<your-grafana-token>"
    registryPass: "<your-registry-robot-password>"
    registryAdminPass: "<your-registry-admin-password>"
    # ...其他密钥...
```

## 更新方式

```bash
helm upgrade --install crater ./charts/crater \
--namespace crater \
--create-namespace
-f customValues.yaml
```

`customValues.yaml` 请联系管理员获取。

