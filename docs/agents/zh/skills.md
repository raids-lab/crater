# 技能系统

> 通过 YAML 文件进行诊断知识注入——赋予智能体领域专业知识，无需工具调用。

---

## 目的

技能是结构化的知识文件，将诊断模式、触发信号和常见解决方案直接注入智能体的系统提示词中。与 RAG（检索增强生成）不同，技能是确定性加载的——所有技能始终可用，而非基于相似度搜索进行检索。

这种方式以 token 成本换取可靠性：智能体始终能够访问已知故障模式的诊断知识，无需经历检索步骤的延迟或召回率的不确定性。

---

## 技能格式

每个技能是 `crater_agent/skills/` 目录下的一个 YAML 文件：

```yaml
name: "OOM Diagnosis"
description: "Diagnose Out-of-Memory failures in job containers"

trigger_signals:
  exit_codes: [137, 143]
  event_reasons: ["OOMKilled", "Evicted"]
  log_keywords: ["out of memory", "OOM", "Cannot allocate memory"]
  job_status: ["Failed"]

diagnosis_knowledge: |
  当作业因 OOM 被杀时，通常表明：
  1. 请求内存不足 (memory request < actual usage)
  2. 容器进程内存泄漏
  3. 数据加载一次性加载过大

common_solutions:
  - condition: "GPU 显存不足"
    suggestion: "增加 GPU 显存分配或使用梯度检查点"
  - condition: "CPU 内存不足"
    suggestion: "减少 batch size 或启用数据流式处理"
```

### 字段说明

| 字段 | 用途 |
|-------|---------|
| `name` | 技能标识符 |
| `description` | 该技能的适用场景 |
| `trigger_signals.exit_codes` | 激活该知识的容器退出码 |
| `trigger_signals.event_reasons` | K8s 事件原因 |
| `trigger_signals.log_keywords` | 需要监控的日志模式 |
| `trigger_signals.job_status` | 作业状态值 |
| `diagnosis_knowledge` | 供 LLM 使用的自由文本诊断推理 |
| `common_solutions` | 条件 → 建议的映射 |

---

## 当前技能

| 文件 | 覆盖范围 |
|------|----------|
| `oom_diagnosis.yaml` | OOMKilled、内存耗尽 |
| `image_pull_error.yaml` | ImagePullBackOff、镜像仓库故障 |
| `queue_pending.yaml` | 调度延迟、资源争用 |
| `scheduling_failed.yaml` | 污点/亲和性不匹配、资源不足 |

---

## 注入机制

```python
# loader.py
load_all_skills(skills_dir) → str
  1. Scan *.yaml files in skills_dir
  2. Parse each with yaml.safe_load()
  3. Format each with format_skill_for_prompt(skill)
  4. Concatenate under "## 诊断参考知识" header

# prompts.py
build_system_prompt(context, skills_context=skills_text)
  → Appends skills_context to system prompt template
```

格式化后的输出如下所示：

```markdown
## 诊断参考知识

### OOM Diagnosis
OOM 诊断参考...

触发信号: exit_codes=[137, 143], event_reasons=["OOMKilled"]

诊断知识:
当作业因 OOM 被杀时...

常见方案:
- GPU 显存不足 → 增加 GPU 显存分配或使用梯度检查点
- CPU 内存不足 → 减少 batch size

### Image Pull Error
...
```

---

## 添加新技能

1. 创建 `crater_agent/skills/your_skill.yaml`
2. 填写 YAML 结构（name、trigger_signals、diagnosis_knowledge、common_solutions）
3. 无需修改代码——`load_all_skills()` 会自动发现新的 YAML 文件

---

## 设计决策

| 决策 | 理由 |
|----------|-----------|
| YAML 而非数据库 | 可版本控制、可审查、无运行时依赖 |
| 全量注入而非 RAG | 确定性、无召回失败、token 成本可接受（总计约 1500 tokens） |
| 触发信号作为元数据 | 支持未来基于作业状态的选择性注入（尚未实现） |

---

## 代码

| 组件 | 文件 |
|-----------|------|
| 技能加载器 | `crater_agent/skills/loader.py` |
| 技能 YAML 文件 | `crater_agent/skills/*.yaml` |
| 提示词注入 | `crater_agent/agent/prompts.py` |
