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

import { configAtom, initializeConfig } from '@/utils/store/config'
import { useQuery } from '@tanstack/react-query'
import { useAtom } from 'jotai'
import { useEffect } from 'react'

/**
 * useConfigLoader
 * @description 监听配置变化，更新配置
 */
const useConfigLoader = () => {
  const [appConfig, setAppConfig] = useAtom(configAtom)

  const { data } = useQuery({
    queryKey: ['appConfig'],
    queryFn: initializeConfig,
  })

  useEffect(() => {
    if (data) {
      // check if appConfig deep equal to data
      const isEqual = JSON.stringify(appConfig) === JSON.stringify(data)
      if (!isEqual) {
        setAppConfig(data)
        // refresh page if config changed
        window.location.reload()
      }
    }
  }, [appConfig, data, setAppConfig])
}

export default useConfigLoader
