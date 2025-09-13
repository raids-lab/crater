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

export interface AppConfig {
  version: string
  url: {
    apiBase: string
    document: string
  }
  grafana: {
    overview: {
      main: string
      schedule: string
      network: string
    }
    node: {
      basic: string
      nvidia: string
    }
    job: {
      basic: string
      nvidia: string
      pod: string
    }
    user: {
      nvidia: string
    }
  }
}
