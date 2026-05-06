"""Platform/runtime discovery for agent-side local/core tools.

Default behavior:
- discover and read backend debug-config first
- optionally merge an agent-side override config if explicitly provided
- expose a small routing helper for local vs backend tool execution
"""

from __future__ import annotations

import json
import os
import shutil
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

import yaml

from config import settings
from paths import repo_root

DEFAULT_LOCAL_CORE_TOOLS = {
    "get_agent_runtime_summary",
    "web_search",
    "fetch_url",
    "k8s_list_nodes",
    "k8s_list_pods",
    "k8s_get_events",
    "k8s_describe_resource",
    "k8s_get_pod_logs",
    "prometheus_query",
    "harbor_check",
    "k8s_get_service",
    "k8s_get_endpoints",
    "k8s_get_ingress",
    "get_volcano_queue_state",
    "k8s_get_configmap",
    "k8s_get_networkpolicy",
    "aggregate_image_pull_errors",
    "detect_zombie_jobs",
    "get_ddp_rank_mapping",
    "get_node_kernel_diagnostics",
    "get_rdma_interface_status",
    "get_node_gpu_info",
    "get_nccl_env_config",
    "check_node_nic_status",
    "detect_training_anomaly_patterns",
    "get_distributed_job_overview",
    "get_node_accelerator_info",
    "k8s_top_nodes",
    "k8s_top_pods",
    "k8s_rollout_status",
}

DEFAULT_LOCAL_WRITE_TOOLS: set[str] = set()

DEFAULT_PROTECTED_NAMESPACES = [
    "kube-system",
    "kube-public",
    "kube-node-lease",
]

DEFAULT_BLOCKED_COMMAND_PATTERNS = [
    "delete namespace",
    "delete node",
    "delete pv",
    "delete crd",
    "cluster-info dump",
    "auth can-i",
    "--as=",
    "exec -it",
    "port-forward",
    "proxy",
    "apply -f http",
]


def _load_json_dict(path: Path | None) -> dict[str, Any]:
    if path is None or not path.exists():
        return {}
    try:
        raw = path.read_text(encoding="utf-8").strip()
        loaded = json.loads(raw) if raw else {}
        return dict(loaded) if isinstance(loaded, dict) else {}
    except Exception:
        return {}


def _load_yaml_dict(path: Path | None) -> dict[str, Any]:
    if path is None or not path.exists():
        return {}
    try:
        raw = path.read_text(encoding="utf-8").strip()
        loaded = yaml.safe_load(raw) if raw else {}
        return dict(loaded) if isinstance(loaded, dict) else {}
    except Exception:
        return {}


def _resolve_relative_path(raw_path: str, *, base_dir: Path) -> Path:
    candidate = Path(raw_path).expanduser()
    if candidate.is_absolute():
        return candidate
    return (base_dir / candidate).resolve()


def _as_string_list(value: Any) -> list[str]:
    if not isinstance(value, list):
        return []
    return [str(item).strip() for item in value if str(item).strip()]


def _as_string_dict(value: Any) -> dict[str, str]:
    if not isinstance(value, dict):
        return {}
    result: dict[str, str] = {}
    for key, item in value.items():
        normalized_key = str(key).strip()
        normalized_value = str(item).strip()
        if normalized_key and normalized_value:
            result[normalized_key] = normalized_value
    return result


def _default_repo_root() -> Path:
    return repo_root()


def _discover_backend_config_path() -> Path | None:
    # Agent setting override (if configured)
    configured = settings.resolve_backend_debug_config_path()
    if configured is not None and configured.exists():
        return configured

    # Backend convention env var
    env_path = str(os.getenv("CRATER_DEBUG_CONFIG_PATH") or "").strip()
    if env_path:
        resolved = _resolve_relative_path(env_path, base_dir=Path.cwd())
        if resolved.exists():
            return resolved

    # Common local dev defaults
    repo_root = _default_repo_root()
    for candidate in (
        repo_root / "backend/etc/debug-config.yaml",
        repo_root / "backend/etc/debug-config.yml",
        repo_root / "backend/etc/example-config.yaml",
    ):
        if candidate.exists():
            return candidate
    return None


@dataclass(frozen=True)
class PlatformRuntimeConfig:
    """Resolved runtime config for agent-local tools."""

    local_core_tools: set[str] = field(
        default_factory=lambda: set(DEFAULT_LOCAL_CORE_TOOLS)
    )
    local_write_tools: set[str] = field(default_factory=lambda: set(DEFAULT_LOCAL_WRITE_TOOLS))
    web_search_enabled: bool = False
    web_search_allowed_domains: list[str] = field(default_factory=list)
    web_search_seed_urls: list[str] = field(default_factory=list)
    web_search_timeout_seconds: int = 10
    web_search_max_content_chars: int = 4_000
    backend_url: str = ""
    agent_service_url: str = ""
    platform_host: str = ""
    model_download_image: str = ""
    prometheus_api: str = ""
    prometheus_timeout_seconds: int = 15
    prometheus_headers: dict[str, str] = field(default_factory=dict)
    prometheus_verify_tls: bool = True
    registry_server: str = ""
    registry_project: str = ""
    registry_username: str = ""
    registry_password: str = ""
    registry_verify_tls: bool = True
    registry_enabled: bool = False
    build_proxy: dict[str, str] = field(default_factory=dict)
    build_tool_images: dict[str, str] = field(default_factory=dict)
    grafana: dict[str, str] = field(default_factory=dict)
    kubeconfig_path: str = ""
    kube_context: str = ""
    kube_namespace: str = ""
    kubectl_bin: str = "kubectl"
    kubectl_timeout_seconds: int = 30
    k8s_protected_namespaces: list[str] = field(default_factory=lambda: list(DEFAULT_PROTECTED_NAMESPACES))
    k8s_protected_node_names: list[str] = field(default_factory=list)
    k8s_blocked_command_patterns: list[str] = field(
        default_factory=lambda: list(DEFAULT_BLOCKED_COMMAND_PATTERNS)
    )
    namespaces: dict[str, str] = field(default_factory=dict)
    storage_prefixes: dict[str, str] = field(default_factory=dict)
    storage_pvcs: dict[str, str] = field(default_factory=dict)
    sources: dict[str, str] = field(default_factory=dict)

    def is_local_core_tool(self, tool_name: str) -> bool:
        return tool_name in self.local_core_tools

    def is_local_write_tool(self, tool_name: str) -> bool:
        return tool_name in self.local_write_tools

    def default_route_for_tool(self, tool_name: str) -> str:
        if self.is_local_core_tool(tool_name) or self.is_local_write_tool(tool_name):
            return "local"
        return "backend"

    def is_allowed_web_url(self, url: str) -> bool:
        if not self.web_search_allowed_domains:
            return True
        hostname = (urlparse(url).hostname or "").lower()
        if not hostname:
            return False
        return any(
            hostname == domain or hostname.endswith(f".{domain}")
            for domain in (d.lower() for d in self.web_search_allowed_domains)
        )

    def to_summary(self) -> dict[str, Any]:
        return {
            "localCoreTools": sorted(self.local_core_tools),
            "localWriteTools": sorted(self.local_write_tools),
            "recommendedPlatformRuntimeFields": self.recommended_platform_runtime_fields(),
            "discouragedConfigSections": self.discouraged_platform_runtime_fields(),
            "toolReadiness": self.tool_readiness(),
            "webSearchEnabled": self.web_search_enabled,
            "webSearchAllowedDomains": list(self.web_search_allowed_domains),
            "webSearchSeedUrls": list(self.web_search_seed_urls),
            "backendURL": self.backend_url,
            "agentServiceURL": self.agent_service_url,
            "platformHost": self.platform_host,
            "modelDownloadImage": self.model_download_image,
            "prometheusAPI": self.prometheus_api,
            "prometheus": {
                "baseURL": self.prometheus_api,
                "timeoutSeconds": self.prometheus_timeout_seconds,
                "headers": dict(self.prometheus_headers),
                "verifyTLS": self.prometheus_verify_tls,
            },
            "registry": {
                "enabled": self.registry_enabled,
                "server": self.registry_server,
                "project": self.registry_project,
                "username": self.registry_username,
                "verifyTLS": self.registry_verify_tls,
                "buildProxy": dict(self.build_proxy),
                "buildToolImages": dict(self.build_tool_images),
            },
            "kubernetes": {
                "kubeconfigPath": self.kubeconfig_path,
                "context": self.kube_context,
                "namespace": self.kube_namespace,
                "kubectlBin": self.kubectl_bin,
                "timeoutSeconds": self.kubectl_timeout_seconds,
                "safety": {
                    "protectedNamespaces": list(self.k8s_protected_namespaces),
                    "protectedNodeNames": list(self.k8s_protected_node_names),
                    "blockedCommandPatterns": list(self.k8s_blocked_command_patterns),
                },
            },
            "namespaces": dict(self.namespaces),
            "storage": {
                "prefixes": dict(self.storage_prefixes),
                "pvc": dict(self.storage_pvcs),
            },
            "grafana": {
                "address": str(self.grafana.get("address") or "").strip(),
                "host": str(self.grafana.get("host") or "").strip(),
                "hasToken": bool(str(self.grafana.get("token") or "").strip()),
            },
            "sources": dict(self.sources),
        }

    @staticmethod
    def recommended_platform_runtime_fields() -> dict[str, list[str]]:
        return {
            "toolRouting": [
                "toolRouting.localCoreTools",
                "toolRouting.localWriteTools",
            ],
            "sharedPlatformBaseline": [
                "host",
                "modelDownloadImage",
                "namespaces.job",
                "namespaces.image",
                "storage.prefix.user",
                "storage.prefix.account",
                "storage.prefix.public",
                "storage.pvc.readWriteMany",
                "storage.pvc.readOnlyMany",
            ],
            "localKubernetesTools": [
                "kubernetes.kubeconfigPath",
                "kubernetes.context",
                "kubernetes.namespace",
                "kubernetes.kubectlBin",
                "kubernetes.timeoutSeconds",
                "kubernetes.safety.protectedNamespaces",
                "kubernetes.safety.protectedNodeNames",
                "kubernetes.safety.blockedCommandPatterns",
            ],
            "localPrometheusTools": [
                "prometheus.baseURL",
                "prometheus.timeoutSeconds",
                "prometheus.headers",
                "prometheus.verifyTLS",
            ],
            "localRegistryTools": [
                "registry.enabled",
                "registry.server",
                "registry.username",
                "registry.password",
                "registry.verifyTLS",
                "registry.buildProxy.httpsProxy",
                "registry.buildToolImages.buildx",
                "registry.buildToolImages.nerdctl",
                "registry.buildToolImages.envd",
            ],
            "optionalGrafanaContext": [
                "grafana.address",
                "grafana.host",
                "grafana.token",
            ],
            "optionalWebSearch": [
                "webSearch.enabled",
                "webSearch.allowedDomains",
                "webSearch.seedUrls",
                "webSearch.timeoutSeconds",
                "webSearch.maxContentChars",
            ],
        }

    @staticmethod
    def discouraged_platform_runtime_fields() -> list[str]:
        return [
            "postgres.* unless a direct local DB tool is added",
            "smtp.*",
            "auth.*",
            "secrets.*",
            "inline kubeconfig certificate/key blobs",
            "backend-only chart values that the agent never reads",
        ]

    def tool_readiness(self) -> dict[str, Any]:
        def build_state(
            *,
            missing_fields: list[str] | None = None,
            blocking_issues: list[str] | None = None,
            warnings: list[str] | None = None,
        ) -> dict[str, Any]:
            missing = list(missing_fields or [])
            blocking = list(blocking_issues or [])
            warning_list = list(warnings or [])
            return {
                "ready": not missing and not blocking,
                "missingFields": missing,
                "blockingIssues": blocking,
                "warnings": warning_list,
            }

        k8s_missing: list[str] = []
        k8s_blocking: list[str] = []
        if not self.kubeconfig_path:
            k8s_missing.append("kubernetes.kubeconfigPath")
        else:
            kubeconfig_path = Path(self.kubeconfig_path).expanduser()
            if not kubeconfig_path.exists():
                k8s_blocking.append(f"kubeconfig file does not exist: {kubeconfig_path}")
        kubectl_bin = str(self.kubectl_bin or "").strip()
        if not kubectl_bin:
            k8s_missing.append("kubernetes.kubectlBin")
        elif shutil.which(kubectl_bin) is None:
            k8s_blocking.append(f"kubectl binary not found on PATH: {kubectl_bin}")
        k8s_ready = build_state(
            missing_fields=k8s_missing,
            blocking_issues=k8s_blocking,
            warnings=[] if self.kube_context else ["kubernetes.context not set; kubectl default context will be used"],
        )

        prometheus_ready = build_state(
            missing_fields=[] if self.prometheus_api else ["prometheus.baseURL"],
        )

        registry_missing: list[str] = []
        registry_warnings: list[str] = []
        if not self.registry_enabled:
            registry_missing.append("registry.enabled=true")
        if not self.registry_server:
            registry_missing.append("registry.server")
        if not self.registry_username or not self.registry_password:
            registry_warnings.append("registry credentials are empty; private Harbor checks may return 401")
        registry_ready = build_state(
            missing_fields=registry_missing,
            warnings=registry_warnings,
        )

        return {
            "get_agent_runtime_summary": build_state(),
            "k8s_list_nodes": k8s_ready,
            "k8s_list_pods": k8s_ready,
            "k8s_get_events": k8s_ready,
            "k8s_describe_resource": k8s_ready,
            "k8s_get_pod_logs": k8s_ready,
            "prometheus_query": prometheus_ready,
            "harbor_check": registry_ready,
        }


def load_platform_runtime_config() -> PlatformRuntimeConfig:
    """Load backend debug config first, then optional agent-side overrides."""

    runtime_path = settings.resolve_platform_runtime_config_path()
    platform_payload = {}
    if runtime_path is not None:
        platform_payload = _load_json_dict(runtime_path) or _load_yaml_dict(runtime_path)

    backend_path = _discover_backend_config_path()
    backend_payload = _load_yaml_dict(backend_path)

    repo_root = _default_repo_root()
    web_cfg = platform_payload.get("webSearch") or {}
    prometheus_cfg = platform_payload.get("prometheus") or {}
    registry_cfg = platform_payload.get("registry") or {}
    grafana_cfg = platform_payload.get("grafana") or {}
    k8s_cfg = platform_payload.get("kubernetes") or {}
    namespace_cfg = platform_payload.get("namespaces") or {}
    storage_cfg = platform_payload.get("storage") or {}
    routing_cfg = platform_payload.get("toolRouting") or {}
    agent_cfg = backend_payload.get("agent") or {}
    ops_cfg = agent_cfg.get("ops") or {}
    backend_web_cfg = ops_cfg.get("webSearch") or {}
    backend_k8s_cfg = ops_cfg.get("kubernetes") or {}
    backend_registry_cfg = (backend_payload.get("registry") or {}).get("harbor") or {}
    backend_build_tools_cfg = ((backend_payload.get("registry") or {}).get("buildTools") or {})
    backend_storage_cfg = backend_payload.get("storage") or {}
    backend_namespace_cfg = backend_payload.get("namespaces") or {}
    model_download_cfg = backend_payload.get("modelDownload") or {}

    local_core_tools = set(_as_string_list(routing_cfg.get("localCoreTools"))) or set(DEFAULT_LOCAL_CORE_TOOLS)
    local_write_tools = set(_as_string_list(routing_cfg.get("localWriteTools"))) or set(DEFAULT_LOCAL_WRITE_TOOLS)

    allowed_domains = _as_string_list(web_cfg.get("allowedDomains"))
    if not allowed_domains:
        allowed_domains = _as_string_list(backend_web_cfg.get("allowedDomains"))

    seed_urls = _as_string_list(web_cfg.get("seedUrls"))

    web_enabled = bool(web_cfg.get("enabled", backend_web_cfg.get("enabled", False)))
    web_timeout = int(web_cfg.get("timeoutSeconds") or backend_web_cfg.get("timeoutSeconds") or 10)

    kubeconfig_raw = str(k8s_cfg.get("kubeconfigPath") or "").strip()
    if kubeconfig_raw:
        kubeconfig_path = str(
            _resolve_relative_path(
                kubeconfig_raw,
                base_dir=runtime_path.parent if runtime_path else repo_root,
            )
        )
    else:
        backend_kubeconfig_raw = str(backend_k8s_cfg.get("kubeconfigPath") or "").strip()
        if backend_kubeconfig_raw:
            kubeconfig_path = str(
                _resolve_relative_path(
                    backend_kubeconfig_raw,
                    base_dir=backend_path.parent if backend_path else repo_root,
                )
            )
        else:
            kubeconfig_path = str(os.getenv("KUBECONFIG") or "").strip()

    # Env override knobs (useful in CI/dev)
    if (os.getenv("CRATER_AGENT_LOCAL_WEB_SEARCH_ENABLED") or "").strip():
        raw = os.getenv("CRATER_AGENT_LOCAL_WEB_SEARCH_ENABLED", "").strip().lower()
        web_enabled = raw in ("1", "true", "yes", "on")
    env_domains = (os.getenv("CRATER_AGENT_LOCAL_WEB_SEARCH_ALLOWED_DOMAINS") or "").strip()
    if env_domains:
        allowed_domains = [d.strip().lower() for d in env_domains.split(",") if d.strip()]

    safety_cfg = k8s_cfg.get("safety") or {}
    backend_safety_cfg = backend_k8s_cfg.get("safety") or {}
    protected_namespaces = _as_string_list(safety_cfg.get("protectedNamespaces"))
    if not protected_namespaces:
        protected_namespaces = _as_string_list(backend_safety_cfg.get("protectedNamespaces"))
    if not protected_namespaces:
        protected_namespaces = list(DEFAULT_PROTECTED_NAMESPACES)

    protected_node_names = _as_string_list(safety_cfg.get("protectedNodeNames"))
    if not protected_node_names:
        protected_node_names = _as_string_list(backend_safety_cfg.get("protectedNodeNames"))

    blocked_command_patterns = _as_string_list(safety_cfg.get("blockedCommandPatterns"))
    if not blocked_command_patterns:
        blocked_command_patterns = _as_string_list(
            backend_safety_cfg.get("blockedCommandPatterns")
        )
    if not blocked_command_patterns:
        blocked_command_patterns = list(DEFAULT_BLOCKED_COMMAND_PATTERNS)

    config = PlatformRuntimeConfig(
        local_core_tools=local_core_tools,
        local_write_tools=local_write_tools,
        web_search_enabled=web_enabled,
        web_search_allowed_domains=allowed_domains,
        web_search_seed_urls=seed_urls,
        web_search_timeout_seconds=web_timeout,
        web_search_max_content_chars=int(web_cfg.get("maxContentChars") or 4_000),
        backend_url=str(settings.crater_backend_url or "").strip(),
        agent_service_url=str(agent_cfg.get("serviceURL") or "").strip(),
        platform_host=str(platform_payload.get("host") or backend_payload.get("host") or "").strip(),
        model_download_image=str(
            platform_payload.get("modelDownloadImage")
            or model_download_cfg.get("image")
            or ""
        ).strip(),
        prometheus_api=str(
            prometheus_cfg.get("baseURL")
            or platform_payload.get("prometheusAPI")
            or backend_payload.get("prometheusAPI")
            or ""
        ).strip(),
        prometheus_timeout_seconds=int(prometheus_cfg.get("timeoutSeconds") or 15),
        prometheus_headers=_as_string_dict(prometheus_cfg.get("headers")),
        prometheus_verify_tls=bool(prometheus_cfg.get("verifyTLS", True)),
        registry_server=str(
            registry_cfg.get("server") or backend_registry_cfg.get("server") or ""
        ).strip(),
        registry_project=str(
            registry_cfg.get("project") or backend_registry_cfg.get("project") or ""
        ).strip(),
        registry_username=str(
            registry_cfg.get("username") or backend_registry_cfg.get("user") or ""
        ).strip(),
        registry_password=str(
            registry_cfg.get("password") or backend_registry_cfg.get("password") or ""
        ).strip(),
        registry_verify_tls=bool(registry_cfg.get("verifyTLS", True)),
        registry_enabled=bool(
            registry_cfg.get("enabled", (backend_payload.get("registry") or {}).get("enable", False))
        ),
        build_proxy=_as_string_dict(
            registry_cfg.get("buildProxy")
            or backend_build_tools_cfg.get("proxyConfig")
        ),
        build_tool_images=_as_string_dict(
            registry_cfg.get("buildToolImages")
            or backend_build_tools_cfg.get("images")
        ),
        grafana=_as_string_dict(grafana_cfg),
        kubeconfig_path=kubeconfig_path,
        kube_context=str(k8s_cfg.get("context") or backend_k8s_cfg.get("context") or "").strip(),
        kube_namespace=str(
            k8s_cfg.get("namespace") or backend_k8s_cfg.get("namespace") or ""
        ).strip(),
        kubectl_bin=str(
            k8s_cfg.get("kubectlBin") or backend_k8s_cfg.get("kubectlBin") or "kubectl"
        ).strip() or "kubectl",
        kubectl_timeout_seconds=int(
            k8s_cfg.get("timeoutSeconds") or backend_k8s_cfg.get("timeoutSeconds") or 30
        ),
        k8s_protected_namespaces=protected_namespaces,
        k8s_protected_node_names=protected_node_names,
        k8s_blocked_command_patterns=blocked_command_patterns,
        namespaces={
            "job": str(namespace_cfg.get("job") or backend_namespace_cfg.get("job") or "").strip(),
            "image": str(namespace_cfg.get("image") or backend_namespace_cfg.get("image") or "").strip(),
        },
        storage_prefixes={
            "user": str(
                ((storage_cfg.get("prefix") or {}).get("user"))
                or ((backend_storage_cfg.get("prefix") or {}).get("user"))
                or ""
            ).strip(),
            "account": str(
                ((storage_cfg.get("prefix") or {}).get("account"))
                or ((backend_storage_cfg.get("prefix") or {}).get("account"))
                or ""
            ).strip(),
            "public": str(
                ((storage_cfg.get("prefix") or {}).get("public"))
                or ((backend_storage_cfg.get("prefix") or {}).get("public"))
                or ""
            ).strip(),
        },
        storage_pvcs={
            "readWriteMany": str(
                ((storage_cfg.get("pvc") or {}).get("readWriteMany"))
                or ((backend_storage_cfg.get("pvc") or {}).get("readWriteMany"))
                or ""
            ).strip(),
            "readOnlyMany": str(
                ((storage_cfg.get("pvc") or {}).get("readOnlyMany"))
                or ((backend_storage_cfg.get("pvc") or {}).get("readOnlyMany"))
                or ""
            ).strip(),
        },
        sources={
            "platform_runtime_config_path": str(runtime_path) if runtime_path else "",
            "backend_debug_config_path": str(backend_path) if backend_path else "",
        },
    )
    return config


def route_for_tool(runtime: PlatformRuntimeConfig, tool_name: str, *, default: str = "backend") -> str:
    """Decide where to execute a tool: 'local' or 'backend'.

    Env override:
      CRATER_AGENT_TOOL_ROUTE_<tool_name>=local|backend
      Example: CRATER_AGENT_TOOL_ROUTE_web_search=backend
    """

    normalized_default = "local" if str(default).strip().lower() == "local" else "backend"
    env_key = f"CRATER_AGENT_TOOL_ROUTE_{tool_name}"
    raw = str(os.getenv(env_key) or "").strip().lower()
    if raw in ("local", "backend"):
        return raw
    return normalized_default
