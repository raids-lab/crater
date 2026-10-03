---
title: TensorBoard 面板
description: 创建、访问和管理 TensorBoard 日志面板
---

# TensorBoard 面板

Crater 可以从个人目录或已有作业的日志创建 TensorBoard 面板。入口位于个人面板页以及作业操作菜单。

## 写入日志

创建单机、Jupyter、WebIDE、PyTorch DDP 或 TensorFlow PS 作业时，可设置 `TENSORBOARD_LOGDIR`。默认目录由后端在作业创建后解析为 `/home/<用户>/tensorboard-runs/<唯一作业名>`，因此展示名称相同的作业也不会混用日志目录；自定义目录必须是绝对路径。

TensorFlow PS 模板默认由 worker-0（chief）写入摘要，避免多个 worker 重复写入同一事件流。

## 创建面板

1. 打开 **TensorBoard** 创建页。
2. 添加个人目录，或选择当前用户拥有的作业作为来源。来源作业列表会加载全部分页。
3. 可添加 0 到 10 个来源。多来源会分别挂载到 `/tensorboard-runs/<作业名>`，TensorBoard 会递归发现其中的 run。
4. 提交面板。它会进入你的调度队列，Pod 启动后最多运行四天。每位用户最多同时保有 10 个等待中或活动面板。

个人目录只能位于当前用户的 `/home/<用户>` 下。作业来源只能引用自己的作业及其允许的存储挂载。创建页不会探测目录内容，请确认目录存在且包含 TensorBoard event 文件。

## 状态、访问与清理

面板状态为 `pending`、`starting`、`ready`、`failed` 或 `expired`，由 Volcano Job 阶段与 Pod Ready 条件判断；其中 `pending` 与普通 VCJob 的 Pending 阶段一致。它不代表 Service、Ingress 或 EndpointSlice 的独立健康检查。

打开面板时，Crater 会验证登录用户、面板是否仍存在以及面板所有权，再签发仅用于该面板路径的短期访问会话。单独获知面板 URL 不能绕过此校验。面板删除或自动回收后，URL 会随资源清理而失效。

## 常见问题

- 一直处于 `pending`：面板正在调度队列中等待兼容的 CPU 和内存资源，可联系管理员检查队列与节点容量。
- 一直处于 `starting`：联系管理员检查镜像拉取和 Pod 事件。
- 状态为 `failed`：查看面板错误信息；其中只返回安全的英文错误摘要，不包含集群内部细节。
- 页面可打开但没有曲线：确认选中的路径中存在 event 文件，并检查训练程序写入的实际目录。
