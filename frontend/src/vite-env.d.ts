/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_APP_VERSION: string;

  // URLs
  readonly VITE_HOST: string;
  readonly VITE_API_BASE_URL: string;
  readonly VITE_API_VERSION: string;
  readonly VITE_WEBSITE_BASE_URL: string;

  // Grafana URLs
  readonly VITE_GRAFANA_NODE: string;
  readonly VITE_GRAFANA_POD_MONITOR: string;
  readonly VITE_GRAFANA_JOB_MONITOR: string;
  readonly VITE_GRAFANA_JOB_GPU_MONITOR: string;
  readonly VITE_GRAFANA_GPU_DASHBOARD: string;
  readonly VITE_GRAFANA_OVERVIEW: string;
  readonly VITE_GRAFANA_SCHEDULE: string;
  readonly VITE_GRAFANA_NETWORK: string;

  // Development
  readonly VITE_USE_MSW: string;
}
