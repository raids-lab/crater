# Crater CLI Download Create

用户需要创建模型或数据集下载任务时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

## 典型范例

下载 Hugging Face 模型：

```bash
crater download model qwen/Qwen2.5-Coder-7B-Instruct --source hf
```

下载 ModelScope 数据集：

```bash
crater download dataset AI-ModelScope/alpaca-gpt4-data-zh --source ms
```

使用通用 create 形式：

```bash
crater download create --name qwen/Qwen2.5-Coder-7B-Instruct --category model --source hf
```

提交后等待完成或失败：

```bash
crater download model qwen/Qwen2.5-Coder-7B-Instruct --source hf --wait
```

从环境变量读取 gated/private 仓库 token：

```bash
crater download model meta-llama/Llama-2-7b-hf --source hf --token-env HF_TOKEN
```

从 stdin 读取 token，适合脚本管道：

```bash
printf '%s' "$HF_TOKEN" | crater download model meta-llama/Llama-2-7b-hf --source hf --token-stdin
```

## 命令说明

- `crater download model <NAME>`：创建模型下载任务，类别固定为 `model`。
- `crater download dataset <NAME>`：创建数据集下载任务，类别固定为 `dataset`。
- `crater download create --name <NAME> --category model|dataset`：通用创建形式。
- `--source`：请求的下载来源，支持 `modelscope` / `ms`、`huggingface` / `hf`，默认 `modelscope`；复用已有资源时可能不采用该来源。
- `--revision`：请求的分支、tag 或 revision；复用已有资源时可能不采用该版本，不能仅凭传入此参数认定已取得指定版本。
- `--wait`：创建后轮询任务，直到状态为 `Ready`、`Failed` 或 `Paused`。
- `--poll-interval`：轮询间隔，默认 `5s`。
- `--timeout`：最大等待时间，默认 `0` 表示不超时。

## 判断规则

- 如果用户明确说“模型”，优先用 `crater download model <NAME>`。
- 如果用户明确说“数据集”，优先用 `crater download dataset <NAME>`。
- 如果用户需要脚本消费结果，添加 `--json`，并从 stdout 的 `data.download` 读取任务信息。
- 如果用户需要脚本等待下载结束，可以组合 `--wait --json`，最终读取 `data.download.status`。
- 如果用户只给了来源缩写，`hf` 表示 Hugging Face，`ms` 表示 ModelScope。
- 如果用户给出 gated/private 仓库，优先建议 `--token-env`，不要让用户把 token 发到聊天里。

## 共享资源与完成判定

平台按 `name + category` 共享一份公共资源。已有 Ready、Pending、Downloading 或 Paused 记录时，后端可能复用其他来源或版本的记录，而不是新建下载任务。因此命令成功只表示请求成功处理，不保证请求的来源、版本已被采用，也不保证下载完成。

1. 创建时优先使用 `--json`；若已执行普通输出命令，用 `crater download get <id> --json` 补查。
2. 必须读取 `data.download.source`、`data.download.revision`、`data.download.status` 和 `data.download.path`，将实际来源、版本与用户要求对照；来源缩写先按 `hf` → `huggingface`、`ms` → `modelscope` 规范化后比较。
3. 来源或版本不一致时，明确说明复用了已有资源，展示实际来源、版本、状态与路径。用户明确要求的来源或版本未匹配时，报告“指定来源或版本要求尚未满足”，不能宣称指定版本已下载。返回空 `revision` 时，说明记录未指定版本，不能推断它就是用户要求的分支、tag 或 commit。
4. `Ready` 才表示资源已可用；Pending、Downloading 表示尚未完成，Paused 表示已暂停，Failed 表示失败。即使使用 `--wait`，也必须按最终状态判断，并再次核对实际来源、版本与路径。
5. 不得擅自删除共享下载记录或公共目录，也不得擅自清理、重试其他任务来绕过复用规则。用户确实需要不同来源或版本时，说明当前限制，并请平台管理员协调，不把已有公共资源当作可随意覆盖的私人文件。

例如：用户要求 Hugging Face 的 `v2`，返回的却是 ModelScope 的 `v1`、状态 Ready，应报告“已复用 ModelScope v1，位于返回路径；Hugging Face v2 的要求尚未满足”，而不是“v2 下载成功”。

## 输出与字段

`--json` 成功时：

```json
{
  "status": "OK",
  "data": {
    "download": {
      "id": 123,
      "name": "owner/name",
      "source": "huggingface",
      "category": "model",
      "revision": "main",
      "status": "Downloading",
      "path": "public/Models/owner/name"
    }
  }
}
```

实际对象还包含 `revision`、`sizeBytes`、`downloadedBytes`、`downloadSpeed`、`message`、`jobName`、`creatorId`、`referenceCount`、`createdAt`、`updatedAt` 等字段。

## 安全注意

- 不要推荐直接写 `--token <token>`，除非用户明确在安全脚本环境中需要；普通终端会留下 shell history。
- 不要在聊天中展示 token、环境变量值或完整命令历史。
- `--token-env`、`--token-stdin`、`--token` 三者互斥。

## 排查重点

- `owner/name` 格式错误会在本地直接失败。
- `crater download model <NAME>` 和 `crater download dataset <NAME>` 使用位置参数；不要给用户提示不存在的 `--name`。
- 401 通常表示 Crater 登录 token 失效，需要重新 `crater auth login`。
- 403 通常表示当前 Crater 账号权限不足。
- gated/private 仓库下载失败时，优先检查来源 token 是否可用，而不是 Crater 登录态。
