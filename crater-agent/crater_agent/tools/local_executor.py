"""Agent-side local/core tool executor for portable read-only tools.

Goal:
- Keep a small set of platform-agnostic tools runnable inside crater-agent.
- Make them portable to other platforms/backends by avoiding backend-specific APIs.

Currently supported:
- web_search: DuckDuckGo search via CAMEL SearchToolkit.
- execute_code: run Python in CAMEL CodeExecutionToolkit sandbox.
- get_agent_runtime_summary: return agent + runtime config summaries.

Extended local/core tools (platform-agnostic):
- K8s read-only tools via kubectl subprocess.
- Prometheus/Harbor direct HTTP queries.
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
            "get_agent_runtime_summary": self._handle_get_agent_runtime_summary,
            # --- DEPRECATED: migrating to LLM-native built-in tools ---
            # "web_search": self._handle_web_search,
            # "fetch_url": self._handle_fetch_url,
            # "execute_code": self._handle_execute_code,
            # ---------------------------------------------------------
            "k8s_list_nodes": self._handle_k8s_list_nodes,
            "k8s_list_pods": self._handle_k8s_list_pods,
            "k8s_get_events": self._handle_k8s_get_events,
            "k8s_describe_resource": self._handle_k8s_describe_resource,
            "k8s_get_pod_logs": self._handle_k8s_get_pod_logs,
            "prometheus_query": self._handle_prometheus_query,
            "harbor_check": self._handle_harbor_check,
            # "execute_code": self._handle_execute_code,  # DEPRECATED — see above
            "k8s_get_service": self._handle_k8s_get_service,
            "k8s_get_endpoints": self._handle_k8s_get_endpoints,
            "k8s_get_ingress": self._handle_k8s_get_ingress,
            "get_volcano_queue_state": self._handle_get_volcano_queue_state,
            "k8s_get_configmap": self._handle_k8s_get_configmap,
            "k8s_get_networkpolicy": self._handle_k8s_get_networkpolicy,
            "aggregate_image_pull_errors": self._handle_aggregate_image_pull_errors,
            "detect_zombie_jobs": self._handle_detect_zombie_jobs,
            "get_ddp_rank_mapping": self._handle_get_ddp_rank_mapping,
            "get_node_kernel_diagnostics": self._handle_get_node_kernel_diagnostics,
            "get_rdma_interface_status": self._handle_get_rdma_interface_status,
            # B8: GPU / Distributed Training Diagnostics
            "get_node_gpu_info": self._handle_get_node_gpu_info,
            "get_nccl_env_config": self._handle_get_nccl_env_config,
            "check_node_nic_status": self._handle_check_node_nic_status,
            "detect_training_anomaly_patterns": self._handle_detect_training_anomaly_patterns,
            "get_distributed_job_overview": self._handle_get_distributed_job_overview,
            "get_node_accelerator_info": self._handle_get_node_accelerator_info,
            "k8s_top_nodes": self._handle_k8s_top_nodes,
            "k8s_top_pods": self._handle_k8s_top_pods,
            "k8s_rollout_status": self._handle_k8s_rollout_status,
            "k8s_scale_workload": self._handle_k8s_scale_workload,
            "k8s_label_node": self._handle_k8s_label_node,
            "k8s_taint_node": self._handle_k8s_taint_node,
            "cordon_node": self._handle_cordon_node,
            "uncordon_node": self._handle_uncordon_node,
            "drain_node": self._handle_drain_node,
            "delete_pod": self._handle_delete_pod,
            "restart_workload": self._handle_restart_workload,
            "execute_admin_command": self._handle_execute_admin_command,
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

    # --- DEPRECATED: web_search handler ---
    # Migrating to LLM-native enable_search. Handler kept for reference.
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

    # --- DEPRECATED: fetch_url handler ---
    # Migrating to LLM-native web_extractor / search_strategy=agent_max. Handler kept for reference.
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


    # --- DEPRECATED: execute_code handler ---
    # Subprocess sandbox has no real isolation. Migrating to LLM-native code_interpreter.
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

    # ------------------------------------------------------------------
    # P1: K8s Resource Query Tools
    # ------------------------------------------------------------------

    async def _handle_k8s_get_service(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "services"]
        if name:
            args.append(name)
        label_selector = str(tool_args.get("label_selector") or "").strip()
        field_selector = str(tool_args.get("field_selector") or "").strip()
        if label_selector:
            args.extend(["-l", label_selector])
        if field_selector:
            args.extend(["--field-selector", field_selector])

        payload = await self._run_kubectl_json(args, namespace=namespace)
        if name and payload.get("kind") == "Service":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}
            ports = [
                {
                    "name": str(p.get("name") or ""),
                    "port": p.get("port"),
                    "target_port": p.get("targetPort"),
                    "protocol": str(p.get("protocol") or "TCP"),
                }
                for p in (spec.get("ports") or [])
            ]
            results.append({
                "name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "type": str(spec.get("type") or "ClusterIP"),
                "cluster_ip": str(spec.get("clusterIP") or ""),
                "ports": ports,
                "selector": spec.get("selector") or {},
            })

        return {
            "status": "success",
            "result": {"count": len(results), "services": results},
        }

    async def _handle_k8s_get_endpoints(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "endpoints"]
        if name:
            args.append(name)

        payload = await self._run_kubectl_json(args, namespace=namespace)
        if name and payload.get("kind") == "Endpoints":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            subsets_raw = item.get("subsets") or []
            subsets = []
            for ss in subsets_raw:
                addresses = [
                    {
                        "ip": str(a.get("ip") or ""),
                        "node_name": str(a.get("nodeName") or ""),
                        "target_ref": str((a.get("targetRef") or {}).get("name") or ""),
                    }
                    for a in (ss.get("addresses") or [])
                ]
                not_ready = [
                    {
                        "ip": str(a.get("ip") or ""),
                        "node_name": str(a.get("nodeName") or ""),
                        "target_ref": str((a.get("targetRef") or {}).get("name") or ""),
                    }
                    for a in (ss.get("notReadyAddresses") or [])
                ]
                ports = [
                    {"port": p.get("port"), "protocol": str(p.get("protocol") or "TCP")}
                    for p in (ss.get("ports") or [])
                ]
                subsets.append({
                    "addresses": addresses,
                    "not_ready_addresses": not_ready,
                    "ports": ports,
                })
            results.append({
                "name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "subsets": subsets,
            })

        return {
            "status": "success",
            "result": {"count": len(results), "endpoints": results},
        }

    async def _handle_k8s_get_ingress(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "ingress"]
        if name:
            args.append(name)
        label_selector = str(tool_args.get("label_selector") or "").strip()
        if label_selector:
            args.extend(["-l", label_selector])

        payload = await self._run_kubectl_json(args, namespace=namespace)
        if name and payload.get("kind") == "Ingress":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}
            status = item.get("status") or {}
            rules = []
            for rule in (spec.get("rules") or []):
                paths = []
                for p in ((rule.get("http") or {}).get("paths") or []):
                    backend = p.get("backend") or {}
                    svc = backend.get("service") or {}
                    port = svc.get("port") or {}
                    paths.append({
                        "path": str(p.get("path") or "/"),
                        "path_type": str(p.get("pathType") or ""),
                        "service": str(svc.get("name") or ""),
                        "port": port.get("number") or port.get("name") or "",
                    })
                rules.append({
                    "host": str(rule.get("host") or ""),
                    "paths": paths,
                })
            tls = [
                {"hosts": t.get("hosts") or [], "secret_name": str(t.get("secretName") or "")}
                for t in (spec.get("tls") or [])
            ]
            lb_ingress = (status.get("loadBalancer") or {}).get("ingress") or []
            lb_ips = [str(lb.get("ip") or lb.get("hostname") or "") for lb in lb_ingress]
            results.append({
                "name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "ingress_class": str(spec.get("ingressClassName") or metadata.get("annotations", {}).get("kubernetes.io/ingress.class", "")),
                "rules": rules,
                "tls": tls,
                "load_balancer_ips": lb_ips,
            })

        return {
            "status": "success",
            "result": {"count": len(results), "ingresses": results},
        }

    async def _handle_get_volcano_queue_state(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "queue.scheduling.volcano.sh"]
        if name:
            args.append(name)

        payload = await self._run_kubectl_json(args, include_namespace=False)
        if name and payload.get("kind") == "Queue":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}
            status = item.get("status") or {}
            capability = spec.get("capability") or {}
            allocated = status.get("allocated") or {}
            results.append({
                "name": str(metadata.get("name") or ""),
                "state": str(status.get("state") or ""),
                "weight": spec.get("weight", 1),
                "capability": {
                    "cpu": str(capability.get("cpu") or ""),
                    "memory": str(capability.get("memory") or ""),
                    "gpu": str(capability.get("nvidia.com/gpu") or capability.get("gpu") or ""),
                },
                "allocated": {
                    "cpu": str(allocated.get("cpu") or ""),
                    "memory": str(allocated.get("memory") or ""),
                    "gpu": str(allocated.get("nvidia.com/gpu") or allocated.get("gpu") or ""),
                },
                "running": status.get("running", 0),
                "pending": status.get("pending", 0),
                "inqueue": status.get("inqueue", 0),
            })

        return {
            "status": "success",
            "result": {"count": len(results), "queues": results},
        }

    async def _handle_k8s_get_configmap(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "configmaps"]
        if name:
            args.append(name)
        label_selector = str(tool_args.get("label_selector") or "").strip()
        if label_selector:
            args.extend(["-l", label_selector])

        payload = await self._run_kubectl_json(args, namespace=namespace)
        if name and payload.get("kind") == "ConfigMap":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            data = item.get("data") or {}
            # Truncate large values to keep response manageable
            truncated_data = {}
            for k, v in data.items():
                val = str(v)
                truncated_data[k] = val[:2000] + "...(truncated)" if len(val) > 2000 else val
            results.append({
                "name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "keys": list(data.keys()),
                "data": truncated_data,
            })

        return {
            "status": "success",
            "result": {"count": len(results), "configmaps": results},
        }

    async def _handle_k8s_get_networkpolicy(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        name = str(tool_args.get("name") or "").strip()
        args = ["get", "networkpolicies"]
        if name:
            args.append(name)

        payload = await self._run_kubectl_json(args, namespace=namespace)
        if name and payload.get("kind") == "NetworkPolicy":
            items = [payload]
        else:
            items = list(payload.get("items") or [])

        results: list[dict[str, Any]] = []
        for item in items[:limit]:
            metadata = item.get("metadata") or {}
            spec = item.get("spec") or {}

            pod_selector = spec.get("podSelector") or {}
            policy_types = spec.get("policyTypes") or []

            ingress_rules = []
            for rule in (spec.get("ingress") or []):
                ingress_rules.append({
                    "from": rule.get("from") or [],
                    "ports": rule.get("ports") or [],
                })

            egress_rules = []
            for rule in (spec.get("egress") or []):
                egress_rules.append({
                    "to": rule.get("to") or [],
                    "ports": rule.get("ports") or [],
                })

            results.append({
                "name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "pod_selector": pod_selector,
                "policy_types": policy_types,
                "ingress_rules": ingress_rules,
                "egress_rules": egress_rules,
            })

        return {
            "status": "success",
            "result": {"count": len(results), "network_policies": results},
        }

    # ------------------------------------------------------------------
    # P2: Enrichment Tools
    # ------------------------------------------------------------------

    async def _handle_aggregate_image_pull_errors(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 100), 500))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        time_window_minutes = max(1, min(int(tool_args.get("time_window_minutes") or 60), 1440))

        args = ["get", "events"]
        include_ns = True
        if namespace:
            include_ns = True
        else:
            args.append("--all-namespaces")
            include_ns = False

        payload = await self._run_kubectl_json(args, namespace=namespace, include_namespace=include_ns)
        items = payload.get("items") or []

        pull_reasons = {"Failed", "ErrImagePull", "ImagePullBackOff", "BackOff", "InspectFailed"}
        from datetime import datetime, timezone, timedelta  # noqa: PLC0415
        cutoff = datetime.now(timezone.utc) - timedelta(minutes=time_window_minutes)

        aggregated: dict[str, dict[str, Any]] = {}
        for ev in items:
            reason = str(ev.get("reason") or "")
            if reason not in pull_reasons:
                continue
            message = str(ev.get("message") or "")
            if not any(kw in message.lower() for kw in ("image", "pull", "manifest", "registry", "unauthorized")):
                if reason not in ("ErrImagePull", "ImagePullBackOff"):
                    continue

            last_ts_raw = ev.get("lastTimestamp") or ev.get("eventTime") or ""
            if last_ts_raw:
                try:
                    last_ts = datetime.fromisoformat(str(last_ts_raw).replace("Z", "+00:00"))
                    if last_ts < cutoff:
                        continue
                except (ValueError, TypeError):
                    pass

            # Extract image from message
            image = ""
            for token in message.split():
                if "/" in token and (":" in token or "." in token):
                    image = token.strip('"').strip("'").rstrip(":")
                    break
            if not image:
                image = "(unknown)"

            involved = ev.get("involvedObject") or {}
            pod_name = str(involved.get("name") or "")

            if image not in aggregated:
                aggregated[image] = {
                    "image": image,
                    "error_count": 0,
                    "affected_pods": [],
                    "first_seen": str(ev.get("firstTimestamp") or last_ts_raw),
                    "last_seen": str(last_ts_raw),
                    "sample_message": message[:500],
                }
            entry = aggregated[image]
            entry["error_count"] += int(ev.get("count") or 1)
            if pod_name and pod_name not in entry["affected_pods"] and len(entry["affected_pods"]) < 20:
                entry["affected_pods"].append(pod_name)
            if str(last_ts_raw) > entry["last_seen"]:
                entry["last_seen"] = str(last_ts_raw)

        sorted_results = sorted(aggregated.values(), key=lambda x: x["error_count"], reverse=True)[:limit]
        return {
            "status": "success",
            "result": {
                "count": len(sorted_results),
                "time_window_minutes": time_window_minutes,
                "image_pull_errors": sorted_results,
            },
        }

    async def _handle_detect_zombie_jobs(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        threshold_hours = max(1, int(tool_args.get("running_hours_threshold") or 72))

        args = ["get", "pods", "--all-namespaces", "--field-selector", "status.phase=Running",
                "-l", "crater.raids.io/task-type"]
        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = payload.get("items") or []

        from datetime import datetime, timezone  # noqa: PLC0415
        now = datetime.now(timezone.utc)
        zombies: list[dict[str, Any]] = []

        for pod in items:
            metadata = pod.get("metadata") or {}
            spec = pod.get("spec") or {}
            status = pod.get("status") or {}
            start_raw = status.get("startTime")
            if not start_raw:
                continue
            try:
                start_time = datetime.fromisoformat(str(start_raw).replace("Z", "+00:00"))
            except (ValueError, TypeError):
                continue
            running_hours = (now - start_time).total_seconds() / 3600
            if running_hours < threshold_hours:
                continue

            labels = metadata.get("labels") or {}
            zombies.append({
                "pod_name": str(metadata.get("name") or ""),
                "namespace": str(metadata.get("namespace") or ""),
                "node": str(spec.get("nodeName") or ""),
                "running_hours": round(running_hours, 1),
                "job_name": str(labels.get("volcano.sh/job-name") or labels.get("crater.raids.io/job-name") or ""),
                "owner": str(labels.get("crater.raids.io/task-user") or ""),
                "job_type": str(labels.get("crater.raids.io/task-type") or ""),
                "start_time": str(start_raw),
            })

        zombies.sort(key=lambda x: x["running_hours"], reverse=True)
        return {
            "status": "success",
            "result": {
                "count": len(zombies[:limit]),
                "threshold_hours": threshold_hours,
                "zombie_candidates": zombies[:limit],
            },
        }

    async def _handle_get_ddp_rank_mapping(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        job_name = str(tool_args.get("job_name") or "").strip()
        if not job_name:
            raise ValueError("job_name is required")

        args = ["get", "pods", "--all-namespaces", "-l", f"volcano.sh/job-name={job_name}"]
        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = payload.get("items") or []

        ranks: list[dict[str, Any]] = []
        for pod in items:
            metadata = pod.get("metadata") or {}
            spec = pod.get("spec") or {}
            status = pod.get("status") or {}
            labels = metadata.get("labels") or {}
            pod_name = str(metadata.get("name") or "")
            task_spec = str(labels.get("volcano.sh/task-spec") or "")

            # Derive ordinal from pod name suffix: <job>-<task>-<ordinal>
            ordinal = 0
            parts = pod_name.rsplit("-", 1)
            if len(parts) == 2:
                try:
                    ordinal = int(parts[1])
                except ValueError:
                    pass

            container_statuses = status.get("containerStatuses") or []
            ready = all(cs.get("ready", False) for cs in container_statuses) if container_statuses else False

            ranks.append({
                "pod_name": pod_name,
                "task_spec": task_spec,
                "ordinal": ordinal,
                "node_name": str(spec.get("nodeName") or ""),
                "pod_ip": str(status.get("podIP") or ""),
                "host_ip": str(status.get("hostIP") or ""),
                "phase": str(status.get("phase") or ""),
                "ready": ready,
            })

        ranks.sort(key=lambda x: (x["task_spec"], x["ordinal"]))
        # Assign sequential rank
        for i, r in enumerate(ranks):
            r["rank"] = i

        return {
            "status": "success",
            "result": {
                "job_name": job_name,
                "total_ranks": len(ranks),
                "ranks": ranks,
            },
        }

    # ------------------------------------------------------------------
    # P2: Node-level Diagnostics (kubectl debug node)
    # ------------------------------------------------------------------

    async def _run_kubectl_debug_node(
        self,
        node_name: str,
        commands: str,
        *,
        image: str = "busybox:latest",
        timeout: int = 60,
    ) -> str:
        if not self.runtime.kubeconfig_path:
            raise PermissionError("no kubeconfig configured for kubectl debug")
        cmd = self._kubectl_cmd(include_namespace=False) + [
            "debug", f"node/{node_name}", "-it", "--image", image,
            "--", "nsenter", "-t", "1", "-m", "-u", "-i", "-n", "--",
            "sh", "-c", commands,
        ]
        code, stdout, stderr = await self._run_subprocess(cmd, timeout_seconds=timeout)
        if code != 0:
            msg = stderr.strip() or f"kubectl debug failed with code {code}"
            raise ValueError(f"kubectl debug node/{node_name} failed: {msg}")
        return stdout

    async def _handle_get_node_kernel_diagnostics(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")
        dmesg_lines = max(50, min(int(tool_args.get("dmesg_lines") or 200), 1000))
        check_d_state = tool_args.get("check_d_state", True)

        commands_parts = [
            f"echo '===KERNEL_VERSION===' && uname -r",
            f"echo '===LOAD_AVG===' && cat /proc/loadavg",
            f"echo '===DMESG===' && dmesg --time-format iso 2>/dev/null | tail -{dmesg_lines} | grep -iE 'gpu|nccl|rdma|error|warn|oom|hung|rcu|nvrm|xid|nv-|mlx|infiniband|fail|kill|panic' || echo '(no matches)'",
        ]
        if check_d_state:
            commands_parts.append(
                "echo '===D_STATE===' && ps aux 2>/dev/null | awk '$8 ~ /D/ {print}' | head -50 || echo '(none)'"
            )

        raw = await self._run_kubectl_debug_node(
            node_name,
            " && ".join(commands_parts),
            timeout=90,
        )

        sections: dict[str, str] = {}
        current_section = ""
        current_lines: list[str] = []
        for line in raw.splitlines():
            if line.startswith("===") and line.endswith("==="):
                if current_section:
                    sections[current_section] = "\n".join(current_lines)
                current_section = line.strip("=")
                current_lines = []
            else:
                current_lines.append(line)
        if current_section:
            sections[current_section] = "\n".join(current_lines)

        dmesg_lines_list = [l.strip() for l in sections.get("DMESG", "").splitlines() if l.strip() and l.strip() != "(no matches)"]
        d_state_lines = [l.strip() for l in sections.get("D_STATE", "").splitlines() if l.strip() and l.strip() != "(none)"]

        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "kernel_version": sections.get("KERNEL_VERSION", "").strip(),
                "load_avg": sections.get("LOAD_AVG", "").strip(),
                "dmesg_highlights": dmesg_lines_list,
                "dmesg_highlight_count": len(dmesg_lines_list),
                "d_state_processes": d_state_lines,
                "d_state_count": len(d_state_lines),
            },
        }

    async def _handle_get_rdma_interface_status(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")

        commands = (
            "echo '===IB_DEVICES===' && "
            "(ibstat 2>/dev/null || echo '(ibstat not available)') && "
            "echo '===RDMA_LINKS===' && "
            "(rdma link show 2>/dev/null || echo '(rdma tool not available)') && "
            "echo '===IB_PORT_STATE===' && "
            "(for f in /sys/class/infiniband/*/ports/*/state; do echo \"$f: $(cat $f 2>/dev/null)\"; done 2>/dev/null || echo '(no IB ports found)') && "
            "echo '===IB_LINK===' && "
            "(ip link show type infiniband 2>/dev/null || echo '(no IB interfaces)') && "
            "echo '===KERNEL_MODULES===' && "
            "(lsmod 2>/dev/null | grep -iE 'mlx|ib_|rdma|nv_peer|gpu' || echo '(no relevant modules)')"
        )

        raw = await self._run_kubectl_debug_node(
            node_name,
            commands,
            image="busybox:latest",
            timeout=90,
        )

        sections: dict[str, str] = {}
        current_section = ""
        current_lines: list[str] = []
        for line in raw.splitlines():
            if line.startswith("===") and line.endswith("==="):
                if current_section:
                    sections[current_section] = "\n".join(current_lines)
                current_section = line.strip("=")
                current_lines = []
            else:
                current_lines.append(line)
        if current_section:
            sections[current_section] = "\n".join(current_lines)

        def parse_section(key: str) -> list[str]:
            text = sections.get(key, "").strip()
            if not text or text.startswith("("):
                return []
            return [l.strip() for l in text.splitlines() if l.strip()]

        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "ib_devices": parse_section("IB_DEVICES"),
                "rdma_links": parse_section("RDMA_LINKS"),
                "ib_port_states": parse_section("IB_PORT_STATE"),
                "ib_interfaces": parse_section("IB_LINK"),
                "kernel_modules": parse_section("KERNEL_MODULES"),
            },
        }

    # ------------------------------------------------------------------
    # B8: GPU / Distributed Training Diagnostic Tools
    # ------------------------------------------------------------------

    async def _handle_get_node_gpu_info(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")

        commands = (
            "echo '===GPU_QUERY===' && "
            "(nvidia-smi --query-gpu=index,name,driver_version,pci.bus_id,memory.total,memory.used,"
            "memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw,power.limit,"
            "ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total,compute_mode"
            " --format=csv,noheader 2>/dev/null || echo '(nvidia-smi not available)') && "
            "echo '===CUDA_VERSION===' && "
            "(nvidia-smi 2>/dev/null | head -3 || echo '(unavailable)') && "
            "echo '===GPU_PROCS===' && "
            "(nvidia-smi --query-compute-apps=pid,name,used_memory --format=csv,noheader 2>/dev/null "
            "|| echo '(none)') && "
            "echo '===NVIDIA_MODULES===' && "
            "(lsmod 2>/dev/null | grep -i nvidia || echo '(no nvidia modules)')"
        )

        raw = await self._run_kubectl_debug_node(node_name, commands, timeout=90)

        sections: dict[str, str] = {}
        current_section = ""
        current_lines: list[str] = []
        for line in raw.splitlines():
            if line.startswith("===") and line.endswith("==="):
                if current_section:
                    sections[current_section] = "\n".join(current_lines)
                current_section = line.strip("=")
                current_lines = []
            else:
                current_lines.append(line)
        if current_section:
            sections[current_section] = "\n".join(current_lines)

        # Parse CSV GPU info
        gpus: list[dict[str, Any]] = []
        gpu_text = sections.get("GPU_QUERY", "").strip()
        if gpu_text and not gpu_text.startswith("("):
            for line in gpu_text.splitlines():
                parts = [p.strip() for p in line.split(",")]
                if len(parts) >= 15:
                    gpus.append({
                        "index": parts[0],
                        "name": parts[1],
                        "driver_version": parts[2],
                        "pci_bus_id": parts[3],
                        "memory_total": parts[4],
                        "memory_used": parts[5],
                        "memory_free": parts[6],
                        "gpu_utilization": parts[7],
                        "memory_utilization": parts[8],
                        "temperature_c": parts[9],
                        "power_draw": parts[10],
                        "power_limit": parts[11],
                        "ecc_corrected": parts[12],
                        "ecc_uncorrected": parts[13],
                        "compute_mode": parts[14],
                    })

        # Extract CUDA version from nvidia-smi header
        cuda_version = ""
        header_text = sections.get("CUDA_VERSION", "").strip()
        for line in header_text.splitlines():
            if "CUDA Version" in line:
                import re as _re  # noqa: PLC0415
                match = _re.search(r"CUDA Version:\s*([\d.]+)", line)
                if match:
                    cuda_version = match.group(1)
                break

        procs_text = sections.get("GPU_PROCS", "").strip()
        gpu_processes: list[str] = []
        if procs_text and not procs_text.startswith("("):
            gpu_processes = [l.strip() for l in procs_text.splitlines() if l.strip()]

        modules_text = sections.get("NVIDIA_MODULES", "").strip()
        nvidia_modules: list[str] = []
        if modules_text and not modules_text.startswith("("):
            nvidia_modules = [l.strip() for l in modules_text.splitlines() if l.strip()]

        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "gpu_count": len(gpus),
                "cuda_version": cuda_version,
                "driver_version": gpus[0]["driver_version"] if gpus else "",
                "gpus": gpus,
                "gpu_processes": gpu_processes[:50],
                "nvidia_modules": nvidia_modules,
            },
        }

    async def _handle_get_nccl_env_config(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        job_name = str(tool_args.get("job_name") or "").strip()
        if not job_name:
            raise ValueError("job_name is required")

        args = ["get", "pods", "--all-namespaces", "-l", f"volcano.sh/job-name={job_name}"]
        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = payload.get("items") or []

        _NCCL_PREFIXES = frozenset({
            "NCCL_", "MASTER_ADDR", "MASTER_PORT", "WORLD_SIZE", "RANK",
            "LOCAL_RANK", "LOCAL_WORLD_SIZE", "NODE_RANK", "GROUP_RANK",
            "ROLE_RANK", "CUDA_VISIBLE_DEVICES", "OMP_NUM_THREADS",
            "GLOO_", "MPI_",
        })

        def _is_distributed_env(name: str) -> bool:
            return any(name.startswith(prefix) for prefix in _NCCL_PREFIXES)

        ranks: list[dict[str, Any]] = []
        for pod in items:
            metadata = pod.get("metadata") or {}
            spec = pod.get("spec") or {}
            status = pod.get("status") or {}
            labels = metadata.get("labels") or {}
            pod_name = str(metadata.get("name") or "")

            env_vars: dict[str, str] = {}
            for container in spec.get("containers") or []:
                for env in container.get("env") or []:
                    name = str(env.get("name") or "")
                    value = str(env.get("value") or "")
                    if _is_distributed_env(name):
                        env_vars[name] = value

            # Derive ordinal
            ordinal = 0
            parts = pod_name.rsplit("-", 1)
            if len(parts) == 2:
                try:
                    ordinal = int(parts[1])
                except ValueError:
                    pass

            ranks.append({
                "pod_name": pod_name,
                "node_name": str(spec.get("nodeName") or ""),
                "pod_ip": str(status.get("podIP") or ""),
                "task_spec": str(labels.get("volcano.sh/task-spec") or ""),
                "ordinal": ordinal,
                "phase": str(status.get("phase") or ""),
                "env_vars": env_vars,
            })

        ranks.sort(key=lambda x: (x["task_spec"], x["ordinal"]))

        # Aggregate config summary
        all_env_keys: set[str] = set()
        for r in ranks:
            all_env_keys.update(r["env_vars"].keys())

        # Detect inconsistencies
        inconsistencies: list[str] = []
        for key in sorted(all_env_keys):
            if key in ("RANK", "LOCAL_RANK", "NODE_RANK", "GROUP_RANK", "ROLE_RANK",
                        "CUDA_VISIBLE_DEVICES", "MASTER_ADDR"):
                continue  # Expected to differ per rank
            values = {r["env_vars"].get(key, "(unset)") for r in ranks if key in r["env_vars"]}
            if len(values) > 1:
                inconsistencies.append(f"{key}: {sorted(values)}")

        return {
            "status": "success",
            "result": {
                "job_name": job_name,
                "total_ranks": len(ranks),
                "common_env_keys": sorted(all_env_keys),
                "inconsistencies": inconsistencies,
                "ranks": ranks,
            },
        }

    async def _handle_check_node_nic_status(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")
        include_virtual = bool(tool_args.get("include_virtual", False))

        commands = (
            "echo '===IP_LINK===' && "
            "ip -s -d link show 2>/dev/null && "
            "echo '===NET_STATS===' && "
            "for iface in /sys/class/net/*; do "
            "  name=$(basename $iface); "
            "  carrier=$(cat $iface/carrier 2>/dev/null || echo 0); "
            "  speed=$(cat $iface/speed 2>/dev/null || echo -1); "
            "  rx_errors=$(cat $iface/statistics/rx_errors 2>/dev/null || echo 0); "
            "  tx_errors=$(cat $iface/statistics/tx_errors 2>/dev/null || echo 0); "
            "  rx_dropped=$(cat $iface/statistics/rx_dropped 2>/dev/null || echo 0); "
            "  tx_dropped=$(cat $iface/statistics/tx_dropped 2>/dev/null || echo 0); "
            "  rx_bytes=$(cat $iface/statistics/rx_bytes 2>/dev/null || echo 0); "
            "  tx_bytes=$(cat $iface/statistics/tx_bytes 2>/dev/null || echo 0); "
            "  operstate=$(cat $iface/operstate 2>/dev/null || echo unknown); "
            "  echo \"$name|$carrier|$speed|$operstate|$rx_errors|$tx_errors|$rx_dropped|$tx_dropped|$rx_bytes|$tx_bytes\"; "
            "done && "
            "echo '===ETHTOOL===' && "
            "(for iface in $(ls /sys/class/net/ 2>/dev/null); do "
            "  echo \"--- $iface ---\"; "
            "  ethtool $iface 2>/dev/null | grep -iE 'speed|duplex|link detected|auto-negotiation' || echo '(ethtool unavailable)'; "
            "done)"
        )

        raw = await self._run_kubectl_debug_node(node_name, commands, timeout=90)

        sections: dict[str, str] = {}
        current_section = ""
        current_lines: list[str] = []
        for line in raw.splitlines():
            if line.startswith("===") and line.endswith("==="):
                if current_section:
                    sections[current_section] = "\n".join(current_lines)
                current_section = line.strip("=")
                current_lines = []
            else:
                current_lines.append(line)
        if current_section:
            sections[current_section] = "\n".join(current_lines)

        _VIRTUAL_PREFIXES = ("veth", "docker", "br-", "cni", "flannel", "calico", "tunl", "kube-")

        nics: list[dict[str, Any]] = []
        net_stats_text = sections.get("NET_STATS", "").strip()
        for line in net_stats_text.splitlines():
            parts = line.split("|")
            if len(parts) < 10:
                continue
            name = parts[0].strip()
            if name == "lo":
                continue
            if not include_virtual and any(name.startswith(p) for p in _VIRTUAL_PREFIXES):
                continue
            rx_errors = int(parts[4] or 0)
            tx_errors = int(parts[5] or 0)
            rx_dropped = int(parts[6] or 0)
            tx_dropped = int(parts[7] or 0)
            has_errors = (rx_errors + tx_errors + rx_dropped + tx_dropped) > 0
            nics.append({
                "name": name,
                "carrier": parts[1].strip() == "1",
                "speed_mbps": int(parts[2] or -1),
                "operstate": parts[3].strip(),
                "rx_errors": rx_errors,
                "tx_errors": tx_errors,
                "rx_dropped": rx_dropped,
                "tx_dropped": tx_dropped,
                "rx_bytes": int(parts[8] or 0),
                "tx_bytes": int(parts[9] or 0),
                "has_errors": has_errors,
            })

        # Parse ethtool
        ethtool_info: dict[str, list[str]] = {}
        current_iface = ""
        for line in sections.get("ETHTOOL", "").splitlines():
            if line.startswith("--- ") and line.endswith(" ---"):
                current_iface = line.strip("- ").strip()
            elif current_iface and line.strip() and not line.strip().startswith("("):
                ethtool_info.setdefault(current_iface, []).append(line.strip())

        # Flag issues
        issues: list[str] = []
        for nic in nics:
            if nic["operstate"] != "up" and nic["carrier"]:
                issues.append(f"{nic['name']}: operstate={nic['operstate']} but carrier=1")
            if not nic["carrier"] and nic["operstate"] == "up":
                issues.append(f"{nic['name']}: no carrier but operstate=up (cable/switch issue?)")
            if nic["has_errors"]:
                issues.append(
                    f"{nic['name']}: errors rx={nic['rx_errors']} tx={nic['tx_errors']} "
                    f"drops rx={nic['rx_dropped']} tx={nic['tx_dropped']}"
                )
            if 0 < nic["speed_mbps"] < 10000 and not any(nic["name"].startswith(p) for p in _VIRTUAL_PREFIXES):
                issues.append(f"{nic['name']}: low speed {nic['speed_mbps']}Mbps (expected >=10Gbps for GPU cluster)")

        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "nic_count": len(nics),
                "nics": nics,
                "ethtool": ethtool_info,
                "issues": issues,
                "issue_count": len(issues),
            },
        }

    _TRAINING_ANOMALY_PATTERNS: list[tuple[str, str, str]] = [
        # (category, severity, regex_pattern)
        # Loss anomalies
        ("loss_nan_inf", "critical", r"(?i)\b(nan|inf)\b.*\b(loss|diverge)|\b(loss|train).*\b(nan|inf)\b"),
        ("loss_explosion", "warning", r"(?i)loss\s*[:=]\s*[\d.]*e\+[3-9]\d*|loss\s*[:=]\s*\d{6,}"),
        # Gradient anomalies
        ("gradient_overflow", "warning", r"(?i)gradient.*overflow|grad.*norm.*inf|overflow.*detected"),
        ("gradient_underflow", "warning", r"(?i)gradient.*underflow|grad.*norm.*0\.0"),
        # Memory errors (CUDA + generic)
        ("cuda_oom", "critical", r"(?i)CUDA\s*out\s*of\s*memory|RuntimeError.*OOM|torch\.cuda\.OutOfMemoryError"),
        ("cpu_oom", "critical", r"(?i)Killed.*oom|oom-kill|out.of.memory.*killed|Cannot allocate memory"),
        # NCCL communication errors
        ("nccl_timeout", "critical", r"(?i)NCCL\s*(timeout|watchdog)|ProcessGroupNCCL.*timeout|Work.*timed out"),
        ("nccl_connection", "critical", r"(?i)NCCL.*connection\s*(refused|reset|closed)|ncclSystemError|ncclInternalError"),
        ("nccl_unhandled", "warning", r"(?i)NCCL.*unhandled\s*system\s*error|NCCL\s*WARN"),
        # CUDA runtime errors
        ("cuda_assert", "critical", r"(?i)device-side\s*assert|CUDA\s*error.*assert"),
        ("cuda_illegal_mem", "critical", r"(?i)illegal\s*memory\s*access|CUDA.*illegal.*address"),
        ("cuda_init_error", "critical", r"(?i)CUDA.*initialization|cudaErrorNoDevice|no\s*CUDA.*capable\s*device"),
        ("cuda_driver", "critical", r"(?i)CUDA\s*driver\s*version.*insufficient|driver.*mismatch"),
        # ---- Ascend NPU (华为昇腾 CANN) ----
        ("ascend_op_not_supported", "critical",
         r"(?i)op.*not\s*support.*on\s*(NPU|Ascend)|NotImplementedError.*acl|"
         r"RuntimeError.*CANN.*unsupported|AscendCL.*EZ\d+|not\s*supported.*ascend"),
        ("ascend_hccl_error", "critical",
         r"(?i)HCCL\s*(timeout|error|failed)|HcclCommInitRootInfo.*failed|"
         r"hccl.*connection.*refused|HCCL_WORLD_SIZE"),
        ("ascend_device_error", "critical",
         r"(?i)acl.*error|AscendCL.*Error|device\s*error.*Ascend|"
         r"NPU\s*error|davinci.*error|aicore.*error"),
        ("ascend_memory", "critical",
         r"(?i)NPU\s*out\s*of\s*memory|Ascend.*memory.*insufficient|"
         r"HBM.*out\s*of\s*memory|malloc.*device.*memory.*failed"),
        # ---- Hygon DCU / AMD ROCm ----
        ("rocm_hip_error", "critical",
         r"(?i)HIP\s*error|hipError|ROCm.*error|"
         r"hip.*Memory|hipErrorOutOfMemory|hipErrorNoBinaryForGpu"),
        ("rocm_op_not_supported", "critical",
         r"(?i)op.*not\s*support.*(ROCm|DCU|HIP)|"
         r"NotImplementedError.*(rocm|hip)|MIOpen.*unsupported"),
        ("rccl_error", "critical",
         r"(?i)RCCL\s*(timeout|error|failed)|rcclSystemError|"
         r"rcclInternalError|RCCL.*connection"),
        # ---- Generic accelerator errors ----
        ("device_not_available", "critical",
         r"(?i)no\s*(GPU|NPU|DCU|accelerator)\s*(device\s*)?available|"
         r"device\s*count.*0|RuntimeError.*Expected.*device"),
        ("op_not_implemented", "critical",
         r"(?i)NotImplementedError.*(?:backend|device|kernel)|"
         r"op.*not.*implemented.*for.*(?:device|backend)|"
         r"aten::.*not\s*implemented|could\s*not\s*run.*on\s*device"),
        # Data loading
        ("dataloader_error", "warning", r"(?i)DataLoader\s*worker.*exited|broken\s*pipe.*DataLoader|EOF.*DataLoader"),
        ("data_corruption", "warning", r"(?i)corrupted.*data|unexpected\s*EOF|truncated\s*file"),
        # Checkpoint issues
        ("checkpoint_fail", "warning", r"(?i)checkpoint.*fail|save.*state.*error|load.*state.*error|corrupted.*checkpoint"),
        # Hung/stuck process
        ("process_hung", "critical", r"(?i)no\s*progress|heartbeat.*timeout|watchdog.*expired|deadlock\s*detected"),
        ("stuck_rank", "critical", r"(?i)rank\s*\d+.*stuck|barrier.*timeout|rendezvous.*timeout"),
    ]

    async def _handle_detect_training_anomaly_patterns(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        job_name = str(tool_args.get("job_name") or "").strip()
        if not job_name:
            raise ValueError("job_name is required")
        tail = max(50, min(int(tool_args.get("tail") or 500), 5000))
        container = str(tool_args.get("container") or "").strip()

        # Get all pods for this job
        args = ["get", "pods", "--all-namespaces", "-l", f"volcano.sh/job-name={job_name}",
                "--field-selector", "status.phase!=Pending"]
        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = payload.get("items") or []

        if not items:
            return {
                "status": "success",
                "result": {"job_name": job_name, "pod_count": 0, "categories": {}, "summary": "no pods found"},
            }

        import re as _re  # noqa: PLC0415

        compiled_patterns = [
            (cat, sev, _re.compile(pat)) for cat, sev, pat in self._TRAINING_ANOMALY_PATTERNS
        ]

        categories: dict[str, dict[str, Any]] = {}
        pod_results: list[dict[str, Any]] = []

        for pod in items[:20]:  # Cap to 20 pods
            metadata = pod.get("metadata") or {}
            pod_name = str(metadata.get("name") or "")
            namespace = str(metadata.get("namespace") or "")
            if not pod_name:
                continue

            # Fetch logs
            cmd = self._kubectl_cmd(namespace=namespace, include_namespace=True) + [
                "logs", pod_name, "--tail", str(tail),
            ]
            if container:
                cmd.extend(["-c", container])
            try:
                code, stdout, stderr = await self._run_subprocess(cmd, timeout_seconds=30)
                if code != 0:
                    pod_results.append({"pod_name": pod_name, "error": stderr.strip()[:200]})
                    continue
            except ValueError:
                pod_results.append({"pod_name": pod_name, "error": "log fetch timed out"})
                continue

            pod_hits: list[dict[str, Any]] = []
            for line_num, line in enumerate(stdout.splitlines(), 1):
                for cat, sev, pattern in compiled_patterns:
                    if pattern.search(line):
                        pod_hits.append({
                            "category": cat,
                            "severity": sev,
                            "line": line_num,
                            "sample": line.strip()[:300],
                        })
                        # Update category aggregation
                        if cat not in categories:
                            categories[cat] = {"severity": sev, "count": 0, "pods": [], "samples": []}
                        categories[cat]["count"] += 1
                        if pod_name not in categories[cat]["pods"]:
                            categories[cat]["pods"].append(pod_name)
                        if len(categories[cat]["samples"]) < 3:
                            categories[cat]["samples"].append(line.strip()[:200])
                        break  # One match per line

            pod_results.append({
                "pod_name": pod_name,
                "hit_count": len(pod_hits),
                "hits": pod_hits[:30],  # Cap per-pod hits
            })

        # Determine overall severity
        severities = [c["severity"] for c in categories.values()]
        overall = "critical" if "critical" in severities else ("warning" if "warning" in severities else "ok")

        return {
            "status": "success",
            "result": {
                "job_name": job_name,
                "pod_count": len(pod_results),
                "overall_severity": overall,
                "category_count": len(categories),
                "categories": categories,
                "pods": pod_results,
            },
        }

    async def _handle_get_distributed_job_overview(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        job_name = str(tool_args.get("job_name") or "").strip()
        if not job_name:
            raise ValueError("job_name is required")

        # Get all pods
        args = ["get", "pods", "--all-namespaces", "-l", f"volcano.sh/job-name={job_name}"]
        payload = await self._run_kubectl_json(args, include_namespace=False)
        items = payload.get("items") or []

        _DIST_ENV_PREFIXES = ("NCCL_", "MASTER_ADDR", "MASTER_PORT", "WORLD_SIZE", "RANK",
                               "LOCAL_RANK", "CUDA_VISIBLE_DEVICES")

        ranks: list[dict[str, Any]] = []
        nodes_involved: dict[str, dict[str, Any]] = {}
        master_addr = ""
        master_port = ""
        world_size = ""

        for pod in items:
            metadata = pod.get("metadata") or {}
            spec = pod.get("spec") or {}
            status = pod.get("status") or {}
            labels = metadata.get("labels") or {}
            pod_name = str(metadata.get("name") or "")
            node_name = str(spec.get("nodeName") or "")
            pod_ip = str(status.get("podIP") or "")
            phase = str(status.get("phase") or "")

            # Extract key env vars
            key_env: dict[str, str] = {}
            for container in spec.get("containers") or []:
                for env in container.get("env") or []:
                    name = str(env.get("name") or "")
                    value = str(env.get("value") or "")
                    if any(name.startswith(p) for p in _DIST_ENV_PREFIXES):
                        key_env[name] = value

            if not master_addr and "MASTER_ADDR" in key_env:
                master_addr = key_env["MASTER_ADDR"]
            if not master_port and "MASTER_PORT" in key_env:
                master_port = key_env["MASTER_PORT"]
            if not world_size and "WORLD_SIZE" in key_env:
                world_size = key_env["WORLD_SIZE"]

            # Derive ordinal
            ordinal = 0
            parts = pod_name.rsplit("-", 1)
            if len(parts) == 2:
                try:
                    ordinal = int(parts[1])
                except ValueError:
                    pass

            container_statuses = status.get("containerStatuses") or []
            ready = all(cs.get("ready", False) for cs in container_statuses) if container_statuses else False
            restart_count = sum(int(cs.get("restartCount") or 0) for cs in container_statuses)

            ranks.append({
                "pod_name": pod_name,
                "task_spec": str(labels.get("volcano.sh/task-spec") or ""),
                "ordinal": ordinal,
                "node_name": node_name,
                "pod_ip": pod_ip,
                "host_ip": str(status.get("hostIP") or ""),
                "phase": phase,
                "ready": ready,
                "restart_count": restart_count,
                "nccl_env": {k: v for k, v in key_env.items() if k.startswith("NCCL_")},
                "cuda_visible_devices": key_env.get("CUDA_VISIBLE_DEVICES", ""),
            })

            # Track nodes
            if node_name and node_name not in nodes_involved:
                nodes_involved[node_name] = {"pod_count": 0, "ready_pods": 0, "not_ready_pods": 0}
            if node_name:
                nodes_involved[node_name]["pod_count"] += 1
                if ready:
                    nodes_involved[node_name]["ready_pods"] += 1
                else:
                    nodes_involved[node_name]["not_ready_pods"] += 1

        ranks.sort(key=lambda x: (x["task_spec"], x["ordinal"]))
        for i, r in enumerate(ranks):
            r["rank"] = i

        # Diagnose issues
        issues: list[str] = []
        not_ready = [r for r in ranks if not r["ready"]]
        if not_ready:
            issues.append(f"{len(not_ready)}/{len(ranks)} ranks not ready: {[r['pod_name'] for r in not_ready[:5]]}")
        restarted = [r for r in ranks if r["restart_count"] > 0]
        if restarted:
            issues.append(f"{len(restarted)} ranks have restarts: {[(r['pod_name'], r['restart_count']) for r in restarted[:5]]}")
        pending = [r for r in ranks if r["phase"] == "Pending"]
        if pending:
            issues.append(f"{len(pending)} ranks still Pending")
        if world_size and int(world_size) != len(ranks):
            issues.append(f"WORLD_SIZE={world_size} but found {len(ranks)} pods")

        return {
            "status": "success",
            "result": {
                "job_name": job_name,
                "total_ranks": len(ranks),
                "master_addr": master_addr,
                "master_port": master_port,
                "world_size": world_size,
                "node_count": len(nodes_involved),
                "nodes": nodes_involved,
                "issues": issues,
                "issue_count": len(issues),
                "ranks": ranks,
            },
        }

    async def _handle_get_node_accelerator_info(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")

        # Multi-vendor detection: NVIDIA, Hygon DCU, Ascend NPU, Cambricon MLU, generic PCIe
        commands = (
            "echo '===PCIE_DEVICES===' && "
            "(lspci 2>/dev/null | grep -iE 'NVIDIA|AMD|Hygon|Huawei|Ascend|Cambricon|display|3D|VGA|accelerat|process' "
            "|| echo '(lspci unavailable)') && "

            "echo '===NVIDIA===' && "
            "(nvidia-smi --query-gpu=index,name,driver_version,memory.total,memory.free,temperature.gpu,"
            "ecc.errors.uncorrected.volatile.total --format=csv,noheader 2>/dev/null "
            "|| echo '(nvidia-smi not found)') && "
            "echo '===NVIDIA_CUDA===' && "
            "(nvidia-smi 2>/dev/null | grep 'CUDA Version' || echo '(no cuda)') && "

            "echo '===HYGON_DCU===' && "
            "(rocm-smi --showid --showtemp --showuse --showmeminfo vram 2>/dev/null "
            "|| hy-smi 2>/dev/null "
            "|| echo '(no hygon/rocm tools)') && "
            "echo '===HYGON_DRIVER===' && "
            "(cat /sys/module/amdgpu/version 2>/dev/null || modinfo amdgpu 2>/dev/null | grep ^version "
            "|| echo '(no amdgpu driver)') && "

            "echo '===ASCEND_NPU===' && "
            "(npu-smi info 2>/dev/null || echo '(npu-smi not found)') && "
            "echo '===ASCEND_DRIVER===' && "
            "(cat /usr/local/Ascend/driver/version.info 2>/dev/null "
            "|| npu-smi info -t board 2>/dev/null | head -20 "
            "|| echo '(no ascend driver)') && "

            "echo '===CAMBRICON_MLU===' && "
            "(cnmon 2>/dev/null | head -40 || echo '(cnmon not found)') && "

            "echo '===COMM_LIBS===' && "
            "(echo 'NCCL:' && (find /usr -name 'libnccl*' -o -name 'nccl.h' 2>/dev/null | head -5 || echo '  not found') && "
            "echo 'HCCL:' && (find /usr -name 'libhccl*' -o -name 'hccl.h' 2>/dev/null | head -5 || echo '  not found') && "
            "echo 'RCCL:' && (find /usr -name 'librccl*' -o -name 'rccl.h' 2>/dev/null | head -5 || echo '  not found') && "
            "echo 'MPI:' && (which mpirun 2>/dev/null && mpirun --version 2>/dev/null | head -1 || echo '  not found')) && "

            "echo '===ACCEL_MODULES===' && "
            "(lsmod 2>/dev/null | grep -iE 'nvidia|amdgpu|hygon|drm_vram|habanalabs|davinci|cambricon' "
            "|| echo '(no accelerator modules)')"
        )

        raw = await self._run_kubectl_debug_node(node_name, commands, timeout=120)

        sections: dict[str, str] = {}
        current_section = ""
        current_lines: list[str] = []
        for line in raw.splitlines():
            if line.startswith("===") and line.endswith("==="):
                if current_section:
                    sections[current_section] = "\n".join(current_lines)
                current_section = line.strip("=")
                current_lines = []
            else:
                current_lines.append(line)
        if current_section:
            sections[current_section] = "\n".join(current_lines)

        def is_available(key: str) -> bool:
            text = sections.get(key, "").strip()
            return bool(text) and not text.startswith("(")

        # Detect accelerator type
        detected_type = "unknown"
        if is_available("NVIDIA"):
            detected_type = "nvidia"
        elif is_available("HYGON_DCU"):
            detected_type = "hygon_dcu"
        elif is_available("ASCEND_NPU"):
            detected_type = "ascend_npu"
        elif is_available("CAMBRICON_MLU"):
            detected_type = "cambricon_mlu"

        # Parse NVIDIA GPUs
        nvidia_gpus: list[dict[str, str]] = []
        if is_available("NVIDIA"):
            for line in sections["NVIDIA"].splitlines():
                parts = [p.strip() for p in line.split(",")]
                if len(parts) >= 7:
                    nvidia_gpus.append({
                        "index": parts[0], "name": parts[1], "driver": parts[2],
                        "memory_total": parts[3], "memory_free": parts[4],
                        "temp_c": parts[5], "ecc_uncorrected": parts[6],
                    })

        cuda_version = ""
        for line in sections.get("NVIDIA_CUDA", "").splitlines():
            if "CUDA Version" in line:
                import re as _re  # noqa: PLC0415
                m = _re.search(r"CUDA Version:\s*([\d.]+)", line)
                if m:
                    cuda_version = m.group(1)

        # Parse comm lib availability
        comm_text = sections.get("COMM_LIBS", "")
        comm_libs: dict[str, bool] = {}
        for lib in ("NCCL", "HCCL", "RCCL", "MPI"):
            comm_libs[lib.lower()] = lib + ":" in comm_text and "not found" not in comm_text.split(lib + ":")[1].split("\n")[0] if lib + ":" in comm_text else False

        # Parse PCIe devices
        pcie_text = sections.get("PCIE_DEVICES", "").strip()
        pcie_devices = [l.strip() for l in pcie_text.splitlines() if l.strip()] if not pcie_text.startswith("(") else []

        # Parse kernel modules
        modules_text = sections.get("ACCEL_MODULES", "").strip()
        accel_modules = [l.strip() for l in modules_text.splitlines() if l.strip()] if not modules_text.startswith("(") else []

        # Compatibility notes
        compat_notes: list[str] = []
        if detected_type == "hygon_dcu" and not comm_libs.get("rccl"):
            compat_notes.append("Hygon DCU detected but RCCL not found — distributed training may not work")
        if detected_type == "ascend_npu" and not comm_libs.get("hccl"):
            compat_notes.append("Ascend NPU detected but HCCL not found — distributed training may not work")
        if detected_type == "nvidia" and not comm_libs.get("nccl"):
            compat_notes.append("NVIDIA GPU detected but NCCL not found — distributed training may not work")
        if detected_type == "unknown" and pcie_devices:
            compat_notes.append("Accelerator devices found via PCIe but no management tool (nvidia-smi/npu-smi/rocm-smi) available")

        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "detected_type": detected_type,
                "nvidia": {
                    "gpu_count": len(nvidia_gpus),
                    "cuda_version": cuda_version,
                    "gpus": nvidia_gpus,
                } if nvidia_gpus else None,
                "hygon_dcu": sections.get("HYGON_DCU", "").strip()[:3000] if is_available("HYGON_DCU") else None,
                "hygon_driver": sections.get("HYGON_DRIVER", "").strip()[:500] if is_available("HYGON_DRIVER") else None,
                "ascend_npu": sections.get("ASCEND_NPU", "").strip()[:3000] if is_available("ASCEND_NPU") else None,
                "ascend_driver": sections.get("ASCEND_DRIVER", "").strip()[:500] if is_available("ASCEND_DRIVER") else None,
                "cambricon_mlu": sections.get("CAMBRICON_MLU", "").strip()[:3000] if is_available("CAMBRICON_MLU") else None,
                "comm_libs": comm_libs,
                "pcie_accelerator_devices": pcie_devices[:20],
                "accel_kernel_modules": accel_modules,
                "compatibility_notes": compat_notes,
            },
        }

    # ------------------------------------------------------------------
    # P3: Extended K8s Tools (top, rollout, scale, label, taint)
    # ------------------------------------------------------------------

    async def _handle_k8s_top_nodes(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        args = ["top", "nodes", "--no-headers"]
        node_name = str(tool_args.get("node_name") or "").strip()
        if node_name:
            args.insert(2, node_name)
        result = await self._run_kubectl_text(args, include_namespace=False)
        lines = [l for l in result.get("content", "").splitlines() if l.strip()]
        nodes = []
        for line in lines:
            parts = line.split()
            if len(parts) >= 5:
                nodes.append({
                    "name": parts[0],
                    "cpu_usage": parts[1],
                    "cpu_percent": parts[2],
                    "memory_usage": parts[3],
                    "memory_percent": parts[4],
                })
        return {"status": "success", "result": {"count": len(nodes), "nodes": nodes}}

    async def _handle_k8s_top_pods(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        namespace = str(tool_args.get("namespace") or "").strip()
        label_selector = str(tool_args.get("label_selector") or "").strip()
        limit = max(1, min(int(tool_args.get("limit") or 50), 500))
        args = ["top", "pods", "--no-headers"]
        if namespace:
            args.extend(["-n", namespace])
        else:
            args.append("--all-namespaces")
        if label_selector:
            args.extend(["-l", label_selector])
        result = await self._run_kubectl_text(args, include_namespace=False)
        lines = [l for l in result.get("content", "").splitlines() if l.strip()]
        pods = []
        for line in lines[:limit]:
            parts = line.split()
            if len(parts) >= 3:
                pods.append({
                    "name": parts[0] if namespace else parts[1],
                    "namespace": namespace or (parts[0] if not namespace else ""),
                    "cpu_usage": parts[-2] if len(parts) >= 4 else parts[-1],
                    "memory_usage": parts[-1],
                })
        return {"status": "success", "result": {"count": len(pods), "pods": pods, "truncated": len(lines) > limit}}

    async def _handle_k8s_rollout_status(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        kind = str(tool_args.get("kind") or "").strip()
        name = str(tool_args.get("name") or "").strip()
        namespace = str(tool_args.get("namespace") or "").strip() or None
        if not kind or not name:
            raise ValueError("kind and name are required")
        allowed_kinds = {"Deployment", "StatefulSet", "DaemonSet"}
        if kind not in allowed_kinds:
            raise ValueError(f"kind must be one of {allowed_kinds}, got: {kind}")
        args = ["rollout", "status", f"{kind.lower()}/{name}"]
        result = await self._run_kubectl_text(args, namespace=namespace)
        return {"status": "success", "result": {"kind": kind, "name": name, "output": result.get("content", "")}}

    async def _handle_k8s_scale_workload(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        kind = str(tool_args.get("kind") or "").strip()
        name = str(tool_args.get("name") or "").strip()
        replicas = int(tool_args.get("replicas", -1))
        namespace = str(tool_args.get("namespace") or "").strip() or None
        if not kind or not name:
            raise ValueError("kind and name are required")
        if kind not in ("Deployment", "StatefulSet"):
            raise ValueError(f"kind must be Deployment or StatefulSet, got: {kind}")
        if replicas < 0 or replicas > 100:
            raise ValueError(f"replicas must be 0-100, got: {replicas}")
        args = ["scale", f"{kind.lower()}/{name}", f"--replicas={replicas}"]
        result = await self._run_kubectl_text(args, namespace=namespace)
        return {"status": "success", "result": {"kind": kind, "name": name, "replicas": replicas, "output": result.get("content", "")}}

    async def _handle_k8s_label_node(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        key = str(tool_args.get("key") or "").strip()
        value = str(tool_args.get("value") or "").strip()
        overwrite = bool(tool_args.get("overwrite", False))
        if not node_name or not key:
            raise ValueError("node_name and key are required")
        args = ["label", "nodes", node_name, f"{key}={value}"]
        if overwrite:
            args.append("--overwrite")
        result = await self._run_kubectl_text(args, include_namespace=False)
        return {"status": "success", "result": {"node_name": node_name, "label": f"{key}={value}", "output": result.get("content", "")}}

    async def _handle_k8s_taint_node(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        key = str(tool_args.get("key") or "").strip()
        value = str(tool_args.get("value") or "")
        effect = str(tool_args.get("effect") or "NoSchedule").strip()
        if not node_name or not key:
            raise ValueError("node_name and key are required")
        allowed_effects = {"NoSchedule", "PreferNoSchedule", "NoExecute"}
        if effect not in allowed_effects:
            raise ValueError(f"effect must be one of {allowed_effects}, got: {effect}")
        taint_spec = f"{key}={value}:{effect}" if value else f"{key}:{effect}"
        args = ["taint", "nodes", node_name, taint_spec]
        result = await self._run_kubectl_text(args, include_namespace=False)
        return {"status": "success", "result": {"node_name": node_name, "taint": taint_spec, "output": result.get("content", "")}}

    # ------------------------------------------------------------------
    # K8s Write Tools: Node/Pod management
    # ------------------------------------------------------------------

    async def _handle_cordon_node(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")
        args = ["cordon", node_name]
        result = await self._run_kubectl_text(args, include_namespace=False)
        # Verify: node should show SchedulingDisabled
        verify = await self._run_kubectl_json(["get", "node", node_name], include_namespace=False)
        unschedulable = (verify.get("spec") or {}).get("unschedulable", False)
        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "action": "cordon",
                "verified_unschedulable": unschedulable,
                "output": result.get("content", ""),
            },
        }

    async def _handle_uncordon_node(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")
        args = ["uncordon", node_name]
        result = await self._run_kubectl_text(args, include_namespace=False)
        verify = await self._run_kubectl_json(["get", "node", node_name], include_namespace=False)
        unschedulable = (verify.get("spec") or {}).get("unschedulable", False)
        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "action": "uncordon",
                "verified_unschedulable": unschedulable,
                "output": result.get("content", ""),
            },
        }

    async def _handle_drain_node(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        node_name = str(tool_args.get("node_name") or "").strip()
        if not node_name:
            raise ValueError("node_name is required")
        args = [
            "drain", node_name,
            "--ignore-daemonsets",
            "--delete-emptydir-data",
            "--force",
            "--grace-period=60",
            "--timeout=300s",
        ]
        result = await self._run_kubectl_text(args, include_namespace=False, max_chars=50_000)
        return {
            "status": "success",
            "result": {
                "node_name": node_name,
                "action": "drain",
                "output": result.get("content", ""),
            },
        }

    async def _handle_delete_pod(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        name = str(tool_args.get("name") or "").strip()
        namespace = str(tool_args.get("namespace") or "").strip() or None
        force = bool(tool_args.get("force", False))
        grace_period = tool_args.get("grace_period_seconds")
        if not name:
            raise ValueError("name is required")
        args = ["delete", "pod", name]
        if force:
            args.extend(["--force", "--grace-period=0"])
        elif grace_period is not None:
            args.extend([f"--grace-period={int(grace_period)}"])
        result = await self._run_kubectl_text(args, namespace=namespace)
        return {
            "status": "success",
            "result": {
                "pod_name": name,
                "namespace": namespace,
                "action": "delete_pod",
                "force": force,
                "output": result.get("content", ""),
            },
        }

    async def _handle_restart_workload(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        kind = str(tool_args.get("kind") or "").strip()
        name = str(tool_args.get("name") or "").strip()
        namespace = str(tool_args.get("namespace") or "").strip() or None
        if not kind or not name:
            raise ValueError("kind and name are required")
        allowed_kinds = {"Deployment", "StatefulSet", "DaemonSet"}
        if kind not in allowed_kinds:
            raise ValueError(f"kind must be one of {allowed_kinds}, got: {kind}")
        args = ["rollout", "restart", f"{kind.lower()}/{name}"]
        result = await self._run_kubectl_text(args, namespace=namespace)
        return {
            "status": "success",
            "result": {
                "kind": kind,
                "name": name,
                "namespace": namespace,
                "action": "restart_workload",
                "output": result.get("content", ""),
            },
        }

    # ------------------------------------------------------------------
    # execute_admin_command: Generalized admin command execution
    # ------------------------------------------------------------------

    _ADMIN_CMD_ALLOWLIST = {"kubectl", "helm", "velero", "istioctl"}
    _ADMIN_CMD_BLOCKLIST_PATTERNS = [
        "delete namespace",
        "delete node",
        "delete pv ",
        "delete crd",
        "cluster-info dump",
        "auth can-i",
        "--as=",
        "exec -it",
        "port-forward",
        "proxy",
        "apply -f http",
    ]

    async def _handle_execute_admin_command(self, tool_args: dict[str, Any]) -> dict[str, Any]:
        command = str(tool_args.get("command") or "").strip()
        reason = str(tool_args.get("reason") or "").strip()
        if not command:
            raise ValueError("command is required")
        if not reason:
            raise ValueError("reason is required (explain why this command is needed)")

        parts = command.split()
        if not parts:
            raise ValueError("command cannot be empty")
        binary = parts[0]
        if binary not in self._ADMIN_CMD_ALLOWLIST:
            raise ValueError(
                f"command must start with one of {sorted(self._ADMIN_CMD_ALLOWLIST)}, got: {binary}"
            )
        cmd_lower = command.lower()
        for pattern in self._ADMIN_CMD_BLOCKLIST_PATTERNS:
            if pattern in cmd_lower:
                raise ValueError(f"command contains blocked pattern: {pattern!r}")

        # Build the actual command with kubeconfig injection
        if binary == "kubectl":
            base = self._kubectl_cmd(include_namespace=False)
            # Remove the binary itself since _kubectl_cmd already includes it
            full_cmd = base + parts[1:]
        else:
            full_cmd = parts
            if self.runtime.kubeconfig_path:
                full_cmd = [binary, "--kubeconfig", self.runtime.kubeconfig_path] + parts[1:]

        code, stdout, stderr = await self._run_subprocess(full_cmd, timeout_seconds=300)
        success = code == 0
        output = stdout.strip() if success else stderr.strip()

        return {
            "status": "success" if success else "error",
            "result": {
                "command": command,
                "reason": reason,
                "exit_code": code,
                "output": output[:50_000],
                "truncated": len(output) > 50_000,
            },
        }
