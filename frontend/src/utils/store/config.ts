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
// src/atoms/configAtom.ts
import { atom } from 'jotai'
import { atomWithStorage } from 'jotai/utils'

import { AppConfig } from './type'

// 添加配置存储 key
export const CONFIG_STORAGE_KEY = 'app_config'
export const CONFIG_VERSION_KEY = 'app_config_version'

export const configAtom = atomWithStorage<AppConfig>(
  CONFIG_STORAGE_KEY,
  {
    version: '',
    url: { apiBase: '', document: '' },
    grafana: {
      overview: { main: '', schedule: '', network: '' },
      node: { basic: '', nvidia: '' },
      job: { basic: '', nvidia: '', pod: '' },
      user: { nvidia: '' },
    },
  },
  undefined,
  {
    getOnInit: true,
  }
)

// 创建异步原子来加载配置
export const apiGetConfig = async () => {
  let config: AppConfig
  // 首先尝试从 config.json 读取配置
  const response = await fetch('/config.json')
  if (!response.ok) {
    throw new Error('Failed to load config')
  }
  config = await response.json()

  // 如果是开发环境，用环境变量覆盖部分配置
  if (import.meta.env.MODE === 'development') {
    config = {
      ...config,
      version: import.meta.env.VITE_APP_VERSION,
      url: {
        ...config.url,
        document: import.meta.env.VITE_DOCS_BASE_URL,
      },
    } as AppConfig
  }

  return config
}

// 派生原子

export const configAPIBaseAtom = atom((get) => get(configAtom).url.apiBase)

export const configAppVersionAtom = atom((get) => get(configAtom).version)

export const configUrlWebsiteBaseAtom = atom((get) => get(configAtom).url.document)

// 派生原子 - 按类别导出
export const configGrafanaOverviewAtom = atom((get) => get(configAtom).grafana.overview)
export const configGrafanaNodeAtom = atom((get) => get(configAtom).grafana.node)
export const configGrafanaJobAtom = atom((get) => get(configAtom).grafana.job)
export const configGrafanaUserAtom = atom((get) => get(configAtom).grafana.user)
