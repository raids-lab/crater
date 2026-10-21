// Copyright 2026 The Crater Project Team, RAIDS-Lab
import type { IPage, IResponse } from '@/services/types'

// Keep complete-list selectors on the same paginated API as management tables.
export async function fetchAllPages<T>(
  fetchPage: (page: number) => Promise<IResponse<IPage<T>>>
): Promise<IResponse<T[]>> {
  const items: T[] = []
  for (let page = 1; ; page += 1) {
    const response = await fetchPage(page)
    items.push(...response.data.items)
    if (response.data.items.length === 0 || items.length >= response.data.total) {
      return { ...response, data: items }
    }
  }
}
