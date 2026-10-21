// Copyright 2026 The Crater Project Team, RAIDS-Lab
import type { IPage, IResponse } from '@/services/types'

const MAX_PAGE_REQUESTS = 10000

// Keep complete-list selectors on the same paginated API as management tables.
export async function fetchAllPages<T>(
  fetchPage: (page: number) => Promise<IResponse<IPage<T>>>
): Promise<IResponse<T[]>> {
  const items: T[] = []
  for (let page = 1; page <= MAX_PAGE_REQUESTS; page += 1) {
    const response = await fetchPage(page)
    items.push(...response.data.items)
    if (items.length >= response.data.total) {
      return { ...response, data: items }
    }
    if (response.data.items.length === 0) {
      throw new Error(
        `Pagination ended on empty page ${page} after collecting ${items.length} of ${response.data.total} reported items`
      )
    }
  }
  throw new Error(`Pagination exceeded the maximum of ${MAX_PAGE_REQUESTS} page requests`)
}
