/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
export type DataListOwnerFilter = 'all' | 'mine' | 'others'
export type DataListSortField = 'createdAt' | 'mountCount'
export type DataListSortDirection = 'ascending' | 'descending'

export interface DataListRemoteQuery {
  page: number
  pageSize: number
  search: string
  owner: DataListOwnerFilter
  tag: string
  sortField: DataListSortField
  sortDirection: DataListSortDirection
}

export type DataListRemoteChange =
  | { type: 'page'; page: number }
  | { type: 'pageSize'; pageSize: number }
  | { type: 'search'; search: string }
  | { type: 'owner'; owner: DataListOwnerFilter }
  | { type: 'tag'; tag: string }
  | { type: 'sortField'; sortField: DataListSortField }
  | { type: 'sortDirection'; sortDirection: DataListSortDirection }

// Applies one user interaction while preserving the remaining query state.
export function reduceDataListRemoteQuery(
  query: DataListRemoteQuery,
  change: DataListRemoteChange
): DataListRemoteQuery {
  switch (change.type) {
    case 'page':
      return { ...query, page: change.page }
    case 'pageSize':
      return { ...query, page: 1, pageSize: change.pageSize }
    case 'search':
      return { ...query, page: 1, search: change.search }
    case 'owner':
      return { ...query, page: 1, owner: change.owner }
    case 'tag':
      return { ...query, page: 1, tag: change.tag }
    case 'sortField':
      return { ...query, page: 1, sortField: change.sortField }
    case 'sortDirection':
      return { ...query, page: 1, sortDirection: change.sortDirection }
  }
}

// Moves an out-of-range page to the last valid page after the total changes.
export function clampDataListRemotePage(
  query: DataListRemoteQuery,
  total: number
): DataListRemoteQuery {
  const totalPages = Math.max(1, Math.ceil(total / query.pageSize))
  return query.page > totalPages ? { ...query, page: totalPages } : query
}
