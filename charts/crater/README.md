# crater

![Version: 0.1.4](https://img.shields.io/badge/Version-0.1.4-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

A comprehensive AI development platform for Kubernetes that provides GPU resource management, containerized development environments, and workflow orchestration.

**Homepage:** <https://github.com/raids-lab/crater>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## Source Code

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{"nodeAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"preference":{"matchExpressions":[{"key":"nvidia.com/gpu.present","operator":"NotIn","values":["true"]}]},"weight":100}]}}` | Pod affinity configuration |
| backendConfig | object | `{"auth":{"ldap":{"attributeMapping":{"displayName":"cn","email":"mail","username":"uid"},"enable":false,"server":{"address":"ldap://ldap.example.com:389","baseDN":"dc=example,dc=org","bindDN":"cn=admin,dc=example,dc=org","bindPassword":"<MUSTEDIT>"},"uid":{"source":"default"}},"normal":{"allowLogin":true,"allowRegister":true},"token":{"accessTokenSecret":"example-access-token","refreshTokenSecret":"example-refresh-token"}},"enableLeaderElection":false,"modelDownload":{"image":"crater-harbor.act.buaa.edu.cn/docker.io/library/python:3.11-slim"},"port":":8088","postgres":{"TimeZone":"Asia/Shanghai","dbname":"postgres","host":"crater-postgresql.crater-system.svc.cluster.local","password":"<MUSTEDIT>","port":5432,"sslmode":"disable","user":"postgres"},"prometheusAPI":"http://192.168.0.1:12345","registry":{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":false,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}},"secrets":{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"},"smtp":{"enable":false,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"},"storage":{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}}` | Backend configuration |
| backendConfig.auth | object | `{"ldap":{"attributeMapping":{"displayName":"cn","email":"mail","username":"uid"},"enable":false,"server":{"address":"ldap://ldap.example.com:389","baseDN":"dc=example,dc=org","bindDN":"cn=admin,dc=example,dc=org","bindPassword":"<MUSTEDIT>"},"uid":{"source":"default"}},"normal":{"allowLogin":true,"allowRegister":true},"token":{"accessTokenSecret":"example-access-token","refreshTokenSecret":"example-refresh-token"}}` | Configuration for authentication methods and tokens |
| backendConfig.auth.ldap | object | `{"attributeMapping":{"displayName":"cn","email":"mail","username":"uid"},"enable":false,"server":{"address":"ldap://ldap.example.com:389","baseDN":"dc=example,dc=org","bindDN":"cn=admin,dc=example,dc=org","bindPassword":"<MUSTEDIT>"},"uid":{"source":"default"}}` | LDAP authentication settings |
| backendConfig.auth.ldap.attributeMapping | object | `{"displayName":"cn","email":"mail","username":"uid"}` | LDAP attribute mapping configuration |
| backendConfig.auth.ldap.attributeMapping.displayName | string | `"cn"` | LDAP attribute for display name (Required) |
| backendConfig.auth.ldap.attributeMapping.email | string | `"mail"` | LDAP attribute for email |
| backendConfig.auth.ldap.attributeMapping.username | string | `"uid"` | LDAP attribute for username |
| backendConfig.auth.ldap.enable | bool | `false` | Enable LDAP authentication |
| backendConfig.auth.ldap.server | object | `{"address":"ldap://ldap.example.com:389","baseDN":"dc=example,dc=org","bindDN":"cn=admin,dc=example,dc=org","bindPassword":"<MUSTEDIT>"}` | LDAP server connection settings |
| backendConfig.auth.ldap.server.address | string | `"ldap://ldap.example.com:389"` | LDAP server address |
| backendConfig.auth.ldap.server.baseDN | string | `"dc=example,dc=org"` | Base DN for search |
| backendConfig.auth.ldap.server.bindDN | string | `"cn=admin,dc=example,dc=org"` | Admin DN for binding |
| backendConfig.auth.ldap.server.bindPassword | string | `"<MUSTEDIT>"` | Admin password for binding |
| backendConfig.auth.ldap.uid | object | `{"source":"default"}` | UID/GID acquisition settings |
| backendConfig.auth.ldap.uid.source | string | `"default"` | Source of UID/GID ("default", "ldap", "external") |
| backendConfig.auth.normal | object | `{"allowLogin":true,"allowRegister":true}` | Normal login/registration settings |
| backendConfig.auth.normal.allowLogin | bool | `true` | Allow local login |
| backendConfig.auth.normal.allowRegister | bool | `true` | Allow local registration |
| backendConfig.auth.token | object | `{"accessTokenSecret":"example-access-token","refreshTokenSecret":"example-refresh-token"}` | Authentication token configuration for JWT-based authentication (Required) Both token secrets must be specified for secure authentication |
| backendConfig.auth.token.accessTokenSecret | string | `"example-access-token"` | Secret key used to sign JWT access tokens (Required) Must be a secure, randomly generated string |
| backendConfig.auth.token.refreshTokenSecret | string | `"example-refresh-token"` | Secret key used to sign JWT refresh tokens (Required) Must be a secure, randomly generated string |
| backendConfig.enableLeaderElection | bool | `false` | Enable leader election for controller manager to ensure high availability Defaults to false if not specified |
| backendConfig.modelDownload | object | `{"image":"crater-harbor.act.buaa.edu.cn/docker.io/library/python:3.11-slim"}` | Model download functionality configurations |
| backendConfig.modelDownload.image | string | `"crater-harbor.act.buaa.edu.cn/docker.io/library/python:3.11-slim"` | Container image used for model download jobs Can be customized for local deployments with network restrictions Default: python:3.11-slim (public Docker Hub) Internal deployment uses: crater-harbor.act.buaa.edu.cn/docker.io/library/python:3.11-slim |
| backendConfig.port | string | `":8088"` | Network port that the server endpoint will listen on (Required) Must be specified for the server to start |
| backendConfig.postgres | object | `{"TimeZone":"Asia/Shanghai","dbname":"postgres","host":"crater-postgresql.crater-system.svc.cluster.local","password":"<MUSTEDIT>","port":5432,"sslmode":"disable","user":"postgres"}` | PostgreSQL database connection configuration (Required) All fields must be specified for database connectivity |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | Time zone for database connections Defaults to system time zone if not specified |
| backendConfig.postgres.dbname | string | `"postgres"` | Name of the database to connect to (Required) Database must exist and be accessible |
| backendConfig.postgres.host | string | `"crater-postgresql.crater-system.svc.cluster.local"` | PostgreSQL server hostname or IP address (Required) Must be reachable from the application |
| backendConfig.postgres.password | string | `"<MUSTEDIT>"` | Database password for authentication (Required) Must match the specified user's password |
| backendConfig.postgres.port | int | `5432` | PostgreSQL server port number (Required) Typically 5432 for PostgreSQL |
| backendConfig.postgres.sslmode | string | `"disable"` | SSL/TLS mode for database connection Defaults to "disable" if not specified |
| backendConfig.postgres.user | string | `"postgres"` | Database username for authentication (Required) User must have appropriate permissions |
| backendConfig.prometheusAPI | string | `"http://192.168.0.1:12345"` | Endpoint URL for Prometheus API used for metrics and monitoring If not specified, Prometheus integration will be disabled |
| backendConfig.registry | object | `{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":false,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}}` | Container registry configuration for image storage and building If Enable is false, registry functionality will be disabled |
| backendConfig.registry.buildTools | object | `{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}}` | Configuration for container image building tools and proxies Required if Registry.Enable is true |
| backendConfig.registry.buildTools.proxyConfig | object | `{"httpProxy":null,"httpsProxy":null,"noProxy":null}` | HTTP proxy settings for build environments If not specified, no proxy will be configured for builds |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `nil` | HTTP proxy URL for build environments If not specified, HTTP traffic will not be proxied |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `nil` | HTTPS proxy URL for build environments If not specified, HTTPS traffic will not be proxied |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `nil` | Comma-separated list of domains that should not be proxied If not specified, all traffic will go through the proxy |
| backendConfig.registry.enable | bool | `false` | Enable container registry integration Defaults to false if not specified |
| backendConfig.registry.harbor | object | `{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}` | Configuration for Harbor container registry integration Required if Registry.Enable is true: All Harbor fields must be specified |
| backendConfig.registry.harbor.password | string | `"<MASKED>"` | Admin password for Harbor authentication (Required) Must match the specified user's password |
| backendConfig.registry.harbor.server | string | `"harbor.example.com"` | Harbor registry server URL (Required) Must be a valid Harbor instance URL |
| backendConfig.registry.harbor.user | string | `"admin"` | Admin username for Harbor authentication (Required) User must have appropriate permissions in Harbor |
| backendConfig.secrets | object | `{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"}` | Kubernetes secret names for various security components (Required) All secret names must correspond to existing Kubernetes secrets |
| backendConfig.secrets.imagePullSecretName | string | `""` | Name of the Kubernetes secret for pulling container images from private registries If not specified, no image pull secret will be used |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | Name of the Kubernetes secret for TLS forwarding configuration (Required) Secret must contain appropriate forwarding certificates |
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | Name of the Kubernetes secret containing TLS certificates for HTTPS (Required) Secret must contain 'tls.crt' and 'tls.key' keys |
| backendConfig.smtp | object | `{"enable":false,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"}` | Configuration for email notifications via SMTP If Enable is false, email notifications will be disabled |
| backendConfig.smtp.enable | bool | `false` | Enable SMTP email functionality Defaults to false if not specified |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP server hostname or IP address (Required if Enable is true) Must be a valid SMTP server |
| backendConfig.smtp.notify | string | `"example@example.com"` | Default email address for system notifications (Required if Enable is true) Must be a valid email address |
| backendConfig.smtp.password | string | `"<MASKED>"` | Password for SMTP authentication (Required if Enable is true) Must match the specified user's password |
| backendConfig.smtp.port | int | `25` | SMTP server port number (Required if Enable is true) Typically 25, 465, or 587 |
| backendConfig.smtp.user | string | `"example"` | Username for SMTP authentication (Required if Enable is true) Must be a valid SMTP user |
| backendConfig.storage | object | `{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}` | Persistent volume claim and path prefix configurations (Required) All PVC names and prefix paths must be specified |
| backendConfig.storage.prefix | object | `{"account":"accounts","public":"public","user":"users"}` | Path prefixes for different types of storage locations (Required) All prefix paths must be specified |
| backendConfig.storage.prefix.account | string | `"accounts"` | Account prefix for account-related storage paths (Required) Must be a valid path within the storage system |
| backendConfig.storage.prefix.public | string | `"public"` | Public prefix for publicly accessible storage paths (Required) Must be a valid path within the storage system |
| backendConfig.storage.prefix.user | string | `"users"` | User prefix for user-specific storage paths (Required) Must be a valid path within the storage system |
| backendConfig.storage.pvc.readOnlyMany | string | `nil` | Name of the ReadOnlyMany Persistent Volume Claim for datasets and models It should be a link to the same underlying storage as ReadWriteMany If not specified, datasets and models will be mounted as read-write |
| backendConfig.storage.pvc.readWriteMany | string | `"crater-rw-storage"` | Name of the ReadWriteMany Persistent Volume Claim for shared storage (Required) PVC must exist in the cluster with ReadWriteMany access mode |
| buildkitConfig | object | `{"amdConfig":{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":false,"replicas":3},"armConfig":{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2},"generalConfig":{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}}` | Image building pipeline configuration Only fully available when you have self-hosted image registries like Harbor |
| buildkitConfig.amdConfig | object | `{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":false,"replicas":3}` | AMD architecture configuration |
| buildkitConfig.amdConfig.cache | object | `{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"}` | Cache configuration for AMD builds |
| buildkitConfig.amdConfig.cache.maxUsedSpace | string | `"400GB"` | Maximum used space for AMD build cache |
| buildkitConfig.amdConfig.cache.minFreeSpace | string | `"50GB"` | Minimum free space for AMD build cache |
| buildkitConfig.amdConfig.cache.reservedSpace | string | `"50GB"` | Reserved space for AMD build cache |
| buildkitConfig.amdConfig.cache.storageClass | string | `"openebs-hostpath"` | Storage class for AMD build cache |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | Storage size for AMD build cache |
| buildkitConfig.amdConfig.enabled | bool | `false` | Enable AMD architecture builds |
| buildkitConfig.amdConfig.replicas | int | `3` | Number of AMD build replicas |
| buildkitConfig.armConfig | object | `{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2}` | ARM architecture configuration |
| buildkitConfig.armConfig.cache | object | `{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"}` | Cache configuration for ARM builds |
| buildkitConfig.armConfig.cache.maxUsedSpace | string | `"80GB"` | Maximum used space for ARM build cache |
| buildkitConfig.armConfig.cache.minFreeSpace | string | `"10GB"` | Minimum free space for ARM build cache |
| buildkitConfig.armConfig.cache.reservedSpace | string | `"10GB"` | Reserved space for ARM build cache |
| buildkitConfig.armConfig.cache.storageClass | string | `"openebs-hostpath"` | Storage class for ARM build cache |
| buildkitConfig.armConfig.cache.storageSize | string | `"80Gi"` | Storage size for ARM build cache |
| buildkitConfig.armConfig.enabled | bool | `false` | Enable ARM architecture builds (Can only be true when ARM nodes exist) |
| buildkitConfig.armConfig.replicas | int | `2` | Number of ARM build replicas |
| buildkitConfig.generalConfig | object | `{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}` | General configuration for all architectures |
| buildkitConfig.generalConfig.resources | object | `{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}` | Resource configuration |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | CPU limit |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | Memory limit |
| buildkitConfig.generalConfig.resources.requests.cpu | int | `8` | CPU request |
| buildkitConfig.generalConfig.resources.requests.memory | string | `"24Gi"` | Memory request |
| cronjobConfig | object | `{"jobs":{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}}` | Cronjob management strategy configuration Job scheduling management strategies such as low utilization email reminders and cleanup, long-time usage email reminders and cleanup, etc. |
| cronjobConfig.jobs | object | `{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}` | Job management tasks configuration |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | Batch job maximum days |
| cronjobConfig.jobs.longTime.INTERACTIVE_DAYS | string | `"4"` | Interactive job maximum days |
| cronjobConfig.jobs.longTime.schedule | string | `"*/5 * * * *"` | Schedule for long-time usage check |
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | Time range for monitoring (minutes) |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | GPU utilization threshold |
| cronjobConfig.jobs.lowGPUUtil.WAIT_TIME | string | `"30"` | Wait time before action (minutes) |
| cronjobConfig.jobs.lowGPUUtil.schedule | string | `"*/5 * * * *"` | Schedule for low GPU utilization check |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Jupyter wait time in minutes |
| cronjobConfig.jobs.waitingJupyter.schedule | string | `"*/5 * * * *"` | Schedule for waiting Jupyter check |
| dbBackup | object | `{"backoffLimit":3,"config":{"accessModes":["ReadWriteMany"],"backupDir":"/backups","createPVC":true,"enabled":true,"pvcName":"pg-backup-pvc","retentionCount":7,"size":"10Gi","storageClass":"nfs"},"enabled":true,"failedJobsHistoryLimit":1,"schedule":"0 2 * * *","successfulJobsHistoryLimit":3,"timeZone":"Asia/Shanghai"}` | Backup configuratiton |
| dbBackup.backoffLimit | int | `3` | Number of retries before marking the job as failed |
| dbBackup.config | object | `{"accessModes":["ReadWriteMany"],"backupDir":"/backups","createPVC":true,"enabled":true,"pvcName":"pg-backup-pvc","retentionCount":7,"size":"10Gi","storageClass":"nfs"}` | Internal configuration for backup behavior (can be overridden via ConfigMap in the future) |
| dbBackup.config.createPVC | bool | `true` | Whether to create PVC or use existing pvc. |
| dbBackup.config.enabled | bool | `true` | Whether to actually perform backup (can be used to disable logic without removing job) |
| dbBackup.config.pvcName | string | `"pg-backup-pvc"` | PVC name (existing or created, also used in dbBackup) |
| dbBackup.config.retentionCount | int | `7` | Number of most recent backup files to retain Old backups beyond this number will be deleted during cleanup |
| dbBackup.config.size | string | `"10Gi"` | If create, Storage request size |
| dbBackup.config.storageClass | string | `"nfs"` | If create, Storage class name (e.g. cephfs, nfs, must support ReadWriteMany) |
| dbBackup.enabled | bool | `true` | Whether to enable the database backup CronJob |
| dbBackup.failedJobsHistoryLimit | int | `1` | Number of failed jobs to retain in history |
| dbBackup.schedule | string | `"0 2 * * *"` | Cron schedule for running the backup job Default: runs daily at 02:00 AM (Shanghai time) |
| dbBackup.successfulJobsHistoryLimit | int | `3` | Number of successful jobs to retain in history |
| dbBackup.timeZone | string | `"Asia/Shanghai"` | Time zone used for cron scheduling |
| firstUser | object | `{"password":"Masked@Password","username":"crater-admin"}` | First user configuration When connecting to the database for the first time, creates the first user account with administrator privileges |
| firstUser.password | string | `"Masked@Password"` | Password for the first administrator user (Please reset this password, can not be only number) |
| firstUser.username | string | `"crater-admin"` | Username for the first administrator user |
| frontendConfig | object | `{"grafana":{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}},"url":{"apiPrefix":"/api/v1","document":"https://raids-lab.github.io/crater/zh"},"version":"1.0.0"}` | Frontend configuration |
| frontendConfig.grafana | object | `{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}}` | Grafana dashboard configurations References: https://github.com/raids-lab/crater/tree/main/grafana-dashboards |
| frontendConfig.grafana.job.basic | string | `"/d/R4ZPFfyIz/crater-job-basic-dashboard"` | Basic job dashboard URL |
| frontendConfig.grafana.job.nvidia | string | `"/d/2CDE0AC7/crater-job-nvidia-dashboard"` | NVIDIA job dashboard URL |
| frontendConfig.grafana.job.pod | string | `"/d/MhnFUFLSz/crater-pod-dashboard"` | Pod dashboard URL |
| frontendConfig.grafana.node.basic | string | `"/d/k8s_views_nodes/crater-node-basic-dashboard"` | Basic node dashboard URL |
| frontendConfig.grafana.node.nvidia | string | `"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"` | NVIDIA node dashboard URL |
| frontendConfig.grafana.overview.main | string | `"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad"` | Main overview dashboard URL |
| frontendConfig.grafana.overview.network | string | `"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking"` | Network dashboard URL |
| frontendConfig.grafana.overview.schedule | string | `"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"` | Schedule dashboard URL |
| frontendConfig.grafana.user.nvidia | string | `"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"` | User NVIDIA dashboard URL |
| frontendConfig.url.apiPrefix | string | `"/api/v1"` | Backend API prefix (not modifiable currently) |
| frontendConfig.url.document | string | `"https://raids-lab.github.io/crater/zh"` | Documentation base URL |
| frontendConfig.version | string | `"1.0.0"` | Frontend version |
| grafanaProxy | object | `{"address":"http://prometheus-grafana.monitoring","enable":false,"host":"gpu-grafana.example.com","token":"<MASKED>"}` | Grafana proxy configuration Only Grafana Pro has password-free login feature. We use Nginx proxy to support password-free login for Iframe |
| grafanaProxy.address | string | `"http://prometheus-grafana.monitoring"` | Grafana service address in cluster |
| grafanaProxy.enable | bool | `false` | Whether to enable Grafana proxy |
| grafanaProxy.host | string | `"gpu-grafana.example.com"` | Domain name for exposing Grafana via Ingress Also used in frontend config even if grafanaProxy.enable is false |
| grafanaProxy.token | string | `"<MASKED>"` | Grafana access token (masked, please apply for read-only token in Grafana) |
| host | string | `"crater.example.com"` | Domain name or IP address that the server will bind to (Required) Must be specified for the server to start |
| imagePullPolicy | string | `"Always"` | Image pull policy ("IfNotPresent" | "Always", for development, use Always) |
| imagePullSecrets | list | `[]` | Image pull secrets |
| images | object | `{"backend":{"repository":"ghcr.io/raids-lab/crater-backend","tag":"latest"},"buildkit":{"repository":"docker.io/moby/buildkit","tag":"v0.23.1"},"buildx":{"repository":"ghcr.io/raids-lab/buildx-client","tag":"latest"},"cronjob":{"repository":"docker.io/badouralix/curl-jq","tag":"latest"},"dbBackup":{"repository":"docker.io/library/postgres","tag":"16.4"},"envd":{"repository":"ghcr.io/raids-lab/envd-client","tag":"latest"},"frontend":{"repository":"ghcr.io/raids-lab/crater-frontend","tag":"latest"},"grafanaProxy":{"repository":"docker.io/library/nginx","tag":"1.27.3-bookworm"},"nerdctl":{"repository":"ghcr.io/raids-lab/nerdctl-client","tag":"latest"},"storage":{"repository":"ghcr.io/raids-lab/storage-server","tag":"latest"}}` | Container images configuration |
| images.backend.repository | string | `"ghcr.io/raids-lab/crater-backend"` | Backend service image repository |
| images.backend.tag | string | `"latest"` | Backend service image tag |
| images.buildkit.repository | string | `"docker.io/moby/buildkit"` | Buildkit image repository for containerd-based builds |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit image tag |
| images.buildx.repository | string | `"ghcr.io/raids-lab/buildx-client"` | Buildx image repository for Docker Buildx multi-platform builds |
| images.buildx.tag | string | `"latest"` | Buildx image tag |
| images.cronjob.repository | string | `"docker.io/badouralix/curl-jq"` | Cronjob image repository |
| images.cronjob.tag | string | `"latest"` | Cronjob image tag |
| images.dbBackup.repository | string | `"docker.io/library/postgres"` | DBBackup image repository |
| images.dbBackup.tag | string | `"16.4"` | DBBackup image tag |
| images.envd.repository | string | `"ghcr.io/raids-lab/envd-client"` | Envd image repository for environment-based development builds |
| images.envd.tag | string | `"latest"` | Envd image tag |
| images.frontend.repository | string | `"ghcr.io/raids-lab/crater-frontend"` | Frontend service image repository |
| images.frontend.tag | string | `"latest"` | Frontend service image tag |
| images.grafanaProxy.repository | string | `"docker.io/library/nginx"` | Grafana proxy image repository |
| images.grafanaProxy.tag | string | `"1.27.3-bookworm"` | Grafana proxy image tag |
| images.nerdctl.repository | string | `"ghcr.io/raids-lab/nerdctl-client"` | Nerdctl image repository for containerd-based builds |
| images.nerdctl.tag | string | `"latest"` | Nerdctl image tag |
| images.storage.repository | string | `"ghcr.io/raids-lab/storage-server"` | Storage server image repository |
| images.storage.tag | string | `"latest"` | Storage server image tag |
| namespaces | object | `{"create":true,"image":"crater-images","job":"crater-workspace"}` | Namespace configuration for crater components By default, crater components run in crater namespace, while jobs and images are in separate namespaces |
| namespaces.create | bool | `true` | Whether to create namespaces along with the deployment |
| namespaces.image | string | `"crater-images"` | Namespace for building images |
| namespaces.job | string | `"crater-workspace"` | Namespace for running jobs |
| nodeSelector | object | `{"node-role.kubernetes.io/control-plane":""}` | Node selector for all Deployments Prevents control components from being scheduled to GPU worker nodes |
| protocol | string | `"http"` | Protocol for server communication ("http" or "https") |
| storage | object | `{"create":true,"pvcName":"crater-rw-storage","request":"10Gi","storageClass":"nfs"}` | Persistent Volume Claim configuration |
| storage.create | bool | `true` | Whether to create PVC or use existing pvc. |
| storage.pvcName | string | `"crater-rw-storage"` | PVC name (existing or created, also used in backendConfig) |
| storage.request | string | `"10Gi"` | If create, Storage request size |
| storage.storageClass | string | `"nfs"` | If create, Storage class name (e.g. cephfs, nfs, must support ReadWriteMany) |
| tls | object | `{"base":{"cert":"<MASKED>","create":false,"key":"<MASKED>"},"forward":{"cert":"<MASKED>","create":false,"key":"<MASKED>"}}` | TLS certificate configuration for exposing services via Ingress cert-manager configuration variables |
| tls.base | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | Base certificate configuration (Standard mode, e.g., crater.example.com certificate) |
| tls.base.cert | string | `"<MASKED>"` | Base certificate content (masked) |
| tls.base.create | bool | `false` | Whether to create base certificate |
| tls.base.key | string | `"<MASKED>"` | Base certificate private key (masked) |
| tls.forward | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | Forward certificate configuration (Subdomain mode, e.g., xxx.crater.example.com certificate for exposing internal job services externally) |
| tls.forward.cert | string | `"<MASKED>"` | Forward certificate content (masked) |
| tls.forward.create | bool | `false` | Whether to create forward certificate |
| tls.forward.key | string | `"<MASKED>"` | Forward certificate private key (masked) |
| tolerations | list | `[{"effect":"NoSchedule","key":"node-role.kubernetes.io/control-plane","operator":"Exists"}]` | Pod tolerations |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
