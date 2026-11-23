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
import { CancelledError, queryOptions } from '@tanstack/react-query'
import { getDefaultStore } from 'jotai'

import { ComboboxItem } from '@/components/form/combobox'

import { apiAdminGetKaniko } from '@/services/api/admin/imagepack'
import { ImageInfoResponse, apiUserGetKaniko } from '@/services/api/imagepack'
import { JobType, apiJTaskImageList } from '@/services/api/vcjob'

import { atomUserContext } from '@/utils/store'

export const buildDetailKey = (name: string, isAdmin: boolean, account?: string) =>
  [
    'imagepack',
    'get',
    isAdmin ? 'admin' : 'user',
    isAdmin ? null : (account ?? null),
    name,
  ] as const

export const queryBuildDetail = (name: string, isAdmin: boolean, account?: string) =>
  queryOptions({
    queryKey: buildDetailKey(name, isAdmin, account),
    queryFn: async ({ signal }) => {
      if (isAdmin) return apiAdminGetKaniko(name)

      const store = getDefaultStore()
      const context = store.get(atomUserContext)
      // Account switching can refetch the old query before React observes the new key.
      if (!account || context?.queue !== account) throw new CancelledError({ revert: true })

      const response = await apiUserGetKaniko(name, signal)
      if (store.get(atomUserContext) !== context) throw new CancelledError({ revert: true })
      return response
    },
    select: (res) => res.data,
    enabled: !!name && (isAdmin || !!account),
    retry: false,
    staleTime: 0,
  })

export const queryBaseImages = (type?: JobType) => {
  return queryOptions({
    queryKey: ['images', type],
    queryFn: () => apiJTaskImageList(type || JobType.Jupyter),
    select: (res) => {
      const items = Array.from(
        new Map(res.data.images.map((item) => [item.imageLink, item])).values()
      )
        // Filter out items with invalid or empty data
        .filter(
          (item) =>
            item &&
            item.imageLink &&
            item.description &&
            item.userInfo &&
            item.userInfo.nickname &&
            item.userInfo.username
        )
        // Sort by creation time, newest first
        .sort((a, b) => {
          // Adjust the field name if needed based on your data structure
          return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        })
        .map(
          (item) =>
            ({
              value: item.imageLink,
              label: `${item.description} (${item.imageLink}) [${item.userInfo.nickname} ${item.userInfo.username}]`,
              selectedLabel: item.description,
              detail: item,
              tags: item.tags,
            }) as ComboboxItem<ImageInfoResponse>
        )
      return items
    },
  })
}
