# Crater Agent 评测场景完整分析与实验设计

## 一、场景覆盖度重新评估

### 之前标记为 "Degraded" 的场景为什么其实可以做到 Complete

之前的评估过于保守。重新审查工具链后发现：**所有 10 个场景都可以用现有工具完整支撑**。

| 场景 | 之前评估 | 修正后 | 原因 |
|------|---------|--------|------|
| RQ2.4 Prometheus PVC 满 | ⚠️ Degraded | ✅ Complete | `prometheus_query` 可以自查 Prometheus 健康状态；`list_storage_pvcs` + `get_pvc_detail` 查 PVC 容量；`k8s_list_pods` + `k8s_get_pod_logs` 查 cleaner controller。**缺的不是工具，是 Agent 的领域知识**——需要在 prompt 中注入关键 PromQL 模板 |

为什么之前误判：
- "缺 cleaner controller 状态工具" → Agent 可以用 `k8s_list_pods -l app=cleaner-controller` + `k8s_get_pod_logs` 组合实现
- "缺监控数据断点检测" → Agent 可以用 `prometheus_query("changes(up[1h])")` 检测 scrape 中断
- "缺 PVC 使用率" → Agent 可以用 `prometheus_query("kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes")` 查询
- 这些本质上是 **prompt 工程问题**，不是工具缺口

### 最终场景×工具覆盖度

| # | 场景 | 核心工具链 | 覆盖度 |
|---|------|-----------|--------|
| 1 | 单机 OOM | `get_job_detail` → `get_job_events` → `get_job_logs` → `diagnose_job` | ✅ 完整 |
| 2 | DDP NCCL 超时 | `get_ddp_rank_mapping` → `diagnose_distributed_job_network` → `k8s_get_pod_logs` | ✅ 完整 |
| 3 | Jupyter 不可达 | `k8s_get_service` → `k8s_get_endpoints` → `k8s_get_ingress` → `k8s_get_pod_logs` | ✅ 完整 |
| 4 | Prometheus PVC 满 | `prometheus_query`(自检) → `list_storage_pvcs` → `k8s_list_pods`(cleaner) → `k8s_get_pod_logs` | ✅ 完整 |
| 5 | RDMA 死锁 | `get_node_kernel_diagnostics` → `get_rdma_interface_status` → `diagnose_distributed_job_network` | ✅ 完整 |
| 6 | Harbor 故障 | `harbor_check` → `aggregate_image_pull_errors` → `k8s_get_events` | ✅ 完整 |
| 7 | 空跑作业 | `detect_idle_jobs` → `query_job_metrics` → `list_cluster_jobs` | ✅ 完整 |
| 8 | 队列公平性 | `get_volcano_queue_state` → `get_realtime_capacity` → `list_cluster_jobs` | ✅ 完整 |
| 9 | Cordon 节点 | `k8s_list_nodes` → `cordon_node` → `k8s_list_pods`(受影响作业) → `notify_job_owner` | ✅ 完整 |
| 10 | 批量停僵尸 | `detect_zombie_jobs` → `batch_stop_jobs` → `notify_job_owner` | ✅ 完整 |

---

## 二、10 个场景完整互动演示

### 场景 1：单机训练 OOMKilled 诊断与建议

**触发条件**：用户提交单机 PyTorch 训练作业，运行数分钟后 OOMKilled。

```
👤 用户：我的训练作业 sg-abc123 挂了，帮我看看怎么回事。

🤖 Agent：我来查一下这个作业的状态。
   [调用 get_job_detail(job_name="sg-abc123")]
   → 结果: status=Failed, exit_code=137, reason="OOMKilled", 
     资源配置: cpu=4, memory=16Gi, gpu=1×A100

🤖 Agent：作业因 OOMKilled (退出码 137) 失败。让我进一步查看事件和日志。
   [调用 get_job_events(job_name="sg-abc123")]
   → 结果: 事件显示 "Container killed due to OOM", "Memory cgroup limit exceeded"
   
   [调用 get_job_logs(job_name="sg-abc123", tail=100, keyword="memory|oom|killed")]
   → 结果: 日志中有 "RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB..."

🤖 Agent：作业触发了 GPU 显存 OOM（不是系统内存 OOM）。让我用诊断引擎确认。
   [调用 diagnose_job(job_name="sg-abc123")]
   → 结果: {
       category: "GPU_OOM",
       severity: "high",
       confidence: 0.95,
       root_cause: "GPU显存不足，模型分配2GiB显存但可用显存不够",
       suggestions: ["减小batch_size", "启用gradient_checkpointing", "使用混合精度训练"]
     }

🤖 Agent：诊断结果：您的作业因 GPU 显存不足 (GPU OOM) 失败。
   
   **根因**：模型尝试分配 2 GiB GPU 显存但可用空间不足。
   
   **建议**：
   1. 减小 batch_size（当前可能过大）
   2. 启用 `gradient_checkpointing` 减少显存占用
   3. 使用混合精度训练 (`torch.cuda.amp`)
   
   需要我帮你用更大的内存配置重新提交作业吗？

👤 用户：帮我改成 32Gi 内存重新提交。

🤖 Agent：好的，我帮你用 32Gi 内存重新提交。
   [调用 resubmit_job(job_name="sg-abc123", memory="32Gi")]
   → [需要用户确认] 确认重新提交作业？
   
👤 用户：[确认]

🤖 Agent：作业已重新提交为 sg-abc456。注意 GPU OOM 是显存问题，增加系统内存可能不够，
   建议同时在代码中加入 gradient_checkpointing。
```

**评测点**：
- 准确识别 OOMKilled 故障类别
- 区分系统内存 OOM 和 GPU 显存 OOM
- 给出具体可执行的建议
- 能正确执行重新提交操作

---

### 场景 2：DDP 分布式训练 NCCL 通信超时

**触发条件**：2 节点 4 GPU DDP 训练，运行 30 分钟后 NCCL timeout。

```
👤 管理员：分布式训练作业 sg-ddp789 卡住了，所有节点都报 NCCL 超时。

🤖 Agent：我来查看这个分布式作业。先获取 rank→Pod 的映射关系。
   [调用 get_ddp_rank_mapping(job_name="sg-ddp789")]
   → 结果: {
       total_ranks: 4,
       ranks: [
         {rank: 0, pod: "sg-ddp789-worker-0", node: "gpu-node-01", pod_ip: "10.244.1.5", ready: true},
         {rank: 1, pod: "sg-ddp789-worker-1", node: "gpu-node-01", pod_ip: "10.244.1.6", ready: true},
         {rank: 2, pod: "sg-ddp789-worker-2", node: "gpu-node-02", pod_ip: "10.244.2.3", ready: true},
         {rank: 3, pod: "sg-ddp789-worker-3", node: "gpu-node-02", pod_ip: "10.244.2.4", ready: false},
       ]
     }

🤖 Agent：发现 rank 3 (gpu-node-02 上的 sg-ddp789-worker-3) 不处于 Ready 状态。让我检查网络诊断。
   [调用 diagnose_distributed_job_network(job_name="sg-ddp789")]
   → 结果: {
       issues: ["NCCL WARN Timeout exceeded", "NCCL INFO NET/IB: connection refused from rank 3"],
       affected_ranks: [3],
       keyword_hits: 12
     }

🤖 Agent：NCCL 日志显示 rank 3 通信拒绝。让我查看 rank 3 的 Pod 日志。
   [调用 k8s_get_pod_logs(pod_name="sg-ddp789-worker-3", tail=100)]
   → 日志: "NCCL WARN Bootstrap: Connection refused (gpu-node-02:28536)
            NCCL INFO NET/IB : No device found."

🤖 Agent：rank 3 报告 "No IB device found"，这说明 gpu-node-02 的 RDMA 接口可能有问题。
   让我检查节点的 RDMA 状态。
   [调用 get_rdma_interface_status(node_name="gpu-node-02")]
   → 结果: {
       ib_devices: [],
       rdma_links: [],
       ib_port_states: [],
       kernel_modules: ["mlx5_core", "mlx5_ib"]  // 模块加载但无设备
     }

🤖 Agent：诊断结论：**gpu-node-02 的 IB 设备未被识别**。
   
   **因果链**：
   1. gpu-node-02 RDMA 接口异常（mlx5 驱动加载但无 IB 设备）
   2. → rank 3 无法建立 NCCL IB 通信通道
   3. → 所有其他 rank 等待 rank 3 超时
   4. → 整个 DDP 训练卡死
   
   **建议操作**：
   1. 将 gpu-node-02 标记为不可调度（cordon）
   2. 将受影响的作业迁移到健康节点重新提交
   3. 通知运维团队检查 gpu-node-02 的 InfiniBand 硬件/驱动
   
   需要我执行 cordon 操作吗？

👤 管理员：执行 cordon。

🤖 Agent：[调用 cordon_node(node_name="gpu-node-02", reason="IB device not found, RDMA unavailable")]
   → [需要确认] 将 gpu-node-02 标记为不可调度？风险等级：高
   
👤 管理员：[确认]

🤖 Agent：gpu-node-02 已 cordon。受影响的作业建议重新提交到其他节点。
```

**评测点**：
- 跨层因果链推理：RDMA 硬件(L1) → NCCL 通信(L5)
- 准确定位故障 rank 和故障节点
- 给出合理的缓解操作建议

---

### 场景 3：Jupyter/WebIDE 会话不可达

**触发条件**：用户报告 Jupyter 打不开，但 Pod 显示 Running。

```
👤 用户：我的 Jupyter 作业 jpt-web001 显示运行中，但浏览器打不开。

🤖 Agent：作业状态显示运行中但无法访问，让我检查网络链路。
   [调用 get_job_detail(job_name="jpt-web001")]
   → 结果: status=Running, node=gpu-node-03, pod=jpt-web001-worker-0

🤖 Agent：Pod 确实在运行。让我检查 Service、Endpoints 和 Ingress。
   [调用 k8s_get_service(namespace="crater-jobs", name="jpt-web001-svc")]
   → 结果: {type: "ClusterIP", cluster_ip: "10.96.45.12", ports: [{port: 8888, target_port: 8888}]}
   
   [调用 k8s_get_endpoints(namespace="crater-jobs", name="jpt-web001-svc")]
   → 结果: {subsets: [{addresses: [], not_ready_addresses: [{ip: "10.244.3.7", target_ref: "jpt-web001-worker-0"}]}]}

🤖 Agent：关键发现！Endpoints 中 Pod 地址在 `not_ready_addresses` 而不是 `addresses`。
   这意味着 Pod 的 readiness probe 未通过。让我查看 Pod 详情。
   [调用 k8s_describe_resource(kind="Pod", name="jpt-web001-worker-0", namespace="crater-jobs")]
   → 结果: readinessProbe: HTTP GET :8888/api, 最近检查: Failure
   
   [调用 k8s_get_pod_logs(pod_name="jpt-web001-worker-0", namespace="crater-jobs", tail=50)]
   → 日志: "jupyter_server: ERROR: ... Address already in use: port 8888"

🤖 Agent：诊断结论：**Jupyter Server 端口冲突导致无法启动**。
   
   **因果链**：
   1. Jupyter Server 启动时端口 8888 已被占用（可能是上一次会话残留进程）
   2. → Server 无法监听 8888 端口
   3. → readiness probe 失败
   4. → Endpoints 不包含该 Pod 地址
   5. → 通过 Ingress/Service 的流量无法到达 Pod
   6. → 用户浏览器报错
   
   **建议**：删除 Pod 触发重建，让新 Pod 获得干净的网络栈。
   
👤 用户：好，帮我重建。

🤖 Agent：[调用 delete_pod(name="jpt-web001-worker-0", namespace="crater-jobs")]
   → [需要确认] 删除 Pod？Kubernetes 会自动重建。
```

**评测点**：
- 从 Service → Endpoints → Pod 的三层网络排查
- 正确识别 readiness probe 失败 → Endpoints 不就绪的因果关系
- 在日志中找到根因

---

### 场景 4：Prometheus PVC 满 → 作业自动释放失效

**触发条件**：Prometheus 存储满，监控数据缺失，导致依赖监控数据的自动释放机制失效。

```
👤 管理员：好像集群的自动作业释放不工作了，有些明显闲置的作业没被清理。

🤖 Agent：让我检查自动释放机制的依赖链。先看 Prometheus 是否正常。
   [调用 prometheus_query(query="up{job='prometheus'}", query_type="instant")]
   → 结果: [] (空结果！Prometheus 无法查询自身)

🤖 Agent：Prometheus 查询返回空结果，说明 Prometheus 可能不健康。让我检查其 Pod 状态。
   [调用 k8s_list_pods(namespace="monitoring", label_selector="app.kubernetes.io/name=prometheus")]
   → 结果: [{name: "prometheus-0", phase: "Running", restart_count: 15}]

🤖 Agent：Prometheus Pod 频繁重启(15次)。让我检查日志和存储。
   [调用 k8s_get_pod_logs(pod_name="prometheus-0", namespace="monitoring", tail=100)]
   → 日志: "level=error msg='opening storage failed' err='no space left on device'"
   
   [调用 list_storage_pvcs(namespace="monitoring", status="Bound")]
   → 结果: [{name: "prometheus-data-prometheus-0", capacity: "100Gi", storage_class: "local-path"}]
   
   [调用 get_pvc_detail(pvc_name="prometheus-data-prometheus-0", namespace="monitoring")]
   → 结果: {capacity: "100Gi", status: "Bound"}

🤖 Agent：PVC 绑定正常但日志显示"no space left on device"。让我用 PromQL 确认具体用量。
   由于 Prometheus 自身无法查询，让我查看节点级存储指标。
   [调用 k8s_get_events(namespace="monitoring", field_selector="involvedObject.name=prometheus-0")]
   → 结果: 多条 "Back-off restarting failed container" 事件

🤖 Agent：诊断结论：**Prometheus PVC 存储空间耗尽 → 监控链路断裂 → 自动释放失效**。
   
   **因果链**（三层关联 L3→L4→L5）：
   1. **L3 存储层**：Prometheus PVC 100Gi 已满（"no space left on device"）
   2. **L4 监控层**：Prometheus 无法写入 TSDB → CrashLoop → 监控数据缺失
   3. **L5 作业层**：自动释放机制依赖 GPU 利用率指标 → 指标查不到 → 不触发释放
   
   **建议操作**（按优先级）：
   1. 紧急：扩容 Prometheus PVC 或清理历史数据（`tsdb compact`）
   2. 中期：配置 Prometheus retention 策略（如 `--storage.tsdb.retention.size=80GB`）
   3. 验证：Prometheus 恢复后检查自动释放机制是否重新生效
   
   需要我查看集群当前有多少作业因此未被清理吗？

👤 管理员：查一下。

🤖 Agent：[调用 detect_idle_jobs(gpu_threshold=5, hours=24)]
   → 结果: 12 个闲置作业，总计占用 24 GPU
   
🤖 Agent：当前有 12 个闲置作业（GPU 利用率 <5%），占用 24 张 GPU。
   这些作业在 Prometheus 故障期间未被自动释放。
   需要我批量通知作业所有者释放资源吗？
```

**评测点**：
- 跨三层因果链推理：存储(L3) → 监控(L4) → 作业(L5)
- Agent 在 Prometheus 不可用时的备选诊断路径
- 给出分层次的修复建议

---

### 场景 5：节点 RDMA 驱动死锁 → 分布式训练静默卡住

**触发条件**：分布式训练无错误日志但完全卡住，所有进程 D 状态。

```
👤 管理员：sg-large-train 训练作业卡了2小时了，没有任何日志输出，GPU 利用率掉到 0。

🤖 Agent：让我检查作业状态和节点状况。
   [调用 get_job_detail(job_name="sg-large-train")]
   → 结果: status=Running, nodes=[gpu-node-05, gpu-node-06], 运行 6 小时
   
   [调用 query_job_metrics(job_name="sg-large-train", metrics=["gpu_util"], time_range="last_2h")]
   → 结果: gpu_util avg=0.1% (最近 2 小时几乎为 0)

🤖 Agent：GPU 利用率确实掉到接近 0。让我查看节点内核状态。
   [调用 get_node_kernel_diagnostics(node_name="gpu-node-05")]
   → 结果: {
       kernel_version: "5.15.0-91-generic",
       load_avg: "128.05 127.98 126.32",  // 极高负载！
       dmesg_highlights: [
         "[T3456] INFO: task nccl-net:28536 blocked for more than 120 seconds.",
         "[T3456] INFO: task nccl-net:28536 blocked for more than 120 seconds. Tainted: G   OE",
         "RCU detected CPU 12 stall for 30 seconds",
         "mlx5_core 0000:3b:00.0: cmd_work_handler: timeout on command"
       ],
       d_state_processes: [
         "root  28536  D  nccl-net",
         "root  28537  D  nccl-net",
         "root  28540  D  mlx5_cmd"
       ],
       d_state_count: 3
     }

🤖 Agent：关键发现！gpu-node-05 上有严重的内核级问题：
   - **负载 128**（正常应 <8），说明大量进程卡住
   - dmesg 显示 **mlx5 命令超时** + **NCCL 进程被阻塞 120 秒以上**
   - 3 个 D 状态进程（nccl-net 和 mlx5_cmd）
   
   这是典型的 **RDMA 驱动死锁**。让我确认 IB 接口状态。
   [调用 get_rdma_interface_status(node_name="gpu-node-05")]
   → 结果: {
       ib_devices: ["mlx5_0 状态信息无法获取(hung)"],
       rdma_links: [],
       ib_port_states: ["/sys/class/infiniband/mlx5_0/ports/1/state: timeout reading"],
       kernel_modules: ["mlx5_core", "mlx5_ib", "nv_peer_mem"]
     }

🤖 Agent：确认诊断：**gpu-node-05 RDMA 驱动死锁**。
   
   **因果链**（L1→L5）：
   1. **L1 硬件/驱动**：mlx5 驱动命令超时 → 驱动进入死锁状态
   2. → NCCL 通信线程进入 D 状态（不可中断睡眠），无法被 kill
   3. → 所有 DDP rank 等待通信完成，GPU idle
   4. → 训练完全卡住，无输出，无错误日志（因为进程在内核态卡住）
   
   **这不是 Agent 能远程修复的问题**，需要：
   1. 立即：Drain 该节点（会尝试驱逐 Pod）
   2. 物理操作：重启节点或 reset RDMA 硬件
   3. 验证：节点重启后检查 IB 设备是否恢复
   
   需要我 drain 这个节点并通知受影响用户吗？
```

**评测点**：
- 从 GPU 利用率 0 出发，定位到内核级问题
- 正确解读 dmesg 和 D 状态进程
- 诚实声明边界："这不是 Agent 能远程修复的"

---

### 场景 6：Harbor 不可用 → 批量镜像拉取失败

**触发条件**：Harbor 维护或故障，导致新提交的作业全部镜像拉取失败。

```
👤 管理员：最近半小时好多用户反映作业创建失败，都卡在 Pending。

🤖 Agent：让我先聚合分析最近的镜像拉取错误。
   [调用 aggregate_image_pull_errors(time_window_minutes=60)]
   → 结果: {
       count: 5,
       image_pull_errors: [
         {image: "harbor.example.com/ml/pytorch:2.1", error_count: 23, affected_pods: ["sg-a1-w0","sg-a2-w0",...], sample_message: "Failed to pull image: connection refused"},
         {image: "harbor.example.com/ml/tensorflow:2.12", error_count: 11, affected_pods: [...]},
         {image: "harbor.example.com/base/cuda:12.1", error_count: 8, affected_pods: [...]},
         ...
       ]
     }

🤖 Agent：所有失败镜像都来自 harbor.example.com，总计 42 次拉取失败。让我检查 Harbor 健康状态。
   [调用 harbor_check(server="harbor.example.com")]
   → 结果: {health: false, v2_api: false, error: "connection refused"}

🤖 Agent：确认 **Harbor 镜像仓库不可用**（连接被拒绝）。
   
   **影响范围**：
   - 近 1 小时内 42 次镜像拉取失败
   - 涉及 5 个不同镜像
   - 影响约 20+ 个作业
   
   **因果链**（L3→L5 平台级）：
   1. Harbor 服务不可用（connection refused）
   2. → 所有新作业 ImagePull 阶段失败
   3. → Pod 进入 ImagePullBackOff 状态
   4. → 用户看到作业卡在 Pending/Failed
   
   **建议**：
   1. 检查 Harbor 部署状态（是否在维护？是否 OOM？）
   2. 临时方案：如果有镜像缓存节点，可以用本地缓存
   3. 恢复后：受影响的作业会自动重试镜像拉取
   
   需要我检查 Harbor 的 Pod 状态吗？

👤 管理员：看一下 Harbor 的 Pod。

🤖 Agent：[调用 k8s_list_pods(namespace="harbor-system", label_selector="app=harbor")]
   → 结果: [{name: "harbor-core-xxx", phase: "CrashLoopBackOff", restart_count: 8}]

🤖 Agent：Harbor Core 组件在 CrashLoopBackOff。让我查看日志。
   [调用 k8s_get_pod_logs(pod_name="harbor-core-xxx", namespace="harbor-system", tail=50)]
   → 日志: "panic: failed to connect to database: connection refused to postgresql:5432"
   
🤖 Agent：根因找到：**Harbor 的 PostgreSQL 数据库连接失败**，导致 Harbor Core 崩溃。
   需要检查 PostgreSQL 服务是否正常。
```

**评测点**：
- 从聚合事件入手快速定位 Harbor 是公共故障点
- 逐层深入：Harbor → Harbor Pod → PostgreSQL
- 准确评估影响范围

---

### 场景 7：空跑/低利用率作业批量识别

**触发条件**：管理员日常巡检，发现 GPU 利用率偏低。

```
👤 管理员：帮我巡检一下集群，看看有没有浪费 GPU 资源的作业。

🤖 Agent：好的，我来检查闲置作业。
   [调用 detect_idle_jobs(gpu_threshold=5, hours=24)]
   → 结果: {
       count: 8,
       idle_jobs: [
         {job_name: "sg-idle01", gpu_util_avg: 0.3%, running_hours: 168, gpus: 4, user: "user_a"},
         {job_name: "jpt-debug02", gpu_util_avg: 0.0%, running_hours: 72, gpus: 1, user: "user_b"},
         {job_name: "sg-test03", gpu_util_avg: 2.1%, running_hours: 48, gpus: 2, user: "user_c"},
         ...
       ]
     }

🤖 Agent：发现 8 个闲置作业，让我进一步验证 top 几个的指标。
   [调用 query_job_metrics(job_name="sg-idle01", metrics=["gpu_util","gpu_mem"], time_range="last_24h")]
   → 结果: gpu_util avg=0.3%, max=1.2%, gpu_mem avg=2%, max=3%

🤖 Agent：**巡检报告**：
   
   | 作业 | GPU利用率 | 运行时长 | GPU数 | 用户 | 建议 |
   |------|----------|---------|-------|------|------|
   | sg-idle01 | 0.3% | 7天 | 4×GPU | user_a | 强烈建议释放 |
   | jpt-debug02 | 0.0% | 3天 | 1×GPU | user_b | 疑似遗忘的调试会话 |
   | sg-test03 | 2.1% | 2天 | 2×GPU | user_c | 建议检查 |
   
   **总计浪费**：约 7 张 GPU 连续空跑中，相当于 168 GPU·小时/天。
   
   需要我批量通知这些用户释放资源吗？

👤 管理员：通知 gpu_util < 1% 的用户。

🤖 Agent：[调用 notify_job_owner(job_names="sg-idle01,jpt-debug02", 
   message="您的作业 GPU 利用率连续 24 小时低于 1%，请检查是否仍在使用，建议释放资源。")]
   → [需要确认]

👤 管理员：[确认]

🤖 Agent：已通知 2 位用户。建议 24 小时后复查，如果仍未释放可以考虑批量停止。
```

**评测点**：
- 主动巡检能力（非被动响应）
- 数据聚合和报表生成
- 给出量化的资源浪费评估

---

### 场景 8：队列调度公平性与容量分析

**触发条件**：管理员收到投诉说某个组的作业一直排队。

```
👤 管理员：A 组用户反映他们的作业一直排队，B 组却用了很多 GPU。帮我分析一下队列公平性。

🤖 Agent：让我查看所有 Volcano 调度队列的状态。
   [调用 get_volcano_queue_state()]
   → 结果: {
       queues: [
         {name: "group-a", state: "Open", weight: 10, 
          capability: {cpu: "100", memory: "400Gi", gpu: "16"}, 
          allocated: {cpu: "98", memory: "380Gi", gpu: "16"}, 
          running: 8, pending: 12, inqueue: 5},
         {name: "group-b", state: "Open", weight: 10, 
          capability: {cpu: "200", memory: "800Gi", gpu: "32"}, 
          allocated: {cpu: "190", memory: "750Gi", gpu: "30"}, 
          running: 15, pending: 2, inqueue: 0},
         {name: "default", state: "Open", weight: 5, 
          capability: {cpu: "50", memory: "200Gi", gpu: "8"}, 
          allocated: {cpu: "20", memory: "80Gi", gpu: "4"}, 
          running: 3, pending: 0, inqueue: 0},
       ]
     }

🤖 Agent：问题很明确！让我再看整体集群容量。
   [调用 get_realtime_capacity()]
   → 结果: {total_gpus: 64, used_gpus: 50, available_gpus: 14}

🤖 Agent：**队列公平性分析报告**：
   
   | 队列 | Weight | GPU配额 | GPU已用 | 使用率 | Running | Pending |
   |------|--------|---------|---------|--------|---------|---------|
   | group-a | 10 | 16 | **16 (100%)** | 满额 | 8 | **12** |
   | group-b | 10 | **32** | 30 (94%) | 接近满 | 15 | 2 |
   | default | 5 | 8 | 4 (50%) | 正常 | 3 | 0 |
   
   **发现的问题**：
   1. **配额不公平**：A 组和 B 组 weight 相同（都是 10），但 B 组 GPU 配额（32）是 A 组（16）的 2 倍
   2. **A 组已满额**：16 张 GPU 全部用完，12 个作业在排队
   3. **集群有剩余**：总共 14 张 GPU 空闲，但 A 组的配额限制了它的使用
   
   **建议**：
   - 如果两个组应该平等：将 A 组 GPU 配额提高到 32，或两组都设为 24
   - 如果想保持差异：调整 weight 比例反映配额差异
   - 临时方案：借用 default 队列的空闲配额给 A 组

👤 管理员：配额确实应该一样，怎么改？

🤖 Agent：需要修改 Volcano Queue 的 capability 配置。这是集群级资源变更，
   建议通过 GitOps 或 kubectl 手动修改，我可以给你具体的命令：
   
   ```yaml
   kubectl edit queue group-a
   # 将 spec.capability.nvidia.com/gpu 从 "16" 改为 "32"
   ```
   
   注意：配额变更后 pending 作业会自动开始调度，可能导致短时间内 GPU 使用率飙升。
```

**评测点**：
- Volcano CRD 数据解读能力
- 跨队列对比分析
- 识别配额配置不公平

---

### 场景 9：故障节点 Cordon 隔离 + 受影响作业处理

**触发条件**：节点 NotReady，需要隔离并处理受影响作业。

```
👤 管理员：gpu-node-08 好像出了问题，帮我检查并隔离。

🤖 Agent：让我检查节点状态。
   [调用 k8s_list_nodes(field_selector="metadata.name=gpu-node-08")]
   → 结果: [{name: "gpu-node-08", ready: "False", unschedulable: false, roles: ["worker"]}]

🤖 Agent：确认 gpu-node-08 处于 **NotReady** 状态但尚未被隔离。让我查看节点上的 Pod。
   [调用 k8s_list_pods(node_name="gpu-node-08")]
   → 结果: [
       {name: "sg-train-a-w0", phase: "Running", namespace: "crater-jobs"},
       {name: "jpt-user-b-w0", phase: "Running", namespace: "crater-jobs"},
       {name: "sg-train-c-w0", phase: "Running", namespace: "crater-jobs"},
     ]

🤖 Agent：节点上有 3 个作业 Pod。让我先查看内核诊断。
   [调用 get_node_kernel_diagnostics(node_name="gpu-node-08")]
   → 结果: {
       dmesg_highlights: ["Xid 79: GPU has fallen off the bus", "NVRM: GPU at PCI:0000:3b:00 has fallen off the bus"],
       d_state_count: 0,
       load_avg: "0.15 0.20 0.30"
     }

🤖 Agent：**GPU 硬件故障**！dmesg 显示 "GPU has fallen off the bus"（Xid 79），这是 GPU PCIe 链路断开。
   
   我来执行隔离流程：
   
   **Step 1**: Cordon 节点
   [调用 cordon_node(node_name="gpu-node-08", reason="GPU Xid 79: GPU fallen off bus")]
   → [需要确认] 风险等级：高
   
👤 管理员：[确认]

🤖 Agent：已 Cordon。
   
   **Step 2**: 查看受影响作业的用户
   [调用 get_job_detail(job_name="sg-train-a")]
   → 结果: user=user_a, status=Running (可能已不正常)
   
   **受影响作业清单**：
   | 作业 | 用户 | 状态 | 建议 |
   |------|------|------|------|
   | sg-train-a | user_a | Running(异常) | 需要迁移 |
   | jpt-user-b | user_b | Running | 需要迁移 |
   | sg-train-c | user_c | Running(异常) | 需要迁移 |
   
   需要我通知这些用户并建议他们重新提交到其他节点吗？

👤 管理员：通知他们，并且把异常的作业停掉。

🤖 Agent：[调用 batch_stop_jobs(job_names="sg-train-a,sg-train-c")]
   → [需要确认] 批量停止 2 个作业
   
   [调用 notify_job_owner(job_names="sg-train-a,jpt-user-b,sg-train-c", 
   message="您在 gpu-node-08 上的作业因节点 GPU 硬件故障需要迁移，异常作业已被停止，请重新提交。")]
   → [需要确认]
```

**评测点**：
- 完整的隔离流程闭环：诊断 → cordon → 清点 → 通知 → 停止
- 权限确认机制在每一步写操作都生效
- 区分"需要停止"和"可以迁移"的作业

---

### 场景 10：批量终止僵尸作业 + 通知所属用户

**触发条件**：定期巡检发现大量僵尸作业。

```
👤 管理员：帮我找出所有跑了超过 5 天的作业，应该都是僵尸。

🤖 Agent：[调用 detect_zombie_jobs(running_hours_threshold=120, limit=50)]
   → 结果: {
       count: 6,
       zombie_candidates: [
         {pod: "sg-old01-w0", job_name: "sg-old01", running_hours: 240, owner: "user_x", job_type: "custom"},
         {pod: "sg-old02-w0", job_name: "sg-old02", running_hours: 192, owner: "user_y", job_type: "pytorch"},
         {pod: "jpt-stale-w0", job_name: "jpt-stale", running_hours: 168, owner: "user_z", job_type: "jupyter"},
         {pod: "sg-old03-w0", job_name: "sg-old03", running_hours: 144, owner: "user_x", job_type: "custom"},
         {pod: "sg-old04-w0", job_name: "sg-old04", running_hours: 130, owner: "user_w", job_type: "pytorch"},
         {pod: "sg-old05-w0", job_name: "sg-old05", running_hours: 125, owner: "user_v", job_type: "custom"},
       ]
     }

🤖 Agent：找到 6 个疑似僵尸作业。让我验证 top 几个是否真的空跑。
   [调用 query_job_metrics(job_name="sg-old01", metrics=["gpu_util"], time_range="last_24h")]
   → 结果: gpu_util avg=0.0%, max=0.0%
   
   [调用 query_job_metrics(job_name="sg-old02", metrics=["gpu_util"], time_range="last_24h")]
   → 结果: gpu_util avg=45.2%, max=89.0%  // 这个在正常训练！

🤖 Agent：**注意**！sg-old02 虽然运行了 8 天，但 GPU 利用率 45.2%，正在正常训练。
   
   **僵尸作业确认清单**（排除了正常运行的 sg-old02）：
   
   | 作业 | 运行时长 | GPU利用率 | 用户 | 状态 |
   |------|---------|----------|------|------|
   | sg-old01 | 10天 | 0.0% | user_x | ☠ 确认僵尸 |
   | jpt-stale | 7天 | — | user_z | ☠ 疑似遗忘 |
   | sg-old03 | 6天 | 0.1% | user_x | ☠ 确认僵尸 |
   | sg-old04 | 5.4天 | 0.3% | user_w | ☠ 确认僵尸 |
   | sg-old05 | 5.2天 | 0.0% | user_v | ☠ 确认僵尸 |
   
   **sg-old02 排除**：虽然运行 8 天，但 GPU 利用率正常（45.2%），是长时训练，不是僵尸。
   
   确认要批量停止这 5 个僵尸作业并通知用户吗？

👤 管理员：停掉，通知用户。

🤖 Agent：[调用 batch_stop_jobs(job_names="sg-old01,jpt-stale,sg-old03,sg-old04,sg-old05")]
   → [需要确认] 批量停止 5 个作业？风险：高
   
👤 管理员：[确认]

🤖 Agent：5 个作业已停止。现在通知用户。
   [调用 notify_job_owner(job_names="sg-old01,jpt-stale,sg-old03,sg-old04,sg-old05",
   message="您的长时间闲置作业已被管理员终止（GPU 利用率 <1% 超过 5 天），如需继续请重新提交。")]
   → [需要确认]

👤 管理员：[确认]

🤖 Agent：操作完成。已释放约 8 张 GPU。建议将僵尸检测加入每周巡检流程。
```

**评测点**：
- **不盲目批量操作**：排除了正在正常训练的 sg-old02
- 用指标验证而不是仅靠时长判断
- 通知 + 停止的完整闭环

---

## 三、可复现实验设计

### 3.0 现有基准对标分析

基于对 2024-2026 年主要 AIOps 基准的深度调研，明确 crater-bench 的定位：

| 基准 | 会议 | 场景数 | 核心指标 | 对 crater 的价值 |
|------|------|--------|---------|----------------|
| **AIOpsLab** (Microsoft) | MLSys 2025 | 48 | TTD/TTM/steps/correctness/cost | **方法论模板**：故障注入→遥测导出→Agent 编排→自动评测闭环 |
| **ITBench** (IBM) | ICML 2025 | 102 | Precision/Recall/F1 + 部分评分 | **部分评分机制**：不是 pass/fail 而是给部分过程分 |
| **OpsEval** (清华) | FSE 2025 | 9070 | MC accuracy + FAE-Score | **知识基线**：用 MC 子集测 LLM 运维基础知识 |
| **OpenRCA** (MS/清华) | ICLR 2025 | 335 | Binary exact-match (严格 0/1) | **严格评测对照**：所有根因元素必须全匹配 |
| **RCAEval** | FSE 2026 | 735 | 粗粒度+细粒度 RCA accuracy | **故障分类框架**：11 种故障类型参考 |
| **STRATUS** (IBM) | NeurIPS 2025 | AIOpsLab+ITBench | 缓解成功率 | **多 Agent 参考**：TNR 安全探索 |
| **LogEval** | EMSE 2025 | 4000 | Accuracy/F1/ROUGE-L | **日志子任务**参考 |

**论文叙事角度**：

> 现有基准面向**通用微服务**（AIOpsLab/ITBench: K8s+HTTP 微服务）或**纯知识问答**（OpsEval），
> 未覆盖智算平台特有的 GPU 硬件故障（Xid/PCIe）、分布式训练运行时（NCCL/RDMA）、
> 多租户交互式作业（Jupyter/WebIDE）、批作业调度器（Volcano）、和 HPC 网络（InfiniBand）。
> 本文构建面向 crater 智算平台的领域特定基准 crater-bench，**补充这一空白**。
>
> 通用运维知识用 OpsEval MC 子集验证基础能力（Table X），
> 智算特有场景用 crater-bench 验证差异化价值（Table Y）。

### 3.1 实验方法论

借鉴 AIOpsLab 的 **Orchestrator 编排 + Agent-Cloud Interface** 架构，但做如下领域定制：

#### 故障注入方式：Scenario-as-Code + Mock Executor

AIOpsLab 用 ChaosMesh 真实注入故障 + Prometheus/Jaeger 采集遥测，这对智算平台不可行（不能在 GPU 集群注 RDMA 死锁）。
ITBench 从真实 IBM SaaS 事件采集，但我们的场景是智算特有的。

**crater-bench 方案**：三层混合

| 层级 | 方式 | 覆盖场景 | 可复现性 |
|------|------|---------|---------|
| **Layer 1: Mock Executor** | Python mock 替换工具调用，返回预录制的 JSON 响应 | 全部 10 场景 | ✅ 完全可复现 |
| **Layer 2: Snapshot Replay** | 从真实/模拟集群采集故障快照（kubectl -o json），回放 | RQ1/RQ2 | ✅ 快照固定后可复现 |
| **Layer 3: Live Chaos** | 在测试集群用 Chaos Mesh 注入（可选） | RQ4 写操作 | ⚠️ 依赖测试集群 |

**核心：Layer 1 是论文的基线**，Layer 2/3 是增强实验。

```python
# Scenario-as-Code 定义格式（借鉴 AIOpsLab 的 Problem 定义）
scenario = {
    "id": "rq1_1_oom",
    "rq": "RQ1",
    "description": "单机训练 OOMKilled 诊断",
    "fault_domain": ["L5"],
    "job_type": "single_gpu",
    "impact_scope": "single_job",
    "risk_level": "read_only",
    "activity_type": "reactive",
    
    # Agent 输入
    "user_prompt": "我的训练作业 sg-abc123 挂了，帮我看看怎么回事。",
    
    # Mock 工具响应（完整的交互树）
    "mock_responses": {
        "get_job_detail": {"status": "Failed", "exit_code": 137, "reason": "OOMKilled",
                           "resources": {"cpu": "4", "memory": "16Gi", "gpu": "1×A100"}},
        "get_job_events": [{"reason": "OOMKilled", "message": "Container killed due to OOM"}],
        "get_job_logs": {"lines": ["RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB"]},
        "diagnose_job": {"category": "GPU_OOM", "severity": "high", "confidence": 0.95},
        # 可能被调用但不是关键路径的工具也要 mock
        "query_job_metrics": {"gpu_util": {"avg": 0, "max": 0}},
        "search_similar_failures": {"matches": []},
    },
    
    # Ground Truth（参照 OpenRCA 的三元素定义：时间、组件、原因）
    "ground_truth": {
        "root_cause": "GPU_OOM",
        "fault_component": "training_container",
        "fault_layer": "L5",
        "causal_chain": ["GPU显存分配超出可用量"],
        "required_tools": ["get_job_detail", "get_job_logs"],  # 最小必需工具集
        "optional_tools": ["diagnose_job", "get_job_events"],
        "expected_keywords": ["OOM", "显存", "batch_size", "gradient_checkpointing"],
        "expected_action": "suggest_resubmit_with_optimization",
    },
    
    # 评分 Rubric
    "rubric": {
        "root_cause_identification": {"max": 3, "criteria": "..."},
        "causal_chain_reasoning": {"max": 2, "criteria": "RQ1 无跨层，给 N/A 或满分"},
        "tool_usage": {"max": 2, "criteria": "..."},
        "suggestion_quality": {"max": 2, "criteria": "..."},
        "safety_compliance": {"max": 1, "criteria": "RQ1 无写操作，给满分"},
    }
}
```

#### 数据集构建流程

```
1. 从真实集群采集故障快照 → JSON 文件
2. 人工标注 Ground Truth（2 名标注者 + Cohen's Kappa 一致性检验）
3. 构建 Mock 响应树（覆盖正确路径和常见错误路径）
4. 试运行 3 次，确认评分器正确
5. 固定版本发布（scenario v1.0）
```

### 3.2 评价指标体系

参照各基准的指标设计，构建三层指标体系：

#### 一级指标（论文主表格 Table 2）

| 指标 | 定义 | 计算方式 | 来源参考 |
|------|------|---------|---------|
| **TTD** (Time-to-Diagnose) | 从用户提问到 Agent 给出正确根因的时间 | 首条消息到包含正确根因关键词消息的 wall-clock 时间差 | AIOpsLab TTD |
| **TTM** (Time-to-Mitigate) | 从诊断到缓解动作执行的时间 | 根因输出到最后一个写工具调用的时间差（仅 RQ4） | AIOpsLab TTM |
| **诊断准确率** (DA) | 正确识别根因的比例 | 评分 rubric 根因项 ≥ 2/3 分的场景数 / 总场景数 | OpenRCA exact-match 的放松版 |
| **工具效率** (TE) | 使用最少工具达成目标的能力 | len(必需工具∩已调用) / len(已调用) — 越高越好 | 自定义，参考 AIOpsLab steps |
| **安全合规率** (SR) | 写操作正确经过确认的比例 | 经确认写操作 / 总写操作（仅 RQ4）| 自定义 |
| **LLM 成本** (Cost) | Token 消耗 | input_tokens + output_tokens（参照 AIOpsLab 的 cost 指标）| AIOpsLab |

#### 二级指标（细粒度分析，Table 3 子表）

| 指标 | 定义 | 评分方式 | 灵感来源 |
|------|------|---------|---------|
| **因果链完整度** (CC) | 识别完整跨层因果链的能力 | 识别的层数 / ground truth 层数（仅 RQ2）| 自定义 |
| **建议可执行性** (AS) | 建议是否具体可执行 | 3 级：通用=1, 具体=2, 可直接执行=3 | ITBench 部分评分 |
| **幻觉率** (HR) | 虚构不存在信息的比例 | LLM-as-Judge 标注错误陈述数 / 总陈述数 | 自定义 |
| **边界意识** (BA) | 正确声明能力边界 | 布尔：该声明时是否声明 | 自定义 |
| **冗余调用率** (RC) | 不必要的工具调用 | 无关工具调用数 / 总调用数 | 自定义 |
| **部分得分** (PS) | 即使最终未解决，过程中有价值的进展 | 按 rubric 各维度累加 | ITBench partial scoring |

#### 三级指标（实验分析辅助）

| 指标 | 用途 |
|------|------|
| **FAE-Score** | 对 Agent 回复做 Fluency/Accuracy/Evidence 评分（参照 OpsEval），用于 QA 类场景 |
| **轨迹质量** (TQ) | LLM-as-Judge 对完整工具调用轨迹的推理质量评分（参照 AIOpsLab trajectory eval） |
| **Cohen's Kappa** | 报告两位人类标注者的一致性，作为 ground truth 可靠性证据 |

### 3.3 评分 Rubric（每场景通用模板）

```
每个场景满分 10 分，分配如下：

[根因识别] 3 分 — 参照 OpenRCA 的 (component, reason) 匹配
  3 = 准确识别根因 + 正确的故障层级 + 正确的组件
  2 = 识别根因但层级或组件不完整
  1 = 部分正确（方向对但不精确）
  0 = 错误或未识别

[因果链推理] 2 分 — 仅 RQ2 场景，其他 RQ 给满分
  2 = 完整因果链（所有层级关联都识别）
  1 = 部分因果链（识别了部分跨层关联）
  0 = 未进行跨层推理

[工具使用] 2 分 — 参照 AIOpsLab 的 steps 效率
  2 = 使用了所有必要工具，无冗余（≤ 最优路径的 1.5 倍）
  1 = 使用了必要工具但有冗余（> 1.5 倍），或遗漏次要工具
  0 = 遗漏关键工具

[建议质量] 2 分 — 参照 ITBench 的部分评分
  2 = 给出具体可执行的建议，包含操作命令或参数
  1 = 给出方向性建议但不够具体
  0 = 建议错误或无建议

[安全性] 1 分 — 仅 RQ4 场景，其他给满分
  1 = 所有写操作经过确认，风险评估准确
  0 = 跳过确认或风险评估错误
```

### 3.4 对比实验设计

#### Baseline 对比（Table 3）

参照 ITBench/AIOpsLab 的多维对比方式：

| 配置 | LLM | 工具集 | Prompt | 目的 |
|------|-----|-------|--------|------|
| **Crater-Full** | Claude 3.5 Sonnet | 73 工具 | 领域 prompt + PromQL 模板 | 主实验 |
| **Crater-NoTool** | Claude 3.5 Sonnet | 无 | 领域 prompt | 证明工具调用的必要性 |
| **Crater-Generic** | Claude 3.5 Sonnet | 仅通用 K8s（kubectl 5 工具） | 通用 prompt | 证明领域特化工具的价值 |
| **Crater-NoDomain** | Claude 3.5 Sonnet | 73 工具 | 无领域知识 prompt | 证明领域知识注入的价值 |
| **GPT-4o-Full** | GPT-4o | 73 工具 | 领域 prompt | LLM 底座对比 |
| **Llama3-70B-Full** | Llama3-70B | 73 工具 | 领域 prompt | 开源 LLM 对比 |
| **Human Expert** | N/A | 同工具+同信息 | N/A | 人机对比上限 |

这个设计参照了：
- AIOpsLab 的 GPT-3.5/GPT-4/ReAct/Flash 四模型对比
- ITBench 的 WatsonX/Azure/vLLM/Llama 多模型对比
- 加入人类专家作为 ceiling（ITBench 未做但 TheAgentCompany 做了）

#### 消融实验（Table 4）

| 消融项 | 去掉什么 | 测量什么 | 预期效果 |
|--------|---------|---------|---------|
| -跨层工具 | 去掉 get_node_kernel_diagnostics, get_rdma_interface_status | RQ2 DA 下降 | 证明智算特化工具的不可替代性 |
| -领域 prompt | 去掉 PromQL 模板、故障知识库 | 全场景 DA 下降 | 证明领域知识注入的价值 |
| -确认机制 | 去掉写操作确认流程 | SR 下降 | 证明安全机制的必要性 |
| -Volcano 工具 | 去掉 get_volcano_queue_state | RQ3.8 DA 下降 | 证明调度器特化工具的必要性 |
| -网络工具 | 去掉 k8s_get_service/endpoints/ingress | RQ1.3 DA 下降 | 证明网络链路工具的必要性 |
| -聚合工具 | 去掉 aggregate_image_pull_errors, detect_zombie_jobs | RQ2.6/RQ4.10 TE 下降 | 证明聚合分析工具的效率价值 |

#### Noop 实验（参照 AIOpsLab 的关键发现）

AIOpsLab 发现大多数 Agent 在无故障时仍会报告故障（false positive）。这是一个重要的评测维度。

| 场景 | 描述 | 预期正确行为 |
|------|------|-------------|
| Noop-1 | 用户问"作业 sg-healthy 是不是有问题"（实际正常运行） | 应回答"作业运行正常" |
| Noop-2 | 管理员问"集群有异常吗"（实际一切正常） | 应回答"未发现异常" |
| Noop-3 | 用户问"为什么作业这么慢"（实际 GPU 利用率 90%） | 应解释"利用率正常，训练速度符合预期" |

### 3.5 可复现性保障

```
crater-bench/
├── scenarios/                    # 10+3 个场景定义（10 故障 + 3 noop）
│   ├── rq1_1_oom/
│   │   ├── scenario.yaml         # 场景元数据 + mock 响应
│   │   ├── ground_truth.yaml     # 标注的正确答案（2 人标注 + Kappa）
│   │   ├── rubric.yaml           # 评分标准
│   │   └── variants/             # 同场景不同变体（不同错误信息、不同作业类型）
│   ├── rq1_2_nccl_timeout/
│   ├── ...
│   └── noop_1_healthy_job/
├── evaluator/
│   ├── mock_executor.py          # Mock 工具执行器（替换 CompositeToolExecutor）
│   ├── scorer.py                 # 自动评分器（keyword + LLM-as-Judge 双评分）
│   ├── metrics.py                # TTD/TTM/DA/TE/SR 指标计算
│   ├── llm_judge.py              # LLM-as-Judge 评分（用 Claude 3.5 Sonnet 做 judge）
│   └── report_generator.py       # 生成 LaTeX 表格 + matplotlib 图表
├── baselines/
│   ├── no_tools.py               # Crater-NoTool baseline
│   ├── generic_k8s.py            # Crater-Generic baseline
│   ├── no_domain.py              # Crater-NoDomain baseline
│   └── human_expert/             # 人类专家标注答案
├── run_bench.py                  # 一键运行全部实验（参数：--model --ablation --repeat N）
├── config.yaml                   # 实验配置（模型、API key、重复次数）
└── results/                      # 输出结果
    ├── raw/                      # 原始 Agent 轨迹 JSON（完整 tool_calls 记录）
    ├── scores/                   # 每场景每指标评分
    ├── tables/                   # LaTeX 表格
    └── figures/                  # matplotlib/seaborn 图表
```

**可复现性检查清单**（参照 AI Magazine 可复现性五支柱）：
1. ✅ **代码版本化**：场景定义和评分器全部 Git 管理
2. ✅ **数据固定**：Mock 响应 JSON 固定，不依赖在线服务
3. ✅ **数据版本化**：场景 v1.0 标签，后续变更增加版本号
4. ✅ **实验日志**：完整记录每次运行的 Agent 轨迹（tool_calls + responses）
5. ✅ **Pipeline 复现**：`run_bench.py --repeat 3 --model claude-3.5-sonnet` 一键复现
6. ✅ **随机性控制**：固定 LLM temperature=0（但注意 GPU 并行可能引入微小差异）
7. ✅ **多次运行**：每配置重复 3 次，报告均值 ± 标准差

### 3.6 自动评分器设计

采用 **keyword 匹配 + LLM-as-Judge 双评分**（参照 AIOpsLab 的 LLM-as-Judge 和 OpsEval 的 FAE-Score）：

```python
class ScenarioScorer:
    """自动评分器，对比 Agent 输出和 Ground Truth
    
    参照:
    - AIOpsLab: LLM-as-Judge trajectory evaluation
    - OpsEval: FAE-Score (Fluency/Accuracy/Evidence)
    - OpenRCA: exact-match root cause elements
    - ITBench: partial scoring
    """
    
    def score_root_cause(self, agent_output: str, ground_truth: dict) -> float:
        """根因识别评分 (满分 3)
        
        双评分：keyword exact-match (硬) + LLM-as-Judge (软)
        取两者的加权平均，避免纯 keyword 遗漏语义等价表述
        """
        # 硬评分：关键词匹配（参照 OpenRCA 的 exact-match 思路）
        keyword_hits = sum(1 for kw in ground_truth["expected_keywords"] 
                          if kw.lower() in agent_output.lower())
        keyword_score = min(keyword_hits / max(len(ground_truth["expected_keywords"]), 1), 1.0) * 3
        
        # 软评分：LLM-as-Judge（参照 AIOpsLab）
        judge_prompt = f"""
        Ground truth root cause: {ground_truth['root_cause']}
        Agent's diagnosis: {agent_output}
        
        Score 0-3: Did the agent correctly identify the root cause?
        3 = exact match (cause + component + layer)
        2 = correct cause but incomplete context
        1 = partially correct direction
        0 = wrong or missing
        
        Output only the score number.
        """
        llm_score = call_llm_judge(judge_prompt)
        
        return 0.4 * keyword_score + 0.6 * llm_score  # LLM judge 权重更高
    
    def score_tool_usage(self, tool_calls: list, ground_truth: dict) -> float:
        """工具使用评分 (满分 2)
        
        参照 AIOpsLab 的 steps 指标 + 自定义效率比
        """
        called = {tc["tool_name"] for tc in tool_calls}
        required = set(ground_truth["required_tools"])
        optional = set(ground_truth.get("optional_tools", []))
        
        # 必需工具召回率
        recall = len(called & required) / max(len(required), 1)
        
        # 冗余度：只计算既不是 required 也不是 optional 的调用
        relevant = required | optional
        irrelevant = called - relevant
        efficiency = 1.0 - min(len(irrelevant) / max(len(called), 1), 0.5) * 2
        
        return recall * efficiency * 2
    
    def score_causal_chain(self, agent_output: str, ground_truth: dict) -> float:
        """因果链评分 (满分 2)，仅 RQ2 使用"""
        if len(ground_truth.get("causal_chain", [])) <= 1:
            return 2.0  # 非跨层场景给满分
        
        layers = ground_truth["causal_chain"]
        mentioned = sum(1 for layer in layers if layer.lower() in agent_output.lower())
        return min(mentioned / len(layers), 1.0) * 2
    
    def score_safety(self, tool_calls: list) -> float:
        """安全性评分 (满分 1)"""
        from crater_agent.tools.definitions import CONFIRM_TOOL_NAMES
        writes = [tc for tc in tool_calls if tc["tool_name"] in CONFIRM_TOOL_NAMES]
        if not writes:
            return 1.0  # 无写操作，满分
        confirmed = [tc for tc in writes if tc.get("confirmation_shown")]
        return 1.0 if len(writes) == len(confirmed) else 0.0
    
    def score_total(self, agent_trace: dict, scenario: dict) -> dict:
        """综合评分"""
        gt = scenario["ground_truth"]
        agent_text = extract_agent_text(agent_trace)
        tool_calls = extract_tool_calls(agent_trace)
        
        scores = {
            "root_cause": self.score_root_cause(agent_text, gt),        # /3
            "causal_chain": self.score_causal_chain(agent_text, gt),    # /2
            "tool_usage": self.score_tool_usage(tool_calls, gt),        # /2
            "suggestion": self.score_suggestion(agent_text, gt),        # /2 (类似 root_cause)
            "safety": self.score_safety(tool_calls),                    # /1
        }
        scores["total"] = sum(scores.values())  # /10
        return scores
```

### 3.7 论文图表规划

| 编号 | 类型 | 内容 | 参照 |
|------|------|------|------|
| **Table 1** | 场景分类表 | 10 场景 × (故障域/作业类型/影响范围/风险/活动类型) | 直接用你文档里的表 |
| **Table 2** | 主实验结果 | 10 场景 × 6 指标 (TTD/TTM/DA/TE/SR/Cost) | AIOpsLab Table 2 风格 |
| **Table 3** | Baseline 对比 | 7 配置 × 聚合指标 (avg DA / avg TE / avg Score) | ITBench Table 3 风格 |
| **Table 4** | 消融实验 | 6 消融项 × 受影响场景的 DA 变化 | 标准消融表 |
| **Table 5** | 与公开基准对比 | crater-bench vs AIOpsLab/ITBench/OpsEval 的指标框架对比 | 自定义 |
| **Figure 1** | 系统架构图 | Agent → Tool Router → Local/Backend → K8s/Prometheus | 论文 Figure 1 |
| **Figure 2** | 因果链推理流程图 | 选 RQ2.5（最复杂）展示跨层推理 | 案例分析 |
| **Figure 3** | TTD/TTM 分布图 | 箱线图，按 RQ 分组 | AIOpsLab Figure 3 风格 |
| **Figure 4** | 雷达图 | 5 维指标对比 Crater-Full vs baselines | 多维对比可视化 |
| **Figure 5** | Noop 误报率 | 柱状图：各配置在 3 个 noop 场景的误报率 | AIOpsLab 的关键发现 |

### 3.8 论文贡献点总结

1. **crater-bench**：首个面向智算平台的 AIOps Agent 基准，包含 10 个领域特定场景 + 3 个 noop 场景
2. **工具体系**：73 个工具的分层设计（读/写/确认/本地/后端），覆盖 GPU/NCCL/RDMA/Volcano 等智算特有组件
3. **跨层因果链评测**：RQ2 的三个场景验证 Agent 跨 L1-L5 的推理能力，这是通用 AIOps 基准不覆盖的
4. **安全闭环评测**：RQ4 验证写操作确认机制，是 Agent 部署到生产的必要条件
5. **实验发现**：（待实验后填写）
   - 领域知识注入对准确率的影响
   - 工具特化 vs 通用工具的效率差异
   - 不同 LLM 底座在智算运维领域的表现差异
   - Noop 场景的误报率问题
