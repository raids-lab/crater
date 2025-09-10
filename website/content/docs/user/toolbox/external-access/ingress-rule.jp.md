---
title: Ingress アクセスルール
description: Ingress ルールを使用すると、外部の訪問者が特定のパスを通じてサービスにアクセスできるようになります。
---

## 1.1 機能紹介

**Ingress ルール**はHTTPまたはHTTPSプロトコルを通じてKubernetesクラスター内のサービスを外部に公開します。これは、**TensorBoard**、**Visdom**、**Jupyter**などのウェブベースのアプリケーションに適しています。Ingress ルールを使用することで、外部訪問者が特定のパスを通じてサービスにアクセスできるようになります。

例えば、TensorBoardやVisdomを通じてサービスにアクセスしたい場合、コンテナ内でサービスを適切なポートに公開し、それらをクラスター内のサービスにマッピングします。Ingress コントローラーはこれらのリクエストを自動的に処理し、クラスター内の対応するサービスに転送し、必要に応じてHTTPSおよびHTTPプロトコルをサポートします。

**利点**:

- ウェブサービスの公開に適しています。
- HTTP/HTTPSプロトコルをサポートしています。

**使用シーン**:

- TensorBoardへのアクセス。
- Visdomへのアクセス。
- Jupyter Notebookへのアクセス。

**転送パス**: すべてのアクセスパスは統一された形式で使用されます：`crater.act.buaa.edu.cn/ingress/{userName}-{uuid}`。ここで、`userName`はユーザー名、`uuid`は自動生成された5桁の識別子で、特定のサービスを指します。

![ingress-intro](./img/ingress-intro.webp)

設定後、対応するPodの`Annotations`に以下の内容が表示されます。`ingress.crater.raids.io`を`key`として使用します：

```yaml
metadata:
  annotations:
    crater.raids.io/task-name: tensorboard-example
    ingress.crater.raids.io/lars: '{"Name":"lars","Port":4210,"Prefix":"/ingress/liuxw24-eb05b/"}'
    ingress.crater.raids.io/tensorboard: '{"Name":"tensorboard","Port":6006,"Prefix":"/ingress/liuxw24-379e0/"}'
    ingress.crater.raids.io/notebook: '{"Name":"notebook","Port":8888,"Prefix":"/ingress/liuxw24-cce14/"}'
```

## 1.2 使用例

ウェブアプリを外部に公開したい場合、**Ingress ルール**を使用できます。例えば、TensorBoardに対してIngress ルールを設定することで、外部ユーザーがブラウザを通じてそのサービスにアクセスできるようになります。

### Ingress 外部アクセスルールの設定

**Ingress 外部アクセスルールの設定手順は以下の通りです：**

1. 作業の詳細ページで **外部アクセスルールを設定** をクリックします。

   ![ingress-entrance](./img/ingress-entrance.webp)

2. ポップアップされたダイアログで **「Ingress ルールを追加」** をクリックし、対応する**ルール名**（小文字のみで、20文字以内で、重複しない）および**コンテナポート**を入力し、保存します。

   ![ingress-new](./img/ingress-new.webp)

3. 保存が成功すると、**対応する Ingress ルール**が表示されます。

   ![ingress-tensorboard](./img/ingress-tensorboard.webp)

**設定例**：

```json
{
  "Name": "tensorboard",
  "Port": 6006,
  "Prefix": "/ingress/liuxw24-379e0/"
}
```

**フィールドの説明**：

- **ポート番号** (`port`): カスタムポート番号、ここでは `6006` と設定しています。これはTensorBoardがデフォルトで使用するポートです。
- **アクセスパス** (`prefix`): アクセスパスは `crater.act.buaa.edu.cn/ingress/{userName}-{uuid}` にマッピングされます。ここに、`userName` はユーザー名、`uuid` は自動生成された5桁の識別子です。

### コンテナ内でTensorBoardを起動する

> TensorBoardは、深層学習モデルのトレーニングプロセスなどの関連データを可視化するためのツールです。通常、ローカルのデフォルトURL（例：`http://localhost:6006/`）でサービスを起動してデータを表示します。しかし、サーバー環境やリバースプロキシなどを通じてアクセスするようなケースでは、TensorBoardを正しくアクセスするためにカスタムのbaseurlを指定する必要があります。

**baseurlを指定する方法**（コマンドラインから起動する例）：

コマンドラインからTensorBoardを起動するとき、`--logdir`パラメータを使ってログディレクトリを指定し、`--bind_all`と`--path_prefix`パラメータを使ってbaseurlに関連する設定を指定できます。

ログディレクトリが`/path/to/logs`で、baseurlを`/tensorboard`にしたい場合、以下のコマンドを使用できます：

```bash
tensorboard --logdir=/path/to/logs --bind_all --path_prefix=/tensorboard
```

ここで、`--bind_all`パラメータはTensorBoardがすべてのネットワークインターフェースにバインドするようにし、他のマシンからアクセスできるようにします（必要であれば）。

`--path_prefix`パラメータはbaseurlを指定するために使われます。この例では、`http://your_server_ip:6006/tensorboard`というURLでTensorBoardにアクセスできます（ここではデフォルトポートが6006であると仮定しています）。

**コンテナ内でTensorBoardを起動し、関連設定を行う手順は以下の通りです：**

ターミナルまたはコマンドプロンプトを開き、以下のコマンドを実行します：

```bash
tensorboard --port {port} --logdir {your-logs-dir} --bind_all --path_prefix={your-ingress-prefix}
```

各パラメータの説明は以下の通りです：

- `port`：指定するポート。デフォルトは6006です。

- `{your-logs-dir}`：ユーザーが指定するトレーニングデータの出力ディレクトリ（例：`./logs`）

- `--bind_all`：TensorBoardがすべてのネットワークインターフェースにバインドするようにします。これにより、他のマシンからアクセスできます。

- `{your-ingress-prefix}`：指定したIngressアクセスパス。この例では`/ingress/liuxw24-379e0`です（Ingress アクセスルールの設定を参照してください）。

**アクセス方法**：

- ユーザーは`gpu.act.buaa.edu.cn/ingress/{userName}-{uuid}`のパスを通じてTensorBoardにアクセスできます。以下のページが表示されます：

  ![ingress-tensorboard](./img/ingress-tb-1.webp)