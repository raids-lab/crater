"""一次性生成全部 20 张 .drawio 源文件 (Ch1-Ch4)。"""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.drawio_builder import (
    DrawioBuilder,
    FONT_FAMILY,
    PALETTE,
    actor_style,
    edge_style,
    lifeline_style,
    message_style,
    node_style,
)


CH1 = ROOT / "ch1"
CH2 = ROOT / "ch2"
CH3 = ROOT / "ch3"
CH4 = ROOT / "ch4"
for d in (CH1, CH2, CH3, CH4):
    d.mkdir(parents=True, exist_ok=True)


# =============================================================================
# Ch1
# =============================================================================

def fig1_1_research_positioning():
    b = DrawioBuilder("研究问题定位", width=900, height=520)
    # 四个对比块
    lane1 = b.lane("通用多智能体系统", 40, 40, 380, 180, fill=PALETTE["light_purple"])
    b.node("AutoGen / MetaGPT / CrewAI", 60, 80, 340, 36,
           style=node_style(fill="#FFFFFF"), parent=lane1)
    b.node("通用任务编排与角色协作\n（缺少运维对象语义）", 60, 130, 340, 70,
           style=node_style(fill="#FFFFFF", font_size=9), parent=lane1)

    lane2 = b.lane("传统 AIOps / 运维智能体", 480, 40, 380, 180, fill=PALETTE["light_green"])
    b.node("RCACopilot / STRATUS / D-Bot", 500, 80, 340, 36,
           style=node_style(fill="#FFFFFF"), parent=lane2)
    b.node("微服务故障 检测 → 定位 → 缓解\n（缺少智算平台业务语义）", 500, 130, 340, 70,
           style=node_style(fill="#FFFFFF"), parent=lane2)

    lane3 = b.lane("Kubernetes Agent", 40, 250, 380, 180, fill=PALETTE["light_blue"])
    b.node("K8sGPT / Kagent / KubeIntellect", 60, 290, 340, 36,
           style=node_style(fill="#FFFFFF"), parent=lane3)
    b.node("K8s 控制面诊断与管理\n（缺少 GPU / 作业 / 审批）", 60, 340, 340, 70,
           style=node_style(fill="#FFFFFF"), parent=lane3)

    center = b.lane("本文：智算平台智能运维", 480, 250, 380, 230, fill=PALETTE["highlight"])
    b.node("Mops 多智能体运维框架", 500, 290, 340, 36,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF",
                             font_size=10, font_style=1), parent=center)
    b.node("智算平台多类型任务\n(GPU / 作业 / Jupyter / 审批 / 确认流)",
           500, 340, 340, 56,
           style=node_style(fill="#FFFFFF"), parent=center)
    b.node("工具增强 + 可信执行 + 场景基准", 500, 410, 340, 36,
           style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"),
           parent=center)

    b.write(CH1 / "fig1-1-research-positioning.drawio")


def fig1_2_research_roadmap():
    b = DrawioBuilder("研究内容与技术路线", width=1100, height=540)

    rc1 = b.lane("研究内容一：任务分析", 60, 60, 290, 380,
                 fill=PALETTE["light_blue"])
    b.node("智算平台运维任务分析", 80, 110, 250, 50,
           style=node_style(fill="#FFFFFF"), parent=rc1)
    b.node("场景基准设计\n(Crater-Bench 66 场景)", 80, 180, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc1)
    b.node("评测维度建模\n(任务/工具/安全合规)", 80, 260, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc1)
    b.node("数据集与工具快照", 80, 340, 250, 50,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"), parent=rc1)

    rc2 = b.lane("研究内容二：Mops 框架设计", 400, 60, 290, 380,
                 fill=PALETTE["light_green"])
    b.node("两级意图路由 + 协调器", 420, 110, 250, 50,
           style=node_style(fill="#FFFFFF"), parent=rc2)
    b.node("多智能体角色协作\n(Planner/Explorer/Executor/Verifier)",
           420, 180, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc2)
    b.node("MASState 共享状态\n+ StateView 投影", 420, 260, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc2)
    b.node("工具体系 + 安全控制", 420, 340, 250, 50,
           style=node_style(fill=PALETTE["ps"], text_color="#FFFFFF"), parent=rc2)

    rc3 = b.lane("研究内容三：实现与验证", 740, 60, 290, 380,
                 fill=PALETTE["light_purple"])
    b.node("系统实现\n(前端 + Go + Python)", 760, 110, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc3)
    b.node("离线评测 (Mock 工具快照)", 760, 190, 250, 50,
           style=node_style(fill="#FFFFFF"), parent=rc3)
    b.node("线上质量分析\n(LLM-as-Judge)", 760, 260, 250, 60,
           style=node_style(fill="#FFFFFF"), parent=rc3)
    b.node("消融实验 + 案例分析", 760, 340, 250, 50,
           style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"), parent=rc3)

    # 连线
    e_style = edge_style(stroke=PALETTE["subtle"])
    b.edge(rc1, rc2, "任务驱动设计", style=e_style)
    b.edge(rc2, rc3, "落地工程实现", style=e_style)
    b.edge(rc3, rc1, "实验反馈", style=edge_style(stroke=PALETTE["accent"], dashed=True))

    b.write(CH1 / "fig1-2-research-roadmap.drawio")


def fig1_3_five_layer_model():
    b = DrawioBuilder("智算平台五层运维对象模型", width=1000, height=560)

    layers = [
        ("用户负载层", "训练脚本 · 推理服务 · Jupyter 会话", PALETTE["light_purple"], 80),
        ("作业运行时层", "容器运行时 · NCCL/HCCL · 数据加载器", PALETTE["light_blue"], 160),
        ("平台服务层", "Prometheus · Harbor · 分布式存储 · Jupyter 服务", PALETTE["light_green"], 240),
        ("集群编排层", "Kubernetes 控制面 · Volcano 调度器 · 节点管理", PALETTE["highlight"], 320),
        ("硬件与驱动层", "GPU/NPU · InfiniBand/RoCE · 设备驱动", "#FFD8C2", 400),
    ]
    for name, detail, fill, y in layers:
        lane = b.lane(name, 80, y, 840, 64, fill=fill)
        b.node(detail, 100, y + 28, 800, 30,
               style=node_style(fill="#FFFFFF", font_size=9), parent=lane)

    # 故障传播箭头（右侧标注）
    b.text_label("故障跨层传播", 940, 90, 100, 20, font_size=9,
                 color=PALETTE["accent"])
    arrow_style = edge_style(stroke=PALETTE["accent"], orthogonal=False)
    # 用纯文本label在右侧作为视觉线索
    b.text_label("↓ NCCL 通信超时", 940, 180, 120, 20, font_size=8,
                 color=PALETTE["accent"])
    b.text_label("↓ 存储写入失败", 940, 260, 120, 20, font_size=8,
                 color=PALETTE["accent"])
    b.text_label("↓ 调度饥饿", 940, 340, 120, 20, font_size=8,
                 color=PALETTE["accent"])
    b.text_label("↓ Xid 错误", 940, 420, 120, 20, font_size=8,
                 color=PALETTE["accent"])

    # 左侧：运维智能体观测点
    b.text_label("← 运维智能体观测", 20, 90, 130, 20, font_size=9,
                 color=PALETTE["mops"])
    b.text_label("← 工具调用切入点", 20, 250, 130, 20, font_size=9,
                 color=PALETTE["mops"])
    b.text_label("← 写操作受确认流约束", 0, 420, 160, 20, font_size=9,
                 color=PALETTE["mops"])

    b.write(CH1 / "fig1-3-five-layer-model.drawio")


# =============================================================================
# Ch2
# =============================================================================

def fig2_1_cloud_native_stack():
    b = DrawioBuilder("智算平台云原生技术栈", width=1000, height=560)

    user = b.lane("用户层 (User)", 60, 60, 880, 80, fill=PALETTE["light_purple"])
    b.node("PyTorch / TensorFlow", 90, 90, 200, 40,
           style=node_style(fill="#FFFFFF"), parent=user)
    b.node("Jupyter Lab / WebIDE", 320, 90, 200, 40,
           style=node_style(fill="#FFFFFF"), parent=user)
    b.node("推理服务 (Triton/vLLM)", 550, 90, 200, 40,
           style=node_style(fill="#FFFFFF"), parent=user)
    b.node("控制台 (React 18)", 780, 90, 130, 40,
           style=node_style(fill="#FFFFFF"), parent=user)

    runtime = b.lane("作业运行时层 (Runtime)", 60, 160, 880, 80,
                     fill=PALETTE["light_blue"])
    b.node("containerd / runc", 90, 190, 220, 40,
           style=node_style(fill="#FFFFFF"), parent=runtime)
    b.node("NCCL / HCCL 分布式通信", 340, 190, 250, 40,
           style=node_style(fill="#FFFFFF"), parent=runtime)
    b.node("DataLoader / Ceph CSI", 610, 190, 250, 40,
           style=node_style(fill="#FFFFFF"), parent=runtime)

    svc = b.lane("平台服务层 (Platform Services)", 60, 260, 880, 100,
                 fill=PALETTE["light_green"])
    b.node("Prometheus + DCGM", 90, 295, 180, 40,
           style=node_style(fill="#FFFFFF"), parent=svc)
    b.node("Harbor 镜像仓库", 290, 295, 160, 40,
           style=node_style(fill="#FFFFFF"), parent=svc)
    b.node("Ceph / NFS 存储", 470, 295, 160, 40,
           style=node_style(fill="#FFFFFF"), parent=svc)
    b.node("Grafana 可视化", 650, 295, 140, 40,
           style=node_style(fill="#FFFFFF"), parent=svc)
    b.node("审计日志系统", 810, 295, 110, 40,
           style=node_style(fill="#FFFFFF"), parent=svc)

    orch = b.lane("集群编排层 (Orchestration)", 60, 380, 880, 80,
                  fill=PALETTE["highlight"])
    b.node("Kubernetes 控制面", 90, 410, 220, 40,
           style=node_style(fill="#FFFFFF"), parent=orch)
    b.node("Volcano Queue / PodGroup", 330, 410, 250, 40,
           style=node_style(fill="#FFFFFF"), parent=orch)
    b.node("Device Plugin / GPU Operator", 600, 410, 260, 40,
           style=node_style(fill="#FFFFFF"), parent=orch)

    hw = b.lane("硬件与驱动层 (Hardware)", 60, 480, 880, 60, fill="#FFD8C2")
    b.node("GPU / NPU 加速卡 · InfiniBand / RoCE v2 · NVMe SSD · 设备驱动",
           90, 502, 820, 32,
           style=node_style(fill="#FFFFFF", font_size=9), parent=hw)

    b.write(CH2 / "fig2-1-cloud-native-stack.drawio")


def fig2_2_react_loop():
    b = DrawioBuilder("ReAct 推理-行动循环", width=900, height=480)

    user = b.node("用户提问", 60, 200, 130, 50,
                  style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"))
    think = b.node("Thought\n（推理下一步）", 240, 100, 160, 70,
                   style=node_style(fill=PALETTE["light_blue"]))
    action = b.node("Action\n（调用工具）", 470, 100, 160, 70,
                    style=node_style(fill=PALETTE["light_green"]))
    obs = b.node("Observation\n（工具结果）", 470, 260, 160, 70,
                 style=node_style(fill=PALETTE["light_purple"]))
    decide = b.node("证据是否充分？", 240, 260, 160, 70,
                    style=node_style(fill=PALETTE["highlight"]))
    final = b.node("最终回答", 700, 180, 140, 70,
                   style=node_style(fill=PALETTE["accent"],
                                     text_color="#FFFFFF", font_style=1))

    e = edge_style()
    b.edge(user, think, "1. 接收任务", style=e)
    b.edge(think, action, "2. 选择工具", style=e)
    b.edge(action, obs, "3. 执行", style=e)
    b.edge(obs, decide, "4. 解析结果", style=e)
    b.edge(decide, think, "否：继续推理", style=edge_style(stroke=PALETTE["subtle"], dashed=True))
    b.edge(decide, final, "是：生成回答", style=edge_style(stroke=PALETTE["accent"]))

    b.write(CH2 / "fig2-2-react-loop.drawio")


# =============================================================================
# Ch3 框架设计（重点）
# =============================================================================

def fig3_1_mops_architecture():
    b = DrawioBuilder("Mops 五层架构", width=1100, height=700)

    # L1 用户交互
    l1 = b.lane("L1 用户交互层", 60, 50, 980, 70, fill=PALETTE["light_purple"])
    b.node("AIChatDrawer (SSE)", 80, 78, 180, 36,
           style=node_style(fill="#FFFFFF"), parent=l1)
    b.node("AgentTimeline", 280, 78, 160, 36,
           style=node_style(fill="#FFFFFF"), parent=l1)
    b.node("ConfirmActionCard", 460, 78, 180, 36,
           style=node_style(fill="#FFFFFF"), parent=l1)
    b.node("FeedbackBar", 660, 78, 140, 36,
           style=node_style(fill="#FFFFFF"), parent=l1)
    b.node("AdminAuditView", 820, 78, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=l1)

    # L2 任务编排
    l2 = b.lane("L2 任务编排层", 60, 140, 980, 80, fill=PALETTE["light_blue"])
    b.node("Coordinator\n(IntentRouter + 阶段决策)", 80, 170, 260, 46,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF",
                             font_style=1), parent=l2)
    b.node("MASState 共享状态对象", 360, 170, 260, 46,
           style=node_style(fill="#FFFFFF"), parent=l2)
    b.node("Token 预算 / Checkpoint 调度", 640, 170, 260, 46,
           style=node_style(fill="#FFFFFF"), parent=l2)
    b.node("LangGraph 编排引擎", 920, 170, 100, 46,
           style=node_style(fill="#FFFFFF", font_size=8), parent=l2)

    # L3 智能体角色
    l3 = b.lane("L3 智能体角色层", 60, 240, 980, 130, fill=PALETTE["light_green"])
    roles = [
        ("Planner", "PlanArtifact"),
        ("Explorer", "ObservationArtifact"),
        ("Executor", "ExecutionArtifact"),
        ("Verifier", "验证判断"),
    ]
    for i, (role, art) in enumerate(roles):
        x = 80 + i * 235
        b.node(role, x, 275, 90, 36,
               style=node_style(fill=PALETTE["ps"], text_color="#FFFFFF",
                                 font_style=1), parent=l3)
        b.node(art, x + 100, 275, 130, 36,
               style=node_style(fill="#FFFFFF", font_size=8), parent=l3)

    task_lane = b.lane("任务型智能体", 80, 320, 940, 42,
                       fill="#FFFFFF", stroke=PALETTE["muted"])
    for i, (name, _) in enumerate([
        ("Guide 帮助", PALETTE["ps"]),
        ("General 通用", PALETTE["ps"]),
        ("ApprovalAgent 审批", PALETTE["accent"]),
        ("Inspection 巡检", PALETTE["accent"]),
    ]):
        b.node(name, 100 + i * 230, 340, 200, 28,
               style=node_style(fill=PALETTE["bg_lane2"], font_size=9),
               parent=task_lane)

    # L4 工具与知识
    l4 = b.lane("L4 工具与知识层", 60, 390, 980, 90, fill=PALETTE["highlight"])
    b.node("工具声明注册表 (96 tools)", 80, 420, 240, 46,
           style=node_style(fill="#FFFFFF"), parent=l4)
    b.node("Go REST 执行器", 340, 420, 160, 46,
           style=node_style(fill="#FFFFFF"), parent=l4)
    b.node("Python 本地执行器", 520, 420, 180, 46,
           style=node_style(fill="#FFFFFF"), parent=l4)
    b.node("Mock 快照执行器", 720, 420, 160, 46,
           style=node_style(fill="#FFFFFF"), parent=l4)
    b.node("诊断技能库", 900, 420, 120, 46,
           style=node_style(fill="#FFFFFF"), parent=l4)

    # L5 平台基础设施
    l5 = b.lane("L5 平台基础设施层", 60, 500, 980, 80, fill="#FFD8C2")
    for i, name in enumerate(["Kubernetes API", "Volcano",
                              "Prometheus", "Harbor",
                              "PostgreSQL", "Ceph 存储", "审计日志"]):
        b.node(name, 80 + i * 135, 530, 125, 36,
               style=node_style(fill="#FFFFFF", font_size=9), parent=l5)

    b.text_label("→ 自上而下：抽象语义 ↓ ；自下而上：实时观测 ↑",
                 60, 595, 700, 20, font_size=9, color=PALETTE["subtle"])

    b.write(CH3 / "fig3-1-mops-architecture.drawio")


def fig3_2_coordinator_intent_router():
    b = DrawioBuilder("协调器与意图路由器内部结构", width=1100, height=620)

    entry = b.node("用户请求 / 续接事件", 60, 60, 200, 50,
                   style=node_style(fill=PALETTE["mops"],
                                     text_color="#FFFFFF", font_style=1))

    coord = b.lane("Coordinator (协调器)", 320, 40, 720, 480, fill=PALETTE["light_blue"])

    # Stage 0 IntentRouter
    ir = b.lane("IntentRouter (两级意图路由)", 340, 80, 680, 200,
                fill="#FFFFFF", stroke=PALETTE["mops"])
    b.node("Level 1 确定性启发式\n· 续接状态\n· 页面上下文\n· 简单帮助匹配",
           360, 120, 280, 130,
           style=node_style(fill=PALETTE["light_green"], font_size=9,
                             font_style=0), parent=ir)
    b.node("Level 2 LLM 分类\n· entry_mode / op_mode\n· requested_action\n· complexity / confidence",
           660, 120, 280, 130,
           style=node_style(fill=PALETTE["light_purple"], font_size=9), parent=ir)
    b.node("置信度合并 (公式 3.1)", 480, 250, 320, 26,
           style=node_style(fill=PALETTE["highlight"], font_size=8.5), parent=ir)

    # 阶段决策
    decide = b.lane("阶段决策 (Phase Decision)", 340, 300, 680, 200,
                    fill="#FFFFFF", stroke=PALETTE["accent"])
    b.node("快速路径\n首轮无计划 → plan\n恢复无动作 → finalize\n已知写操作 → execute",
           360, 340, 280, 130,
           style=node_style(fill=PALETTE["light_blue"], font_size=9), parent=decide)
    b.node("LLM 兜底\n阅读完整 MASState\n输出下一阶段\n受护栏配置约束",
           660, 340, 280, 130,
           style=node_style(fill=PALETTE["light_green"], font_size=9), parent=decide)

    # 出口
    out_help = b.node("Guide / General\n快速回答", 60, 220, 200, 60,
                      style=node_style(fill=PALETTE["ps"],
                                        text_color="#FFFFFF"))
    out_agent = b.node("Planner → Explorer\n→ Executor → Verifier",
                       60, 320, 200, 70,
                       style=node_style(fill=PALETTE["mops"],
                                         text_color="#FFFFFF"))
    out_resume = b.node("Checkpoint Resume", 60, 410, 200, 60,
                        style=node_style(fill=PALETTE["accent"],
                                          text_color="#FFFFFF"))

    b.edge(entry, ir, "进入")
    b.edge(ir, out_help, "entry_mode = help",
           style=edge_style(stroke=PALETTE["ps"]))
    b.edge(decide, out_agent, "phase ∈ {plan,explore,execute,verify}",
           style=edge_style(stroke=PALETTE["mops"]))
    b.edge(decide, out_resume, "resume",
           style=edge_style(stroke=PALETTE["accent"], dashed=True))

    b.text_label("护栏 (MASRuntimeConfig): lead_max_rounds=8,  subagent_max_iter=25,  no_progress=2,  tool_timeout=60s",
                 60, 540, 1000, 24, font_size=9, color=PALETTE["subtle"])

    b.write(CH3 / "fig3-2-coordinator-intent-router.drawio")


def fig3_3_multi_agent_roles():
    b = DrawioBuilder("多智能体角色协作架构", width=1100, height=620)

    coord = b.node("Coordinator\n(阶段决策 + 收束)", 470, 40, 200, 60,
                   style=node_style(fill=PALETTE["mops"],
                                     text_color="#FFFFFF", font_style=1))

    # 4 角色协作链
    planner = b.node("Planner\n规划候选工具\n输出 PlanArtifact",
                     60, 180, 200, 90,
                     style=node_style(fill=PALETTE["ps"], text_color="#FFFFFF"))
    explorer = b.node("Explorer\n迭代式证据收集\n输出 ObservationArtifact",
                      300, 180, 200, 90,
                      style=node_style(fill=PALETTE["ps"], text_color="#FFFFFF"))
    executor = b.node("Executor\n操作计划与执行\n生成确认请求",
                      540, 180, 200, 90,
                      style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"))
    verifier = b.node("Verifier\n挑战式验证\n输出 pass/risk/不足",
                      780, 180, 200, 90,
                      style=node_style(fill=PALETTE["ps"], text_color="#FFFFFF"))

    # 工具权限标注
    b.text_label("只读工具", 60, 280, 200, 16, font_size=8, color=PALETTE["mops"])
    b.text_label("只读工具", 300, 280, 200, 16, font_size=8, color=PALETTE["mops"])
    b.text_label("只读 + 写操作 + 确认", 540, 280, 200, 16, font_size=8,
                 color=PALETTE["accent"])
    b.text_label("只读工具", 780, 280, 200, 16, font_size=8, color=PALETTE["mops"])

    # 任务型智能体
    task_lane = b.lane("任务型智能体（非诊断路径）", 60, 340, 920, 130,
                       fill=PALETTE["light_blue"])
    items = [
        ("Guide", "用户帮助引导", "无工具，按角色生成结构化说明"),
        ("General", "通用问答", "无工具，处理问候/平台介绍"),
        ("ApprovalAgent", "锁定延期审批", "7 个只读工具白名单"),
        ("Inspection", "定时巡检", "计算→存储→网络分步骤收集"),
    ]
    for i, (name, role, detail) in enumerate(items):
        x = 80 + i * 225
        b.node(name, x, 380, 200, 26,
               style=node_style(fill=PALETTE["accent"],
                                 text_color="#FFFFFF", font_style=1),
               parent=task_lane)
        b.node(role, x, 408, 200, 22,
               style=node_style(fill="#FFFFFF", font_size=8.5), parent=task_lane)
        b.node(detail, x, 432, 200, 32,
               style=node_style(fill=PALETTE["bg_lane2"], font_size=8),
               parent=task_lane)

    # 数据流
    e_main = edge_style(stroke=PALETTE["mops"])
    e_ret = edge_style(stroke=PALETTE["subtle"], dashed=True)
    b.edge(coord, planner, "①任务目标", style=e_main)
    b.edge(planner, explorer, "②计划", style=e_main)
    b.edge(explorer, executor, "③证据", style=e_main)
    b.edge(executor, verifier, "④执行结果", style=e_main)
    b.edge(verifier, coord, "⑤验证报告", style=e_ret)

    b.edge(coord, task_lane, "快速路径", style=edge_style(stroke=PALETTE["accent"],
                                                          dashed=True))

    b.text_label("共享状态 MASState · StateView 投影 · 工作流 Checkpoint",
                 60, 490, 920, 22, font_size=9, color=PALETTE["subtle"])

    b.write(CH3 / "fig3-3-multi-agent-roles.drawio")


def fig3_4_mops_sequence():
    b = DrawioBuilder("Mops 协作时序图", width=1200, height=820)

    # 7 个 actor / lifeline
    actors = [
        ("用户", 60),
        ("Coordinator", 220),
        ("IntentRouter", 380),
        ("Planner", 540),
        ("Explorer", 700),
        ("Executor", 860),
        ("Verifier", 1020),
    ]
    lifelines = {}
    for name, x in actors:
        lid = b.node(name, x, 40, 130, 700, style=lifeline_style())
        lifelines[name] = lid

    # 消息序列 (y 增加)
    msgs = [
        ("用户", "Coordinator", "1. 提交请求 + 上下文", 80, False, False),
        ("Coordinator", "IntentRouter", "2. 路由分类", 130, False, False),
        ("IntentRouter", "Coordinator", "3. 返回 entry_mode/op_mode/action/conf", 170, True, False),
        ("Coordinator", "Planner", "4. 启动规划 (goal)", 220, False, False),
        ("Planner", "Coordinator", "5. PlanArtifact (候选工具 + 风险)", 280, True, False),
        ("Coordinator", "Explorer", "6. 启动证据收集 (plan)", 330, False, False),
        ("Explorer", "Explorer", "7. 迭代式调用只读工具 (max 25 iter)", 380, False, True),
        ("Explorer", "Coordinator", "8. ObservationArtifact", 440, True, False),
        ("Coordinator", "Executor", "9. 执行操作 (含写)", 490, False, False),
        ("Executor", "用户", "10. 高风险写 → 触发确认 (SSE)", 540, False, False),
        ("用户", "Executor", "11. 确认 / 拒绝 / 修改", 590, True, False),
        ("Executor", "Coordinator", "12. ExecutionArtifact", 630, True, False),
        ("Coordinator", "Verifier", "13. 验证 (plan, obs, exec, answer)", 670, False, False),
        ("Verifier", "Coordinator", "14. pass | risk | insufficient", 700, True, False),
    ]

    for src, dst, label, y, dashed, async_msg in msgs:
        s_id = lifelines[src]
        d_id = lifelines[dst]
        style = message_style(dashed=dashed, async_msg=async_msg)
        b.edge(s_id, d_id, label, style=style)

    # 最终回复
    final = b.node("Coordinator → 用户：15. 最终回答 + 引用 + 审计 ID",
                   60, 760, 1090, 30,
                   style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF",
                                     font_size=9))

    b.write(CH3 / "fig3-4-mops-sequence.drawio")


def fig3_5_masstate_stateview():
    b = DrawioBuilder("MASState 上下文与 StateView 投影", width=1100, height=620)

    center = b.lane("MASState 共享状态对象", 320, 40, 460, 540,
                    fill=PALETTE["highlight"])
    fields = [
        ("goal", "任务目标 + 路由决策"),
        ("plan", "PlanArtifact (Planner)"),
        ("observation", "ObservationArtifact (Explorer)"),
        ("execution", "ExecutionArtifact (Executor)"),
        ("action_history", "历史操作 + 状态机轨迹"),
        ("tool_records", "结构化工具调用记录"),
        ("workflow", "Checkpoint 用于确认后恢复"),
        ("page_context", "前端页面/路由/实体绑定"),
        ("user", "user_id / role / account"),
    ]
    for i, (name, desc) in enumerate(fields):
        y = 80 + i * 56
        b.node(name, 340, y, 130, 44,
               style=node_style(fill=PALETTE["mops"],
                                 text_color="#FFFFFF", font_style=1), parent=center)
        b.node(desc, 480, y, 280, 44,
               style=node_style(fill="#FFFFFF", font_size=9), parent=center)

    # StateView 投影
    views = [
        ("Coordinator View", "全量可见", 60, 80, PALETTE["mops"]),
        ("Planner View", "goal · 已有观察 · 工具声明", 60, 200, PALETTE["ps"]),
        ("Explorer View", "goal · plan · 已收集证据", 60, 320, PALETTE["ps"]),
        ("Executor View", "goal · plan · obs · 写操作白名单", 60, 440, PALETTE["accent"]),
    ]
    for name, detail, x, y, color in views:
        b.node(name, x, y, 220, 36,
               style=node_style(fill=color, text_color="#FFFFFF", font_style=1))
        b.node(detail, x, y + 40, 220, 56,
               style=node_style(fill="#FFFFFF", font_size=9))

    views_right = [
        ("Verifier View", "plan · obs · exec · answer (只读)", 820, 80, PALETTE["ps"]),
        ("Guide View", "user.role + 平台能力清单", 820, 200, PALETTE["ps"]),
        ("ApprovalAgent View", "ticket · 申请上下文", 820, 320, PALETTE["accent"]),
        ("Audit View (后台)", "tool_records · run_event", 820, 440, PALETTE["mops"]),
    ]
    for name, detail, x, y, color in views_right:
        b.node(name, x, y, 220, 36,
               style=node_style(fill=color, text_color="#FFFFFF", font_style=1))
        b.node(detail, x, y + 40, 220, 56,
               style=node_style(fill="#FFFFFF", font_size=9))

    b.text_label("StateView 提供按角色裁剪的只读投影，避免无关字段噪声",
                 60, 560, 800, 22, font_size=9, color=PALETTE["subtle"])

    b.write(CH3 / "fig3-5-masstate-stateview.drawio")


def fig3_6_tool_decoupling():
    b = DrawioBuilder("工具声明 / 执行解耦架构", width=1100, height=560)

    # 顶层 智能体
    top = b.lane("智能体角色（消费者）", 60, 40, 980, 80, fill=PALETTE["light_purple"])
    for i, name in enumerate(["Coordinator", "Planner", "Explorer", "Executor", "Verifier",
                              "ApprovalAgent"]):
        b.node(name, 80 + i * 160, 72, 150, 38,
               style=node_style(fill="#FFFFFF"), parent=top)

    # 声明层
    decl = b.lane("工具声明层 (Declaration)", 60, 140, 980, 110,
                  fill=PALETTE["highlight"])
    fields = [
        ("name + 描述", PALETTE["bg_card"]),
        ("参数 JSON Schema", PALETTE["bg_card"]),
        ("返回值语义", PALETTE["bg_card"]),
        ("风险等级 / 确认需求", PALETTE["accent"]),
        ("角色权限 (allow_roles)", PALETTE["ps"]),
        ("管理员标记", PALETTE["danger"]),
    ]
    for i, (name, color) in enumerate(fields):
        text_color = "#FFFFFF" if color != PALETTE["bg_card"] else PALETTE["text"]
        b.node(name, 80 + i * 160, 175, 150, 50,
               style=node_style(fill=color, text_color=text_color, font_size=9),
               parent=decl)

    # 执行层
    exec_lane = b.lane("工具执行层 (Execution)", 60, 270, 980, 110,
                       fill=PALETTE["light_blue"])
    backends = [
        ("Go REST Executor", "统一 HTTP + JWT 鉴权"),
        ("Python 本地执行器", "kubectl / PromQL / Harbor SDK"),
        ("Mock 快照执行器", "预录制工具快照，离线评测"),
        ("复合路由执行器", "运行时按配置分发"),
    ]
    for i, (name, detail) in enumerate(backends):
        x = 80 + i * 240
        b.node(name, x, 300, 220, 36,
               style=node_style(fill=PALETTE["mops"],
                                 text_color="#FFFFFF", font_style=1),
               parent=exec_lane)
        b.node(detail, x, 338, 220, 32,
               style=node_style(fill="#FFFFFF", font_size=8.5),
               parent=exec_lane)

    # 底层 平台
    infra = b.lane("平台基础设施", 60, 400, 980, 80, fill="#FFD8C2")
    for i, name in enumerate(["Crater 后端 API", "Kubernetes",
                              "Prometheus", "Harbor", "PostgreSQL"]):
        b.node(name, 80 + i * 195, 432, 180, 36,
               style=node_style(fill="#FFFFFF", font_size=9), parent=infra)

    b.text_label("智能体仅依赖声明层；执行层多后端可替换 ⇒ 跨平台迁移仅替换执行器实现",
                 60, 495, 980, 22, font_size=9, color=PALETTE["subtle"])

    b.write(CH3 / "fig3-6-tool-decoupling.drawio")


def fig3_7_confirm_flow():
    b = DrawioBuilder("安全控制与确认流时序", width=1180, height=720)

    actors = [
        ("前端", 60),
        ("Go 后端", 230),
        ("Coordinator", 410),
        ("Executor", 590),
        ("ConfirmRegistry", 770),
        ("工具执行器", 960),
    ]
    lifelines = {}
    for name, x in actors:
        lid = b.node(name, x, 40, 130, 620, style=lifeline_style())
        lifelines[name] = lid

    msgs = [
        ("前端", "Go 后端", "1. POST /agent/chat", 80, False, False),
        ("Go 后端", "Coordinator", "2. 转发 (含 user_id, role, page_context)", 130, False, False),
        ("Coordinator", "Executor", "3. 触发写操作 (e.g. stop_job)", 190, False, False),
        ("Executor", "ConfirmRegistry", "4. 查询是否需要确认", 240, False, False),
        ("ConfirmRegistry", "Executor", "5. risk_level=high → 是", 280, True, False),
        ("Executor", "Coordinator", "6. 冻结 MASState 快照", 320, False, False),
        ("Coordinator", "Go 后端", "7. 发出 confirmation_required SSE", 370, False, False),
        ("Go 后端", "前端", "8. 渲染 ConfirmActionCard + 参数表单", 410, False, False),
        ("前端", "Go 后端", "9. 用户点击 确认 / 拒绝 / 修改", 470, False, False),
        ("Go 后端", "Coordinator", "10. /agent/chat/confirm", 510, False, False),
        ("Coordinator", "Executor", "11. 从快照恢复并执行 (或拒绝)", 550, False, False),
        ("Executor", "工具执行器", "12. 真实写操作", 590, False, False),
        ("工具执行器", "Go 后端", "13. 审计落库 tool_call + run_event", 630, True, False),
    ]
    for src, dst, label, y, dashed, async_msg in msgs:
        b.edge(lifelines[src], lifelines[dst], label,
               style=message_style(dashed=dashed, async_msg=async_msg))

    b.write(CH3 / "fig3-7-confirm-flow.drawio")


# =============================================================================
# Ch4 实现
# =============================================================================

def fig4_1_microservice_deploy():
    b = DrawioBuilder("MOps 微服务部署架构", width=1100, height=700)

    user = b.lane("API 层 (Gin Router)", 60, 40, 980, 90, fill=PALETTE["light_purple"])
    b.node("POST /agent/chat", 80, 78, 170, 38,
           style=node_style(fill=PALETTE["light_blue"]), parent=user)
    b.node("POST /agent/chat/confirm", 270, 78, 200, 38,
           style=node_style(fill=PALETTE["light_blue"]), parent=user)
    b.node("POST /agent/chat/resume", 490, 78, 200, 38,
           style=node_style(fill=PALETTE["light_blue"]), parent=user)
    b.node("GET /admin/agent/...", 710, 78, 180, 38,
           style=node_style(fill=PALETTE["light_green"]), parent=user)
    b.node("POST /pipeline/admin-ops-report", 910, 78, 110, 38,
           style=node_style(fill=PALETTE["light_green"], font_size=8), parent=user)

    biz = b.lane("应用逻辑", 60, 150, 980, 260, fill=PALETTE["light_blue"])

    go_app = b.lane("Go 后端 (Gin + GORM)", 80, 190, 460, 200,
                    fill="#FFFFFF", stroke=PALETTE["mops"])
    b.node("JWT 认证 / 权限", 100, 230, 200, 36,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"),
           parent=go_app)
    b.node("会话管理 + 上下文构造", 100, 274, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=go_app)
    b.node("工具分发代理", 100, 318, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=go_app)
    b.node("审计持久化", 320, 230, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=go_app)
    b.node("SSE 流转发", 320, 274, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=go_app)
    b.node("CronJobManager", 320, 318, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=go_app)

    py_app = b.lane("Python Agent 服务 (FastAPI + LangGraph)", 560, 190, 460, 200,
                    fill="#FFFFFF", stroke=PALETTE["accent"])
    b.node("Coordinator / IntentRouter", 580, 230, 200, 36,
           style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"),
           parent=py_app)
    b.node("多智能体角色协作", 580, 274, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=py_app)
    b.node("工具执行编排", 580, 318, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=py_app)
    b.node("ApprovalAgent / Inspection", 800, 230, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=py_app)
    b.node("LLM 网关 (DashScope / OpenAI)", 800, 274, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=py_app)
    b.node("Crater-Bench 评测", 800, 318, 200, 36,
           style=node_style(fill="#FFFFFF"), parent=py_app)

    infra = b.lane("共享基础设施", 60, 430, 980, 140, fill="#FFD8C2")
    items = [
        ("PostgreSQL\n会话 + 审计", PALETTE["mops"]),
        ("Redis\n短期缓存 / 锁", PALETTE["accent"]),
        ("RabbitMQ\nCelery 任务队列", PALETTE["warn"]),
        ("Prometheus + Grafana", PALETTE["ps"]),
        ("Ceph / 对象存储\n工具快照 / 模型权重", PALETTE["mops"]),
        ("Kubernetes API\nVolcano / Harbor", PALETTE["accent"]),
    ]
    for i, (text, color) in enumerate(items):
        b.node(text, 80 + i * 158, 470, 145, 80,
               style=node_style(fill=color, text_color="#FFFFFF",
                                 font_size=9), parent=infra)

    b.text_label("React 18 前端 ── HTTPS + JWT ──→ Go (Gin) ── HTTP/SSE ──→ Python (FastAPI) ── HTTP ──→ 共享基础设施",
                 60, 595, 980, 24, font_size=9, color=PALETTE["subtle"])

    b.write(CH4 / "fig4-1-microservice-deploy.drawio")


def fig4_2_er_diagram():
    b = DrawioBuilder("智能体数据库 E-R 图", width=1180, height=720)

    def entity(name: str, fields: list[tuple[str, str]],
               x: int, y: int, accent: str = PALETTE["mops"]) -> str:
        height = 30 + len(fields) * 22
        lane = b.lane(name, x, y, 220, height, fill="#FFFFFF", stroke=accent)
        # 表头使用 swimlane 自带标题
        for i, (col, kind) in enumerate(fields):
            color = PALETTE["accent"] if kind == "PK" else (
                PALETTE["ps"] if kind == "FK" else PALETTE["text"])
            symbol = "★" if kind == "PK" else ("◆" if kind == "FK" else "·")
            b.node(f"{symbol} {col}", x + 6, y + 30 + i * 22, 208, 20,
                   style=node_style(fill="#FFFFFF", font_size=8.5,
                                     text_color=color, rounded=False))
        return lane

    sessions = entity("agent_sessions", [
        ("session_id", "PK"),
        ("user_id", "FK"),
        ("account_id", "FK"),
        ("source", ""),
        ("orchestration_mode", ""),
        ("page_context (JSONB)", ""),
        ("created_at", ""),
    ], 60, 60)

    turns = entity("agent_turns", [
        ("turn_id", "PK"),
        ("session_id", "FK"),
        ("status", ""),
        ("user_message", ""),
        ("orchestration_mode", ""),
        ("created_at", ""),
        ("completed_at", ""),
    ], 320, 60)

    messages = entity("agent_messages", [
        ("message_id", "PK"),
        ("turn_id", "FK"),
        ("role", ""),
        ("content", ""),
        ("metadata (JSONB)", ""),
        ("created_at", ""),
    ], 580, 60)

    tool_calls = entity("agent_tool_calls", [
        ("tool_call_id", "PK"),
        ("turn_id", "FK"),
        ("tool_name", ""),
        ("arguments (JSONB)", ""),
        ("result (JSONB)", ""),
        ("status", ""),
        ("risk_level", ""),
        ("execution_backend", ""),
        ("created_at", ""),
    ], 60, 320, accent=PALETTE["accent"])

    approvals = entity("agent_approvals", [
        ("approval_id", "PK"),
        ("session_id", "FK"),
        ("turn_id", "FK"),
        ("tool_call_id", "FK"),
        ("status", ""),
        ("submitter_id", "FK"),
        ("handler_id", "FK"),
        ("handled_at", ""),
    ], 320, 320, accent=PALETTE["accent"])

    quality = entity("agent_quality_evals", [
        ("id", "PK"),
        ("session_id", "FK"),
        ("turn_id", "FK"),
        ("eval_scope", ""),
        ("eval_type", ""),
        ("score_overall", ""),
        ("score_dimensions (JSONB)", ""),
    ], 580, 320, accent=PALETTE["ps"])

    feedbacks = entity("agent_feedbacks", [
        ("id", "PK"),
        ("session_id", "FK"),
        ("target_type", ""),
        ("target_id", ""),
        ("rating", ""),
        ("tags (JSONB)", ""),
    ], 840, 60, accent=PALETTE["ps"])

    op_logs = entity("operation_logs", [
        ("id", "PK"),
        ("operator_id", "FK"),
        ("op_type", ""),
        ("target", ""),
        ("status", ""),
        ("details (JSONB)", ""),
    ], 840, 320, accent=PALETTE["danger"])

    e = edge_style(stroke=PALETTE["subtle"])
    b.edge(sessions, turns, "1 ─ N", style=e)
    b.edge(turns, messages, "1 ─ N", style=e)
    b.edge(turns, tool_calls, "1 ─ N", style=e)
    b.edge(tool_calls, approvals, "1 ─ 0..1", style=edge_style(stroke=PALETTE["accent"]))
    b.edge(turns, quality, "1 ─ 0..1", style=edge_style(stroke=PALETTE["ps"]))
    b.edge(sessions, feedbacks, "1 ─ N", style=e)
    b.edge(tool_calls, op_logs, "1 ─ 1", style=edge_style(stroke=PALETTE["danger"], dashed=True))

    b.text_label("★ 主键    ◆ 外键    · 普通字段", 60, 640, 600, 20, font_size=9, color=PALETTE["subtle"])

    b.write(CH4 / "fig4-2-er-diagram.drawio")


def fig4_3_context_assembly():
    b = DrawioBuilder("多源上下文组装数据流", width=1100, height=520)

    req = b.node("用户请求 (Web)", 40, 220, 160, 50,
                 style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"))

    auth = b.node("JWT 认证\n→ user_id / role", 240, 220, 160, 50,
                  style=node_style(fill=PALETTE["light_blue"]))

    sources_lane = b.lane("多源上下文加载", 440, 60, 380, 380, fill=PALETTE["highlight"])
    b.node("会话历史\nagent_sessions / messages", 460, 100, 340, 50,
           style=node_style(fill="#FFFFFF"), parent=sources_lane)
    b.node("页面上下文\nroute / jobName / nodeName", 460, 160, 340, 50,
           style=node_style(fill="#FFFFFF"), parent=sources_lane)
    b.node("用户配额 / 角色权限\n(从平台业务库)", 460, 220, 340, 50,
           style=node_style(fill="#FFFFFF"), parent=sources_lane)
    b.node("续接状态\nworkflow checkpoint", 460, 280, 340, 50,
           style=node_style(fill="#FFFFFF"), parent=sources_lane)
    b.node("平台能力清单\nfeatures / accelerators", 460, 340, 340, 50,
           style=node_style(fill="#FFFFFF"), parent=sources_lane)

    assemble = b.node("ChatRequest 组装\n(Go normalize.go)", 860, 220, 170, 70,
                      style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"))

    forward = b.node("→ Python Agent 服务\n(POST /agent/chat)", 860, 320, 170, 60,
                     style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"))

    b.edge(req, auth, "①")
    b.edge(auth, sources_lane, "②")
    b.edge(sources_lane, assemble, "③ 合并")
    b.edge(assemble, forward, "④ HTTP")

    b.write(CH4 / "fig4-3-context-assembly.drawio")


def fig4_4_sse_confirm_sequence():
    b = DrawioBuilder("SSE 协议 + 确认中断与恢复时序", width=1180, height=820)

    actors = [
        ("前端", 60),
        ("Go 后端", 250),
        ("Python Agent", 470),
        ("LLM 网关", 700),
        ("工具执行器", 920),
    ]
    lifelines = {}
    for name, x in actors:
        lid = b.node(name, x, 40, 130, 740, style=lifeline_style())
        lifelines[name] = lid

    msgs = [
        ("前端", "Go 后端", "1. POST /agent/chat", 80, False, False),
        ("Go 后端", "Python Agent", "2. 转发 + 多源上下文", 120, False, False),
        ("Python Agent", "LLM 网关", "3. LLM 调用 (Planner)", 170, False, False),
        ("LLM 网关", "Python Agent", "4. 流式返回", 215, True, False),
        ("Python Agent", "Go 后端", "5. SSE: agent.thinking", 255, False, True),
        ("Go 后端", "前端", "6. 透传 thinking", 295, False, True),
        ("Python Agent", "工具执行器", "7. 只读工具 (Explorer)", 340, False, False),
        ("工具执行器", "Python Agent", "8. 工具结果", 385, True, False),
        ("Python Agent", "Go 后端", "9. SSE: agent.tool_result", 425, False, True),
        ("Python Agent", "Go 后端",
         "10. SSE: agent.confirmation_required (Executor 触发)",
         475, False, False),
        ("Go 后端", "前端", "11. 渲染 ConfirmActionCard", 515, False, False),
        ("前端", "Go 后端", "12. POST /agent/chat/confirm (decision)", 555, False, False),
        ("Go 后端", "Python Agent", "13. 恢复 Checkpoint", 595, False, False),
        ("Python Agent", "工具执行器", "14. 真实写操作", 640, False, False),
        ("工具执行器", "Python Agent", "15. 结果", 680, True, False),
        ("Python Agent", "Go 后端", "16. SSE: agent.final + 引用", 720, False, False),
        ("Go 后端", "前端", "17. 渲染最终回答", 760, False, False),
    ]
    for src, dst, label, y, dashed, async_msg in msgs:
        b.edge(lifelines[src], lifelines[dst], label,
               style=message_style(dashed=dashed, async_msg=async_msg))

    b.write(CH4 / "fig4-4-sse-confirm-sequence.drawio")


def fig4_5_approval_sequence():
    b = DrawioBuilder("审批智能体异步触发时序", width=1180, height=600)

    actors = [
        ("调度器 (Volcano)", 60),
        ("Go 后端", 280),
        ("TicketManager", 500),
        ("ApprovalAgent", 720),
        ("LLM 网关", 920),
    ]
    lifelines = {}
    for name, x in actors:
        lid = b.node(name, x, 40, 130, 520, style=lifeline_style())
        lifelines[name] = lid

    msgs = [
        ("调度器 (Volcano)", "Go 后端", "1. 作业即将达到锁定上限事件", 80, False, False),
        ("Go 后端", "TicketManager", "2. 创建延期审批 ticket", 130, False, False),
        ("TicketManager", "ApprovalAgent", "3. enqueue (ticket_id)", 175, False, True),
        ("ApprovalAgent", "Go 后端", "4. 拉取 7 个只读工具的事实", 230, False, False),
        ("Go 后端", "ApprovalAgent", "5. 结构化证据 JSON", 280, True, False),
        ("ApprovalAgent", "LLM 网关", "6. LLM 评估 (结构化输出)", 325, False, False),
        ("LLM 网关", "ApprovalAgent", "7. verdict ∈ {approve, approve_emergency, escalate}",
         380, True, False),
        ("ApprovalAgent", "TicketManager", "8. 写入 verdict + 理由", 420, False, False),
        ("TicketManager", "Go 后端", "9. 通知用户 / 管理员 (SMTP)", 470, False, True),
    ]
    for src, dst, label, y, dashed, async_msg in msgs:
        b.edge(lifelines[src], lifelines[dst], label,
               style=message_style(dashed=dashed, async_msg=async_msg))

    b.text_label("失败降级链：总结提取 → JSON 修复 → 关键字回退 → 默认 escalate",
                 60, 565, 1000, 22, font_size=9, color=PALETTE["accent"])

    b.write(CH4 / "fig4-5-approval-sequence.drawio")


def fig4_6_inspection_pipeline():
    b = DrawioBuilder("巡检流水线定时任务架构", width=1100, height=520)

    api = b.lane("API 层", 60, 40, 980, 70, fill=PALETTE["light_purple"])
    b.node("POST /pipeline/admin-ops-report", 100, 70, 280, 36,
           style=node_style(fill="#FFFFFF"), parent=api)
    b.node("GET /admin/agent/ops-report/latest", 420, 70, 320, 36,
           style=node_style(fill="#FFFFFF"), parent=api)
    b.node("GET /admin/agent/ops-report/audit", 780, 70, 240, 36,
           style=node_style(fill="#FFFFFF"), parent=api)

    logic = b.lane("应用逻辑", 60, 130, 980, 230, fill=PALETTE["light_blue"])
    b.node("Go CronJobManager\n(Beat-like 触发)", 80, 170, 200, 60,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"),
           parent=logic)
    inspect = b.lane("Inspection Pipeline (Python)", 300, 160, 720, 200,
                     fill="#FFFFFF", stroke=PALETTE["accent"])
    b.node("计算域指标采集\nGPU 利用 / 显存 / 队列",
           320, 200, 220, 50,
           style=node_style(fill=PALETTE["light_green"], font_size=9),
           parent=inspect)
    b.node("存储域指标采集\nPVC 使用 / IO 异常",
           560, 200, 220, 50,
           style=node_style(fill=PALETTE["light_green"], font_size=9),
           parent=inspect)
    b.node("网络域指标采集\nNCCL / RDMA / IB 重传",
           800, 200, 200, 50,
           style=node_style(fill=PALETTE["light_green"], font_size=9),
           parent=inspect)
    b.node("LLM 分析 + 结构化巡检报告\nexecutive_summary · recommendations · ops_audit_items",
           320, 270, 680, 70,
           style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"),
           parent=inspect)

    infra = b.lane("共享基础设施", 60, 380, 980, 100, fill="#FFD8C2")
    b.node("Prometheus", 80, 410, 180, 50,
           style=node_style(fill="#FFFFFF"), parent=infra)
    b.node("Kubernetes API", 280, 410, 180, 50,
           style=node_style(fill="#FFFFFF"), parent=infra)
    b.node("PostgreSQL\n(报告 + 审计项)", 480, 410, 200, 50,
           style=node_style(fill="#FFFFFF"), parent=infra)
    b.node("SMTP\n通知（去重 + 频控）", 700, 410, 180, 50,
           style=node_style(fill="#FFFFFF"), parent=infra)
    b.node("管理员仪表板", 900, 410, 120, 50,
           style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"), parent=infra)

    b.write(CH4 / "fig4-6-inspection-pipeline.drawio")


def fig4_7_tool_backend_routing():
    b = DrawioBuilder("工具执行后端复合路由", width=1100, height=520)

    agent = b.node("Executor 发起工具调用\n(name, arguments)", 60, 200, 220, 80,
                   style=node_style(fill=PALETTE["mops"], text_color="#FFFFFF"))

    router = b.node("CompositeToolExecutor\n按配置 + 角色 + 风险动态路由",
                    330, 200, 240, 80,
                    style=node_style(fill=PALETTE["accent"], text_color="#FFFFFF"))

    branches = [
        ("Go REST 执行器\nHTTP /agent/tools/exec", PALETTE["light_blue"],   620, 60),
        ("Python 本地执行器\nkubectl · PromQL · Harbor", PALETTE["light_green"], 620, 170),
        ("Mock 快照执行器\n预录制结果",                  PALETTE["light_purple"], 620, 280),
        ("诊断技能库\nLLM-as-tool",                     PALETTE["highlight"],  620, 390),
    ]
    for name, color, x, y in branches:
        nid = b.node(name, x, y, 240, 70,
                     style=node_style(fill=color, font_size=9))
        b.edge(router, nid, "")

    b.edge(agent, router, "Tool Spec")

    targets = [
        ("Crater 后端 API", 920, 60),
        ("kubectl / K8s API", 920, 170),
        ("Mock 数据集 (Crater-Bench)", 920, 280),
        ("LLM (DashScope / OpenAI)", 920, 390),
    ]
    for i, (name, x, y) in enumerate(targets):
        nid = b.node(name, x, y, 160, 50,
                     style=node_style(fill="#FFFFFF"))
        # 链接到对应分支
        # 因为我们没保存 branch id，索性用 text label 代替
    b.text_label("分发后端 → 真实平台 / 离线快照 / LLM",
                 600, 470, 500, 20, font_size=9, color=PALETTE["subtle"])

    b.write(CH4 / "fig4-7-tool-backend-routing.drawio")


def fig4_8_crater_bench_pipeline():
    b = DrawioBuilder("Crater-Bench 数据集生成流水线", width=1100, height=520)

    stages = [
        ("①真实历史会话\n(线上日志)",   PALETTE["light_purple"],  60),
        ("②场景抽取与归类\n四类任务",     PALETTE["light_blue"],   240),
        ("③工具调用录制\ntool snapshot", PALETTE["light_green"],  420),
        ("④标准答案标注\nmust/optional", PALETTE["highlight"],     600),
        ("⑤评分规则配置\n13 维 score profile", PALETTE["bg_lane2"], 780),
        ("⑥Mock 执行 + 验证",            PALETTE["accent"],        960),
    ]
    for text, color, x in stages:
        text_color = "#FFFFFF" if color in (PALETTE["accent"],) else PALETTE["text"]
        b.node(text, x, 220, 150, 80,
               style=node_style(fill=color, text_color=text_color,
                                 font_size=9, font_style=1))

    # 中间用箭头
    for i in range(len(stages) - 1):
        b.text_label("→", stages[i][2] + 152, 245, 20, 30,
                     font_size=18, color=PALETTE["subtle"])

    b.node("dataset/v1.0/scenarios/*.yaml",
           60, 360, 460, 50,
           style=node_style(fill=PALETTE["bg_card"]))
    b.node("dataset/v1.0/tool_snapshots/*.json",
           540, 360, 460, 50,
           style=node_style(fill=PALETTE["bg_card"]))

    b.text_label("场景数：66；任务类型：diagnosis (26) / ops (15) / query (14) / submission (11)",
                 60, 430, 1000, 22, font_size=9, color=PALETTE["subtle"])

    b.write(CH4 / "fig4-8-crater-bench-pipeline.drawio")


# =============================================================================
# main
# =============================================================================

def main():
    fig1_1_research_positioning()
    fig1_2_research_roadmap()
    fig1_3_five_layer_model()
    fig2_1_cloud_native_stack()
    fig2_2_react_loop()
    fig3_1_mops_architecture()
    fig3_2_coordinator_intent_router()
    fig3_3_multi_agent_roles()
    fig3_4_mops_sequence()
    fig3_5_masstate_stateview()
    fig3_6_tool_decoupling()
    fig3_7_confirm_flow()
    fig4_1_microservice_deploy()
    fig4_2_er_diagram()
    fig4_3_context_assembly()
    fig4_4_sse_confirm_sequence()
    fig4_5_approval_sequence()
    fig4_6_inspection_pipeline()
    fig4_7_tool_backend_routing()
    fig4_8_crater_bench_pipeline()
    print("all drawio sources generated.")


if __name__ == "__main__":
    main()
