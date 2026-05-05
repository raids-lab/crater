# Mops: Multi-Agent Framework for Intelligent Operations

> **M**ulti-agent **Op**eration**s** — A framework for building intelligent operations agents on AI computing platforms.

---

## Overview

Mops is the multi-agent operations framework powering Crater's intelligent computing platform. It provides a composable architecture for building, orchestrating, and evaluating LLM-powered agents that perform fault diagnosis, resource optimization, proactive inspection, and automated operations on GPU/accelerator clusters.

The framework is designed with three core principles:

1. **Platform-agnostic tool layer** — agents interact with infrastructure through a pluggable tool executor interface, not direct API calls
2. **Composable agent roles** — each agent has a single responsibility and a restricted tool set; new agents are added without modifying existing ones
3. **Safety-first operations** — write operations require explicit confirmation, all actions are auditable, and failures always fall back to human operators

---

## Framework Architecture

```
                         ┌─────────────────────────────────────┐
                         │        Application Layer            │
                         │   FastAPI /chat (SSE streaming)     │
                         │   FastAPI /evaluate/* (synchronous) │
                         └──────────────┬──────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
            ┌───────▼───────┐   ┌──────▼──────┐   ┌───────▼───────┐
            │ Single-Agent  │   │ Multi-Agent  │   │  Task-Specific│
            │ Orchestrator  │   │ Orchestrator │   │  Agents       │
            │ (ReAct loop)  │   │ (Coordinator │   │  (Approval,   │
            │               │   │  pipeline)   │   │   Pipeline)   │
            └───────┬───────┘   └──────┬──────┘   └───────┬───────┘
                    │                  │                   │
                    └──────────────────┼───────────────────┘
                                       │
                         ┌─────────────▼─────────────┐
                         │     Agent Graph Layer      │
                         │  LangGraph ReAct StateGraph│
                         │  (agent → tools → agent)   │
                         └─────────────┬─────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
            ┌───────▼───────┐  ┌──────▼──────┐  ┌───────▼───────┐
            │  Tool Selector│  │    Token     │  │   Skills      │
            │  (role-based) │  │  Management  │  │   Knowledge   │
            └───────┬───────┘  └─────────────┘  └───────────────┘
                    │
         ┌──────────┼──────────┐
         │          │          │
   ┌─────▼────┐ ┌──▼─────┐ ┌──▼──────────┐
   │  Local    │ │   Go   │ │    Mock     │
   │ Executor  │ │Backend │ │  Executor   │
   │(kubectl,  │ │Executor│ │(benchmark)  │
   │PromQL,web)│ │(HTTP)  │ │             │
   └──────────┘ └────────┘ └─────────────┘
```

---

## Agent Catalog

| Agent | Mode | Purpose | Tools | Doc |
|-------|------|---------|-------|-----|
| **Single Agent** | ReAct | General-purpose diagnosis and operations | All (role-filtered) | [single-agent.md](single-agent.md) |
| **Coordinator** | MAS | Request routing and stage orchestration | None (LLM-only) | [coordinator.md](coordinator.md) |
| **Planner** | MAS | Decompose requests into investigation plans | None (LLM-only) | [planner.md](planner.md) |
| **Explorer** | MAS | Evidence collection through read-only tools | Read-only subset | [explorer.md](explorer.md) |
| **Executor** | MAS | Execute write operations with confirmation | Read + Write | [executor.md](executor.md) |
| **Verifier** | MAS | Validate conclusions and flag evidence gaps | None (LLM-only) | [verifier.md](verifier.md) |
| **TicketAgent** | Task | Base class for ticket/order evaluation agents | Configurable whitelist | [ticket-agent.md](ticket-agent.md) |
| **Approval** | Task | Evaluate job lock approval orders | Fixed whitelist (7 tools) | [approval-agent.md](approval-agent.md) |
| **Inspection** | Task | Scheduled cluster health inspection and reporting | Read-only (via PipelineToolClient) | [inspection-pipeline.md](inspection-pipeline.md) |
| **Guide** | MAS | Answer "how-to" questions about the platform | None (LLM-only) | [guide-general.md](guide-general.md) |
| **General** | MAS | Handle greetings and simple platform Q&A | None (LLM-only) | [guide-general.md](guide-general.md) |

---

## Orchestration Modes

### Single-Agent (ReAct)

A single LLM instance with tool access runs a think-act-observe loop until it produces a final answer or hits the tool call limit.

```
User message → [LLM thinks] → [Calls tool] → [Observes result] → [LLM thinks] → ... → Final answer
```

Best for: simple queries, direct diagnoses, single-job operations.

### Multi-Agent (Coordinator Pipeline)

A coordinator LLM routes the request and orchestrates specialized sub-agents through a staged pipeline.

```
User message → [Coordinator routes] → [Planner] → [Explorer loop] → [Executor loop] → [Verifier] → Final answer
                                                                                            ↑
                                                              Coordinator may loop back ─────┘
```

Best for: complex multi-step investigations, cross-object correlation, operations requiring planning.

### Task-Specific (Synchronous)

Standalone agents that reuse the ReAct graph but run synchronously for specific backend hooks (not user-facing chat).

```
Backend event → [Task Agent with restricted tools] → Structured result → Backend action
```

Best for: automated decisions (approval evaluation), batch analysis (GPU audit), scheduled reports.

---

## Extensibility

### Adding a New Agent

1. Create `agents/your_agent.py` inheriting patterns from `BaseRoleAgent`
2. Define a tool whitelist (subset of existing tools)
3. Write a system prompt specific to the agent's role
4. Choose integration mode:
   - **MAS sub-agent**: Add as a new stage in `MultiAgentOrchestrator`
   - **Task agent**: Add a new FastAPI endpoint in `app.py`
   - **ReAct agent**: Reuse `create_agent_graph` with `capabilities.enabled_tools`

No existing code needs modification — the graph, tool executor, and token management are all reusable.

### Adding a New Tool

1. Define the tool function in `tools/definitions.py` with `@tool` decorator
2. Add to `AUTO_TOOLS` (read-only), reserved `AUTO_ACTION_TOOLS` (system-only side effect without HITL), or `CONFIRM_TOOLS` (confirmed write/external action)
3. Add the Go backend handler in `handler/agent/tools_readonly.go` or `tools_dispatch.go`
4. If portable (no Go dependency), implement in `tools/local_executor.py`

### Adapting to a Different Platform

The framework separates **agent logic** from **platform specifics**:

| Layer | What to change | What stays |
|-------|---------------|------------|
| Tools | Tool definitions + Go handlers | Agent graph, orchestration, prompt patterns |
| Auth | `GoBackendToolExecutor` auth headers | Agent roles, tool selection logic |
| Config | `platform-runtime.yaml` endpoints | LLM client factory, token management |
| Skills | Diagnostic YAML knowledge files | Skill loading mechanism |
| Prompts | Domain-specific prompt templates | Prompt injection infrastructure |

The `LocalToolExecutor` enables standalone deployment without the Go backend — agents can operate directly against Kubernetes, Prometheus, and other APIs.

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| LangGraph over workflow engines (Temporal, Dagster) | Lightweight, LLM-native state management; no external dependencies |
| Tool execution via HTTP proxy | Centralized auth, audit, rate limiting in Go backend; agents don't hold cluster credentials |
| Confirmation as graph pause | Preserves conversation state across async user approval; no polling |
| Per-tool token budgets | Prevents oversized tool results from consuming context; tool-specific limits match expected output sizes |
| LLM compaction before hard truncation | Preserves semantic content; hard truncation is a fallback, not the primary strategy |
| Deterministic + LLM routing | Fast pattern matching for common cases; LLM only for ambiguous requests |
| Role-based tool filtering at selection time | Tools are defined once; access control is orthogonal to tool logic |

---

## Project Structure

```
crater-agent/
  crater_agent/
    agent/          # ReAct graph, state, prompts, compaction
    agents/         # Specialized agent roles (planner, explorer, executor, ...)
    orchestrators/  # Single-agent and multi-agent orchestration
    tools/          # Tool definitions, executors, selectors
    skills/         # Diagnostic knowledge YAML files
    memory/         # Session history management
    llm/            # LLM client factory and tokenizer
    pipeline/       # Batch pipelines (GPU audit, ops reports)
    eval/           # Evaluation runner and metrics
    config.py       # Settings and configuration
    app.py          # FastAPI application entry point
  crater_bench/     # Benchmark scenarios and mock responses
  config/           # Runtime configuration files
  dataset/          # Data collection and transformation
  tests/            # Test suite
```

---

## Documentation Index

### System Design
- [architecture.md](architecture.md) — System layers, data flow, state management, safety mechanisms
- [memory-context.md](memory-context.md) — Conversation history, context injection, message compaction
- [tools.md](tools.md) — Tool declaration, execution backends, role filtering, result processing
- [skills.md](skills.md) — Diagnostic knowledge YAML injection

### Agent Roles
- [single-agent.md](single-agent.md) — ReAct loop (foundational building block)
- [coordinator.md](coordinator.md) — Request routing and MAS flow control
- [planner.md](planner.md) — Investigation plan decomposition
- [explorer.md](explorer.md) — Evidence collection (read-only tools)
- [executor.md](executor.md) — Write operations with confirmation
- [verifier.md](verifier.md) — Quality gate and evidence validation
- [guide-general.md](guide-general.md) — Help and conversational agents
- [ticket-agent.md](ticket-agent.md) — Ticket agent base class (extensible framework)
- [approval-agent.md](approval-agent.md) — Job lock approval (TicketAgent instance)
- [inspection-pipeline.md](inspection-pipeline.md) — Scheduled cluster inspection and reporting

### 机制与护栏
- [system-prompt.md](system-prompt.md) — System Prompt 完整模板与动态注入机制
- [confirmation-mechanism.md](confirmation-mechanism.md) — 写操作确认卡片的完整前后端+Agent链路
- [role-separation.md](role-separation.md) — Admin/Portal 角色区分的当前实现
- [guardrails.md](guardrails.md) — 工具调用上限、Token 预算、去重等 Harness 限制

### 问题追踪
- [known-issues.md](known-issues.md) — 已知问题与根因分析（效果调优基线）

### Engineering
- [evaluation.md](evaluation.md) — Benchmark harness, scenarios, metrics, data collection
- [image-authoring-testing.md](image-authoring-testing.md) — Natural-language and offline-replay test cases for image build / import / sharing workflows

### Specifications
- [../specs/agent-approval-hook.md](../specs/agent-approval-hook.md) — Approval agent design spec

---

*Mops is part of the Crater intelligent computing platform. For the overall platform architecture, see the main project documentation.*
