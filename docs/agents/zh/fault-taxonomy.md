# 智算平台故障分类与工具覆盖

本文档对智算平台（基于 Kubernetes + Volcano 调度器的 GPU 集群）中训练与推理作业可能遇到的故障进行系统性分类，并评估当前 Agent 工具链的诊断覆盖度。

---

## 参考文献

| 来源 | 关键发现 |
|------|---------|
| **Meta HPCA 2025**: "Revisiting Reliability in Large-Scale ML Clusters" | 基于 1.5 亿 A100 GPU 小时的统计，NCCL 超时是最常见的分布式训练故障类型，占总故障的显著比例 |
| **ByteDance 2025**: "Robust LLM Training Infrastructure" | 隐性故障（loss 发散、训练挂起）比显性崩溃更难诊断和处理，需要运行时监控与基线对比 |
| **R2CCL 2025** | NIC/线缆故障占分布式训练故障的 50%，凸显网络硬件可靠性对大规模训练的关键影响 |
| **USENIX ATC 2024**: "Characterization of LLM Development in the Datacenter" | 错误多集中在作业启动阶段；长时间预训练中基础设施故障为主要中断原因 |

这些研究共同表明：大规模分布式训练的可靠性瓶颈已从单点 GPU 故障转向**网络通信**和**隐性故障**，诊断工具需要覆盖从硬件到框架的全栈。

---

## 作业类型矩阵

| 作业类型 | 代码标识 | 交互/批处理 | 分布式/单机 | 方向 | 调度器 |
|----------|---------|:---------:|:---------:|------|--------|
| Jupyter Notebook | `jupyter` | 交互式 | 单机 | 探索/调试 | Volcano |
| Web IDE | `webide` | 交互式 | 单机 | 开发/调试 | Volcano |
| 自定义作业 | `custom` | 批处理 | 可配置 | 通用训练 | Volcano |
| PyTorch 分布式 | `pytorch` | 批处理 | 分布式 | DDP 训练 | Volcano |
| TensorFlow 分布式 | `tensorflow` | 批处理 | 分布式 | 分布式训练 | Volcano |
| DeepSpeed | `deepspeed` | 批处理 | 分布式 | 优化分布式训练 | Volcano |
| OpenMPI | `openmpi` | 批处理 | 分布式 | MPI 并行计算 | Volcano |
| 推理服务 | `inference` | 批处理 | 可配置 | 模型推理 | EMIAS |

> **说明**：训练类作业统一由 Volcano 调度器管理，支持 Gang Scheduling 保证分布式作业的原子调度；推理/Serving 作业由 EMIAS 平台独立管理。

---

## 故障分层模型

故障按基础设施栈自底向上分为六层。每一层的故障可能向上传导，导致上层出现级联症状。

```
┌─────────────────────────────────────────┐
│  Layer 6: 交互式作业特有                 │
├─────────────────────────────────────────┤
│  Layer 5: 框架/应用层                    │
├─────────────────────────────────────────┤
│  Layer 4: 平台层 (Crater)               │
├─────────────────────────────────────────┤
│  Layer 3: 分布式通信层 (NCCL/MPI)        │
├─────────────────────────────────────────┤
│  Layer 2: 基础设施层 (K8s/Volcano/网络)   │
├─────────────────────────────────────────┤
│  Layer 1: 硬件层                         │
└─────────────────────────────────────────┘
```

---

### Layer 1: 硬件层

硬件故障是整个故障栈的基础，通常表现为设备不可用或性能降级。

#### 1.1 GPU 故障

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| Xid 错误（Xid 48/63/79 等） | `dmesg` 中出现 `NVRM: Xid` 报错；作业异常退出；GPU 计算结果错误 | `get_node_gpu_info`、`get_pod_logs` | 好 |
| ECC 不可纠正错误 | GPU 被自动隔离或 Pod 被驱逐；`nvidia-smi` 报 ECC 错误计数增加 | `get_node_gpu_info` | 好 |
| GPU 温度过高 / 降频（Thermal Throttle） | 训练吞吐量突然下降；GPU 时钟频率低于基准值 | `get_node_gpu_info`（温度/时钟信息） | 好 |
| GPU 驱动版本不匹配 | CUDA 初始化失败；`nvidia-smi` 报 driver/library mismatch | `get_node_gpu_info`、`get_pod_logs` | 部分 |
| GPU 显存泄漏 | 作业退出后 GPU 显存未释放；后续作业无法分配显存 | `get_node_gpu_info` | 部分 |

#### 1.2 RDMA / InfiniBand 故障

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| IB 端口 Down | NCCL 初始化超时；`ibstat` 显示端口 State: Down | `get_node_gpu_info`（网络信息部分） | 部分 |
| RDMA 驱动异常 | 分布式作业无法启动；`ibv_devinfo` 报错 | `get_pod_logs`、`get_node_gpu_info` | 部分 |
| IB 交换机端口抖动 | 间歇性 NCCL 超时；通信延迟不稳定 | `get_pod_logs`（间接） | 缺失 |

#### 1.3 NIC / 线缆故障

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| 网口链路 Down | 节点网络不可达；Pod 间无法通信 | `get_node_gpu_info`、`get_pod_events` | 部分 |
| CRC 错误累积 | 网络丢包率升高；训练速度逐渐下降 | `get_pod_logs`（间接） | 缺失 |
| 速率协商降级（如 100G→25G） | AllReduce 带宽远低于预期；训练吞吐量异常偏低 | 缺失 | 缺失 |

#### 1.4 存储故障

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| 磁盘故障 | Pod 启动失败，事件中报 volume mount 错误 | `get_pod_events`、`get_pod_logs` | 好 |
| NFS 挂载超时 / 断开 | 数据加载卡住；checkpoint 写入失败 | `get_pod_logs`、`get_pod_events` | 好 |
| 存储 IOPS 不足 | 数据预处理或 checkpoint 耗时异常增长 | `get_pod_logs`（间接） | 部分 |

---

### Layer 2: 基础设施层（K8s / Volcano / 网络）

基础设施层故障直接影响作业的调度、启动和运行。

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| Node NotReady | Pod 被驱逐或无法调度；节点状态变为 NotReady/Unknown | `get_pod_events`、`get_pod_status` | 好 |
| Volcano 调度失败 | 作业长期 Pending；事件中显示 Unschedulable | `get_pod_events`、`get_vcjob_status` | 好 |
| Gang Scheduling 死锁 | 多个作业互相等待资源，均无法启动 | `get_vcjob_status`、`get_pod_events` | 部分 |
| 镜像拉取失败（ImagePullBackOff） | Pod 停留在 ImagePullBackOff 状态；事件报镜像不存在或认证失败 | `get_pod_events`、`get_pod_status` | 好 |
| PVC 挂载失败 | Pod 停留在 ContainerCreating；事件报 volume 挂载超时 | `get_pod_events`、`get_pod_status` | 好 |
| Service / Ingress 不可达 | 交互式作业无法通过浏览器访问；端口转发失败 | `get_pod_status`、`get_pod_logs` | 部分 |
| NetworkPolicy 阻断 | Pod 间通信异常；分布式作业部分 rank 无法连接 | `get_pod_logs`（间接） | 缺失 |
| ConfigMap / Secret 挂载错误 | 配置文件内容不正确或缺失；应用启动时报配置错误 | `get_pod_events`、`get_pod_logs` | 好 |
| Pod CrashLoopBackOff | Pod 反复重启；restart count 持续增加 | `get_pod_status`、`get_pod_logs`、`get_pod_events` | 好 |
| Volcano Queue 队列满 | 新提交的作业全部 Pending；队列资源配额耗尽 | `get_vcjob_status`、`get_pod_events` | 好 |
| DNS 解析失败 | Pod 内无法解析集群内外域名；服务发现异常 | `get_pod_logs`（间接） | 部分 |

---

### Layer 3: 分布式通信层（NCCL / MPI / AllReduce）

分布式通信故障是大规模训练中最常见且最难诊断的故障类别（参考 Meta HPCA 2025）。

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| NCCL 超时（网络原因） | 日志中出现 `NCCL WARN Timeout`；部分 rank 通信不可达 | `get_pod_logs`、`get_nccl_diagnosis`、`get_node_gpu_info` | 好 |
| NCCL 超时（rank 挂起） | 训练进程无响应但不退出；GPU 利用率骤降至 0 | `get_pod_logs`、`get_nccl_diagnosis` | 好 |
| Rank 映射错误 | 训练启动后立即崩溃；日志报 rank/world_size 不匹配 | `get_pod_logs`、`get_vcjob_status` | 好 |
| 跨节点 RDMA 通信失败 | NCCL 初始化阶段超时；日志显示 RDMA 连接建立失败 | `get_nccl_diagnosis`、`get_node_gpu_info` | 好 |
| NCCL 环境变量配置错误 | 通信性能异常；使用了错误的传输协议（如 TCP 而非 IB） | `get_pod_logs`、`get_nccl_diagnosis` | 好 |
| MPI 启动失败 | mpirun 报连接被拒或 host 不可达；slots 配置错误 | `get_pod_logs`、`get_pod_events` | 部分 |
| AllReduce 性能异常 | 训练 step 时间远超基准；通信占比过高 | `get_pod_logs`（间接判断） | 部分 |
| Ring / Tree 拓扑构建失败 | NCCL 初始化阶段报拓扑检测错误；部分 GPU 未被纳入通信组 | `get_nccl_diagnosis`、`get_pod_logs` | 部分 |
| 梯度同步卡死 | 训练在某个 step 完全停滞；所有 rank 的 GPU 利用率同时降为 0 | `get_pod_logs`、`get_nccl_diagnosis` | 好 |

---

### Layer 4: 平台层（Crater）

平台层故障与 Crater 自身的资源管理和作业生命周期控制相关。

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| 配额超限 | 作业创建被拒绝；返回配额不足错误 | `get_user_quota`、平台 API | 好 |
| 作业创建失败 | API 返回错误；作业未出现在列表中 | `list_user_jobs`、`get_pod_events` | 好 |
| 作业长期 Pending 无调度 | 作业状态持续为 Pending；无 Pod 被创建 | `get_vcjob_status`、`get_pod_events`、`search_similar_failures` | 好 |
| 相似故障检索 | 用户遇到未知故障，需要参考历史案例 | `search_similar_failures` | 好 |
| 自动诊断报告 | 需要系统化分析作业故障根因 | `auto_diagnose`（Agent 综合诊断） | 好 |
| 审计与合规检查 | 需要追踪操作记录或验证权限合规性 | `get_audit_logs`、平台审计 | 部分 |
| 作业优先级/抢占 | 高优作业抢占低优作业的资源；低优作业被意外终止 | `get_vcjob_status`、`get_pod_events` | 部分 |
| 回调/Webhook 失败 | 作业状态变更通知未送达；下游系统未感知作业完成 | 平台日志（间接） | 缺失 |

---

### Layer 5: 框架/应用层

框架层故障通常由用户代码或框架配置引起，但也可能由底层资源问题导致。

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| Python 运行时错误 | Pod 日志中出现 Traceback；进程退出码非 0 | `get_pod_logs` | 好 |
| GPU OOM（显存不足） | 日志报 `CUDA out of memory`；进程被 OOM Killer 终止 | `get_pod_logs`、`get_node_gpu_info` | 好 |
| CPU / 系统 OOM | Pod 被 K8s OOMKilled；`dmesg` 中出现 OOM 记录 | `get_pod_status`、`get_pod_events` | 好 |
| Checkpoint 保存失败 | 训练中断后无法恢复；日志报存储写入错误 | `get_pod_logs` | 好 |
| Checkpoint 加载失败 | 恢复训练时报 state_dict 不匹配或文件损坏 | `get_pod_logs` | 好 |
| Loss NaN / 发散 | 训练 loss 突然变为 NaN 或持续发散 | `get_pod_logs`（需人工识别模式） | 部分 |
| 梯度溢出（Gradient Overflow） | 混合精度训练中频繁出现 grad scaler overflow | `get_pod_logs` | 部分 |
| 数据加载错误 | DataLoader worker 崩溃；数据文件损坏或路径错误 | `get_pod_logs` | 好 |
| 依赖库版本冲突 | 导入模块报 ImportError；CUDA/PyTorch 版本不兼容 | `get_pod_logs` | 好 |
| 训练挂起（隐性故障） | 进程存活但无输出；GPU 利用率降至 0（ByteDance 2025 指出此类故障更难诊断） | `get_pod_logs`、`get_node_gpu_info` | 部分 |

---

### Layer 6: 交互式作业特有

交互式作业（Jupyter / WebIDE）具有独特的生命周期和用户交互模式。

| 故障场景 | 典型症状 | 对应诊断工具 | 覆盖度 |
|---------|---------|-------------|:------:|
| Jupyter 无法访问 | 浏览器返回 502/504；Jupyter 服务未启动 | `get_pod_status`、`get_pod_logs`、`get_pod_events` | 好 |
| WebIDE 无法访问 | 页面加载失败；code-server 进程未运行 | `get_pod_status`、`get_pod_logs`、`get_pod_events` | 好 |
| Jupyter Kernel 崩溃 | Kernel 频繁 Restarting；执行代码无响应 | `get_pod_logs` | 好 |
| Kernel OOM | Kernel 死亡并提示内存不足；变量数据丢失 | `get_pod_logs`、`get_pod_status` | 好 |
| 端口转发失败 | TensorBoard 等辅助服务无法通过浏览器访问 | `get_pod_status`（间接） | 部分 |
| 长时间空闲被回收 | 交互式作业因超时被平台自动停止 | `get_pod_events` | 好 |
| SSH 连接失败 | 用户无法通过 SSH 进入交互式容器 | `get_pod_status`、`get_pod_logs` | 部分 |

---

## 训练 vs 推理故障差异

训练作业与推理/Serving 作业在故障模式上存在显著差异，需要不同的诊断策略和工具支持。

### 基本特征对比

| 维度 | 训练作业 | 推理/Serving 作业 |
|------|---------|-----------------|
| 典型持续时间 | 数小时～数周 | 持续运行（长期在线） |
| GPU 使用模式 | 持续高利用率（接近 100%） | 突发请求驱动（波动大） |
| 网络敏感度 | 极高（AllReduce 需要所有 rank 同步） | 中等（请求分发，单点故障可容忍） |
| 恢复策略 | Checkpoint 恢复、重提交作业 | 自动重启、滚动更新、流量摘除 |
| 数据依赖 | 训练数据集完整性与一致性 | 模型权重完整性与版本正确性 |
| 扩缩容需求 | 固定资源，训练期间不变 | 弹性扩缩容，按负载调整 |
| 故障恢复时效要求 | 分钟级～小时级（可接受） | 秒级～分钟级（高可用要求） |

### 故障类型对比

| 故障类别 | 训练常见场景 | 推理常见场景 | 根因差异 |
|---------|-----------|-----------|---------|
| **GPU OOM** | 梯度累积/batch size 过大 | KV Cache 膨胀/长序列/并发请求过多 | 训练可调 batch size；推理需限制并发或分页 |
| **进程崩溃** | rank 0 崩溃→全局失败 | 单 Pod 崩溃→负载均衡摘除 | 训练是全局一致性；推理是独立副本 |
| **通信超时** | NCCL AllReduce 同步失败 | 不常见（推理通常单机或 TP） | 训练涉及多节点同步；推理主要是单请求 |
| **显存碎片** | 罕见（固定计算图） | 常见（变长 KV Cache） | 推理需要 PagedAttention 等显存管理 |
| **性能退化** | loss 不降/吞吐量下降 | P99 延迟升高/首 token 延迟增大 | 监控指标完全不同 |
| **模型问题** | loss NaN/梯度爆炸/发散 | 输出乱码/概率异常/精度下降 | 训练看 loss 曲线；推理看输出质量 |
| **存储故障** | 训练数据读取 I/O 瓶颈 | 模型加载失败/权重文件损坏 | 训练持续读数据；推理主要在启动时加载 |

### 国产卡场景的训练 vs 推理差异

| 维度 | 国产卡训练 | 国产卡推理 |
|------|----------|----------|
| 算子兼容性 | 反向传播算子缺失更多（autograd） | 前向推理算子覆盖较好 |
| 量化支持 | 训练量化（QAT）支持有限 | 推理量化（W4A8/W8A8）各厂商积极支持 |
| 通信库 | 必须用 HCCL/RCCL/CNCL 替代 NCCL | 单机推理通常不依赖通信库 |
| FlashAttention | 训练版本可能不支持 | 推理版本各厂商有定制实现 |
| 动态 shape | 变长序列 padding 开销大 | 同样受限，但 continuous batching 缓解 |
| 性能瓶颈 | MFU 低（35-42% vs NVIDIA 54-58%） | 推理延迟差距相对较小 |
| 最佳场景 | 小模型微调（LoRA/QLoRA/SFT） | 特定模型推理（DeepSeek/Qwen 已优化） |

### 监控指标对比

| 指标 | 训练必需 | 推理必需 | 当前 Agent 覆盖 |
|------|:-------:|:-------:|:-------------:|
| GPU 利用率 | Yes | Yes | **NVIDIA 有**，国产卡无 |
| 显存占用 | Yes | Yes | **NVIDIA 有**，国产卡无 |
| throughput (samples/s) | Yes | - | 无（需应用层上报） |
| loss 曲线 | Yes | - | `detect_training_anomaly_patterns` |
| QPS (请求/秒) | - | Yes | **无**（需推理框架 exporter） |
| P50/P99 延迟 | - | Yes | **无**（需推理框架 exporter） |
| 首 token 延迟 (TTFT) | - | Yes | **无** |
| KV Cache 占用率 | - | Yes | **无** |
| 梯度范数 | Yes | - | `detect_training_anomaly_patterns` |
| PCIe 带宽 | Yes | Yes | `DCGM_FI_PROF_PCIE_TX/RX_BYTES`（仅 NVIDIA） |
| 节点间带宽 | Yes（分布式） | 较少 | `check_node_nic_status`（链路状态，非实时带宽） |

> **现实评估**：
> - 推理基础诊断覆盖度约 **40%**（OOM/崩溃/调度/硬件状态可复用训练工具）
> - 推理请求级指标完全空白，卡点在推理框架 Prometheus exporter 未部署
> - 国产卡场景下，训练和推理的 GPU 运行时指标都缺失（非 Agent 层可解决）

---

## 工具覆盖度总结

基于以上分层分析，各作业类型和故障层级的工具覆盖度评估如下：

### 按作业类型

| 作业类型 | 覆盖度 | 说明 |
|---------|:------:|------|
| 单机训练（custom 单机） | **90%** | 日志、事件、状态、GPU 信息全覆盖 |
| 交互式作业（Jupyter / WebIDE） | **85%** | 基本诊断完备，端口转发和 SSH 诊断较弱 |
| 分布式训练（PyTorch DDP） | **80%** | 新增 NCCL 诊断工具（B8）后从 70% 提升至 80% |
| 硬件故障诊断 | **80%** | `get_node_gpu_info` + `get_node_accelerator_info` 上线后从 65% 提升至 80% |
| 集群运维（admin） | **90%** | 节点管理、配额、审计等管理功能覆盖充分 |
| 推理/Serving | **40%** | 基础故障排查（OOM、崩溃、调度）可复用训练工具；请求级指标（QPS/延迟）完全空白 |

### 按故障层级

| 故障层级 | 覆盖度 | 主要缺口 |
|---------|:------:|---------|
| Layer 1: 硬件层 | **75%** | IB 交换机诊断、NIC 速率协商、CRC 错误计数 |
| Layer 2: 基础设施层 | **85%** | NetworkPolicy 分析、DNS 诊断 |
| Layer 3: 分布式通信层 | **80%** | AllReduce 性能基准对比、MPI 深度诊断 |
| Layer 4: 平台层 | **85%** | Webhook 回调监控、抢占事件追踪 |
| Layer 5: 框架/应用层 | **80%** | 隐性故障（loss 发散、训练挂起）的自动检测 |
| Layer 6: 交互式作业 | **85%** | 端口映射诊断、SSH 连接诊断 |

### 覆盖度整体评估

```
整体覆盖度（训练场景）: ████████████████████░  ~85%
整体覆盖度（推理场景）: ████████░░░░░░░░░░░░  ~40%（基础诊断可复用训练工具）
整体覆盖度（加权平均）: ██████████████░░░░░░  ~70%
```

---

## 国产加速卡兼容性与可观测性现状

### 异构加速卡生态全景

国产加速卡与 NVIDIA CUDA 生态的核心差异在于三条技术路线：

| 路线 | 代表厂商 | 优势 | 劣势 |
|------|---------|------|------|
| **全自研栈** | 华为 Ascend（CANN/HCCL）、寒武纪（Neuware/CNCL） | 自主可控、可深度定制 | 迁移成本最高、生态薄弱 |
| **ROCm 兼容** | 海光 DCU（DTK/RCCL） | 迁移成本较低（HIP 翻译） | 绑定 AMD 生态演进 |
| **CUDA API 翻译层** | 天数智芯（CUDA 10.2 兼容）、壁仞（SUPA） | 迁移门槛最低 | 性能损耗、长期维护风险 |

### 各厂商详细对比

#### 通信库差异

| | NCCL (NVIDIA) | HCCL (华为) | RCCL (海光) | CNCL (寒武纪) |
|---|---|---|---|---|
| 来源 | NVIDIA 原创 | 全自研 | AMD 分支 | 全自研 |
| 开源 | GitHub | Gitee | via AMD | 不开源 |
| 互联 | NVLink/NVSwitch/IB | HCCS/RoCE | PCIe/RoCE | MLU-Link/RoCE/IB |
| 算法 | Ring/Tree/PAT/NVLS | Ring/Mesh/HD/NHR | **仅 Ring** | HDR/DBT |
| 规模 | 10 万+ GPU 验证 | 千卡级 | **256+ GPU 后退化** | 宣称万卡 |
| 成熟度 | 生产级（18 年） | 快速成熟中 | 中等 | 中等 |

> **关键问题**：海光 DCU 的 RCCL 仅支持 Ring 算法，超过 ~256 卡时延迟线性增长；寒武纪 CNCL 不开源，调试困难。

#### 框架支持差异

| 能力 | NVIDIA CUDA | 国产卡 |
|------|-----------|--------|
| PyTorch 原生支持 | 一等公民 | 需插件（torch_npu/torch_mlu/DTK+HIP） |
| 算子库数量 | 400+ 专用库（cuDNN/cuBLAS 等） | 不到 NVIDIA 的 1/3 |
| 自定义 CUDA kernel | 完整支持 | 需重写（海光/天数智芯可部分兼容） |
| torch.compile / Inductor | 完整 | 寒武纪支持 Inductor；昇腾有限 |
| FlashAttention | 原生 | 各厂商需自行实现 |
| 量化（W4A4/W8A8） | 成熟 | 逐模型追赶中 |
| 动态 shape | 完整 | **所有国产卡均受限**（变长序列需 padding/chunking） |
| foreach 操作 | 完整 | **昇腾不支持** `_foreach_norm`（影响 clip_grad_norm） |

#### 性能对比

| 指标 | NVIDIA H100 | Ascend 910B | Hygon DCU Z100 | Cambricon MLU590 |
|------|:-----------:|:-----------:|:--------------:|:----------------:|
| 算力利用率 (MFU) | 54-58% | 35-42% | - | 50-100% (因模型而异) |
| 训练效率 (vs H100) | 100% | ~60-70% | ~65% | ~50-80% |
| 推理性能 | 基准 | DeepSeek/Qwen 良好 | vLLM 可用 | 特定模型接近 A100 |

#### 模型兼容性

| 模型 | Ascend 910B | Hygon DCU | Cambricon MLU | 天数智芯 | 壁仞 |
|------|:-----------:|:---------:|:-------------:|:-------:|:----:|
| Qwen2.5/3 | Yes | Yes | Yes | 部分 | Yes |
| DeepSeek V3/R1 | Yes (MTP/EP) | Yes | - | - | Yes |
| LLaMA 3.x | 部分 (head_dim!=128 有限制) | Yes (via HIP) | 部分 | 部分 | - |
| 自定义 LoRA/QLoRA | 需额外适配 | 较好 (ROCm 兼容) | 需 torch_mlu | 可能兼容 | - |

### 指标采集能力矩阵

| 加速卡类型 | 资源分配检测 | 运行时指标 | 管理工具 | 通信库 | Agent 诊断能力 |
|-----------|:-----------:|:--------:|:-------:|:-----:|:------------:|
| NVIDIA GPU | K8s label + DCGM | 16+ DCGM 指标 | nvidia-smi | NCCL | **完整** |
| Hygon DCU (海光) | K8s label (hygon.com/) | **无 Prometheus 指标** | rocm-smi / hy-smi | RCCL | **基础**（仅硬件检测） |
| AMD GPU | K8s label (amd.com/) | **无标准指标** | rocm-smi | RCCL | **基础** |
| Ascend NPU (昇腾) | **未接入** | **无** | npu-smi | HCCL | **仅 `get_node_accelerator_info`** |
| Cambricon MLU (寒武纪) | **未接入** | **无** | cnmon | CNCL | **仅 `get_node_accelerator_info`** |

### 国产卡常见故障场景

| 故障场景 | NVIDIA 对照 | 国产卡差异 | 当前诊断能力 |
|----------|-----------|----------|:----------:|
| 驱动版本不兼容 | nvidia-smi 版本检查 | 各厂商版本管理混乱，升级频繁 | `get_node_accelerator_info` 可检测 |
| 通信库不支持某框架 | NCCL 广泛支持 | HCCL/RCCL 对 PyTorch 版本有严格要求 | `get_node_accelerator_info` 检查安装情况 |
| 某些模型算子不支持 | CUDA 算子完整 | Ascend CANN / DCU ROCm 算子覆盖不全 | `get_job_logs` + `detect_training_anomaly_patterns` 抓错误日志 |
| 显存碎片化 / HBM 异常 | DCGM ECC 指标 | 无标准指标暴露 | **无法检测** |
| 分布式通信超时 | NCCL debug 日志丰富 | HCCL/RCCL 日志格式不同 | `detect_training_anomaly_patterns` 部分覆盖 |
| GPU 初始化失败 | CUDA_ERROR 日志明确 | 错误信息不统一 | `get_job_logs` 可抓，但模式匹配覆盖有限 |

### 小集群微调场景特别说明

本平台实际场景为小规模集群（卡数有限），主要运行开源小模型的后训练/微调任务（LoRA、QLoRA、SFT）。
在此场景下：

1. **大规模预训练问题（千卡级 NCCL 故障、Ring AllReduce 拓扑）基本不会出现**
2. **最常见问题是兼容性**：某个模型在国产卡上跑不起来、某个算子不支持、通信库版本冲突
3. **显存不够是第二常见**：小卡（16GB/24GB HBM）跑 7B+ 模型 OOM
4. **微调超参问题（loss 发散、梯度爆炸）已被 `detect_training_anomaly_patterns` 覆盖**

### 改善方向

- **短期（已完成）**：`get_node_accelerator_info` 提供多厂商硬件检测和通信库安装验证
- **中期**：需要各国产卡厂商提供 Prometheus exporter（类似 DCGM Exporter），才能补齐运行时指标
- **长期**：`detect_training_anomaly_patterns` 的正则规则库需要持续扩展，覆盖各厂商特有的错误日志格式

---

## 推理作业诊断：现有能力 vs 真实缺口

### 用现有工具可做的推理诊断

| 推理故障场景 | 可用工具 | 效果 |
|------------|---------|:----:|
| 推理 Pod 崩溃/OOM | `k8s_get_pod_logs` + `k8s_get_events` + `detect_training_anomaly_patterns` | **好** |
| GPU 显存不足 | `query_job_metrics`（gpu_mem）+ `get_job_logs` | **好** |
| 调度卡住 | `analyze_queue_status` + `check_quota` | **好** |
| 镜像拉取失败 | `aggregate_image_pull_errors` + `harbor_check` | **好** |
| 节点 GPU 故障 | `get_node_gpu_info` + `get_node_kernel_diagnostics` | **好** |
| 国产卡驱动问题 | `get_node_accelerator_info` | **好** |
| 模型加载失败 | `get_job_logs`（错误日志） | **中等** |

### 真正做不了的（需要基础设施配合）

| 缺失能力 | 卡点 | 依赖 |
|---------|------|------|
| 请求级 QPS/延迟 | 推理框架未部署 Prometheus exporter | vLLM/Triton metrics endpoint |
| P99 延迟监控 | 同上 | 同上 |
| 模型版本管理 | 无模型注册中心 API | 需后端实现 ModelRegistry |
| 自动扩缩容状态 | 无 HPA/KEDA 集成 | 需部署推理自动扩缩 |
| 推理请求追踪 | 无 OpenTelemetry 集成 | 需框架级 tracing |

---

当前 Agent 工具链对推理/Serving 场景的覆盖度为 **0%**。以下列出建设推理诊断能力所需的工具清单。

### 必需工具

| 工具名称 | 功能描述 | 优先级 |
|---------|---------|:------:|
| `create_inference_job` / `create_serving_job` | 创建推理/Serving 作业，支持模型部署配置、副本数设置、资源规格定义 | P0 |
| `get_serving_endpoint_status` | 查询 Serving 端点健康状态，包括副本就绪数、端点 URL、健康检查结果、流量权重 | P0 |
| `query_inference_metrics` | 查询推理服务的核心指标：QPS、P50/P99/P999 延迟、错误率、GPU 利用率、显存占用、批处理队列深度 | P0 |
| `model_version_check` | 验证已部署模型的版本一致性，检查模型文件完整性（checksum）、框架兼容性、权重格式 | P1 |

### 扩展工具

| 工具名称 | 功能描述 | 优先级 |
|---------|---------|:------:|
| `get_serving_logs` | 获取推理服务日志，支持按请求 ID 过滤，区分推理日志和系统日志 | P1 |
| `scale_serving_replicas` | 手动调整推理服务副本数，支持紧急扩容和缩容操作 | P1 |
| `get_inference_request_trace` | 获取单个推理请求的全链路追踪，包括预处理、推理、后处理各阶段耗时 | P2 |
| `rollback_model_version` | 回滚模型版本至指定版本，支持灰度回滚和全量回滚 | P2 |
| `get_serving_autoscaler_status` | 查询自动扩缩容策略状态，包括当前副本数、目标指标、扩缩容历史事件 | P2 |

### 推理场景诊断覆盖路线

1. **第一阶段（P0）**：实现基础的推理作业创建、状态查询和指标监控，覆盖度目标 **50%**
2. **第二阶段（P1）**：补充日志查询、模型版本管理和副本管理，覆盖度目标 **75%**
3. **第三阶段（P2）**：实现全链路追踪、自动扩缩容诊断和智能根因分析，覆盖度目标 **90%**
