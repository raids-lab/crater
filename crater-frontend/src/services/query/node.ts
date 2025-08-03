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
import { queryOptions } from '@tanstack/react-query'

import { NodeRole, apiGetNodes } from '@/services/api/cluster'

export const queryNodes = (onlyWorker?: boolean) => {
  return queryOptions({
    queryKey: ['overview', 'nodes'],
    queryFn: apiGetNodes,
    select: (res) =>
      res.data
        .sort((a, b) => a.name.localeCompare(b.name))
        .sort((a, b) => {
          // 按照 vendor 排序，优先 hygon => shenwei => yitian => 空字符串
          const vendorOrder = ['hygon', 'shenwei', 'yitian', '']
          return vendorOrder.indexOf(a.vendor) - vendorOrder.indexOf(b.vendor)
        })
        .filter((x) => !onlyWorker || x.role === NodeRole.Worker),
  })
}
