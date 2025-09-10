---
title: DeepSeek R1 分散推論の高速展開
description: このプラットフォームは、DeepSeek R1 分散推論のジョブテンプレートを提供しており、これによりすぐに分散タスクを作成して、独自の DeepSeek を高速に展開できます。また、Web UI インターフェースを起動して大規模モデルと対話することも可能です。
---

# DeepSeek R1 分散推論の高速展開

**ジョブテンプレート** メニューには **DeepSeek R1 分散推論** のタスクテンプレートが用意されており、このテンプレートを選択することで、すぐに DeepSeek R1 分散推論を展開できます。また、Web UI インターフェースを起動して大規模モデルと対話することも可能です。

## テンプレートの選択によるジョブの作成

サイドバーにあるジョブテンプレートをクリックし、その後 **DeepSeek R1 分散推論** テンプレートを選択してください。

![](./img/dis-deepseek-32b/dis-temp.webp)

選択後、カスタムジョブの新規作成画面に移動し、関連テンプレートパラメータが既に入力されているのが確認できます。

![](./img/dis-deepseek-32b/dis-submit.webp)

## 起動コマンドおよびよくある問題

### 起動コマンド

テンプレート内の起動コマンドは次の通りです。

```bash
ray start --head --port=6667 --disable-usage-stats;
NCCL_DEBUG=TRACE python3 -m vllm.entrypoints.openai.api_server \
--model=/models/DeepSeek-R1-Distill-Qwen-32B \
--max-model-len 32768 \
--tensor-parallel-size 4 \
--pipeline-parallel-size 2 \
--gpu-memory-utilization 0.90 \
--max-num-seqs 128 \
--trust-remote-code \
--disable-custom-all-reduce \
--port 8000 \
--dtype=half;
```

### よくある問題

> 問題1: ValueError: Bfloat16は、少なくとも計算能力8.0のGPUのみでサポートされます。あなたのTesla V100-SXM2-32GB GPUは計算能力7.0です。CLIでdtypeフラグを明示的に設定してfloat16を使用することができるので、たとえば --dtype=half と設定してください。

回答1: vLLMの起動パラメータに--dtype=halfを追加してください。この問題の主な原因は、多くの加速演算子がv100では動かないため、多くの高性能演算子にはハードウェア制限があるためです。

> 問題2: 必要な計算リソース量をどのように予測すればよいですか？

回答2: メモリ使用量（最小）>= モデルパラメータ数 × 配置ビット数(bit) / 8 です。例えば、インスタンスで32bモデルを使用し、16bitで配置する場合、少なくとも64GBのメモリが必要です。craterで使用する1枚のv100のメモリは32GBなので、実際には3〜4枚のv100で動かすことが可能です。これはテスト用に複数マシンを用意したため、8枚のGPUを使用しています。

> 問題3: sglangを使用できますか？

回答3: sglangフレームワークは7.0のGPUを使用してデプロイすることはできません。また、vllmの一部の機能はv100で使用することはできません（具体的にはvllmのドキュメントやissueセクションをご覧ください）。

これで、ジョブテンプレートを使用して高速に展開したDeepSeek R1 32bモデルと対話できます 🥳！

## Web UI インターフェースの起動と大規模モデルとの対話

**Open WebUI クライアントテンプレート** はモデルデプロイタイプのテンプレートと組み合わせて、大規模モデルの試用体験をより親しみやすく提供します。

サイドバーにあるジョブテンプレートをクリックし、その後 **Open WebUI クライアント** テンプレートを選択してください。

![](./img/dis-deepseek-32b/openweb-temp.webp)

選択後、カスタムジョブの新規作成画面に移動し、関連テンプレートパラメータが既に入力されているのが確認できます。

![](./img/dis-deepseek-32b/openweb-submit.webp)

Crater プラットフォームで **DeepSeek R1 分散推論** のタスクテンプレートを使用して大規模モデル推論サービスを起動した後、環境変数の最初の行を編集する必要があります。OpenAIサービスのアドレスは、作業の「基本情報」に記載されている **Ray Headノードの「内網IP」** です。

![](./img/dis-deepseek-32b/dis-ip.webp)

Open WebUIが正常に起動後、詳細ページに移動し、「外部アクセス」をクリックしてください。すでに転送設定を行っていますので、クリックするだけでアクセスできます。

![](./img/dis-deepseek-32b/openweb-fw.webp)

大規模モデルの冒険を楽しんでください 🥳！