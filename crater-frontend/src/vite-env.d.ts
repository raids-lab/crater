/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_API_VERSION: string;
  readonly VITE_API_BASE_URL: string;
  readonly VITE_GRAFANA_POD_MEMORY: string;
  readonly VITE_GRAFANA_JOB_MONITOR: string;
  readonly VITE_GRAFANA_K8S_VGPU_SCHEDULER_DASHBOARD: string;
}
