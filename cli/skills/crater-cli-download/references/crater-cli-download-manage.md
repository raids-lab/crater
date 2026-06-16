# Crater CLI Download Manage

用户需要查看、排查或控制下载任务时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

## 典型范例

列出当前用户下载任务：

```bash
crater download ls
crater download ls --category model
crater download ls --json
```

查看单个任务：

```bash
crater download get 123 --json
```

查看日志：

```bash
crater download logs 123
crater download logs 123 --follow
```

控制任务：

```bash
crater download pause 123
crater download resume 123
crater download retry 123
crater download rm 123
```

非交互删除：

```bash
crater download rm 123 --yes --no-interactive
```

## 命令说明

- `download ls`：列出当前用户可见的下载任务，可用 `--category model|dataset` 过滤。
- `download get <ID>`：查看任务详情。
- `download logs <ID>`：获取当前日志文本。
- `download logs <ID> --follow`：持续轮询日志，直到任务进入 `Ready`、`Failed` 或 `Paused`。
- `download pause <ID>`：暂停正在下载的任务。
- `download resume <ID>`：恢复已暂停的任务。
- `download retry <ID>`：重试失败的任务。
- `download rm <ID>`：移除当前用户与任务的关联；后端会在无人引用时软删除下载记录并保留文件。

## 判断规则

- 需要脚本消费状态时使用 `--json`。
- `logs --follow` 是长时间运行的人类观察命令，不要与 `--json` 组合。
- 删除任务前确认用户意图；非交互删除必须加 `--yes`。
- `retry` 只适用于失败任务；其它状态由后端拒绝。
- `pause` 通常只适用于下载中任务；`resume` 通常只适用于暂停任务。

## 输出与字段

`download ls --json`：

```json
{
  "status": "OK",
  "data": {
    "downloads": []
  }
}
```

`download get/pause/resume/retry --json`：

```json
{
  "status": "OK",
  "data": {
    "download": {}
  }
}
```

`download logs --json`：

```json
{
  "status": "OK",
  "data": {
    "logs": "..."
  }
}
```

## 安全注意

- 日志中可能包含下载工具输出。转述给用户时避免暴露用户 token、内部路径或其它敏感值。
- 不要静默执行 `rm --yes`；只有用户明确要求跳过确认或非交互删除时才使用。

## 排查重点

- 如果列表为空，确认当前 active credentials 是否正确。
- 如果 `get/logs` 返回 not found，确认任务 ID 是否属于当前用户。
- 如果日志显示等待 Pod 启动，稍后重试或使用 `--follow`。
- 如果任务长时间 `Downloading` 且日志无进展，建议查看平台资源、网络和来源 token。
