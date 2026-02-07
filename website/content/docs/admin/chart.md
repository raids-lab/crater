---
title: "配置说明"
description: "Crater 平台 Helm Chart 详细配置参数说明，涵盖后端、前端、存储、监控及认证等核心模块。"
---

<img src="https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square" alt="Version: 0.1.0"> <img src="https://img.shields.io/badge/Type-application-informational?style=flat-square" alt="Type: application"> <img src="https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square" alt="AppVersion: 1.0.0">

Crater 是一个专为 Kubernetes 设计的综合性 AI 开发平台，提供 GPU 资源管理、容器化开发环境和工作流编排功能。本文档详细列出了通过 Helm 部署 Crater 时的所有可配置项。

**项目主页:** <https://github.com/raids-lab/crater>

## 维护者

| 名称 | 邮箱 | 网址 |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## 源码

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## 基础配置 (Global Values)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| host | string | `"crater.example.com"` | 平台访问的域名或 IP 地址 |
| protocol | string | `"http"` | 访问协议类型（`http` 或 `https`） |
| firstUser.username | string | `"crater-admin"` | 初始管理员用户名 |
| firstUser.password | string | `"Masked@Password"` | 初始管理员密码（请务必修改） |
| imagePullPolicy | string | `"Always"` | 容器镜像拉取策略 |
| namespaces.create | bool | `true` | 是否自动创建命名空间 |
| namespaces.job | string | `"crater-workspace"` | 用于运行作业任务的命名空间 |
| namespaces.image | string | `"crater-images"` | 用于构建镜像的命名空间 |
| storage.create | bool | `true` | 是否自动创建基础持久化存储（PVC） |
| storage.request | string | `"10Gi"` | 存储申请容量 |
| storage.storageClass | string | `"nfs"` | 存储类名称（需支持 ReadWriteMany） |
| storage.pvcName | string | `"crater-rw-storage"` | 共享 PVC 的名称 |

## 监控基础配置 (Monitoring)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| prometheus.enable | bool | `true` | 是否启用内置 Prometheus 监控 |
| prometheus.address | string | `"http://..."` | 集群内部 Prometheus 服务地址 |
| grafana.enable | bool | `true` | 是否启用内置 Grafana 面板 |
| grafana.address | string | `"http://..."` | 集群内部 Grafana 服务地址 |
| monitoring.timezone | string | `"Asia/Shanghai"` | 监控面板显示时区 |

## 后端服务配置 (backendConfig)

### 基础服务项

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.port | string | `":8088"` | 后端 API 服务监听端口 |
| backendConfig.enableLeaderElection | bool | `false` | 是否启用 Leader 选举（用于高可用部署） |
| backendConfig.modelDownload.image | string | `"python:3.11-slim"` | 模型下载作业使用的容器镜像 |
| backendConfig.prometheusAPI | string | `"http://..."` | Prometheus API 地址，用于获取监控指标 |
| backendConfig.auth.token.accessTokenSecret | string | `"..."` | JWT Access Token 签名密钥 |
| backendConfig.auth.token.refreshTokenSecret | string | `"..."` | JWT Refresh Token 签名密钥 |

### 数据库连接 (backendConfig.postgres)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.postgres.host | string | `"..."` | 数据库服务器地址 |
| backendConfig.postgres.port | int | `5432` | 数据库端口 |
| backendConfig.postgres.dbname | string | `"postgres"` | 数据库名称 |
| backendConfig.postgres.user | string | `"postgres"` | 数据库用户名 |
| backendConfig.postgres.password | string | `"..."` | 数据库密码 |
| backendConfig.postgres.sslmode | string | `"disable"` | SSL 模式配置 |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | 数据库连接使用的时区 |

### 存储路径绑定 (backendConfig.storage)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.storage.pvc.readWriteMany | string | `"..."` | 绑定的共享存储 PVC 名称 |
| backendConfig.storage.pvc.readOnlyMany | string | `null` | 可选的只读存储 PVC 名称（用于数据集和模型） |
| backendConfig.storage.prefix.user | string | `"users"` | 用户个人空间的存储路径前缀 |
| backendConfig.storage.prefix.account | string | `"accounts"` | 队列/账户公共空间的存储路径前缀 |
| backendConfig.storage.prefix.public | string | `"public"` | 全局公共数据集的存储路径前缀 |

### 基础资源与密钥 (backendConfig.secrets)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | 用于 HTTPS 的 TLS 证书 Secret 名称 |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | 用于转发的 TLS 证书 Secret 名称 |
| backendConfig.secrets.imagePullSecretName | string | `""` | 用于拉取私有镜像的 Secret 名称 |

### 认证方式配置 (backendConfig.auth)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.auth.ldap.enable | bool | `false` | 是否启用 LDAP 统一身份认证 |
| backendConfig.auth.ldap.server.address | string | `"..."` | LDAP 服务器地址 |
| backendConfig.auth.ldap.server.baseDN | string | `"..."` | 用户搜索的 Base DN |
| backendConfig.auth.ldap.attributeMapping.username | string | `"uid"` | 用户名对应的 LDAP 属性名 |
| backendConfig.auth.ldap.attributeMapping.displayName | string | `"cn"` | 显示名称对应的 LDAP 属性名 |
| backendConfig.auth.ldap.uid.source | string | `"default"` | **UID/GID 获取策略**：可选 `default`, `ldap`, `external` |
| backendConfig.auth.ldap.uid.ldapAttribute.uid | string | `""` | 当 source 为 `ldap` 时的 UID 属性名 |
| backendConfig.auth.normal.allowRegister | bool | `true` | 是否允许平台本地直接注册 |
| backendConfig.auth.normal.allowLogin | bool | `true` | 是否允许使用本地数据库账号登录 |

### 镜像仓库集成 (backendConfig.registry)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.registry.enable | bool | `false` | 是否启用容器镜像仓库（Harbor）集成 |
| backendConfig.registry.harbor.server | string | `"..."` | Harbor 服务访问地址 |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor 管理员账号 |
| backendConfig.registry.harbor.password | string | `"..."` | Harbor 管理员密码 |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `null` | 镜像构建时的 HTTP 代理 |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `null` | 镜像构建时的 HTTPS 代理 |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `null` | 不走代理的域名列表（逗号分隔） |

### 邮件服务配置 (backendConfig.smtp)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| backendConfig.smtp.enable | bool | `false` | 是否启用邮件通知功能 |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP 服务器地址 |
| backendConfig.smtp.port | int | `25` | SMTP 服务器端口 |
| backendConfig.smtp.user | string | `"example"` | SMTP 认证用户名 |
| backendConfig.smtp.password | string | `"..."` | SMTP 认证密码 |
| backendConfig.smtp.notify | string | `"example@example.com"` | 系统通知发送者的邮箱地址 |

## 镜像构建流水线 (buildkitConfig)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| buildkitConfig.amdConfig.enabled | bool | `false` | 是否启用 AMD64 架构构建节点 |
| buildkitConfig.amdConfig.replicas | int | `3` | 构建节点的副本数 |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | 构建节点的缓存卷大小 |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | 构建节点的最大 CPU 限制 |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | 构建节点的最大内存限制 |

## 自动任务策略 (cronjobConfig)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | 低利用率检测的时间范围（分钟） |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | 触发提醒的利用率阈值 |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | 批量作业的最长运行天数 |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Jupyter 作业处于 Waiting 状态的清理阈值 |

## 数据库备份 (dbBackup)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| dbBackup.enabled | bool | `true` | 是否启用数据库自动备份 |
| dbBackup.schedule | string | `"0 2 * * *"` | 备份 Cron 表达式 |
| dbBackup.config.retentionCount | int | `7` | 备份文件的保留天数/个数 |

## 监控展示 (frontendConfig / grafanaProxy)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| frontendConfig.version | string | `"1.0.0"` | 前端应用版本 |
| grafanaProxy.enable | bool | `false` | 是否启用 Grafana 免密代理（用于 Iframe 嵌入） |
| grafanaProxy.address | string | `"..."` | 集群内 Grafana 服务地址 |
| grafanaProxy.token | string | `"..."` | 只读权限的 Grafana API Token |

## TLS 证书配置 (tls)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| tls.base.create | bool | `false` | 是否由 Helm 创建基础证书 Secret |
| tls.base.cert | string | `""` | 基础证书内容 (Base64) |
| tls.forward.create | bool | `false` | 是否由 Helm 创建转发证书 Secret |
| tls.forward.cert | string | `""` | 转发证书内容 (Base64) |

## 组件镜像版本 (images)

| 参数 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| images.backend.repository | string | `"..."` | 后端服务镜像仓库 |
| images.frontend.repository | string | `"..."` | 前端服务镜像仓库 |
| images.storage.repository | string | `"..."` | 存储管理服务镜像仓库 |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit 核心镜像标签 |

---
本文档版本适用于 Crater v0.1.0 及以上版本。
