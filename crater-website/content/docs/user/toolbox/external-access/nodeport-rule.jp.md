---
title: NodePort アクセスルール
description: NodePort ルールはサービスポートを公開し、外部ユーザーがクラスターのノードの IP アドレスと指定されたポート番号を通じてアクセスできるようにします。
---

## 2.1 機能紹介

**NodePort ルール**は、Kubernetes クラスター内のサービスに外部 IP 経由で直接アクセスできるようにします。Ingress ルールとは異なり、NodePort ルールはサービスのポートを公開し、外部ユーザーがクラスターのノードの IP アドレスと指定されたポート番号を通じてアクセスできるようにします。NodePort ルールは、SSH 接続などの外部アクセスが必要なアプリケーションに適しています。

NodePort ルールにおいて、**Kubernetes は自動的にポート範囲 30000 から 32767 からサービスにポート番号を割り当てます**。例えば、クラスター内のノードに SSH 接続したい場合、Kubernetes はそのサービスにポート番号を割り当て、そのポート番号を使って外部から接続できます。

**利点**：

- SSH 接続などの外部から直接アクセスが必要なアプリケーションに適しています。
- サービスに自動的にポートを割り当て、設定を簡略化します。
- HTTP/HTTPSプロトコルに依存しません。

**使用シーン**：

- SSH（ポート 22）を介してクラスター内のノードに接続する。
- 特定のポートを通じてアクセスが必要な他のアプリケーション。

**転送経路**：`{nodeIP}:{nodePort}`を通じてアクセス可能です。`nodeIP`はクラスター内の任意のノードのアドレス、`nodePort`は割り当てられたポート番号です。

![nodeport-intro](./img/nodeport-intro.webp)

設定後、対応する Pod の`Annotations`に以下の内容が表示されます。`nodeport.crater.raids.io`を`key`として使用します：

```yaml
metadata:
  annotations:
    crater.raids.io/task-name: tensorboard-example
    nodeport.crater.raids.io/smtp: '{"name":"smtp","containerPort":25,"address":"192.168.5.82","nodePort":30631}'
    nodeport.crater.raids.io/ssh: '{"name":"ssh","containerPort":22,"address":"192.168.5.82","nodePort":32513}'
    nodeport.crater.raids.io/telnet: '{"name":"telnet","containerPort":23,"address":"192.168.5.82","nodePort":32226}'
```

## 2.2 使用例

外部 IP アドレスを通じて SSH 接続などのアプリケーションにアクセスしたい場合、**NodePort ルール**を使用できます。例えば、VSCode などのツールでリモート開発を行うために、SSH ポート（22 ポート）を公開する NodePort ルールを設定できます。

**NodePort 外部アクセスルールを設定する手順は以下の通りです：**

1. 作業詳細ページで **「外部アクセスルールを設定」** をクリックします。

   ![ingress-entrance](./img/ingress-entrance.webp)

2. 表示されるダイアログで **「NodePort ルールを追加」** をクリックし、**ルール名**（小文字のみ、20 文字以内、重複不可）、**コンテナポート** を入力し、保存をクリックします。

   ![nodeport-new](./img/nodeport-new.webp)

3. 保存後、**対応する NodePort ルール** を確認できます。

   ![nodeport-ssh](./img/nodeport-ssh.webp)

**設定例**：

```json
{
  "name": "ssh",
  "containerPort": 22,
  "address": "192.168.5.82",
  "nodePort": 32513
}
```

**フィールドの説明**：

- **コンテナポート番号** (`containerPort`): 通常、SSH サービスに使用される**22 ポート**を選択します。
- **クラスターノードアドレス** (`address`): クラスター内の任意のノードの IP アドレス。
- **割り当てられた NodePort ポート** (`nodePort`): Kubernetes は自動的にポート範囲 30000 から 32767 からサービスにポート番号を割り当てます。

**アクセス方法**：

- Kubernetes は SSH サービスに自動的にポート番号を割り当てます。このポート番号を使用して、外部 IP を通じて SSH 接続できます（例として VSCode によるリモート開発）。
- 例えば、Kubernetes はこのサービスにポート（例：`32513`）を割り当てます。`ssh user@<node-ip>:32513`で接続できます。

VSCode から NodePort を通じてリモート Jupyter Notebook に接続する例：

![vscode-nodeport-ssh](./img/vscode-nodeport-ssh.webp)