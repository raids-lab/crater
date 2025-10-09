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

> 問題 1: ValueError: Bfloat16 は、少なくとも計算能力 8.0 の GPU のみでサポートされます。あなたの Tesla V100-SXM2-32GB GPU は計算能力 7.0 です。CLI で dtype フラグを明示的に設定して float16 を使用することができるので、たとえば --dtype=half と設定してください。

回答 1: vLLM の起動パラメータに--dtype=half を追加してください。この問題の主な原因は、多くの加速演算子が v100 では動かないため、多くの高性能演算子にはハードウェア制限があるためです。

> 問題 2: 必要な計算リソース量をどのように予測すればよいですか？

回答 2: メモリ使用量（最小）>= モデルパラメータ数 × 配置ビット数 (bit) / 8 です。例えば、インスタンスで 32b モデルを使用し、16bit で配置する場合、少なくとも 64GB のメモリが必要です。crater で使用する 1 枚の v100 のメモリは 32GB なので、実際には 3〜4 枚の v100 で動かすことが可能です。これはテスト用に複数マシンを用意したため、8 枚の GPU を使用しています。

> 問題 3: sglang を使用できますか？

回答 3: sglang フレームワークは 7.0 の GPU を使用してデプロイすることはできません。また、vllm の一部の機能は v100 で使用することはできません（具体的には vllm のドキュメントや issue セクションをご覧ください）。

これで、ジョブテンプレートを使用して高速に展開した DeepSeek R1 32b モデルと対話できます 🥳！

## Web UI インターフェースの起動と大規模モデルとの対話

**Open WebUI クライアントテンプレート** はモデルデプロイタイプのテンプレートと組み合わせて、大規模モデルの試用体験をより親しみやすく提供します。

サイドバーにあるジョブテンプレートをクリックし、その後 **Open WebUI クライアント** テンプレートを選択してください。

![](./img/dis-deepseek-32b/openweb-temp.webp)

選択後、カスタムジョブの新規作成画面に移動し、関連テンプレートパラメータが既に入力されているのが確認できます。

![](./img/dis-deepseek-32b/openweb-submit.webp)

Crater プラットフォームで **DeepSeek R1 分散推論** のタスクテンプレートを使用して大規模モデル推論サービスを起動した後、環境変数の最初の行を編集する必要があります。OpenAI サービスのアドレスは、作業の「基本情報」に記載されている **Ray Head ノードの「内網 IP」** です。

![](./img/dis-deepseek-32b/dis-ip.webp)

Open WebUI が正常に起動後、詳細ページに移動し、「外部アクセス」をクリックしてください。すでに転送設定を行っていますので、クリックするだけでアクセスできます。

![](./img/dis-deepseek-32b/openweb-fw.webp)

大規模モデルの冒険を楽しんでください 🥳！