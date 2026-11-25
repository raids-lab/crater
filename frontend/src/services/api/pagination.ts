// Copyright 2026 The Crater Project Team, RAIDS-Lab
import type { IPage, IResponse } from '@/services/types'

const MAX_PAGE_REQUESTS = 10000

// Keep complete-list selectors on the same paginated API as management tables.
export async function fetchAllPages<T>(
  fetchPage: (page: number) => Promise<IResponse<IPage<T>>>
): Promise<IResponse<T[]>> {
  const items: T[] = []
  let pageSize: number | undefined
  for (let page = 1; page <= MAX_PAGE_REQUESTS; page += 1) {
    const response = await fetchPage(page)
    const data = response.data
    if (data.page !== page) {
      throw new Error(`Pagination requested page ${page}, received page ${data.page}`)
    }
    if (
      !Number.isSafeInteger(data.page_size) ||
      data.page_size < 1 ||
      (pageSize !== undefined && data.page_size !== pageSize)
    ) {
      throw new Error(`Pagination has invalid or changing page_size on page ${page}`)
    }
    pageSize = data.page_size
    if (
      !Number.isSafeInteger(data.total) ||
      data.total < 0 ||
      !Array.isArray(data.items) ||
      data.items.length > pageSize
    ) {
      throw new Error(`Pagination has invalid total or item count on page ${page}`)
    }
    const collected = items.length + data.items.length
    if (collected > data.total) {
      throw new Error(
        `Pagination collected ${collected} items, exceeding reported total ${data.total} on page ${page}`
      )
    }
    items.push(...data.items)
    if (collected === data.total) {
      return { ...response, data: items }
    }
    if (data.items.length === 0) {
      throw new Error(
        `Pagination ended on empty page ${page} after collecting ${collected} of ${data.total} reported items`
      )
    }
    // A short nonterminal offset page would leave a gap before the next page.
    if (data.items.length < pageSize) {
      throw new Error(
        `Pagination ended on short page ${page} after collecting ${collected} of ${data.total} reported items`
      )
    }
  }
  throw new Error(`Pagination exceeded the maximum of ${MAX_PAGE_REQUESTS} page requests`)
}
