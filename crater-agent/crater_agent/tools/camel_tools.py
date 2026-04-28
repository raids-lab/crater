"""CAMEL toolkit adapters for Crater Agent.

web_search remains enabled as the explicit agent-side internet search path.
execute_code stays disabled pending a real sandboxed replacement.

---

Async-safe wrappers around CAMEL's SearchToolkit and CodeExecutionToolkit.
Each public function returns {"status": ..., "result": ...} matching the
LocalToolExecutor contract.

Internal helpers (_ddg_search, _run_code_in_thread) are module-level so
tests can patch them independently of the CAMEL import.
"""

from __future__ import annotations

import asyncio
import json
import os
from typing import Any

_SUPPORTED_CODE_LANGUAGES = frozenset({"python"})


def _ddg_search(query: str, max_results: int) -> list[dict[str, Any]]:
    """Synchronous DuckDuckGo search via CAMEL SearchToolkit.

    Raises ImportError if camel-ai is not installed.

    Note: SearchToolkit.search_duckduckgo uses `number_of_result_pages` to
    control result volume; we pass max_results as a reasonable approximation.
    """
    from camel.toolkits import SearchToolkit  # noqa: PLC0415

    toolkit = SearchToolkit()
    raw = toolkit.search_duckduckgo(query=query, number_of_result_pages=max_results)
    # CAMEL returns list[dict] directly per its type annotation
    if isinstance(raw, list):
        return raw
    if isinstance(raw, str):
        try:
            parsed = json.loads(raw)
            if isinstance(parsed, list):
                return parsed
            raise TypeError(f"SearchToolkit returned JSON non-list: {type(parsed).__name__}")
        except (json.JSONDecodeError, ValueError):
            raise TypeError(f"SearchToolkit returned non-JSON string: {raw[:100]!r}")
    raise TypeError(f"Unexpected result type from SearchToolkit: {type(raw).__name__}")


def _run_code_in_thread(
    code: str,
    language: str,
    sandbox: str,
    timeout: int,
) -> str:
    """Synchronous code execution via CAMEL CodeExecutionToolkit.

    Raises ImportError if camel-ai is not installed.

    CodeExecutionToolkit.execute_code(code: str, code_type: str = 'python') -> str
    """
    from camel.toolkits import CodeExecutionToolkit  # noqa: PLC0415

    toolkit = CodeExecutionToolkit(sandbox=sandbox, timeout=timeout, verbose=False)
    result = toolkit.execute_code(code=code, code_type=language)
    return str(result) if result is not None else ""


async def camel_web_search(
    query: str,
    max_results: int = 10,
    timeout: int = 60,
) -> dict[str, Any]:
    """Search using CAMEL's SearchToolkit (DuckDuckGo, no API key required).

    Returns:
        {"status": "success", "result": {"query": ..., "results": [...], "source": "duckduckgo", "count": N}}
        {"status": "error", "error_type": "dependency_missing"|"search_error"|"timeout", "message": ...}
    """
    try:
        results = await asyncio.wait_for(
            asyncio.to_thread(_ddg_search, query, max_results),
            timeout=timeout,
        )
        return {
            "status": "success",
            "result": {
                "query": query,
                "results": results,
                "source": "duckduckgo",
                "count": len(results),
            },
        }
    except asyncio.TimeoutError:
        return {
            "status": "error",
            "error_type": "timeout",
            "message": f"DuckDuckGo search timed out after {timeout}s",
        }
    except ImportError as exc:
        return {
            "status": "error",
            "error_type": "dependency_missing",
            "message": f"camel-ai not installed: {exc}",
        }
    except Exception as exc:
        return {
            "status": "error",
            "error_type": "search_error",
            "message": str(exc),
        }


def _default_sandbox() -> str:
    """Read sandbox type from CRATER_AGENT_CODE_SANDBOX env var. Default: subprocess."""
    return os.getenv("CRATER_AGENT_CODE_SANDBOX", "subprocess").strip().lower() or "subprocess"


async def camel_execute_code(
    code: str,
    language: str = "python",
    sandbox: str | None = None,
    timeout: int = 30,
) -> dict[str, Any]:
    """Execute code using CAMEL's CodeExecutionToolkit.

    Returns:
        {"status": "success", "result": {"output": ..., "language": ..., "sandbox": ...}}
        {"status": "error", "error_type": "dependency_missing"|"execution_error", "message": ...}
    """
    if language not in _SUPPORTED_CODE_LANGUAGES:
        return {
            "status": "error",
            "error_type": "invalid_input",
            "message": f"Unsupported language: {language!r}. Supported: {sorted(_SUPPORTED_CODE_LANGUAGES)}",
        }
    resolved_sandbox = sandbox or _default_sandbox()
    try:
        output = await asyncio.wait_for(
            asyncio.to_thread(_run_code_in_thread, code, language, resolved_sandbox, timeout),
            timeout=timeout + 5,
        )
        return {
            "status": "success",
            "result": {
                "output": output,
                "language": language,
                "sandbox": resolved_sandbox,
            },
        }
    except asyncio.TimeoutError:
        return {
            "status": "error",
            "error_type": "execution_error",
            "message": f"Code execution timed out after {timeout + 5}s",
        }
    except ImportError as exc:
        return {
            "status": "error",
            "error_type": "dependency_missing",
            "message": f"camel-ai not installed: {exc}",
        }
    except Exception as exc:
        return {
            "status": "error",
            "error_type": "execution_error",
            "message": str(exc),
        }
