"""Prompt templates for quality evaluation judges."""

# ---------------------------------------------------------------------------
# CHAT_JUDGE_PROMPT — for tongyi-xiaomi-analysis-flash
# Analyzes Chinese dialogue surface quality.
# ---------------------------------------------------------------------------

CHAT_JUDGE_SYSTEM = """你是一个专业的对话质量分析师。请根据以下评分标准对用户与AI助手的对话进行打分。

评分维度（每项1-5分）：
- intent_understanding（意图理解）：AI是否准确理解了用户的问题和意图
- completeness（回答完整性）：AI是否完整回答了用户的问题，没有遗漏关键信息
- satisfaction_pred（满意度预测）：综合判断，用户对这次对话可能的满意程度

输出格式（严格JSON，不要包含任何其他文字）：
{
  "intent_understanding": <1-5的整数>,
  "completeness": <1-5的整数>,
  "satisfaction_pred": <1-5的整数>,
  "reasoning": "<一两句简要说明评分依据>"
}
"""

CHAT_JUDGE_USER_TEMPLATE = """请分析以下对话质量：

{dialogue}

请按照系统提示中的格式输出JSON评分。"""


# ---------------------------------------------------------------------------
# CHAIN_JUDGE_PROMPT — for Qwen coordinator model
# Judges technical reasoning chain quality.
# ---------------------------------------------------------------------------

CHAIN_JUDGE_SYSTEM = """You are a technical quality reviewer for an AI agent system managing GPU compute clusters.
Evaluate the quality of the agent's reasoning and tool usage based on the following dimensions (1-5 scale each):

- tool_relevance: Were the tools selected appropriate and necessary for answering the user's query?
- diagnosis_accuracy: Is the root cause analysis or technical diagnosis technically correct and specific?
- suggestion_quality: Are the suggestions/recommendations specific, actionable, and technically sound?
- coherence: Is the reasoning chain logically coherent? Do the tool results support the conclusions?

Output strictly as JSON, no other text:
{
  "tool_relevance": <1-5 integer>,
  "diagnosis_accuracy": <1-5 integer>,
  "suggestion_quality": <1-5 integer>,
  "coherence": <1-5 integer>,
  "reasoning": "<one or two sentences explaining the scores>"
}
"""

CHAIN_JUDGE_USER_TEMPLATE = """Evaluate this agent interaction:

User query: {user_query}

Tool calls made:
{tool_calls_summary}

Agent's final response:
{final_response}

Output JSON scores as instructed."""
