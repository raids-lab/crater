---
title: "設定説明"
description: "大学が開発したクラスタ管理プラットフォームで、インテリジェントなクラスタスケジューリングとモニタリングを提供します。"
---

![バージョン: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![タイプ: アプリケーション](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![アプリバージョン: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

Kubernetes向けの包括的なAI開発プラットフォームで、GPUリソース管理、コンテナ化された開発環境、ワークフローのオーケストレーションを提供します。

**ホームページ:** <https://github.com/raids-lab/crater>

## メンテナ

| 名前 | メール | URL |
| ---- | ------ | --- |
| RAIDS Lab |  | <https://github.com/raids-lab> |

## ソースコード

* <https://github.com/raids-lab/crater/tree/main/charts/crater>

## Values

| Key | 型 | デフォルト | 説明 |
|-----|------|---------|-------------|
| affinity | object | `{"nodeAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"preference":{"matchExpressions":[{"key":"nvidia.com/gpu.present","operator":"NotIn","values":["true"]}]},"weight":100}]}}` | Pod affinity設定 |
| backendConfig | object | `{"auth":{"accessTokenSecret":"<MASKED>","refreshTokenSecret":"<MASKED>"},"enableLeaderElection":false,"port":":8088","postgres":{"TimeZone":"Asia/Shanghai","dbname":"crater","host":"192.168.0.1","password":"<MASKED>","port":6432,"sslmode":"disable","user":"postgres"},"prometheusAPI":"http://192.168.0.1:12345","registry":{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":true,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}},"secrets":{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"},"smtp":{"enable":true,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"},"storage":{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}}` | バックエンド設定 |
| backendConfig.auth | object | `{"accessTokenSecret":"<MASKED>","refreshTokenSecret":"<MASKED>"}` | JWTベース認証用の認証トークン設定（必須）セキュアな認証のために両方のトークンシークレットを指定する必要があります |
| backendConfig.auth.accessTokenSecret | string | `"<MASKED>"` | JWTアクセストークンに署名するために使用されるシークレットキー（必須）セキュアでランダムに生成された文字列でなければなりません |
| backendConfig.auth.refreshTokenSecret | string | `"<MASKED>"` | JWTリフレッシュトークンに署名するために使用されるシークレットキー（必須）セキュアでランダムに生成された文字列でなければなりません |
| backendConfig.enableLeaderElection | bool | `false` | コントローラーマネージャーでリーダー選出を有効にして高可用性を確保します。指定されていない場合、デフォルトでfalseになります |
| backendConfig.port | string | `":8088"` | サーバーエンドポイントがリスンするネットワークポート（必須）サーバーを起動するには指定する必要があります |
| backendConfig.postgres | object | `{"TimeZone":"Asia/Shanghai","dbname":"crater","host":"192.168.0.1","password":"<MASKED>","port":6432,"sslmode":"disable","user":"postgres"}` | PostgreSQLデータベース接続設定（必須）データベース接続のためにすべてのフィールドを指定する必要があります |
| backendConfig.postgres.TimeZone | string | `"Asia/Shanghai"` | データベース接続用のタイムゾーン。指定されていない場合、システムのタイムゾーンがデフォルトになります |
| backendConfig.postgres.dbname | string | `"crater"` | 接続するデータベースの名前（必須）データベースは存在し、アクセス可能でなければなりません |
| backendConfig.postgres.host | string | `"192.168.0.1"` | PostgreSQLサーバーのホスト名またはIPアドレス（必須）アプリケーションからアクセス可能でなければなりません |
| backendConfig.postgres.password | string | `"<MASKED>"` | 認証用データベースパスワード（必須）指定されたユーザーのパスワードと一致する必要があります |
| backendConfig.postgres.port | int | `6432` | PostgreSQLサーバーのポート番号（必須）PostgreSQLでは通常5432です |
| backendConfig.postgres.sslmode | string | `"disable"` | データベース接続用のSSL/TLSモード。指定されていない場合、デフォルトで"disable"になります |
| backendConfig.postgres.user | string | `"postgres"` | 認証用データベースユーザー名（必須）適切な権限を持つユーザーでなければなりません |
| backendConfig.prometheusAPI | string | `"http://192.168.0.1:12345"` | メトリクスおよびモニタリングに使用されるPrometheus APIエンドポイントURL。指定されていない場合、Prometheus統合は無効になります |
| backendConfig.registry | object | `{"buildTools":{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}},"enable":true,"harbor":{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}}` | コンテナレジストリ設定（イメージの保存および構築用）Enableがfalseの場合、レジストリ機能は無効になります |
| backendConfig.registry.buildTools | object | `{"proxyConfig":{"httpProxy":null,"httpsProxy":null,"noProxy":null}}` | コンテナイメージ構築ツールとプロキシの設定。Registry.Enableがtrueの場合、必須です |
| backendConfig.registry.buildTools.proxyConfig | object | `{"httpProxy":null,"httpsProxy":null,"noProxy":null}` | ビルド環境のHTTPプロキシ設定。指定されていない場合、ビルドではプロキシは設定されません |
| backendConfig.registry.buildTools.proxyConfig.httpProxy | string | `nil` | ビルド環境のHTTPプロキシURL。指定されていない場合、HTTPトラフィックはプロキシされません |
| backendConfig.registry.buildTools.proxyConfig.httpsProxy | string | `nil` | ビルド環境のHTTPSプロキシURL。指定されていない場合、HTTPSトラフィックはプロキシされません |
| backendConfig.registry.buildTools.proxyConfig.noProxy | string | `nil` | プロキシを回避するドメインの一覧（カンマ区切り）。指定されていない場合、すべてのトラフィックはプロキシを通ります |
| backendConfig.registry.enable | bool | `true` | コンテナレジストリ統合を有効にする。指定されていない場合、デフォルトでfalseになります |
| backendConfig.registry.harbor | object | `{"password":"<MASKED>","server":"harbor.example.com","user":"admin"}` | Harborコンテナレジストリ統合設定。Registry.Enableがtrueの場合、すべてのHarborフィールドを指定する必要があります |
| backendConfig.registry.harbor.password | string | `"<MASKED>"` | Harbor認証用の管理者パスワード（必須）指定されたユーザーのパスワードと一致する必要があります |
| backendConfig.registry.harbor.server | string | `"harbor.example.com"` | HarborレジストリサーバーURL（必須）有効なHarborインスタンスURLでなければなりません |
| backendConfig.registry.harbor.user | string | `"admin"` | Harbor認証用の管理者ユーザー名（必須）Harbor内で適切な権限を持つユーザーでなければなりません |
| backendConfig.secrets | object | `{"imagePullSecretName":"","tlsForwardSecretName":"crater-tls-forward-secret","tlsSecretName":"crater-tls-secret"}` | 各種セキュリティコンポーネント用のKubernetesシークレット名（必須）すべてのシークレット名は既存のKubernetesシークレットに対応する必要があります |
| backendConfig.secrets.imagePullSecretName | string | `""` | プライベートレジストリからコンテナイメージをプルするためのKubernetesシークレット名。指定されていない場合、イメージプルシークレットは使用されません |
| backendConfig.secrets.tlsForwardSecretName | string | `"crater-tls-forward-secret"` | TLS転送設定用のKubernetesシークレット名（必須）適切な転送証明書を含むシークレットでなければなりません |
| backendConfig.secrets.tlsSecretName | string | `"crater-tls-secret"` | HTTPS用のTLS証明書を含むKubernetesシークレット名（必須）'tls.crt'および'tls.key'のキーを含む必要があります |
| backendConfig.smtp | object | `{"enable":true,"host":"mail.example.com","notify":"example@example.com","password":"<MASKED>","port":25,"user":"example"}` | SMTP経由の電子メール通知設定。Enableがfalseの場合、電子メール通知は無効になります |
| backendConfig.smtp.enable | bool | `true` | SMTP電子メール機能を有効にする。指定されていない場合、デフォルトでfalseになります |
| backendConfig.smtp.host | string | `"mail.example.com"` | SMTPサーバーのホスト名またはIPアドレス（Enableがtrueの場合必須）有効なSMTPサーバーでなければなりません |
| backendConfig.smtp.notify | string | `"example@example.com"` | システム通知用のデフォルト電子メールアドレス（Enableがtrueの場合必須）有効な電子メールアドレスでなければなりません |
| backendConfig.smtp.password | string | `"<MASKED>"` | SMTP認証用のパスワード（Enableがtrueの場合必須）指定されたユーザーのパスワードと一致する必要があります |
| backendConfig.smtp.port | int | `25` | SMTPサーバーのポート番号（Enableがtrueの場合必須）通常は25、465、または587です |
| backendConfig.smtp.user | string | `"example"` | SMTP認証用のユーザー名（Enableがtrueの場合必須）有効なSMTPユーザーでなければなりません |
| backendConfig.storage | object | `{"prefix":{"account":"accounts","public":"public","user":"users"},"pvc":{"readOnlyMany":null,"readWriteMany":"crater-rw-storage"}}` | パーシステントボリュームクラームとパスプレフィックス設定（必須）すべてのPVC名とプレフィックスパスを指定する必要があります |
| backendConfig.storage.prefix | object | `{"account":"accounts","public":"public","user":"users"}` | 異なる種類のストレージ場所用のパスプレフィックス（必須）すべてのプレフィックスパスを指定する必要があります |
| backendConfig.storage.prefix.account | string | `"accounts"` | アカウント関連ストレージパスのプレフィックス（必須）ストレージシステム内での有効なパスでなければなりません |
| backendConfig.storage.prefix.public | string | `"public"` | 公開アクセス可能なストレージパス用のプレフィックス（必須）ストレージシステム内での有効なパスでなければなりません |
| backendConfig.storage.prefix.user | string | `"users"` | ユーザー固有ストレージパス用のプレフィックス（必須）ストレージシステム内での有効なパスでなければなりません |
| backendConfig.storage.pvc.readOnlyMany | string | `nil` | データセットおよびモデル用のReadOnlyManyパーシステントボリュームクラーム名。ReadwriteManyと同じ下位ストレージにリンクする必要があります。指定されていない場合、データセットおよびモデルは読み書き可能になります |
| backendConfig.storage.pvc.readWriteMany | string | `"crater-rw-storage"` | 共有ストレージ用のReadWriteManyパーシステントボリュームクラーム名（必須）クラスター内でReadWriteManyアクセスモードを持つPVCでなければなりません |
| buildkitConfig | object | `{"amdConfig":{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":true,"replicas":3},"armConfig":{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2},"generalConfig":{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}}` | イメージ構築パイプライン設定。Harborなどのセルフホストされたイメージレジストリがある場合のみ完全に利用可能です |
| buildkitConfig.amdConfig | object | `{"cache":{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"},"enabled":true,"replicas":3}` | AMDアーキテクチャ設定 |
| buildkitConfig.amdConfig.cache | object | `{"maxUsedSpace":"400GB","minFreeSpace":"50GB","reservedSpace":"50GB","storageClass":"openebs-hostpath","storageSize":"400Gi"}` | AMD構築用のキャッシュ設定 |
| buildkitConfig.amdConfig.cache.maxUsedSpace | string | `"400GB"` | AMD構築キャッシュの最大使用領域 |
| buildkitConfig.amdConfig.cache.minFreeSpace | string | `"50GB"` | AMD構築キャッシュの最小空き領域 |
| buildkitConfig.amdConfig.cache.reservedSpace | string | `"50GB"` | AMD構築キャッシュの予約領域 |
| buildkitConfig.amdConfig.cache.storageClass | string | `"openebs-hostpath"` | AMD構築キャッシュ用のストレージクラス |
| buildkitConfig.amdConfig.cache.storageSize | string | `"400Gi"` | AMD構築キャッシュ用のストレージサイズ |
| buildkitConfig.amdConfig.enabled | bool | `true` | AMDアーキテクチャ構築を有効にする |
| buildkitConfig.amdConfig.replicas | int | `3` | AMD構築レプリカ数 |
| buildkitConfig.armConfig | object | `{"cache":{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"},"enabled":false,"replicas":2}` | ARMアーキテクチャ設定 |
| buildkitConfig.armConfig.cache | object | `{"maxUsedSpace":"80GB","minFreeSpace":"10GB","reservedSpace":"10GB","storageClass":"openebs-hostpath","storageSize":"80Gi"}` | ARM構築用のキャッシュ設定 |
| buildkitConfig.armConfig.cache.maxUsedSpace | string | `"80GB"` | ARM構築キャッシュの最大使用領域 |
| buildkitConfig.armConfig.cache.minFreeSpace | string | `"10GB"` | ARM構築キャッシュの最小空き領域 |
| buildkitConfig.armConfig.cache.reservedSpace | string | `"10GB"` | ARM構築キャッシュの予約領域 |
| buildkitConfig.armConfig.cache.storageClass | string | `"openebs-hostpath"` | ARM構築キャッシュ用のストレージクラス |
| buildkitConfig.armConfig.cache.storageSize | string | `"80Gi"` | ARM構築キャッシュ用のストレージサイズ |
| buildkitConfig.armConfig.enabled | bool | `false` | ARMアーキテクチャ構築を有効にする（ARMノードが存在する場合のみtrueにできます） |
| buildkitConfig.armConfig.replicas | int | `2` | ARM構築レプリカ数 |
| buildkitConfig.generalConfig | object | `{"resources":{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}}` | すべてのアーキテクチャ用の一般設定 |
| buildkitConfig.generalConfig.resources | object | `{"limits":{"cpu":16,"memory":"48Gi"},"requests":{"cpu":8,"memory":"24Gi"}}` | リソース設定 |
| buildkitConfig.generalConfig.resources.limits.cpu | int | `16` | CPU制限 |
| buildkitConfig.generalConfig.resources.limits.memory | string | `"48Gi"` | メモリ制限 |
| buildkitConfig.generalConfig.resources.requests.cpu | int | `8` | CPUリクエスト |
| buildkitConfig.generalConfig.resources.requests.memory | string | `"24Gi"` | メモリリクエスト |
| cronjobConfig | object | `{"jobs":{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}}` | Cronjob管理戦略設定。低利用率電子メール通知およびクリーンアップ、長時間使用電子メール通知およびクリーンアップなどのジョブスケジューリング管理戦略 |
| cronjobConfig.jobs | object | `{"longTime":{"BATCH_DAYS":"4","INTERACTIVE_DAYS":"4","schedule":"*/5 * * * *"},"lowGPUUtil":{"TIME_RANGE":"90","UTIL":"0","WAIT_TIME":"30","schedule":"*/5 * * * *"},"waitingJupyter":{"JUPYTER_WAIT_MINUTES":"5","schedule":"*/5 * * * *"}}` | ジョブ管理タスク設定 |
| cronjobConfig.jobs.longTime.BATCH_DAYS | string | `"4"` | バッチジョブの最大日数 |
| cronjobConfig.jobs.longTime.INTERACTIVE_DAYS | string | `"4"` | 対話型ジョブの最大日数 |
| cronjobConfig.jobs.longTime.schedule | string | `"*/5 * * * *"` | 長時間使用チェックのスケジュール |
| cronjobConfig.jobs.lowGPUUtil.TIME_RANGE | string | `"90"` | モニタリング用の時間範囲（分） |
| cronjobConfig.jobs.lowGPUUtil.UTIL | string | `"0"` | GPU利用率のしきい値 |
| cronjobConfig.jobs.lowGPUUtil.WAIT_TIME | string | `"30"` | アクション前の待ち時間（分） |
| cronjobConfig.jobs.lowGPUUtil.schedule | string | `"*/5 * * * *"` | 低GPU利用率チェックのスケジュール |
| cronjobConfig.jobs.waitingJupyter.JUPYTER_WAIT_MINUTES | string | `"5"` | Jupyter待機時間（分） |
| cronjobConfig.jobs.waitingJupyter.schedule | string | `"*/5 * * * *"` | 待機Jupyterチェックのスケジュール |
| firstUser | object | `{"password":"<MASKED>","username":"crater-admin"}` | 最初のユーザー設定。データベースに初めて接続するときに、管理者権限を持つ最初のユーザーを作成します |
| firstUser.password | string | `"<MASKED>"` | 最初の管理者ユーザーのパスワード（このパスワードをリセットしてください） |
| firstUser.username | string | `"crater-admin"` | 最初の管理者ユーザーのユーザー名 |
| frontendConfig | object | `{"grafana":{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}},"url":{"apiPrefix":"/api/v1","document":"https://raids-lab.github.io/crater/zh"},"version":"1.0.0"}` | フロントエンド設定 |
| frontendConfig.grafana | object | `{"job":{"basic":"/d/R4ZPFfyIz/crater-job-basic-dashboard","nvidia":"/d/2CDE0AC7/crater-job-nvidia-dashboard","pod":"/d/MhnFUFLSz/crater-pod-dashboard"},"node":{"basic":"/d/k8s_views_nodes/crater-node-basic-dashboard","nvidia":"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"},"overview":{"main":"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad","network":"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking","schedule":"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"},"user":{"nvidia":"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"}}` | Grafanaダッシュボード設定。参考: https://github.com/raids-lab/crater/tree/main/grafana-dashboards |
| frontendConfig.grafana.job.basic | string | `"/d/R4ZPFfyIz/crater-job-basic-dashboard"` | 基本ジョブダッシュボードURL |
| frontendConfig.grafana.job.nvidia | string | `"/d/2CDE0AC7/crater-job-nvidia-dashboard"` | NVIDIAジョブダッシュボードURL |
| frontendConfig.grafana.job.pod | string | `"/d/MhnFUFLSz/crater-pod-dashboard"` | PodダッシュボードURL |
| frontendConfig.grafana.node.basic | string | `"/d/k8s_views_nodes/crater-node-basic-dashboard"` | 基本ノードダッシュボードURL |
| frontendConfig.grafana.node.nvidia | string | `"/d/nvidia-dcgm-dashboard/crater-node-nvidia-dashboard"` | NVIDIAノードダッシュボードURL |
| frontendConfig.grafana.overview.main | string | `"/d/f33ade9f-821d-4e96-a7f2-eb16c8b9c447/838ffad"` | メイン概要ダッシュボードURL |
| frontendConfig.grafana.overview.network | string | `"/d/8b7a8b326d7a6f1f04y7fh66368c67af/networking"` | ネットワークダッシュボードURL |
| frontendConfig.grafana.overview.schedule | string | `"/d/be9oh7yk24jy8f/crater-gpu-e8b083-e5baa6-e58f82-e88083"` | スケジュールダッシュボードURL |
| frontendConfig.grafana.user.nvidia | string | `"/d/user-nvidia-dcgm-dashboard/crater-user-nvidia-dashboard"` | ユーザーNVIDIAダッシュボードURL |
| frontendConfig.url.apiPrefix | string | `"/api/v1"` | バックエンドAPIプレフィックス（現在変更不可） |
| frontendConfig.url.document | string | `"https://raids-lab.github.io/crater/zh"` | ドキュメントのベースURL |
| frontendConfig.version | string | `"1.0.0"` | フロントエンドバージョン |
| grafanaProxy | object | `{"address":"http://prometheus-grafana.monitoring","enable":true,"host":"gpu-grafana.<your-domain>.com","token":"<MASKED>"}` | Grafanaプロキシ設定。Grafana Proのみパスワードなしログイン機能があります。IframeでパスワードなしログインをサポートするためにNginxプロキシを使用します |
| grafanaProxy.address | string | `"http://prometheus-grafana.monitoring"` | クラスター内のGrafanaサービスアドレス |
| grafanaProxy.enable | bool | `true` | Grafanaプロキシを有効にするかどうか |
| grafanaProxy.host | string | `"gpu-grafana.<your-domain>.com"` | Ingressを通じてGrafanaを公開するためのドメイン名 |
| grafanaProxy.token | string | `"<MASKED>"` | Grafanaアクセストークン（マスキング済み、Grafanaで読み取り専用トークンを申請してください） |
| host | string | `"crater.<your-domain>.com"` | サーバーがバインドするドメイン名またはIPアドレス（必須）サーバーを起動するには指定する必要があります |
| imagePullPolicy | string | `"Always"` | イメージプルポリシー（"IfNotPresent" | "Always"、開発ではAlwaysを使用） |
| imagePullSecrets | list | `[]` | イメージプルシークレット |
| images | object | `{"backend":{"repository":"ghcr.io/raids-lab/crater-backend","tag":"latest"},"buildkit":{"repository":"docker.io/moby/buildkit","tag":"v0.23.1"},"buildx":{"repository":"ghcr.io/raids-lab/buildx-client","tag":"latest"},"cronjob":{"repository":"docker.io/badouralix/curl-jq","tag":"latest"},"envd":{"repository":"ghcr.io/raids-lab/envd-client","tag":"latest"},"frontend":{"repository":"ghcr.io/raids-lab/crater-frontend","tag":"latest"},"grafanaProxy":{"repository":"docker.io/library/nginx","tag":"1.27.3-bookworm"},"nerdctl":{"repository":"ghcr.io/raids-lab/nerdctl-client","tag":"latest"},"storage":{"repository":"ghcr.io/raids-lab/storage-server","tag":"latest"}}` | コンテナイメージ設定 |
| images.backend.repository | string | `"ghcr.io/raids-lab/crater-backend"` | バックエンドサービスイメージリポジトリ |
| images.backend.tag | string | `"latest"` | バックエンドサービスイメージタグ |
| images.buildkit.repository | string | `"docker.io/moby/buildkit"` | containerdベースの構築用Buildkitイメージリポジトリ |
| images.buildkit.tag | string | `"v0.23.1"` | Buildkitイメージタグ |
| images.buildx.repository | string | `"ghcr.io/raids-lab/buildx-client"` | Docker Buildxマルチプラットフォーム構築用Buildxイメージリポジトリ |
| images.buildx.tag | string | `"latest"` | Buildxイメージタグ |
| images.cronjob.repository | string | `"docker.io/badouralix/curl-jq"` | Cronjobイメージリポジトリ |
| images.cronjob.tag | string | `"latest"` | Cronjobイメージタグ |
| images.envd.repository | string | `"ghcr.io/raids-lab/envd-client"` | 環境ベースの開発構築用Envdイメージリポジトリ |
| images.envd.tag | string | `"latest"` | Envdイメージタグ |
| images.frontend.repository | string | `"ghcr.io/raids-lab/crater-frontend"` | フロントエンドサービスイメージリポジトリ |
| images.frontend.tag | string | `"latest"` | フロントエンドサービスイメージタグ |
| images.grafanaProxy.repository | string | `"docker.io/library/nginx"` | Grafanaプロキシイメージリポジトリ |
| images.grafanaProxy.tag | string | `"1.27.3-bookworm"` | Grafanaプロキシイメージタグ |
| images.nerdctl.repository | string | `"ghcr.io/raids-lab/nerdctl-client"` | containerdベースの構築用Nerdctlイメージリポジトリ |
| images.nerdctl.tag | string | `"latest"` | Nerdctlイメージタグ |
| images.storage.repository | string | `"ghcr.io/raids-lab/storage-server"` | ストレージサーバーアイメージリポジトリ |
| images.storage.tag | string | `"latest"` | ストレージサーバーアイメージタグ |
| namespaces | object | `{"create":true,"image":"crater-images","job":"crater-workspace"}` | craterコンポーネント用の名前空間設定。デフォルトではcraterコンポーネントはcrater名前空間で実行され、ジョブとイメージは別の名前空間にあります |
| namespaces.create | bool | `true` | デプロイメントとともに名前空間を作成するかどうか |
| namespaces.image | string | `"crater-images"` | イメージを構築するための名前空間 |
| namespaces.job | string | `"crater-workspace"` | ジョブを実行するための名前空間 |
| nodeSelector | object | `{"node-role.kubernetes.io/control-plane":""}` | すべてのDeployment用のノードセレクター。コントロールコンポーネントがGPUワーカーノードにスケジュールされないようにします |
| protocol | string | `"https"` | サーバー通信用プロトコル |
| storage | object | `{"create":true,"pvcName":"crater-rw-storage","request":"2Ti","storageClass":"ceph-fs"}` | パーシステントボリュームクラーム設定 |
| storage.create | bool | `true` | PVCを作成するかどうか |
| storage.pvcName | string | `"crater-rw-storage"` | PVC名（backendConfigでも使用されます） |
| storage.request | string | `"2Ti"` | ストレージリクエストサイズ |
| storage.storageClass | string | `"ceph-fs"` | ストレージクラス名（例：cephfs、nfs、ReadWriteManyをサポートする必要があります） |
| tls | object | `{"base":{"cert":"<MASKED>","create":false,"key":"<MASKED>"},"forward":{"cert":"<MASKED>","create":false,"key":"<MASKED>"}}` | Ingress経由でサービスを公開するためのTLS証明書設定（cert-manager設定変数） |
| tls.base | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | ベース証明書設定（標準モード、例：crater.example.com証明書） |
| tls.base.cert | string | `"<MASKED>"` | ベース証明書の内容（マスキング済み） |
| tls.base.create | bool | `false` | ベース証明書を作成するかどうか |
| tls.base.key | string | `"<MASKED>"` | ベース証明書の秘密鍵（マスキング済み） |
| tls.forward | object | `{"cert":"<MASKED>","create":false,"key":"<MASKED>"}` | フォワード証明書設定（サブドメインモード、例：xxx.crater.example.com証明書で内部ジョブサービスを外部に公開） |
| tls.forward.cert | string | `"<MASKED>"` | フォワード証明書の内容（マスキング済み） |
| tls.forward.create | bool | `false` | フォワード証明書を作成するかどうか |
| tls.forward.key | string | `"<MASKED>"` | フォワード証明書の秘密鍵（マスキング済み） |
| tolerations | list | `[{"effect":"NoSchedule","key":"node-role.kubernetes.io/control-plane","operator":"Exists"}]` | Podのターレレーション |