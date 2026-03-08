---
title: "Configuration Guide"
description: "Detailed explanation of Crater platform Helm Chart configuration parameters, covering core modules such as backend, frontend, storage, monitoring, and authentication."
---

<img src="https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square" alt="Version: 0.1.0"> <img src="https://img.shields.io/badge/Type-application-informational?style=flat-square" alt="Type: application"> <img src="https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square" alt="AppVersion: 1.0.0">

Crater is a comprehensive AI development platform designed for Kubernetes, providing GPU resource management, containerized development environments, and workflow orchestration. This document details all configurable items when deploying Crater via Helm.

**Project Homepage:** <https://github.com/raids-lab/crater>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## Source Code

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## Global Values

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| host | string | `"crater.example.com"` | Domain name or IP address for platform access |
| protocol | string | `"http"` | Protocol type (`http` or `https`) |
| firstUser.username | string | `"crater-admin"` | Initial administrator username |
| firstUser.password | string | `"Masked@Password"` | Initial administrator password (Please change this) |
| imagePullPolicy | string | `"Always"` | Container image pull policy |
| namespaces.create | bool | `true` | Whether to automatically create namespaces |
| namespaces.job | string | `"crater-workspace"` | Namespace for running job tasks |
| namespaces.image | string | `"crater-images"` | Namespace for building images |
| storage.create | bool | `true` | Whether to automatically create base persistent volume claim (PVC) |
| storage.request | string | `"10Gi"` | Storage request capacity |
| storage.storageClass | string | `"nfs"` | Storage class name (must support ReadWriteMany) |
| storage.pvcName | string | `"crater-rw-storage"` | Name of the shared PVC |

## Monitoring (Monitoring)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| prometheus.enable | bool | `true` | Whether to enable integrated Prometheus monitoring |
| prometheus.address | string | `"http://..."` | Internal Prometheus service address in cluster |
| grafana.enable | bool | `true` | Whether to enable integrated Grafana dashboards |
| grafana.address | string | `"http://..."` | Internal Grafana service address in cluster |
| monitoring.timezone | string | `"Asia/Shanghai"` | Timezone for monitoring dashboard display |

## Backend Service Configuration (backendConfig)

### Base Service Items

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.port | string | `":8088"` | Backend API service listening port |
| backendConfig.enableLeaderElection | bool | `false` | Whether to enable Leader election (for HA deployment) |
| backendConfig.modelDownload.image | string | `"python:3.11-slim"` | Container image used for model download jobs |
| backendConfig.prometheusAPI | string | `"http://..."` | Prometheus API address for monitoring metrics |
| backendConfig.auth.token.accessTokenSecret | string | `"..."` | JWT Access Token signing secret |
| backendConfig.auth.token.refreshTokenSecret | string | `"..."` | JWT Refresh Token signing secret |

### Database Connection (backendConfig.postgres)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.postgres.host | string | `"..."` | Database server address |
| backendConfig.postgres.port | int | `5432` | Database port |
| backendConfig.postgres.dbname | string | `"postgres"` | Database name |
| backendConfig.postgres.user | string | `"postgres"` | Database username |
| backendConfig.postgres.password | string | `"..."` | Database password |
| backendConfig.postgres.sslmode | string | `"disable"` | SSL mode configuration |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | Timezone used for database connection |

### Storage Path Binding (backendConfig.storage)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.storage.pvc.readWriteMany | string | `"..."` | Bound shared storage PVC name |
| backendConfig.storage.pvc.readOnlyMany | string | `null` | Optional read-only storage PVC name (for datasets and models) |
| backendConfig.storage.prefix.user | string | `"users"` | Storage path prefix for user personal space |
| backendConfig.storage.prefix.account | string | `"accounts"` | Storage path prefix for queue/account public space |
| backendConfig.storage.prefix.public | string | `"public"` | Storage path prefix for global public datasets |

### Base Resources and Secrets (backendConfig.secrets)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | Name of the TLS certificate Secret for HTTPS |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | Name of the TLS certificate Secret for forwarding |
| backendConfig.secrets.imagePullSecretName | string | `""` | Name of the Secret for pulling private images |

### Authentication Configuration (backendConfig.auth)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.auth.ldap.enable | bool | `false` | Whether to enable LDAP unified identity authentication |
| backendConfig.auth.ldap.server.address | string | `"..."` | LDAP server address |
| backendConfig.auth.ldap.server.baseDN | string | `"..."` | Base DN for user search |
| backendConfig.auth.ldap.attributeMapping.username | string | `"uid"` | LDAP attribute name for username |
| backendConfig.auth.ldap.attributeMapping.displayName | string | `"cn"` | LDAP attribute name for display name |
| backendConfig.auth.ldap.uid.source | string | `"default"` | **UID/GID acquisition strategy**: options `default`, `ldap`, `external` |
| backendConfig.auth.ldap.uid.ldapAttribute.uid | string | `""` | UID attribute name when source is `ldap` |
| backendConfig.auth.normal.allowRegister | bool | `true` | Whether to allow direct registration on the platform |
| backendConfig.auth.normal.allowLogin | bool | `true` | Whether to allow login with local database accounts |

### Image Registry Integration (backendConfig.registry)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.registry.enable | bool | `false` | Whether to enable image registry (Harbor) integration |
| backendConfig.registry.harbor.server | string | `"..."` | Harbor service access address |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor admin account |
| backendConfig.registry.harbor.password | string | `"..."` | Harbor admin password |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `null` | HTTP proxy for image building |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `null` | HTTPS proxy for image building |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `null` | List of domains that should not be proxied (comma-separated) |

### Email Service Configuration (backendConfig.smtp)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| backendConfig.smtp.enable | bool | `false` | Whether to enable email notification functionality |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP server address |
| backendConfig.smtp.port | int | `25` | SMTP server port |
| backendConfig.smtp.user | string | `"example"` | SMTP authentication username |
| backendConfig.smtp.password | string | `"..."` | SMTP authentication password |
| backendConfig.smtp.notify | string | `"example@example.com"` | Email address of the system notification sender |

## Image Building Pipeline (buildkitConfig)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| buildkitConfig.amdConfig.enabled | bool | `false` | Whether to enable AMD64 image build nodes |
| buildkitConfig.amdConfig.replicas | int | `3` | Number of build node replicas |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | Build node cache volume size |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | Maximum CPU limit for build nodes |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | Maximum memory limit for build nodes |

## Automatic Task Strategy (cronjobConfig)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | Time range for low utilization detection (minutes) |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | Utilization threshold for triggering alerts |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | Maximum runtime days for batch jobs |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Cleanup threshold for Jupyter jobs in Waiting state |

## Database Backup (dbBackup)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| dbBackup.enabled | bool | `true` | Whether to enable automatic database backup |
| dbBackup.schedule | string | `"0 2 * * *"` | Backup Cron expression |
| dbBackup.config.retentionCount | int | `7` | Retention count/days for backup files |

## Monitoring Presentation (frontendConfig / grafanaProxy)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| frontendConfig.version | string | `"1.0.0"` | Frontend application version |
| grafanaProxy.enable | bool | `false` | Whether to enable Grafana password-less proxy (for Iframe embedding) |
| grafanaProxy.address | string | `"..."` | Grafana service address in cluster |
| grafanaProxy.token | string | `"..."` | Grafana API Token with read-only permissions |

## TLS Certificate Configuration (tls)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| tls.base.create | bool | `false` | Whether Helm creates the base certificate Secret |
| tls.base.cert | string | `""` | Base certificate content (Base64) |
| tls.forward.create | bool | `false` | Whether Helm creates the forward certificate Secret |
| tls.forward.cert | string | `""` | Forward certificate content (Base64) |

## Component Image Versions (images)

| Parameter | Type | Default | Description |
|-----|------|---------|-------------|
| images.backend.repository | string | `"..."` | Backend service image repository |
| images.frontend.repository | string | `"..."` | Frontend service image repository |
| images.storage.repository | string | `"..."` | Storage management service image repository |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit core image tag |

---
This document version applies to Crater v0.1.0 and above.
