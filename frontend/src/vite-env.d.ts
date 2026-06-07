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
  // Version information
  readonly VITE_APP_VERSION: string
  readonly VITE_APP_COMMIT_SHA: string
  readonly VITE_APP_BUILD_TYPE: string
  readonly VITE_APP_BUILD_TIME: string

  // URLs
  readonly VITE_HOST: string
  readonly VITE_API_PRIFIX: string
  readonly VITE_SERVER_PROXY_BACKEND: string
  readonly VITE_SERVER_PROXY_STORAGE: string
  readonly VITE_DOCS_BASE_URL: string

  // Development
  readonly VITE_USE_MSW: string
  readonly VITE_SET_DOCS_AS_HOME: string
  readonly VITE_TANSTACK_QUERY_DEVTOOLS: string
  readonly VITE_TANSTACK_ROUTER_DEVTOOLS: string
}
