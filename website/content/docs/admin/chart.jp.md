---
title: "設定の説明"
description: "Crater プラットフォーム Helm Chart の詳細な設定パラメータの説明。バックエンド、フロントエンド、ストレージ、監視、認証などのコアモジュールを網羅しています。"
---

<img src="https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square" alt="Version: 0.1.0"> <img src="https://img.shields.io/badge/Type-application-informational?style=flat-square" alt="Type: application"> <img src="https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square" alt="AppVersion: 1.0.0">

Crater は Kubernetes 向けに設計された総合的な AI 開発プラットフォームであり、GPU リソース管理、コンテナ化された開発環境、およびワークフローのオーケストレーション機能を提供します。本ドキュメントでは、Helm を使用して Crater をデプロイする際の設定可能なすべての項目について詳しく説明します。

**プロジェクトホームページ：** <https://github.com/raids-lab/crater>

## メンテナ

| 名前 | メール | URL |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## ソースコード

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## 基本設定 (Global Values)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| host | string | `"crater.example.com"` | プラットフォームにアクセスするためのドメイン名または IP アドレス |
| protocol | string | `"http"` | アクセスプロトコルの種類（`http` または `https`） |
| firstUser.username | string | `"crater-admin"` | 初回管理者ユーザー名 |
| firstUser.password | string | `"Masked@Password"` | 初回管理者パスワード（必ず変更してください） |
| imagePullPolicy | string | `"Always"` | コンテナイメージのプルポリシー |
| namespaces.create | bool | `true` | 名前空間を自動的に作成するかどうか |
| namespaces.job | string | `"crater-workspace"` | ジョブタスクを実行するための名前空間 |
| namespaces.image | string | `"crater-images"` | イメージをビルドするための名前空間 |
| storage.create | bool | `true` | 基本的な永続ストレージ（PVC）を自動的に作成するかどうか |
| storage.request | string | `"10Gi"` | ストレージの申請容量 |
| storage.storageClass | string | `"nfs"` | ストレージクラス名（ReadWriteMany をサポートしている必要があります） |
| storage.pvcName | string | `"crater-rw-storage"` | 共有 PVC の名前 |

## 監視の基本設定 (Monitoring)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| prometheus.enable | bool | `true` | 統合された Prometheus 監視を有効にするかどうか |
| prometheus.address | string | `"http://..."` | クラスタ内部の Prometheus サービスアドレス |
| grafana.enable | bool | `true` | 統合された Grafana ダッシュボードを有効にするかどうか |
| grafana.address | string | `"http://..."` | クラスタ内部の Grafana サービスアドレス |
| monitoring.timezone | string | `"Asia/Shanghai"` | 監視ダッシュボードの表示タイムゾーン |

## バックエンドサービス設定 (backendConfig)

### 基本サービス項目

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.port | string | `":8088"` | バックエンド API サービスのリスニングポート |
| backendConfig.enableLeaderElection | bool | `false` | Leader 選挙を有効にするかどうか（高可用性デプロイ用） |
| backendConfig.modelDownload.image | string | `"python:3.11-slim"` | モデルダウンロードジョブで使用されるコンテナイメージ |
| backendConfig.prometheusAPI | string | `"http://..."` | 監視メトリクスを取得するための Prometheus API アドレス |
| backendConfig.auth.token.accessTokenSecret | string | `"..."` | JWT Access Token の署名シークレット |
| backendConfig.auth.token.refreshTokenSecret | string | `"..."` | JWT Refresh Token の署名シークレット |

### データベース接続 (backendConfig.postgres)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.postgres.host | string | `"..."` | データベースサーバーのアドレス |
| backendConfig.postgres.port | int | `5432` | データベースのポート番号 |
| backendConfig.postgres.dbname | string | `"postgres"` | データベース名 |
| backendConfig.postgres.user | string | `"postgres"` | データベースのユーザー名 |
| backendConfig.postgres.password | string | `"..."` | データベースのパスワード |
| backendConfig.postgres.sslmode | string | `"disable"` | SSL モードの設定 |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | データベース接続に使用されるタイムゾーン |

### ストレージパスのバインド (backendConfig.storage)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.storage.pvc.readWriteMany | string | `"..."` | バインドされた共有ストレージ PVC 名 |
| backendConfig.storage.pvc.readOnlyMany | string | `null` | オプションの読み取り専用ストレージ PVC 名（データセットおよびモデル用） |
| backendConfig.storage.prefix.user | string | `"users"` | ユーザー個人スペースのストレージパスプレフィックス |
| backendConfig.storage.prefix.account | string | `"accounts"` | キュー/アカウント共有スペースのストレージパスプレフィックス |
| backendConfig.storage.prefix.public | string | `"public"` | グローバル公共データセットのストレージパスプレフィックス |

### 基本リソースとシークレット (backendConfig.secrets)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | HTTPS 用の TLS 証明書 Secret 名 |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | 転送用の TLS 証明書 Secret 名 |
| backendConfig.secrets.imagePullSecretName | string | `""` | プライベートイメージをプルするための Secret 名 |

### 認証方式の設定 (backendConfig.auth)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.auth.ldap.enable | bool | `false` | LDAP 統一認証を有効にするかどうか |
| backendConfig.auth.ldap.server.address | string | `"..."` | LDAP サーバーのアドレス |
| backendConfig.auth.ldap.server.baseDN | string | `"..."` | ユーザー検索の Base DN |
| backendConfig.auth.ldap.attributeMapping.username | string | `"uid"` | ユーザー名に対応する LDAP 属性名 |
| backendConfig.auth.ldap.attributeMapping.displayName | string | `"cn"` | 表示名に対応する LDAP 属性名 |
| backendConfig.auth.ldap.uid.source | string | `"default"` | **UID/GID 取得ポリシー**: `default`, `ldap`, `external` から選択 |
| backendConfig.auth.ldap.uid.ldapAttribute.uid | string | `""` | source が `ldap` の場合の UID 属性名 |
| backendConfig.auth.normal.allowRegister | bool | `true` | プラットフォームでの直接登録を許可するかどうか |
| backendConfig.auth.normal.allowLogin | bool | `true` | ローカルデータベースアカウントでのログインを許可するかどうか |

### イメージレジストリの統合 (backendConfig.registry)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.registry.enable | bool | `false` | コンテナイメージレジストリ（Harbor）の統合を有効にするかどうか |
| backendConfig.registry.harbor.server | string | `"..."` | Harbor サービスのアクセスアドレス |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor 管理者アカウント |
| backendConfig.registry.harbor.password | string | `"..."` | Harbor 管理者パスワード |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `null` | イメージビルド時の HTTP プロキシ |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `null` | イメージビルド時の HTTPS プロキシ |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `null` | プロキシを使用しないドメインのリスト（カンマ区切り） |

### 邮件サービス設定 (backendConfig.smtp)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| backendConfig.smtp.enable | bool | `false` | メール通知機能を有効にするかどうか |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTP サーバーのアドレス |
| backendConfig.smtp.port | int | `25` | SMTP サーバーのポート番号 |
| backendConfig.smtp.user | string | `"example"` | SMTP 認証ユーザー名 |
| backendConfig.smtp.password | string | `"..."` | SMTP 認証パスワード |
| backendConfig.smtp.notify | string | `"example@example.com"` | システム通知送信者のメールアドレス |

## イメージビルドパイプライン (buildkitConfig)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| buildkitConfig.amdConfig.enabled | bool | `false` | AMD64 アーキテクチャのビルドノードを有効にするかどうか |
| buildkitConfig.amdConfig.replicas | int | `3` | ビルドノードのレプリカ数 |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | ビルドノードのキャッシュボリュームサイズ |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | ビルドノードの最大 CPU 制限 |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | ビルドノードの最大メモリ制限 |

## 自動タスクポリシー (cronjobConfig)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | 低利用率検出の時間範囲（分） |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | アラートをトリガーする利用率の閾値 |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | バッチジョブの最大実行日数 |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Waiting 状態の Jupyter ジョブのクリーンアップ閾値 |

## データベースバックアップ (dbBackup)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| dbBackup.enabled | bool | `true` | データベースの自動バックアップを有効にするかどうか |
| dbBackup.schedule | string | `"0 2 * * *"` | バックアップの Cron 式 |
| dbBackup.config.retentionCount | int | `7` | バックアップファイルの保持数/日数 |

## 監視の表示 (frontendConfig / grafanaProxy)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| frontendConfig.version | string | `"1.0.0"` | フロントエンドアプリケーションのバージョン |
| grafanaProxy.enable | bool | `false` | Grafana プロキシを有効にするかどうか（Iframe 埋め込み用） |
| grafanaProxy.address | string | `"..."` | クラスタ内の Grafana サービスアドレス |
| grafanaProxy.token | string | `"..."` | 読み取り専用権限を持つ Grafana API トークン |

## TLS 証明書設定 (tls)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| tls.base.create | bool | `false` | Helm で基本証明書シークレットを作成するかどうか |
| tls.base.cert | string | `""` | 基本証明書の内容 (Base64) |
| tls.forward.create | bool | `false` | Helm で転送証明書シークレットを作成するかどうか |
| tls.forward.cert | string | `""` | 転送証明書の内容 (Base64) |

## コンポーネントイメージのバージョン (images)

| パラメータ | 型 | デフォルト値 | 説明 |
|-----|------|---------|-------------|
| images.backend.repository | string | `"..."` | バックエンドサービスのイメージリポジトリ |
| images.frontend.repository | string | `"..."` | フロントエンドサービスのイメージリポジトリ |
| images.storage.repository | string | `"..."` | ストレージ管理サービスのイメージリポジトリ |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkit コアイメージのタグ |

---
本ドキュメントのバージョンは、Crater v0.1.0 以降に対応しています。
