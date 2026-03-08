---
title: "설정 안내"
description: "Crater 플랫폼 Helm Chart 상세 설정 파라미터 설명. 백엔드, 프론트엔드, 스토리지, 모니터링 및 인증 등 핵심 모듈을 다룹니다."
---

<img src="https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square" alt="Version: 0.1.0"> <img src="https://img.shields.io/badge/Type-application-informational?style=flat-square" alt="Type: application"> <img src="https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square" alt="AppVersion: 1.0.0">

Crater 는 Kubernetes 를 위해 설계된 종합 AI 개발 플랫폼으로, GPU 자원 관리, 컨테이너화된 개발 환경 및 워크플로우 오케스트레이션 기능을 제공합니다. 이 문서는 Helm 을 통해 Crater 를 배포할 때 설정 가능한 모든 항목을 상세히 설명합니다.

**프로젝트 홈페이지:** <https://github.com/raids-lab/crater>

## 유지 관리자

| 이름 | 이메일 | URL |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## 소스 코드

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## 기본 설정 (Global Values)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| host | string | `"crater.example.com"` | 플랫폼 접속을 위한 도메인 이름 또는 IP 주소 |
| protocol | string | `"http"` | 접속 프로토콜 유형 (`http` 또는 `https`) |
| firstUser.username | string | `"crater-admin"` | 초기 관리자 사용자 이름 |
| firstUser.password | string | `"Masked@Password"` | 초기 관리자 비밀번호 (반드시 변경하십시오) |
| imagePullPolicy | string | `"Always"` | 컨테이너 이미지 풀 정책 |
| namespaces.create | bool | `true` | 네임스페이스 자동 생성 여부 |
| namespaces.job | string | `"crater-workspace"` | 작업 태스크 실행을 위한 네임스페이스 |
| namespaces.image | string | `"crater-images"` | 이미지 빌드를 위한 네임스페이스 |
| storage.create | bool | `true`| 기본 지속성 스토리지 (PVC) 자동 생성 여부 ||
| storage.request | string | `"10Gi"` | 스토리지 신청 용량 |
| storage.storageClass | string | `"nfs"` | 스토리지 클래스 이름 (ReadWriteMany 지원 필요) |
| storage.pvcName | string | `"crater-rw-storage"` | 공유 PVC 이름 |

## 모니터링 기본 설정 (Monitoring)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| prometheus.enable | bool | `true` | 통합 Prometheus 모니터링 활성화 여부 |
| prometheus.address | string | `"http://..."` | 클러스터 내부 Prometheus 서비스 주소 |
| grafana.enable | bool | `true` | 통합 Grafana 대시보드 활성화 여부 |
| grafana.address | string | `"http://..."` | 클러스터 내부 Grafana 서비스 주소 |
| monitoring.timezone | string | `"Asia/Shanghai"` | 모니터링 대시보드 표시 시간대 |

## 백엔드 서비스 설정 (backendConfig)

### 기본 서비스 항목

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.port | string | `":8088"` | 백엔드 API 서비스 리스닝 포트 |
| backendConfig.enableLeaderElection | bool | `false` | Leader 선출 활성화 여부 (고가용성 배포용) |
| backendConfig.modelDownload.image | string | `"python:3.11-slim"` | 모델 다운로드 작업에 사용되는 컨테이너 이미지 |
| backendConfig.prometheusAPI | string | `"http://..."` | 모니터링 지표 획득을 위한 Prometheus API 주소 |
| backendConfig.auth.token.accessTokenSecret | string | `"..."` | JWT Access Token 서명 키 |
| backendConfig.auth.token.refreshTokenSecret | string | `"..."` | JWT Refresh Token 서명 키 |

### 데이터베이스 연결 (backendConfig.postgres)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.postgres.host | string | `"..."` | 데이터베이스 서버 주소 |
| backendConfig.postgres.port | int | `5432` | 데이터베이스 포트 |
| backendConfig.postgres.dbname | string | `"postgres"` | 데이터베이스 이름 |
| backendConfig.postgres.user | string | `"postgres"` | 데이터베이스 사용자 이름 |
| backendConfig.postgres.password | string | `"..."` | 데이터베이스 비밀번호 |
| backendConfig.postgres.sslmode | string | `"disable"` | SSL 모드 설정 |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | 데이터베이스 연결에 사용되는 시간대 |

### 스토리지 경로 바인딩 (backendConfig.storage)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.storage.pvc.readWriteMany | string | `"..."` | 바인딩된 공유 스토리지 PVC 이름 |
| backendConfig.storage.pvc.readOnlyMany | string | `null` | 선택적인 읽기 전용 스토리지 PVC 이름 (데이터셋 및 모델용) |
| backendConfig.storage.prefix.user | string | `"users"` | 사용자 개인 공간의 스토리지 경로 접두사 |
| backendConfig.storage.prefix.account | string | `"accounts"` | 큐/계정 공용 공간의 스토리지 경로 접두사 |
| backendConfig.storage.prefix.public | string | `"public"` | 전역 공용 데이터셋의 스토리지 경로 접두사 |

### 기본 리소스 및 비밀 키 (backendConfig.secrets)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"`| HTTPS 용 TLS 인증서 Secret 이름 ||
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | 전달용 TLS 인증서 Secret 이름 |
| backendConfig.secrets.imagePullSecretName | string | `""` | 프라이빗 이미지를 가져오기 위한 Secret 이름 |

### 인증 방식 설정 (backendConfig.auth)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.auth.ldap.enable | bool | `false` | LDAP 통합 인증 활성화 여부 |
| backendConfig.auth.ldap.server.address | string | `"..."` | LDAP 서버 주소 |
| backendConfig.auth.ldap.server.baseDN | string | `"..."` | 사용자 검색을 위한 Base DN |
| backendConfig.auth.ldap.attributeMapping.username | string | `"uid"` | 사용자 이름에 대응하는 LDAP 속성 이름 |
| backendConfig.auth.ldap.attributeMapping.displayName | string | `"cn"` | 표시 이름에 대응하는 LDAP 속성 이름 |
| backendConfig.auth.ldap.uid.source | string | `"default"` | **UID/GID 획득 전략**: `default`, `ldap`, `external` 중 선택 |
| backendConfig.auth.ldap.uid.ldapAttribute.uid | string | `""` | source가 `ldap`일 때의 UID 속성 이름 |
| backendConfig.auth.normal.allowRegister | bool | `true` | 플랫폼 내 직접 가입 허용 여부 |
| backendConfig.auth.normal.allowLogin | bool | `true` | 로컬 데이터베이스 계정 로그인 허용 여부 |

### 이미지 레지스트리 통합 (backendConfig.registry)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.registry.enable | bool | `false`| 컨테이너 이미지 레지스트리 (Harbor) 통합 활성화 여부 ||
| backendConfig.registry.harbor.server | string | `"..."` | Harbor 서비스 접속 주소 |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor 관리자 계정 |
| backendConfig.registry.harbor.password | string | `"..."` | Harbor 관리자 비밀번호 |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `null` | 이미지 빌드 시 HTTP 프록시 |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `null` | 이미지 빌드 시 HTTPS 프록시 |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `null` | 프록시를 사용하지 않는 도메인 목록 (쉼표로 구분) |

### 메일 서비스 설정 (backendConfig.smtp)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| backendConfig.smtp.enable | bool | `false` | 메일 알림 기능 활성화 여부 |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP 서버 주소 |
| backendConfig.smtp.port | int | `25` | SMTP 서버 포트 |
| backendConfig.smtp.user | string | `"example"` | SMTP 인증 사용자 이름 |
| backendConfig.smtp.password | string | `"..."` | SMTP 인증 비밀번호 |
| backendConfig.smtp.notify | string | `"example@example.com"` | 시스템 알림 발신자 이메일 주소 |

## 이미지 빌드 파이프라인 (buildkitConfig)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| buildkitConfig.amdConfig.enabled | bool | `false` | AMD64 아키텍처 빌드 노드 활성화 여부 |
| buildkitConfig.amdConfig.replicas | int | `3` | 빌드 노드 복제본 수 |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | 빌드 노드 캐시 볼륨 크기 |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | 빌드 노드 최대 CPU 제한 |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | 빌드 노드 최대 메모리 제한 |

## 자동 작업 전략 (cronjobConfig)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | 저이용률 감지 시간 범위 (분) |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | 알림 트리거 이용률 임계값 |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | 배치 작업 최대 실행 일수 |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Waiting 상태 Jupyter 작업 정리 임계값 |

## 데이터베이스 백업 (dbBackup)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| dbBackup.enabled | bool | `true` | 데이터베이스 자동 백업 활성화 여부 |
| dbBackup.schedule | string | `"0 2 * * *"` | 백업 Cron 표현식 |
| dbBackup.config.retentionCount | int | `7` | 백업 파일 보관 개수/일수 |

## 모니터링 표시 (frontendConfig / grafanaProxy)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| frontendConfig.version | string | `"1.0.0"` | 프론트엔드 애플리케이션 버전 |
| grafanaProxy.enable | bool | `false` | Grafana 프록시 활성화 여부 (Iframe 임베딩용) |
| grafanaProxy.address | string | `"..."` | 클러스터 내 Grafana 서비스 주소 |
| grafanaProxy.token | string | `"..."` | 읽기 전용 권한의 Grafana API 토큰 |

## TLS 인증서 설정 (tls)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| tls.base.create | bool | `false`| Helm 에서 기본 인증서 Secret 생성 여부 ||
| tls.base.cert | string | `""` | 기본 인증서 내용 (Base64) |
| tls.forward.create | bool | `false`| Helm 에서 전달 인증서 Secret 생성 여부 ||
| tls.forward.cert | string | `""` | 전달 인증서 내용 (Base64) |

## 컴포넌트 이미지 버전 (images)

| 파라미터 | 타입 | 기본값 | 설명 |
|-----|------|---------|-------------|
| images.backend.repository | string | `"..."` | 백엔드 서비스 이미지 저장소 |
| images.frontend.repository | string | `"..."` | 프론트엔드 서비스 이미지 저장소 |
| images.storage.repository | string | `"..."` | 스토리지 관리 서비스 이미지 저장소 |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit 핵심 이미지 태그 |

---
이 문서 버전은 Crater v0.1.0 이상 버전과 호환됩니다.
