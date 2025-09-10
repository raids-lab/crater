---
title: "설명서"
description: "대학에서 개발한 지능형 클러스터 스케줄링 및 모니터링을 위한 클러스터 관리 플랫폼입니다."
---

![버전: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![유형: 애플리케이션](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![앱버전: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

Kubernetes를 위한 포괄적인 AI 개발 플랫폼으로, GPU 자원 관리, 컨테이너화된 개발 환경, 워크플로우 오케스트레이션 기능을 제공합니다.

**홈페이지:** <https://github.com/raids-lab/crater>

## 유지자

| 이름 | 이메일 | URL |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## 소스 코드

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## 값

| 키 | 유형 | 기본값 | 설명 |
|-----|------|---------|-------------|
| affinity | object | `{"nodeAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"preference":{"matchExpressions":[{"key":"nvidia.com/gpu.present","operator":"NotIn","values":["true"]}]},"weight":100}]}}` | Pod affinity 설정 |
| backendConfig | object | `{"auth":{"accessTokenSecret":"<MASKED>","refreshTokenSecret":"<MASKED>"},"enableLeaderElection":false,"port":":8088","postgres":{"TimeZone":"Asia/Shanghai","dbname":"crater","host":"192.168.0.1","password":"<MASKED>","port":6432,"sslmode":"disable","user":"postgres"},"prometheusAPI":"http://192.168.0.1:12345","registry":{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":true,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}},"secrets":{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"},"smtp":{"enable":true,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"},"storage":{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}}` | 백엔드 설정 |
| backendConfig.auth | object | `{"accessTokenSecret":"<MASKED>","refreshTokenSecret":"<MASKED>"}` | JWT 기반 인증을 위한 인증 토큰 설정 (필수) 보안 인증을 위해 두 토큰 비밀이 모두 지정되어야 합니다 |
| backendConfig.auth.accessTokenSecret | string | `"<MASKED>"` | JWT 접근 토큰에 서명하는 비밀 키 (필수) 보안이 높고 무작위로 생성된 문자열이어야 합니다 |
| backendConfig.auth.refreshTokenSecret | string | `"<MASKED>"` | JWT 리프레시 토큰에 서명하는 비밀 키 (필수) 보안이 높고 무작위로 생성된 문자열이어야 합니다 |
| backendConfig.enableLeaderElection | bool | `false` | 컨트롤러 관리자에게 리더 선출을 활성화하여 고가용성을 보장합니다. 지정되지 않은 경우 기본값은 false입니다. |
| backendConfig.port | string | `":8088"` | 서버 엔드포인트가 듣는 네트워크 포트 (필수) 서버가 시작되려면 반드시 지정되어야 합니다. |
| backendConfig.postgres | object | `{"TimeZone":"Asia/Shanghai","dbname":"crater","host":"192.168.0.1","password":"<MASKED>","port":6432,"sslmode":"disable","user":"postgres"}` | PostgreSQL 데이터베이스 연결 설정 (필수) 모든 필드가 데이터베이스 연결을 위해 지정되어야 합니다. |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | 데이터베이스 연결을 위한 시간대. 지정되지 않은 경우 시스템 시간대로 설정됩니다. |
| backendConfig.postgres.dbname | string | `"crater"` | 연결할 데이터베이스 이름 (필수) 데이터베이스가 존재하고 접근 가능해야 합니다. |
| backendConfig.postgres.host | string | `"192.168.0.1"` | PostgreSQL 서버의 호스트명 또는 IP 주소 (필수) 애플리케이션에서 접근 가능해야 합니다. |
| backendConfig.postgres.password | string | `"<MASKED>"` | 인증을 위한 데이터베이스 비밀번호 (필수) 지정된 사용자의 비밀번호와 일치해야 합니다. |
| backendConfig.postgres.port | int | `6432` | PostgreSQL 서버의 포트 번호 (필수) 일반적으로 PostgreSQL의 포트는 5432입니다. |
| backendConfig.postgres.sslmode | string | `"disable"` | 데이터베이스 연결을 위한 SSL/TLS 모드. 지정되지 않은 경우 기본값은 "disable"입니다. |
| backendConfig.postgres.user | string | `"postgres"` | 인증을 위한 데이터베이스 사용자 이름 (필수) 사용자가 적절한 권한을 가져야 합니다. |
| backendConfig.prometheusAPI | string | `"http://192.168.0.1:12345"` | 메트릭 및 모니터링을 위해 사용되는 Prometheus API 엔드포인트 URL. 지정되지 않은 경우 Prometheus 통합이 비활성화됩니다. |
| backendConfig.registry | object | `{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":true,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}}` | 이미지 저장 및 빌드를 위한 컨테이너 레지스트리 설정. Enable이 false인 경우 레지스트리 기능이 비활성화됩니다. |
| backendConfig.registry.buildTools | object | `{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}}` | 컨테이너 이미지 빌드 도구 및 프록시 설정. Registry.Enable이 true인 경우 필수입니다. |
| backendConfig.registry.buildTools.proxyConfig | object | `{"httpProxy":null,"httpsProxy":null,"noProxy":null}` | 빌드 환경을 위한 HTTP 프록시 설정. 지정되지 않은 경우 빌드에 프록시가 구성되지 않습니다. |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `nil` | 빌드 환경을 위한 HTTP 프록시 URL. 지정되지 않은 경우 HTTP 트래픽이 프록시를 통해 전달되지 않습니다. |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `nil` | 빌드 환경을 위한 HTTPS 프록시 URL. 지정되지 않은 경우 HTTPS 트래픽이 프록시를 통해 전달되지 않습니다. |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `nil` | 프록시를 사용하지 않는 도메인 목록 (콤마로 구분). 지정되지 않은 경우 모든 트래픽은 프록시를 통해 전달됩니다. |
| backendConfig.registry.enable | bool | `true` | 컨테이너 레지스트리 통합 활성화. 지정되지 않은 경우 기본값은 false입니다. |
| backendConfig.registry.harbor | object | `{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}` | Harbor 컨테이너 레지스트리 통합 설정. Registry.Enable이 true인 경우 모든 Harbor 필드가 지정되어야 합니다. |
| backendConfig.registry.harbor.password | string | `"<MASKED>"` | Harbor 인증을 위한 관리자 비밀번호 (필수) 지정된 사용자의 비밀번호와 일치해야 합니다. |
| backendConfig.registry.harbor.server | string | `"harbor.example.com"` | Harbor 레지스트리 서버 URL (필수) 유효한 Harbor 인스턴스 URL이어야 합니다. |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor 인증을 위한 관리자 사용자 이름 (필수) 사용자가 Harbor에서 적절한 권한을 가져야 합니다. |
| backendConfig.secrets | object | `{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"}` | 다양한 보안 구성 요소를 위한 Kubernetes 비밀 이름 (필수) 모든 비밀 이름은 기존의 Kubernetes 비밀과 일치해야 합니다. |
| backendConfig.secrets.imagePullSecretName | string | `""` | 프라이빗 레지스트리에서 컨테이너 이미지를 끌어오기 위한 Kubernetes 비밀 이름. 지정되지 않은 경우 이미지 끌어오기 비밀이 사용되지 않습니다. |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | TLS 전달 구성에 사용되는 Kubernetes 비밀 이름 (필수) 비밀에는 적절한 전달 인증서가 포함되어야 합니다. |
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | HTTPS를 위한 TLS 인증서를 포함하는 Kubernetes 비밀 이름 (필수) 비밀에는 'tls.crt' 및 'tls.key' 키가 포함되어야 합니다. |
| backendConfig.smtp | object | `{"enable":true,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"}` | SMTP를 통한 이메일 알림 설정. Enable이 false인 경우 이메일 알림이 비활성화됩니다. |
| backendConfig.smtp.enable | bool | `true` | SMTP 이메일 기능 활성화. 지정되지 않은 경우 기본값은 false입니다. |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP 서버 호스트명 또는 IP 주소 (Enable이 true인 경우 필수) 유효한 SMTP 서버이어야 합니다. |
| backendConfig.smtp.notify | string | `"example@example.com"` | 시스템 알림을 위한 기본 이메일 주소 (Enable이 true인 경우 필수) 유효한 이메일 주소이어야 합니다. |
| backendConfig.smtp.password | string | `"<MASKED>"` | SMTP 인증을 위한 비밀번호 (Enable이 true인 경우 필수) 지정된 사용자의 비밀번호와 일치해야 합니다. |
| backendConfig.smtp.port | int | `25` | SMTP 서버 포트 번호 (Enable이 true인 경우 필수) 일반적으로 25, 465, 또는 587입니다. |
| backendConfig.smtp.user | string | `"example"` | SMTP 인증을 위한 사용자 이름 (Enable이 true인 경우 필수) 유효한 SMTP 사용자이어야 합니다. |
| backendConfig.storage | object | `{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}` | 지속 가능한 볼륨 클레임 및 경로 접두사 설정 (필수) 모든 PVC 이름과 접두사 경로가 지정되어야 합니다. |
| backendConfig.storage.prefix | object | `{"account":"accounts","public":"public","user":"users"}` | 다양한 유형의 저장소 위치에 대한 경로 접두사 (필수) 모든 접두사 경로가 지정되어야 합니다. |
| backendConfig.storage.prefix.account | string | `"accounts"` | 계정 관련 저장소 경로의 접두사 (필수) 저장소 시스템 내부의 유효한 경로여야 합니다. |
| backendConfig.storage.prefix.public | string | `"public"` | 공개적으로 접근 가능한 저장소 경로의 접두사 (필수) 저장소 시스템 내부의 유효한 경로여야 합니다. |
| backendConfig.storage.prefix.user | string | `"users"` | 사용자별 저장소 경로의 접두사 (필수) 저장소 시스템 내부의 유효한 경로여야 합니다. |
| backendConfig.storage.pvc.readOnlyMany | string | `nil` | 데이터셋 및 모델을 위한 ReadOnlyMany 지속 가능한 볼륨 클레임 이름. ReadWriteMany와 동일한 하위 저장소에 연결되어야 합니다. 지정되지 않은 경우 데이터셋 및 모델은 읽기-쓰기로 마운트됩니다. |
| backendConfig.storage.pvc.readWriteMany | string | `"crater-rw-storage"` | 공유 저장소를 위한 ReadWriteMany 지속 가능한 볼륨 클레임 이름 (필수) 클러스터 내부에서 ReadWriteMany 액세스 모드로 존재해야 합니다. |
| buildkitConfig | object | `{"amdConfig":{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":true,"replicas":3},"armConfig":{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2},"generalConfig":{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}}` | 이미지 빌드 파이프라인 설정. Harbor와 같은 자체 호스팅 이미지 레지스트리가 있을 때만 완전히 사용 가능합니다. |
| buildkitConfig.amdConfig | object | `{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":true,"replicas":3}` | AMD 아키텍처 설정 |
| buildkitConfig.amdConfig.cache | object | `{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"}` | AMD 빌드에 대한 캐시 설정 |
| buildkitConfig.amdConfig.cache.maxUsedSpace | string | `"400GB"` | AMD 빌드 캐시의 최대 사용 공간 |
| buildkitConfig.amdConfig.cache.minFreeSpace | string | `"50GB"` | AMD 빌드 캐시의 최소 자유 공간 |
| buildkitConfig.amdConfig.cache.reservedSpace | string | `"50GB"` | AMD 빌드 캐시의 예약 공간 |
| buildkitConfig.amdConfig.cache.storageClass | string | `"openebs-hostpath"` | AMD 빌드 캐시의 저장소 클래스 |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | AMD 빌드 캐시의 저장소 크기 |
| buildkitConfig.amdConfig.enabled | bool | `true` | AMD 아키텍처 빌드 활성화 |
| buildkitConfig.amdConfig.replicas | int | `3` | AMD 빌드 복제본 수 |
| buildkitConfig.armConfig | object | `{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2}` | ARM 아키텍처 설정 |
| buildkitConfig.armConfig.cache | object | `{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"}` | ARM 빌드에 대한 캐시 설정 |
| buildkitConfig.armConfig.cache.maxUsedSpace | string | `"80GB"` | ARM 빌드 캐시의 최대 사용 공간 |
| buildkitConfig.armConfig.cache.minFreeSpace | string | `"10GB"` | ARM 빌드 캐시의 최소 자유 공간 |
| buildkitConfig.armConfig.cache.reservedSpace | string | `"10GB"` | ARM 빌드 캐시의 예약 공간 |
| buildkitConfig.armConfig.cache.storageClass | string | `"openebs-hostpath"` | ARM 빌드 캐시의 저장소 클래스 |
| buildkitConfig.armConfig.cache.storageSize | string | `"80Gi"` | ARM 빌드 캐시의 저장소 크기 |
| buildkitConfig.armConfig.enabled | bool | `false` | ARM 아키텍처 빌드 활성화 (ARM 노드가 존재할 때만 true 가능) |
| buildkitConfig.armConfig.replicas | int | `2` | ARM 빌드 복제본 수 |
| buildkitConfig.generalConfig | object | `{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}` | 모든 아키텍처에 대한 일반 설정 |
| buildkitConfig.generalConfig.resources | object | `{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}` | 자원 설정 |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | CPU 제한 |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | 메모리 제한 |
| buildkitConfig.generalConfig.resources.requests.cpu | int | `8` | CPU 요청 |
| buildkitConfig.generalConfig.resources.requests.memory | string | `"24Gi"` | 메모리 요청 |
| cronjobConfig | object | `{"jobs":{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}}` | Cronjob 관리 전략 설정. 낮은 사용률 알림 및 정리, 장시간 사용 알림 및 정리 등과 같은 작업 스케줄링 관리 전략 |
| cronjobConfig.jobs | object | `{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}` | 작업 관리 작업 설정 |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | 배치 작업 최대 일수 |
| cronjobConfig.jobs.longTime.INTERACTIVE_DAYS | string | `"4"` | 인터랙티브 작업 최대 일수 |
| cronjobConfig.jobs.longTime.schedule | string | `"*/5 * * * *"` | 장시간 사용 확인의 스케줄 |
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | 모니터링 시간 범위 (분) |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | GPU 사용률 임계값 |
| cronjobConfig.jobs.lowGPUUtil.WAIT_TIME | string | `"30"` | 작업 이전 대기 시간 (분) |
| cronjobConfig.jobs.lowGPUUtil.schedule | string | `"*/5 * * * *"` | 낮은 GPU 사용률 확인의 스케줄 |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Jupyter 대기 시간 (분) |
| cronjobConfig.jobs.waitingJupyter.schedule | string | `"*/5 * * * *"` | 대기 중인 Jupyter 확인의 스케줄 |
| firstUser | object | `{"password":"<MASKED>","username":"crater-admin"}` | 최초 사용자 설정. 데이터베이스에 처음 연결할 때 관리자 권한을 가진 첫 사용자 계정을 생성합니다. |
| firstUser.password | string | `"<MASKED>"` | 첫 관리자 사용자의 비밀번호 (재설정해야 함) |
| firstUser.username | string | `"crater-admin"` | 첫 관리자 사용자의 사용자 이름 |
| frontendConfig | object | `{"grafana":{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}},"url":{"apiPrefix":"/api/v1","document":"https://raids-lab.github.io/crater/zh"},"version":"1.0.0"}` | 프론트엔드 설정 |
| frontendConfig.grafana | object | `{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}}` | Grafana 대시보드 설정. 참조: https://github.com/raids-lab/crater/tree/main/grafana-dashboards |
| frontendConfig.grafana.job.basic | string | `"/d/R4ZPFfyIz/crater-job-basic-dashboard"` | 기본 작업 대시보드 URL |
| frontendConfig.grafana.job.nvidia | string | `"/d/2CDE0AC7/crater-job-nvidia-dashboard"` | NVIDIA 작업 대시보드 URL |
| frontendConfig.grafana.job.pod | string | `"/d/MhnFUFLSz/crater-pod-dashboard"` | Pod 대시보드 URL |
| frontendConfig.grafana.node.basic | string | `"/d/k8s_views_nodes/crater-node-basic-dashboard"` | 기본 노드 대시보드 URL |
| frontendConfig.grafana.node.nvidia | string | `"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"` | NVIDIA 노드 대시보드 URL |
| frontendConfig.grafana.overview.main | string | `"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad"` | 주요 개요 대시보드 URL |
| frontendConfig.grafana.overview.network | string | `"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking"` | 네트워크 대시보드 URL |
| frontendConfig.grafana.overview.schedule | string | `"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"` | 스케줄 대시보드 URL |
| frontendConfig.grafana.user.nvidia | string | `"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"` | 사용자 NVIDIA 대시보드 URL |
| frontendConfig.url.apiPrefix | string | `"/api/v1"` | 백엔드 API 접두사 (현재 수정 불가) |
| frontendConfig.url.document | string | `"https://raids-lab.github.io/crater/zh"` | 문서 기본 URL |
| frontendConfig.version | string | `"1.0.0"` | 프론트엔드 버전 |
| grafanaProxy | object | `{"address":"http://prometheus-grafana.monitoring","enable":true,"host":"gpu-grafana.<your-domain>.com","token":"<MASKED>"}` | Grafana 프록시 설정. Grafana Pro만 비밀번호 없는 로그인 기능을 제공합니다. 우리는 Nginx 프록시를 사용하여 Iframe을 위한 비밀번호 없는 로그인을 지원합니다. |
| grafanaProxy.address | string | `"http://prometheus-grafana.monitoring"` | 클러스터 내부의 Grafana 서비스 주소 |
| grafanaProxy.enable | bool | `true` | Grafana 프록시 활성화 여부 |
| grafanaProxy.host | string | `"gpu-grafana.<your-domain>.com"` | Ingress를 통해 Grafana를 노출할 도메인 이름 |
| grafanaProxy.token | string | `"<MASKED>"` | Grafana 접근 토큰 (마스킹됨, Grafana에서 읽기 전용 토큰을 신청해야 함) |
| host | string | `"crater.<your-domain>.com"` | 서버가 바인딩할 도메인 이름 또는 IP 주소 (필수) 서버가 시작되려면 반드시 지정되어야 합니다. |
| imagePullPolicy | string | `"Always"` | 이미지 끌어오기 정책 ("IfNotPresent" | "Always", 개발 시에는 Always 사용) |
| imagePullSecrets | list | `[]` | 이미지 끌어오기 비밀 |
| images | object | `{"backend":{"repository":"ghcr.io/raids-lab/crater-backend","tag":"latest"},"buildkit":{"repository":"docker.io/moby/buildkit","tag":"v0.23.1"},"buildx":{"repository":"ghcr.io/raids-lab/buildx-client","tag":"latest"},"cronjob":{"repository":"docker.io/badouralix/curl-jq","tag":"latest"},"envd":{"repository":"ghcr.io/raids-lab/envd-client","tag":"latest"},"frontend":{"repository":"ghcr.io/raids-lab/crater-frontend","tag":"latest"},"grafanaProxy":{"repository":"docker.io/library/nginx","tag":"1.27.3-bookworm"},"nerdctl":{"repository":"ghcr.io/raids-lab/nerdctl-client","tag":"latest"},"storage":{"repository":"ghcr.io/raids-lab/storage-server","tag":"latest"}}` | 컨테이너 이미지 설정 |
| images.backend.repository | string | `"ghcr.io/raids-lab/crater-backend"` | 백엔드 서비스 이미지 저장소 |
| images.backend.tag | string | `"latest"` | 백엔드 서비스 이미지 태그 |
| images.buildkit.repository | string | `"docker.io/moby/buildkit"` | 컨테이너디 기반 빌드를 위한 Buildkit 이미지 저장소 |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit 이미지 태그 |
| images.buildx.repository | string | `"ghcr.io/raids-lab/buildx-client"` | Docker Buildx 멀티 플랫폼 빌드를 위한 Buildx 이미지 저장소 |
| images.buildx.tag | string | `"latest"` | Buildx 이미지 태그 |
| images.cronjob.repository | string | `"docker.io/badouralix/curl-jq"` | Cronjob 이미지 저장소 |
| images.cronjob.tag | string | `"latest"` | Cronjob 이미지 태그 |
| images.envd.repository | string | `"ghcr.io/raids-lab/envd-client"` | 환경 기반 개발 빌드를 위한 Envd 이미지 저장소 |
| images.envd.tag | string | `"latest"` | Envd 이미지 태그 |
| images.frontend.repository | string | `"ghcr.io/raids-lab/crater-frontend"` | 프론트엔드 서비스 이미지 저장소 |
| images.frontend.tag | string | `"latest"` | 프론트엔드 서비스 이미지 태그 |
| images.grafanaProxy.repository | string | `"docker.io/library/nginx"` | Grafana 프록시 이미지 저장소 |
| images.grafanaProxy.tag | string | `"1.27.3-bookworm"` | Grafana 프록시 이미지 태그 |
| images.nerdctl.repository | string | `"ghcr.io/raids-lab/nerdctl-client"` | 컨테이너디 기반 빌드를 위한 Nerdctl 이미지 저장소 |
| images.nerdctl.tag | string | `"latest"` | Nerdctl 이미지 태그 |
| images.storage.repository | string | `"ghcr.io/raids-lab/storage-server"` | 스토리지 서버 이미지 저장소 |
| images.storage.tag | string | `"latest"` | 스토리지 서버 이미지 태그 |
| namespaces | object | `{"create":true,"image":"crater-images","job":"crater-workspace"}` | crater 구성 요소를 위한 네임스페이스 설정. 기본적으로 crater 구성 요소는 crater 네임스페이스에서 실행되고, 작업과 이미지는 별도의 네임스페이스에서 실행됩니다. |
| namespaces.create | bool | `true` | 배포와 함께 네임스페이스를 생성할지 여부 |
| namespaces.image | string | `"crater-images"` | 이미지 빌드를 위한 네임스페이스 |
| namespaces.job | string | `"crater-workspace"` | 작업을 실행하는 네임스페이스 |
| nodeSelector | object | `{"node-role.kubernetes.io/control-plane":""}` | 모든 배포에 대한 노드 선택기. 컨트롤 구성 요소가 GPU 워커 노드에 스케줄되지 않도록 방지합니다. |
| protocol | string | `"https"` | 서버 통신 프로토콜 |
| storage | object | `{"create":true,"pvcName":"crater-rw-storage","request":"2Ti","storageClass":"ceph-fs"}` | 지속 가능한 볼륨 클레임 설정 |
| storage.create | bool | `true` | PVC 생성 여부 |
| storage.pvcName | string | `"crater-rw-storage"` | PVC 이름 (backendConfig에서도 사용됨) |
| storage.request | string | `"2Ti"` | 저장 요청 크기 |
| storage.storageClass | string | `"ceph-fs"` | 저장 클래스 이름 (예: cephfs, nfs, ReadWriteMany를 지원해야 함) |
| tls | object | `{"base":{"cert":"<MASKED>","create":false,"key":"<MASKED>"},"forward":{"cert":"<MASKED>","create":false,"key":"<MASKED>"}}` | Ingress를 통해 서비스를 노출할 때 사용하는 TLS 인증서 설정 cert-manager 설정 변수 |
| tls.base | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | 기본 인증서 설정 (표준 모드, 예: crater.example.com 인증서) |
| tls.base.cert | string | `"<MASKED>"` | 기본 인증서 내용 (마스킹됨) |
| tls.base.create | bool | `false` | 기본 인증서 생성 여부 |
| tls.base.key | string | `"<MASKED>"` | 기본 인증서 개인 키 (마스킹됨) |
| tls.forward | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | 전달 인증서 설정 (서브도메인 모드, 예: xxx.crater.example.com 인증서로 내부 작업 서비스를 외부로 노출) |
| tls.forward.cert | string | `"<MASKED>"` | 전달 인증서 내용 (마스킹됨) |
| tls.forward.create | bool | `false` | 전달 인증서 생성 여부 |
| tls.forward.key | string | `"<MASKED>"` | 전달 인증서 개인 키 (마스킹됨) |
| tolerations | list | `[{"effect":"NoSchedule","key":"node-role.kubernetes.io/control-plane","operator":"Exists"}]` | Pod 톨러레이션 |