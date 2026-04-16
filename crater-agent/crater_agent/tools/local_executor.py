"""Agent-side local/core tool executor for portable read-only tools.

Goal:
- Keep a small set of platform-agnostic tools runnable inside crater-agent.
- Make them portable to other platforms/backends by avoiding backend-specific APIs.

Currently supported (minimum viable set):
- sandbox_grep: grep inside configured sandbox roots.
- web_search: DuckDuckGo search via CAMEL SearchToolkit.
- execute_code: run Python in CAMEL CodeExecutionToolkit sandbox.

Extended local/core tools (platform-agnostic):
- sandbox_list_dir: list directory entries inside sandbox roots.
- sandbox_read_file: read a text file inside sandbox roots with truncation.
- get_agent_runtime_summary: return agent + runtime config summaries.
"""

from __future__ import annotations

import asyncio
import json
import os
import re
import shutil
import time
from pathlib import Path
from typing import Any

import httpx

from crater_agent.runtime.platform import PlatformRuntimeConfig, load_platform_runtime_config
from crater_agent.tools.definitions import is_actor_allowed_for_tool, is_tool_allowed_for_role


def _is_relative_to(path: Path, root: Path) -> bool:
    try:
        path.relative_to(root)
        return True
    except ValueError:
        return False


def _extract_involved_object_name(field_selector: str) -> str | None:
    """Extract pod name from field_selector like 'involvedObject.name=sg-xxx-worker-0'."""
    for part in field_selector.split(","):
        part = part.strip()
        if part.startswith("involvedObject.name="):
            value = part[len("involvedObject.name="):].strip()
            return value or None
    return None


_SCOPED_K8S_TOOLS = frozenset({"k8s_get_events", "k8s_describe_resource", "k8s_get_pod_logs"})


class LocalToolExecutor:
    """Executes portable read-only tools in the Python agent process."""

    def __init__(
        self,
        *,
        runtime: PlatformRuntimeConfig | None = None,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self.runtime = runtime or load_platform_runtime_config()
        self.client = client or httpx.AsyncClient(follow_redirects=True)
        self._owns_client = client is None
        self._handlers = {
            "sandbox_grep": self._handle_sandbox_grep,
            "sandbox_list_dir": self._handle_sandbox_list_dir,
            "sandbox_read_file": self._handle_sandbox_read_file,
            "get_agent_runtime_summary": self._handle_get_agent_runtime_summary,
            "web_search": self._handle_web_search,
            "fetch_url": self._handle_fetch_url,
            "k8s_list_nodes": self._handle_k8s_list_nodes,
            "k8s_list_pods": self._handle_k8s_list_pods,
            "k8s_get_events": self._handle_k8s_get_events,
            "k8s_describe_resource": self._handle_k8s_describe_resource,
            "k8s_get_pod_logs": self._handle_k8s_get_pod_logs,
            "prometheus_query": self._handle_prometheus_query,
            "harbor_check": self._handle_harbor_check,
            "execute_code": self._handle_execute_code,
        }

    def supports(self, tool_name: str) -> bool:
        return tool_name in self._handlers

    @property
    def supported_tool_names(self) -> set[str]:
        return set(self._handlers.keys())

    async def execute(
        self,
        tool_name: str,
        tool_args: dict[str, Any],
        session_id: str,
        user_id: int,
        turn_id: str | None = None,
        tool_call_id: str | None = None,
        agent_id: str | None = None,
        agent_role: str | None = None,
        actor_role: str | None = None,
    ) -> dict[str, Any]:
        del session_id, turn_id, agent_id
        # user_id retained for ownership checks in scoped k8s handlers

        start_time = time.monotonic()
        normalized_role = (agent_role or "single_agent").strip().lower() or "single_agent"
        if not is_tool_allowed_for_role(normalized_role, tool_name):
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"Tool {tool_name} is not allowed for agent role {normalized_role}",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        if not is_actor_allowed_for_tool(actor_role, tool_name):
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"Tool {tool_name} requires admin privileges",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }

        handler = self._handlers.get(tool_name)
        if handler is None:
            return {
                "status": "error",
                "error_type": "not_local",
                "retryable": False,
                "message": f"Tool {tool_name} is not implemented in LocalToolExecutor",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }

        try:
            if tool_name in _SCOPED_K8S_TOOLS:
                result = await handler(tool_args, actor_role=actor_role or "user", user_id=user_id)
            else:
                result = await handler(tool_args)
        except PermissionError as exc:
            result = {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": str(exc),
            }
        except FileNotFoundError as exc:
            result = {
                "status": "error",
                "error_type": "not_found",
                "retryable": False,
                "message": str(exc),
            }
        except ValueError as exc:
            result = {
                "status": "error",
                "error_type": "invalid_input",
                "retryable": False,
                "message": str(exc),
            }
        except httpx.TimeoutException:
            result = {
                "status": "error",
                "error_type": "timeout",
                "retryable": True,
                "message": f"Tool {tool_name} 执行超时",
            }
        except httpx.RequestError as exc:
            result = {
                "status": "error",
                "error_type": "network",
                "retryable": True,
                "message": f"Tool {tool_name} 网络异常: {exc}",
            }
        except Exception as exc:
            result = {
                "status": "error",
                "error_type": "unexpected",
                "retryable": False,
                "message": f"Tool {tool_name} 执行异常: {exc}",
            }

        result.setdefault("status", "success")
        result["_latency_ms"] = int((time.monotonic() - start_time) * 1000)
        if tool_call_id:
            result.setdefault("tool_call_id", tool_call_id)
        return result

    async def close(self) -> None:
        if self._owns_client:
            await self.client.aclose()

    def _expand_sandbox_targets(self, raw_path: str) -> list[Path]:
        requested = str(raw_path or "").strip()
        if not self.runtime.sandbox_roots:
            raise PermissionError("sandbox_grep is unavailable: no sandbox roots configured")
        if requested in {"", ".", "*"}:
            return list(self.runtime.sandbox_roots)

        candidate = Path(requested).expanduser()
        if candidate.is_absolute():
            resolved = candidate.resolve()
            # Resolve roots too to avoid macOS /var -> /private/var mismatches.
            if not any(
                _is_relative_to(resolved, root.resolve()) for root in self.runtime.sandbox_roots
            ):
                raise PermissionError(f"path is outside sandbox roots: {requested}")
            return [resolved]

        matches: list[Path] = []
        for root in self.runtime.sandbox_roots:
            root_resolved = root.resolve()
            resolved = (root_resolved / candidate).resolve()
            if _is_relative_to(resolved, root_resolved) and resolved.exists():
                matches.append(resolved)
        if matches:
            return matches
        raise FileNotFoundError(f"sandbox path not found: {requested}")

    def _resolve_sandbox_path(self, raw_path: str) -> Path:
        requested = str(raw_path or "").strip()
        if not requested:
            raise ValueError("path is required")
        targets = self._expand_sandbox_targets(requested)
        if len(targets) != 1:
            # For list/read we require a concrete single target.
            raise ValueError("path must resolve to a single sandbox target (specify a concrete path)")
        return targets[0]

    async def _handle_sandbox_grep(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        pattern = str(tool_args.get("pattern") or "").strip()
        if not pattern:
            raise ValueError("sandbox_grep requires a non-empty pattern")

        regex = re.compile(
            pattern,
            re.IGNORECASE if bool(tool_args.get("ignore_case")) else 0,
        )
        max_matches = max(1, min(int(tool_args.get("max_matches") or 50), 200))
        targets = self._expand_sandbox_targets(str(tool_args.get("path") or ""))

        matches: list[dict[str, Any]] = []
        truncated = False
        for target in targets:
            files = (
                [target]
                if target.is_file()
                else [path for path in target.rglob("*") if path.is_file()]
            )
            for file_path in files:
                try:
                    if file_path.stat().st_size > self.runtime.sandbox_max_file_size_bytes:
                        continue
                    content = file_path.read_text(encoding="utf-8", errors="ignore")
                except OSError:
                    continue
                if "\x00" in content:
                    continue
                for line_no, line in enumerate(content.splitlines(), start=1):
                    match = regex.search(line)
                    if not match:
                        continue
                    matches.append(
                        {
                            "path": str(file_path),
                            "line": line_no,
                            "text": line.strip()[:300],
                            "match": match.group(0)[:120],
                        }
                    )
                    if len(matches) >= max_matches:
                        truncated = True
                        break
                if truncated:
                    break
            if truncated:
                break

        return {
            "status": "success",
            "result": {
                "pattern": pattern,
                "path": str(tool_args.get("path") or "."),
                "matches": matches,
                "total_matches": len(matches),
                "truncated": truncated,
                "searched_roots": [str(path) for path in targets],
            },
        }

    async def _handle_sandbox_list_dir(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        raw_path = str(tool_args.get("path") or "").strip()
        max_entries = max(1, min(int(tool_args.get("max_entries") or 200), 2000))

        # "" / "*" enumerates configured roots. "." means "default sandbox root"
        # when only one root is configured; otherwise we keep the root listing to
        # avoid ambiguous cross-root merges.
        if raw_path in {"", "*"} or (
            raw_path == "." and len(self.runtime.sandbox_roots) != 1
        ):
            roots = list(self.runtime.sandbox_roots)
            entries = [
                {"path": str(root), "name": root.name, "is_dir": True, "size": None}
                for root in roots[:max_entries]
            ]
            return {
                "status": "success",
                "result": {
                    "path": "<sandbox_roots>",
                    "count": len(entries),
                    "entries": entries,
                    "truncated": len(roots) > len(entries),
                },
            }

        target = (
            self.runtime.sandbox_roots[0].resolve()
            if raw_path == "."
            else self._resolve_sandbox_path(raw_path)
        )
        if not target.exists():
            raise FileNotFoundError(f"sandbox_list_dir: path not found: {raw_path}")
        if not target.is_dir():
            raise ValueError(f"sandbox_list_dir: not a directory: {raw_path}")

        glob = str(tool_args.get("glob") or "").strip() or None

        entries: list[dict[str, Any]] = []
        iterator = target.glob(glob) if glob else target.iterdir()
        for p in iterator:
            if len(entries) >= max_entries:
                break
            try:
                st = p.stat()
                entries.append(
                    {
                        "path": str(p),
                        "name": p.name,
                        "is_dir": p.is_dir(),
                        "size": int(st.st_size) if p.is_file() else None,
                    }
                )
            except OSError:
                entries.append({"path": str(p), "name": p.name, "is_dir": p.is_dir(), "size": None})

        return {
            "status": "success",
            "result": {
                "path": str(target),
                "count": len(entries),
                "entries": entries,
                "truncated": len(entries) >= max_entries,
            },
        }

    async def _handle_sandbox_read_file(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        raw_path = str(tool_args.get("path") or "").strip()
        target = self._resolve_sandbox_path(raw_path)
        if not target.exists() or not target.is_file():
            raise FileNotFoundError(f"sandbox_read_file: file not found: {raw_path}")

        try:
            size = int(target.stat().st_size)
        except OSError:
            size = -1

        if size >= 0 and size > self.runtime.sandbox_max_file_size_bytes:
            raise PermissionError(
                f"sandbox_read_file blocked: file too large ({size} bytes) > "
                f"{self.runtime.sandbox_max_file_size_bytes}"
            )

        max_bytes = max(1, min(int(tool_args.get("max_bytes") or 50000), 2_000_000))

        def _read_bytes() -> bytes:
            try:
                return target.read_bytes()[:max_bytes]
            except OSError:
                return b""

        data = await asyncio.to_thread(_read_bytes)
        content = data.decode("utf-8", errors="replace")
        return {
            "status": "success",
            "result": {
                "path": str(target),
                "max_bytes": max_bytes,
                "truncated": size >= 0 and size > max_bytes,
                "content": content,
            },
        }

    async def _handle_get_agent_runtime_summary(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        del tool_args
        from crater_agent.config import settings as _settings

        return {
            "status": "success",
            "result": {
                "agent": _settings.public_agent_config_summary(),
                "platformRuntime": self.runtime.to_summary(),
                "toolReadiness": self.runtime.tool_readiness(),
            },
        }

    async def _check_job_ownership(self, pod_name: str, user_id: int) -> bool:
        """Check if pod_name belongs to a job owned by user_id via Go backend.

        Returns False (deny) on any error — fail-closed.
        """
        from crater_agent.config import settings

        if not pod_name or not user_id:
            return False
        try:
            resp = await self.client.get(
                f"{settings.crater_backend_url}/api/agent/k8s-ownership",
                params={"pod_name": pod_name, "user_id": str(user_id)},
                headers={"X-Agent-Internal-Token": settings.crater_backend_internal_token},
                timeout=5.0,
            )
            if resp.status_code == 200:
                return bool(resp.json().get("allowed"))
            return False
        except Exception:
            return False

    def _kubectl_cmd(
        self,
        *,
        namespace: str | None = None,
        include_namespace: bool = True,
    ) -> list[str]:
        kubectl_bin = shutil.which(self.runtime.kubectl_bin) or self.runtime.kubectl_bin
        cmd = [kubectl_bin]
        if self.runtime.kubeconfig_path:
            cmd.extend(["--kubeconfig", self.runtime.kubeconfig_path])
        if self.runtime.kube_context:
            cmd.extend(["--context", self.runtime.kube_context])
        resolved_namespace = (
            str(namespace or "").strip()
            or self.runtime.kube_namespace
            or self.runtime.namespaces.get("job", "")
        )
        if include_namespace and resolved_namespace:
            cmd.extend(["-n", resolved_namespace])
        return cmd

    async def _run_subprocess(
        self,
        cmd: list[str],
        *,
        timeout_seconds: int | None = None,
    ) -> tuple[int, str, str]:
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        timeout = timeout_seconds or self.runtime.kubectl_timeout_seconds
        try:
            stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=timeout)
        except asyncio.TimeoutError:
            proc.kill()
            await proc.communicate()
            raise ValueError(f"command timed out after {timeout}s: {' '.join(cmd)}")
        return proc.returncode, stdout.decode("utf-8", errors="replace"), stderr.decode("utf-8", errors="replace")

    async def _run_kubectl_json(
        self,
        args: list[str],
        *,
        namespace: str | None = None,
        include_namespace: bool = True,
    ) -> dict[str, Any]:
        if not self.runtime.kubeconfig_path:
            raise PermissionError("no kubeconfig configured for direct Kubernetes tools")
        cmd = self._kubectl_cmd(namespace=namespace, include_namespace=include_namespace) + args + ["-o", "json"]
        code, stdout, stderr = await self._run_subprocess(cmd)
        if code != 0:
            raise ValueError(stderr.strip() or f"kubectl failed: {' '.join(cmd)}")
        try:
            loaded = json.loads(stdout or "{}")
        except json.JSONDecodeError as exc:
            raise ValueError(f"kubectl returned invalid JSON: {exc}") from exc
        return loaded if isinstance(loaded, dict) else {}

    async def _run_kubectl_text(
        self,
        args: list[str],
        *,
        namespace: str | None = None,
        include_namespace: bool = True,
        max_chars: int = 20_000,
    ) -> dict[str, Any]:
        if not self.runtime.kubeconfig_path:
            raise PermissionError("no kubeconfig configured for direct Kubernetes tools")
        cmd = self._kubectl_cmd(namespace=namespace, include_namespace=include_namespace) + args
        code, stdout, stderr = await self._run_subprocess(cmd)
        if code != 0:
            raise ValueError(stderr.strip() or f"kubectl failed: {' '.join(cmd)}")
        content = stdout.strip()
        return {
            "command": cmd,
            "content": content[:max_chars],
            "truncated": len(content) > max_chars,
        }

    @staticmethod
    def _node_ready_status(item: dict[str, Any]) -> str:
        for condition in (item.get("status") or {}).get("conditions") or []:
            if condition.get("type") == "Ready":
                return str(condition.get("status") or "Unknown")
        return "Unknown"

    async def _handle_k8s_list_nodes(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 100), 500))
        args = ["get", "nodes"]
        label_selector = str(tool_args.get("label_selector") or "").strip()
        field_selector = str(tool_args.get("field_selector") or "").strip()
        if label_selector:
            args.extend(["-l", label_selector])
        if field_selector:
            args.extend(["--field-selector", field_selector])

        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = list(payload.get("items") or [])
        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}
            status = item.get("status") or {}
            labels = metadata.get("labels") or {}
            roles = [
                key.removeprefix("node-role.kubernetes.io/")
                for key in labels.keys()
                if key.startswith("node-role.kubernetes.io/")
            ]
            addresses = status.get("addresses") or []
            internal_ip = next(
                (addr.get("address") for addr in addresses if addr.get("type") == "InternalIP"),
                "",
            )
            results.append(
                {
                    "name": str(metadata.get("name") or ""),
                    "ready": self._node_ready_status(item),
                    "unschedulable": bool(spec.get("unschedulable", False)),
                    "roles": roles,
                    "internal_ip": internal_ip,
                    "kubelet_version": str((status.get("nodeInfo") or {}).get("kubeletVersion") or ""),
                }
            )

        return {
            "status": "success",
            "result": {
                "count": len(results),
                "nodes": results,
                "kubeconfigPath": self.runtime.kubeconfig_path,
                "context": self.runtime.kube_context,
            },
        }

    async def _handle_k8s_list_pods(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 200), 1000))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        args = ["get", "pods"]
        include_namespace = True
        if namespace:
            include_namespace = True
        else:
            args.append("--all-namespaces")
            include_namespace = False
        label_selector = str(tool_args.get("label_selector") or "").strip()
        field_selector = str(tool_args.get("field_selector") or "").strip()
        node_name = str(tool_args.get("node_name") or "").strip()
        selectors: list[str] = []
        if field_selector:
            selectors.append(field_selector)
        if node_name:
            selectors.append(f"spec.nodeName={node_name}")
        if label_selector:
            args.extend(["-l", label_selector])
        if selectors:
            args.extend(["--field-selector", ",".join(selectors)])

        payload = await self._run_kubectl_json(args, namespace=namespace, include_namespace=include_namespace)
        items = list(payload.get("items") or [])
        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}
            status = item.get("status") or {}
            statuses = status.get("containerStatuses") or []
            restart_count = sum(int(cs.get("restartCount") or 0) for cs in statuses)
            ready_count = sum(1 for cs in statuses if cs.get("ready"))
            results.append(
                {
                    "namespace": str(metadata.get("namespace") or ""),
                    "name": str(metadata.get("name") or ""),
                    "phase": str(status.get("phase") or ""),
                    "node_name": str(spec.get("nodeName") or ""),
                    "pod_ip": str(status.get("podIP") or ""),
                    "ready_containers": ready_count,
                    "containers": len(statuses),
                    "restart_count": restart_count,
                    "start_time": str(status.get("startTime") or ""),
                }
            )

        return {
            "status": "success",
            "result": {
                "count": len(results),
                "pods": results,
                "namespace": namespace or "*",
            },
        }

    async def _handle_k8s_get_events(
        self,
        tool_args: dict[str, Any],
        *,
        actor_role: str = "admin",
        user_id: int = 0,
    ) -> dict[str, Any]:
        # --- ownership check for non-admin ---
        if actor_role not in ("admin", "platform_admin", "system_admin"):
            field_selector = str(tool_args.get("field_selector") or "").strip()
            pod_name = _extract_involved_object_name(field_selector)
            if not pod_name:
                return {
                    "status": "error",
                    "error_type": "tool_policy",
                    "retryable": False,
                    "message": (
                        "k8s_get_events: non-admin users must specify "
                        "field_selector with involvedObject.name=<pod_name>"
                    ),
                }
            if not await self._check_job_ownership(pod_name, user_id):
                return {
                    "status": "error",
                    "error_type": "tool_policy",
                    "retryable": False,
                    "message": f"k8s_get_events: pod {pod_name!r} does not belong to current user",
                }
        # --- original logic below (unchanged) ---
        limit = max(1, min(int(tool_args.get("limit") or 100), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        args = ["get", "events"]
        include_namespace = True
        if namespace:
            include_namespace = True
        else:
            args.append("--all-namespaces")
            include_namespace = False
        field_selector = str(tool_args.get("field_selector") or "").strip()
        if field_selector:
            args.extend(["--field-selector", field_selector])

        payload = await self._run_kubectl_json(args, namespace=namespace, include_namespace=include_namespace)
        items = list(payload.get("items") or [])
        items.sort(
            key=lambda item: str(
                (item.get("eventTime") or "")
                or ((item.get("series") or {}).get("lastObservedTime") or "")
                or (item.get("lastTimestamp") or "")
            ),
            reverse=True,
        )
        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            involved = item.get("involvedObject") or item.get("regarding") or {}
            results.append(
                {
                    "namespace": str(metadata.get("namespace") or ""),
                    "name": str(metadata.get("name") or ""),
                    "type": str(item.get("type") or ""),
                    "reason": str(item.get("reason") or ""),
                    "message": str(item.get("message") or "")[:500],
                    "regarding_kind": str(involved.get("kind") or ""),
                    "regarding_name": str(involved.get("name") or ""),
                    "timestamp": str(
                        item.get("eventTime")
                        or ((item.get("series") or {}).get("lastObservedTime") or "")
                        or item.get("lastTimestamp")
                        or ""
                    ),
                }
            )

        return {
            "status": "success",
            "result": {
                "count": len(results),
                "events": results,
                "namespace": namespace or "*",
            },
        }

    async def _handle_k8s_describe_resource(
        self,
        tool_args: dict[str, Any],
        *,
        actor_role: str = "admin",
        user_id: int = 0,
    ) -> dict[str, Any]:
        kind = str(tool_args.get("kind") or "").strip()
        name = str(tool_args.get("name") or "").strip()
        # --- ownership check for non-admin ---
        if actor_role not in ("admin", "platform_admin", "system_admin"):
            if kind.lower() not in ("pod", "vcjob"):
                return {
                    "status": "error",
                    "error_type": "tool_policy",
                    "retryable": False,
                    "message": "k8s_describe_resource: non-admin users can only describe Pod or VCJob resources",
                }
            if not await self._check_job_ownership(name, user_id):
                return {
                    "status": "error",
                    "error_type": "tool_policy",
                    "retryable": False,
                    "message": f"k8s_describe_resource: resource {name!r} does not belong to current user",
                }
        # --- original logic below (unchanged) ---
        namespace = str(tool_args.get("namespace") or "").strip() or None
        if not kind or not name:
            raise ValueError("k8s_describe_resource requires kind and name")

        include_namespace = kind.lower() not in {"node", "nodes", "namespace", "namespaces", "persistentvolume", "pv"}
        text_result = await self._run_kubectl_text(
            ["describe", kind, name],
            namespace=namespace,
            include_namespace=include_namespace,
        )
        return {
            "status": "success",
            "result": {
                "kind": kind,
                "name": name,
                "namespace": namespace or "",
                "command": text_result["command"],
                "content": text_result["content"],
                "truncated": text_result["truncated"],
            },
        }

    async def _handle_k8s_get_pod_logs(
        self,
        tool_args: dict[str, Any],
        *,
        actor_role: str = "admin",
        user_id: int = 0,
    ) -> dict[str, Any]:
        pod_name = str(tool_args.get("pod_name") or "").strip()
        if not pod_name:
            return {
                "status": "error",
                "error_type": "validation_error",
                "retryable": False,
                "message": "k8s_get_pod_logs requires pod_name",
            }
        # --- ownership check for non-admin ---
        if actor_role not in ("admin", "platform_admin", "system_admin"):
            if not await self._check_job_ownership(pod_name, user_id):
                return {
                    "status": "error",
                    "error_type": "tool_policy",
                    "retryable": False,
                    "message": f"k8s_get_pod_logs: pod {pod_name!r} does not belong to current user",
                }
        # --- original logic below (unchanged) ---
        namespace = str(tool_args.get("namespace") or "").strip() or None
        tail = max(1, min(int(tool_args.get("tail") or 200), 5000))
        since_seconds_raw = tool_args.get("since_seconds")
        since_seconds = int(since_seconds_raw) if since_seconds_raw not in (None, "") else None
        previous = bool(tool_args.get("previous", False))
        max_chars = max(1_000, min(int(tool_args.get("max_chars") or 40_000), 200_000))

        cmd = self._kubectl_cmd(namespace=namespace, include_namespace=True) + ["logs", pod_name, "--tail", str(tail)]
        container = str(tool_args.get("container") or "").strip()
        if container:
            cmd.extend(["-c", container])
        if since_seconds and since_seconds > 0:
            cmd.extend(["--since", f"{since_seconds}s"])
        if previous:
            cmd.append("--previous")

        code, stdout, stderr = await self._run_subprocess(cmd)
        if code != 0:
            raise ValueError(stderr.strip() or f"kubectl logs failed: {' '.join(cmd)}")
        content = stdout.strip()
        return {
            "status": "success",
            "result": {
                "pod_name": pod_name,
                "namespace": namespace or self.runtime.kube_namespace or self.runtime.namespaces.get("job", ""),
                "container": container,
                "tail": tail,
                "since_seconds": since_seconds,
                "previous": previous,
                "command": cmd,
                "content": content[:max_chars],
                "truncated": len(content) > max_chars,
            },
        }

    @staticmethod
    def _trim_prometheus_series(
        result_type: str,
        result: Any,
        *,
        max_series: int,
        max_points_per_series: int,
    ) -> tuple[Any, int, bool]:
        if result_type not in {"vector", "matrix"} or not isinstance(result, list):
            return result, 0, False

        series_count = len(result)
        truncated = series_count > max_series
        trimmed: list[dict[str, Any]] = []
        for item in result[:max_series]:
            if not isinstance(item, dict):
                continue
            trimmed_item = {
                "metric": item.get("metric") if isinstance(item.get("metric"), dict) else {},
            }
            if result_type == "vector":
                trimmed_item["value"] = item.get("value")
            else:
                values = item.get("values") if isinstance(item.get("values"), list) else []
                trimmed_item["values"] = values[-max_points_per_series:]
                trimmed_item["points"] = len(values)
                trimmed_item["truncated_points"] = len(values) > max_points_per_series
            trimmed.append(trimmed_item)
        return trimmed, series_count, truncated

    async def _handle_prometheus_query(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        if not self.runtime.prometheus_api:
            raise PermissionError("prometheus_query is unavailable: no Prometheus API configured")

        query = str(tool_args.get("query") or "").strip()
        if not query:
            raise ValueError("prometheus_query requires query")

        query_type = str(tool_args.get("query_type") or "instant").strip().lower() or "instant"
        if query_type not in {"instant", "range"}:
            raise ValueError("query_type must be 'instant' or 'range'")

        timeout_seconds = int(
            tool_args.get("timeout_seconds") or self.runtime.prometheus_timeout_seconds or 15
        )
        max_series = max(1, min(int(tool_args.get("max_series") or 20), 100))
        max_points_per_series = max(1, min(int(tool_args.get("max_points_per_series") or 120), 1000))
        params: dict[str, Any] = {"query": query}
        if query_type == "instant":
            eval_time = str(tool_args.get("time") or "").strip()
            if eval_time:
                params["time"] = eval_time
        else:
            start = str(tool_args.get("start") or "").strip()
            end = str(tool_args.get("end") or "").strip()
            if not start or not end:
                raise ValueError("range query requires start and end")
            params["start"] = start
            params["end"] = end
            params["step"] = str(tool_args.get("step") or "60s").strip() or "60s"

        base_url = self.runtime.prometheus_api.rstrip("/")
        endpoint = "/api/v1/query" if query_type == "instant" else "/api/v1/query_range"
        request_headers = dict(self.runtime.prometheus_headers)

        async with httpx.AsyncClient(
            follow_redirects=True,
            verify=self.runtime.prometheus_verify_tls,
        ) as client:
            response = await client.get(
                f"{base_url}{endpoint}",
                params=params,
                headers=request_headers,
                timeout=timeout_seconds,
            )
            response.raise_for_status()
            payload = response.json()

        if str(payload.get("status") or "").lower() != "success":
            error_message = str(payload.get("error") or "prometheus query failed").strip()
            raise ValueError(error_message)

        data = payload.get("data") if isinstance(payload.get("data"), dict) else {}
        result_type = str(data.get("resultType") or "").strip()
        raw_result = data.get("result")
        trimmed_result, series_count, truncated_series = self._trim_prometheus_series(
            result_type,
            raw_result,
            max_series=max_series,
            max_points_per_series=max_points_per_series,
        )
        return {
            "status": "success",
            "result": {
                "query": query,
                "query_type": query_type,
                "endpoint": f"{base_url}{endpoint}",
                "params": params,
                "result_type": result_type,
                "series_count": series_count,
                "truncated_series": truncated_series,
                "max_series": max_series,
                "max_points_per_series": max_points_per_series,
                "data": trimmed_result,
                "warnings": payload.get("warnings") if isinstance(payload.get("warnings"), list) else [],
            },
        }

    def _resolve_registry_server(self, raw_server: str) -> tuple[str, str]:
        server = str(raw_server or "").strip() or self.runtime.registry_server
        if not server:
            raise PermissionError("harbor_check is unavailable: no registry server configured")
        if server.startswith(("http://", "https://")):
            normalized = server.rstrip("/")
        else:
            normalized = f"https://{server}".rstrip("/")
        return server, normalized

    def _build_registry_auth(self, tool_args: dict[str, Any]) -> tuple[str, str] | None:
        username = str(tool_args.get("username") or "").strip() or self.runtime.registry_username
        password = str(tool_args.get("password") or "").strip() or self.runtime.registry_password
        if username and password:
            return (username, password)
        return None

    @staticmethod
    def _normalize_image_reference(
        *,
        image: str,
        server: str,
    ) -> tuple[str, str]:
        normalized = image.strip()
        if not normalized:
            raise ValueError("image must be non-empty")
        server_prefixes = {server, f"https://{server}", f"http://{server}"}
        for prefix in sorted(server_prefixes, key=len, reverse=True):
            if normalized.startswith(prefix + "/"):
                normalized = normalized[len(prefix) + 1 :]
                break
        if "@" in normalized:
            repository, reference = normalized.split("@", 1)
            return repository.strip("/"), reference
        last_slash = normalized.rfind("/")
        last_colon = normalized.rfind(":")
        if last_colon > last_slash:
            repository = normalized[:last_colon]
            reference = normalized[last_colon + 1 :]
        else:
            repository = normalized
            reference = "latest"
        return repository.strip("/"), reference

    async def _handle_harbor_check(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        raw_server = str(tool_args.get("server") or "").strip()
        server, base_url = self._resolve_registry_server(raw_server)
        timeout_seconds = int(tool_args.get("timeout_seconds") or 10)
        auth = self._build_registry_auth(tool_args)
        verify_tls = self.runtime.registry_verify_tls

        image = str(tool_args.get("image") or "").strip()
        repository = str(tool_args.get("repository") or "").strip()
        reference = str(tool_args.get("reference") or "").strip() or "latest"
        if image:
            repository, reference = self._normalize_image_reference(image=image, server=server)

        async with httpx.AsyncClient(
            follow_redirects=True,
            verify=verify_tls,
            auth=auth,
        ) as client:
            v2_resp = await client.get(f"{base_url}/v2/", timeout=timeout_seconds)
            health_resp = await client.get(f"{base_url}/api/v2.0/health", timeout=timeout_seconds)

            manifest_result: dict[str, Any] | None = None
            if repository:
                manifest_url = f"{base_url}/v2/{repository}/manifests/{reference}"
                manifest_resp = await client.head(
                    manifest_url,
                    headers={
                        "Accept": ",".join(
                            [
                                "application/vnd.oci.image.manifest.v1+json",
                                "application/vnd.docker.distribution.manifest.v2+json",
                                "application/vnd.docker.distribution.manifest.list.v2+json",
                            ]
                        )
                    },
                    timeout=timeout_seconds,
                )
                manifest_result = {
                    "repository": repository,
                    "reference": reference,
                    "status_code": manifest_resp.status_code,
                    "exists": manifest_resp.status_code == 200,
                    "docker_content_digest": manifest_resp.headers.get("Docker-Content-Digest", ""),
                }

            health_payload: dict[str, Any] | str
            try:
                health_payload = health_resp.json()
            except Exception:
                health_payload = health_resp.text[:1000]

        return {
            "status": "success",
            "result": {
                "server": server,
                "base_url": base_url,
                "registry_v2": {
                    "status_code": v2_resp.status_code,
                    "reachable": v2_resp.status_code in {200, 401},
                },
                "harbor_health": {
                    "status_code": health_resp.status_code,
                    "ok": health_resp.status_code == 200,
                    "payload": health_payload,
                },
                "manifest": manifest_result,
            },
        }

    async def _handle_web_search(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        if not self.runtime.web_search_enabled:
            raise PermissionError("web_search is disabled by runtime config")

        query = str(tool_args.get("query") or "").strip()
        if not query:
            raise ValueError("web_search requires a non-empty query")

        limit = max(1, min(int(tool_args.get("limit") or 5), 20))
        timeout = max(10, min(int(tool_args.get("timeout_seconds") or 60), 120))

        from crater_agent.tools.camel_tools import camel_web_search  # noqa: PLC0415
        return await camel_web_search(query=query, max_results=limit, timeout=timeout)

    async def _handle_fetch_url(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        import re  # noqa: PLC0415

        if not self.runtime.web_search_enabled:
            raise PermissionError("fetch_url is disabled by runtime config")

        url = str(tool_args.get("url") or "").strip()
        if not url:
            raise ValueError("fetch_url requires a non-empty url")
        if not url.startswith(("https://", "http://")):
            raise ValueError("fetch_url requires a full URL starting with https:// or http://")
        if not self.runtime.is_allowed_web_url(url):
            raise PermissionError(f"fetch_url: domain not in allowedDomains: {url}")

        max_chars = max(200, min(int(tool_args.get("max_chars") or 4000), 16000))

        try:
            resp = await self.client.get(
                url,
                timeout=60,
                headers={"User-Agent": "Mozilla/5.0 (compatible; crater-agent/1.0)"},
            )
        except Exception as exc:
            return {"status": "error", "error_type": "network", "message": str(exc), "url": url}

        content_type = resp.headers.get("content-type", "")
        if "json" in content_type:
            try:
                return {"status": "success", "url": url, "content_type": "json", "content": resp.json()}
            except Exception:
                pass

        body = resp.text
        # Remove <script>, <style>, <nav>, <header>, <footer> blocks entirely
        for tag in ("script", "style", "nav", "header", "footer", "aside"):
            body = re.sub(rf"<{tag}[^>]*>.*?</{tag}>", " ", body, flags=re.DOTALL | re.IGNORECASE)
        # Strip remaining tags
        body = re.sub(r"<[^>]+>", " ", body)
        # Unescape HTML entities and collapse whitespace
        import html as _html  # noqa: PLC0415
        body = _html.unescape(body)
        body = " ".join(body.split())

        return {
            "status": "ok" if resp.status_code < 400 else "error",
            "url": url,
            "status_code": resp.status_code,
            "content_type": content_type,
            "content": body[:max_chars],
            "truncated": len(body) > max_chars,
        }


    async def _handle_execute_code(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        code = str(tool_args.get("code") or "").strip()
        if not code:
            raise ValueError("execute_code requires non-empty code")

        language = str(tool_args.get("language") or "python").strip().lower() or "python"
        if language not in {"python"}:
            raise ValueError(f"execute_code: unsupported language '{language}' (only 'python' supported)")

        timeout = max(5, min(int(tool_args.get("timeout") or 30), 120))

        from crater_agent.tools.camel_tools import camel_execute_code  # noqa: PLC0415
        return await camel_execute_code(code=code, language=language, timeout=timeout)
