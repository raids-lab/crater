/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_API_VERSION: string;
  readonly VITE_HOST: string;
  readonly VITE_API_BASE_URL: string;
  readonly VITE_GRAFANA_NODE: string;
  readonly VITE_GRAFANA_POD_MONITOR: string;
  readonly VITE_GRAFANA_JOB_MONITOR: string;
  readonly VITE_GRAFANA_GPU_DASHBOARD: string;
  readonly VITE_GRAFANA_OVERVIEW: string;
}
