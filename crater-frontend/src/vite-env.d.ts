/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_APP_VERSION: string

  // URLs
  readonly VITE_HOST: string
  readonly VITE_SERVER_PROXY_DOMAIN: string
  readonly VITE_SERVER_PROXY_PREFIX: string
  readonly VITE_SERVER_PROXY_STORAGE: string
  readonly VITE_WEBSITE_BASE_URL: string

  // Grafana URLs
  readonly VITE_GRAFANA_NODE: string
  readonly VITE_GRAFANA_POD_MONITOR: string
  readonly VITE_GRAFANA_JOB_MONITOR: string
  readonly VITE_GRAFANA_JOB_GPU_MONITOR: string
  readonly VITE_GRAFANA_GPU_DASHBOARD: string
  readonly VITE_GRAFANA_USER_GPU_DASHBOARD: string
  readonly VITE_GRAFANA_OVERVIEW: string
  readonly VITE_GRAFANA_SCHEDULE: string
  readonly VITE_GRAFANA_NETWORK: string

  // Development
  readonly VITE_USE_MSW: string
  readonly VITE_SET_DOCS_AS_HOME: string
  readonly VITE_TANSTACK_QUERY_DEVTOOLS: string
  readonly VITE_TANSTACK_ROUTER_DEVTOOLS: string
}
