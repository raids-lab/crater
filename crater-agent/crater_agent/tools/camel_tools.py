"""CAMEL toolkit adapters for Crater Agent.

web_search remains enabled as the explicit agent-side internet search path.

Async-safe wrappers around CAMEL's SearchToolkit.
Each public function returns {"status": ..., "result": ...} matching the
LocalToolExecutor contract.

Internal helpers are module-level so tests can patch them independently of the
CAMEL import.
"""

from __future__ import annotations

import asyncio
import json
from typing import Any


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
