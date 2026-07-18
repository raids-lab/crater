import type { ColumnFiltersState, PaginationState, SortingState } from '@tanstack/react-table'

export interface RemoteTableParams {
  page: number
  page_size: number
  sort?: string
  search?: string
  filters: Record<string, string[]>
}

interface RemoteTableState {
  pagination: PaginationState
  sorting: SortingState
  search: string
  filters: ColumnFiltersState
}

export function normalizeRemoteTableParams(state: RemoteTableState): RemoteTableParams {
  const sort = state.sorting
    .slice(0, 3)
    .map(({ id, desc }) => `${desc ? '-' : ''}${id}`)
    .join(',')
  const filters = Object.fromEntries(
    state.filters
      .map(({ id, value }) => [id, normalizeFilterValue(value)] as const)
      .filter(([, values]) => values.length > 0)
      .sort(([left], [right]) => left.localeCompare(right))
  )
  const search = state.search.trim()

  return {
    page: state.pagination.pageIndex + 1,
    page_size: state.pagination.pageSize,
    ...(sort ? { sort } : {}),
    ...(search ? { search } : {}),
    filters,
  }
}

export function buildRemoteSearchParams(params: RemoteTableParams): URLSearchParams {
  const searchParams = new URLSearchParams()
  searchParams.set('page', String(params.page))
  searchParams.set('page_size', String(params.page_size))
  if (params.sort) searchParams.set('sort', params.sort)
  if (params.search) searchParams.set('search', params.search)
  appendFilters(searchParams, params.filters)
  return searchParams
}

export function buildFacetSearchParams(params: RemoteTableParams): URLSearchParams {
  const searchParams = new URLSearchParams()
  if (params.search) searchParams.set('search', params.search)
  appendFilters(searchParams, params.filters)
  return searchParams
}

export function buildRemoteQueryKey(resource: string, params: RemoteTableParams) {
  return ['remote-list', resource, params] as const
}

export function buildFacetQueryKey(resource: string, params: RemoteTableParams) {
  return [
    'remote-list-facets',
    resource,
    { search: params.search ?? '', filters: params.filters },
  ] as const
}

export function getLastPageIndex(total: number, pageSize: number): number {
  return Math.max(Math.ceil(total / pageSize) - 1, 0)
}

export function shouldClearRemoteSelection(
  previous: RemoteTableParams,
  next: RemoteTableParams
): boolean {
  return JSON.stringify(previous) !== JSON.stringify(next)
}

function normalizeFilterValue(value: unknown): string[] {
  const values = Array.isArray(value) ? value : value === undefined || value === '' ? [] : [value]
  return values.map(String).filter(Boolean).sort()
}

function appendFilters(searchParams: URLSearchParams, filters: Record<string, string[]>) {
  for (const key of Object.keys(filters).sort()) {
    for (const value of filters[key]) {
      searchParams.append(key, value)
    }
  }
}
