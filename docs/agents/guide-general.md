# Guide & General Agents

> Lightweight agents for help-oriented and conversational interactions.

---

## Guide Agent

Handles "how-to" questions about the platform:
- "How do I submit a training job?"
- "What GPU types are available?"
- "Where can I see my quota?"

Responds with structured help: what to do, where to do it, what to watch out for. Role-aware — only shows features visible to the user's current role (user vs admin).

## General Agent

Handles greetings, casual Q&A, and requests that don't fit other categories:
- "Hi" / "What can you do?"
- Simple platform questions
- Clarification requests

Provides a friendly entry point. Avoids suggesting risky operations or making assumptions when context is insufficient.

---

Both agents are LLM-only (no tool access). They receive user context and capabilities summary but do not call any tools.

## Code

| Component | File |
|-----------|------|
| Guide agent | `crater_agent/agents/guide.py` |
| General agent | `crater_agent/agents/general.py` |
