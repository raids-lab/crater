---
title: NodePort アクセスルール
description: NodePort ルールはサービスポートを公開し、外部ユーザーがクラスターのノードのIPアドレスと指定されたポート番号を通じてアクセスできるようにします。
---

## 2.1 機能紹介

**NodePort ルール**は、Kubernetesクラスター内のサービスに外部IP経由で直接アクセスできるようにします。Ingressルールとは異なり、NodePortルールはサービスのポートを公開し、外部ユーザーがクラスターのノードのIPアドレスと指定されたポート番号を通じてアクセスできるようにします。NodePortルールは、SSH接続などの外部アクセスが必要なアプリケーションに適しています。

NodePortルールにおいて、**Kubernetesは自動的にポート範囲30000から32767からサービスにポート番号を割り当てます**。例えば、クラスター内のノードにSSH接続したい場合、Kubernetesはそのサービスにポート番号を割り当て、そのポート番号を使って外部から接続できます。

**利点**：

- SSH接続などの外部から直接アクセスが必要なアプリケーションに適しています。
- サービスに自動的にポートを割り当て、設定を簡略化します。
- HTTP/HTTPSプロトコルに依存しません。

**使用シーン**：

- SSH（ポート22）を介してクラスター内のノードに接続する。
- 特定のポートを通じてアクセスが必要な他のアプリケーション。

**転送経路**：`{nodeIP}:{nodePort}`を通じてアクセス可能です。`nodeIP`はクラスター内の任意のノードのアドレス、`nodePort`は割り当てられたポート番号です。

![nodeport-intro](./img/nodeport-intro.webp)

設定後、対応するPodの`Annotations`に以下の内容が表示されます。`nodeport.crater.raids.io`を`key`として使用します：

```yaml
metadata:
  annotations:
    crater.raids.io/task-name: tensorboard-example
    nodeport.crater.raids.io/smtp: '{"name":"smtp","containerPort":25,"address":"192.168.5.82","nodePort":30631}'
    nodeport.crater.raids.io/ssh: '{"name":"ssh","containerPort":22,"address":"192.168.5.82","nodePort":32513}'
    nodeport.crater.raids.io/telnet: '{"name":"telnet","containerPort":23,"address":"192.168.5.82","nodePort":32226}'
```

## 2.2 使用例

外部IPアドレスを通じてSSH接続などのアプリケーションにアクセスしたい場合、**NodePortルール**を使用できます。例えば、VSCodeなどのツールでリモート開発を行うために、SSHポート（22ポート）を公開するNodePortルールを設定できます。

**NodePort外部アクセスルールを設定する手順は以下の通りです：**

1. 作業詳細ページで **「外部アクセスルールを設定」** をクリックします。

   ![ingress-entrance](./img/ingress-entrance.webp)

2. 表示されるダイアログで **「NodePortルールを追加」** をクリックし、**ルール名**（小文字のみ、20文字以内、重複不可）、**コンテナポート** を入力し、保存をクリックします。

   ![nodeport-new](./img/nodeport-new.webp)

3. 保存後、**対応するNodePortルール** を確認できます。

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

- **コンテナポート番号** (`containerPort`): 通常、SSHサービスに使用される**22ポート**を選択します。
- **クラスターノードアドレス** (`address`): クラスター内の任意のノードのIPアドレス。
- **割り当てられたNodePortポート** (`nodePort`): Kubernetesは自動的にポート範囲30000から32767からサービスにポート番号を割り当てます。

**アクセス方法**：

- KubernetesはSSHサービスに自動的にポート番号を割り当てます。このポート番号を使用して、外部IPを通じてSSH接続できます（例としてVSCodeによるリモート開発）。
- 例えば、Kubernetesはこのサービスにポート（例：`32513`）を割り当てます。`ssh user@<node-ip>:32513`で接続できます。

VSCodeからNodePortを通じてリモートJupyter Notebookに接続する例：

![vscode-nodeport-ssh](./img/vscode-nodeport-ssh.webp)