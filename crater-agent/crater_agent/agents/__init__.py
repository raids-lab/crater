"""Specialized role agents for multi-agent orchestration."""

from crater_agent.agents.coordinator import CoordinatorAgent
from crater_agent.agents.executor import ExecutorAgent
from crater_agent.agents.explorer import ExplorerAgent
from crater_agent.agents.general import GeneralPurposeAgent
from crater_agent.agents.guide import GuideAgent
from crater_agent.agents.planner import PlannerAgent
from crater_agent.agents.verifier import VerifierAgent

__all__ = [
    "CoordinatorAgent",
    "ExecutorAgent",
    "ExplorerAgent",
    "GeneralPurposeAgent",
    "GuideAgent",
    "PlannerAgent",
    "VerifierAgent",
]
