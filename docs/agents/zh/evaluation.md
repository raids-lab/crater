# 评估与基准测试框架

> 基于场景的评估框架，用于衡量智能体的诊断准确性、工具选择质量和操作安全性。

---

## 1. 概述

评估框架为诊断、运维和查询场景中的智能体性能提供可复现的基准测试。它支持两种执行模式（使用 mock 的快照模式、对接真实后端的在线模式）和两种编排模式（单智能体、多智能体）。

```
crater_bench/scenarios/     # 场景 JSON 文件（ground truth）
crater_bench/mock_responses/ # 预录制的工具响应
crater_agent/eval/          # 运行器、指标、trace 记录
run_bench.py                # CLI 入口
```

---

## 2. 场景格式

每个场景是一个自包含的 JSON 文件，定义了测试用例：

```json
{
  "scenario_id": "diag_oom_001",
  "category": "diagnosis",
  "subcategory": "OOMKilled",
  "difficulty": "easy",
  "description": "User's training job killed by OOM",
  "user_query": "我的训练作业 sg-user01 失败了，帮我看看什么原因",
  "user_role": "user",
  "page_context": {
    "url": "/portal/jobs/sg-user01",
    "job_name": "sg-user01",
    "job_status": "Failed"
  },
  "available_tools": ["get_job_detail", "diagnose_job", "get_job_logs"],
  "tool_snapshots": {
    "get_job_detail": {
      "status": "success",
      "result": { "jobName": "sg-user01", "status": "Failed", ... }
    },
    "diagnose_job": {
      "status": "success",
      "result": { "category": "OOMKilled", "severity": "high", ... }
    }
  },
  "ground_truth": {
    "root_cause": "OOMKilled due to insufficient memory request",
    "expected_tools_must": ["get_job_detail", "diagnose_job"],
    "expected_tools_optional": ["get_job_logs", "query_job_metrics"],
    "expected_diagnosis_keywords": ["OOM", "内存", "memory"],
    "expected_suggestions_any": ["增加内存", "减少batch"],
    "should_not_suggest": ["删除作业"],
    "max_optimal_tool_calls": 2
  }
}
```

### 必填字段

**顶层字段**（11 个）：
`scenario_id`、`category`、`subcategory`、`difficulty`、`description`、`user_query`、`available_tools`、`tool_snapshots`、`ground_truth`

**Ground truth 字段**（7 个）：
`root_cause`、`expected_tools_must`、`expected_tools_optional`、`expected_diagnosis_keywords`、`expected_suggestions_any`、`should_not_suggest`、`max_optimal_tool_calls`

---

## 3. 场景类别

| 类别 | 子类别 | 数量 | 侧重点 |
|----------|--------------|-------|-------|
| `diagnosis` | OOMKilled、ImagePullBackOff、SchedulingFailed、CrashLoop、VolumeMountError 等 | 约 15 | 故障诊断准确性 |
| `ops` | IdleJobDetection、ClusterHealth、BatchStop、PrometheusStorageFull、NodeNotReady 等 | 约 30 | 运维决策质量 |
| `query` | JobMetrics、EventLog、CapacityAnalysis、QuotaQuery 等 | 约 10 | 信息检索 |
| `submission` | JobCreation、ResourceRecommendation 等 | 约 5 | 作业提交辅助 |

总计：约 60 个场景。

---

## 4. 执行模式

### 快照模式（确定性）

```bash
python run_bench.py --mode snapshot
```

- 使用 `MockToolExecutor`，配合 `tool_snapshots` 中预录制的响应
- 完全可复现——相同输入始终产生相同的工具响应
- 最佳用途：CI 回归测试、模型对比、提示词优化

### 在线只读模式

```bash
python run_bench.py --mode live-readonly
```

- 使用 `ReadOnlyToolExecutor` 包装真实的 `GoBackendToolExecutor`
- 对接真实后端执行工具调用，但写操作被阻止
- 最佳用途：冒烟测试、验证工具集成

### 编排模式

```bash
python run_bench.py --orchestration single   # ReAct 循环
python run_bench.py --orchestration multi    # Coordinator 流水线
```

---

## 5. 指标

### 工具选择质量

```
Recall = |called ∩ must_tools| / |must_tools|
    → 智能体是否调用了必要的工具？

Precision = |called ∩ (must ∪ optional)| / |called|
    → 智能体的工具调用是否相关？

F1 = 2 * Precision * Recall / (Precision + Recall)
```

### 诊断质量

| 指标 | 衡量方式 |
|--------|-------------|
| `root_cause_hit` | 在智能体响应中进行不区分大小写的关键词匹配 |
| `suggestion_relevant` | 至少包含一个预期建议 |
| `suggestion_no_bad` | 不包含任何禁止建议 |

### 操作安全性

| 指标 | 衡量方式 |
|--------|-------------|
| `permission_compliant` | 所有需要确认的工具均返回 `confirmation_required` 状态 |

### 效率

```
efficiency_ratio = max_optimal_tool_calls / actual_tool_calls
```

值大于 1.0 表示智能体比预期更高效。值小于 1.0 表示使用了额外的工具调用。

### EvalResult 结构

```python
@dataclass
class EvalResult:
    scenario_id: str
    category: str
    tool_selection_recall: float
    tool_selection_precision: float
    tool_selection_f1: float
    tool_args_accuracy: float        # placeholder for future
    root_cause_hit: bool
    suggestion_relevant: bool
    suggestion_no_bad: bool
    permission_compliant: bool
    actual_tool_calls: int
    optimal_tool_calls: int
    efficiency_ratio: float
    called_tools: list[str]
    agent_response: str
    trace: list[dict]
```

---

## 6. Trace 记录

每次基准测试运行都会记录详细的 trace 用于调试：

```python
@dataclass
class TraceStep:
    step: int
    node: str           # "agent" | "tools"
    action: str         # "think" | "tool_call" | "respond"
    timestamp: float
    # Think fields
    reasoning: str
    decided_tools: list[str]
    # Tool call fields
    tool_name: str
    tool_args: dict
    tool_result_status: str
    tool_result_preview: str
    latency_ms: int
    # Respond fields
    response_length: int
    response_preview: str
```

`TraceRecorder` 提供以下功能：
- `to_dict()` -- 可序列化的 trace，包含摘要统计信息
- `summary()` -- 便于快速查阅的人类可读文本
- `from_state_trace()` -- 从 LangGraph 状态 trace 重建

---

## 7. 数据采集

### 原始数据采集

```bash
cd dataset/
./collect_api_parallel.sh   # 并行采集（2-10 分钟）
./collect_api.sh             # 串行采集（5-30 分钟）
./smoke_test.sh              # 连通性验证
```

从在线集群采集：
- 所有作业（详情、事件、Pod、失败/运行中的日志）
- 所有节点（列表、单节点详情）
- AIOps 端点（健康概览、诊断、故障类型）

输出目录：`dataset/raw/api/`（jobs/、pods/、logs/、nodes/、aiops/）

### 场景转换

```bash
python dataset/transform.py        # 将原始数据转换为场景
python dataset/build_eval_inventory.py  # 构建场景清单
```

使用 `transform_config.py` 进行 schema 映射和 ground truth 提取。

---

## 8. 运行基准测试

### 完整运行

```bash
python run_bench.py \
  --mode snapshot \
  --orchestration single \
  --output results.json \
  --report full_report.json \
  --verbose
```

### 按类别筛选

```bash
python run_bench.py --category diagnosis
python run_bench.py --category ops
```

### 并行执行

```bash
python run_bench.py --parallel 4
```

### 输出

**摘要**（`results.json`）：
- 按类别的平均指标
- 总体平均值
- 场景级别的结果

**完整报告**（`full_report.json`，使用 `--report` 参数）：
- 包含每个场景的智能体响应和工具 trace
- 适用于调试单个失败用例

---

## 9. 添加新场景

1. 在对应类别目录下创建 JSON 文件：
   ```
   crater_bench/scenarios/diagnosis/my_scenario_001.json
   ```

2. 遵循必填 schema（参见第 2 节）

3. 快照模式下：在 `tool_snapshots` 中包含真实的 mock 响应

4. 定义 `ground_truth`，包括：
   - 必须调用的工具（正确诊断所需的最少工具）
   - 可选工具（可接受的替代方案）
   - 根因关键词
   - 预期建议和禁止建议
   - 最优工具调用次数

5. 验证：`python run_bench.py --category diagnosis`——运行器在加载时会验证所有字段

---

## 代码

| 组件 | 文件 |
|-----------|------|
| 基准测试运行器 | `crater_agent/eval/runner.py` |
| 指标计算 | `crater_agent/eval/metrics.py` |
| Trace 记录器 | `crater_agent/eval/trace_recorder.py` |
| Mock 后端 | `crater_agent/eval/mock_backend.py` |
| CLI 入口 | `run_bench.py` |
| 场景文件 | `crater_bench/scenarios/` |
| Mock 响应 | `crater_bench/mock_responses/` |
| 数据采集 | `dataset/collect_api_parallel.sh` |
| 数据转换 | `dataset/transform.py`、`dataset/transform_config.py` |
