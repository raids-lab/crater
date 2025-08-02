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

import { Resource, apiResourceList } from '@/services/api/resource'

export const queryResources = <T = Resource>(
  withVendorDomain: boolean,
  filter?: (resource: Resource) => boolean,
  mapper?: (resource: Resource) => T
) => {
  return queryOptions({
    queryKey: ['resources', 'list', withVendorDomain],
    queryFn: () => apiResourceList(withVendorDomain),
    select: (res) => {
      return res.data
        .sort((a, b) => {
          return a.name.localeCompare(b.name)
        })
        .filter((resource) => {
          if (!filter) return true
          return filter(resource)
        })
        .map((resource) => {
          return mapper ? mapper(resource) : (resource as T)
        })
    },
  })
}
