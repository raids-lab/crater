import type {
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  SortingState,
  Updater,
} from '@tanstack/react-table'
import { useCallback, useMemo } from 'react'
import { useDebounceValue, useLocalStorage } from 'usehooks-ts'

import { normalizeRemoteTableParams } from '@/components/query-table/remote-state'

const SEARCH_DEBOUNCE_MS = 300

interface RemoteTableInitialState {
  pageSize?: number
  sorting?: SortingState
  columnFilters?: ColumnFiltersState
  search?: string
}

function resolveUpdater<T>(updater: Updater<T>, current: T): T {
  return typeof updater === 'function' ? (updater as (value: T) => T)(current) : updater
}

export default function useRemoteTableState(
  storageKey: string,
  initialState: RemoteTableInitialState = {}
) {
  const [pageIndex, setPageIndex] = useLocalStorage(`${storageKey}-page-index`, 0)
  const [pageSize, setPageSize] = useLocalStorage(
    `${storageKey}-page-size`,
    initialState.pageSize ?? 10
  )
  const [sorting, persistSorting] = useLocalStorage<SortingState>(
    `${storageKey}-sorting`,
    initialState.sorting ?? []
  )
  const [columnFilters, persistColumnFilters] = useLocalStorage<ColumnFiltersState>(
    `${storageKey}-column-filters`,
    initialState.columnFilters ?? []
  )
  const [search, persistSearch] = useLocalStorage(`${storageKey}-search`, initialState.search ?? '')
  const [debouncedSearch, setDebouncedSearch] = useDebounceValue(search, SEARCH_DEBOUNCE_MS)

  const pagination = useMemo(() => ({ pageIndex, pageSize }), [pageIndex, pageSize])
  const resetPage = useCallback(() => setPageIndex(0), [setPageIndex])

  const setPagination = useCallback<OnChangeFn<PaginationState>>(
    (updater) => {
      const next = resolveUpdater(updater, pagination)
      setPageIndex(next.pageIndex)
      setPageSize(next.pageSize)
    },
    [pagination, setPageIndex, setPageSize]
  )

  const setSorting = useCallback<OnChangeFn<SortingState>>(
    (updater) => {
      persistSorting((current) => resolveUpdater(updater, current))
      resetPage()
    },
    [persistSorting, resetPage]
  )

  const setColumnFilters = useCallback<OnChangeFn<ColumnFiltersState>>(
    (updater) => {
      persistColumnFilters((current) => resolveUpdater(updater, current))
      resetPage()
    },
    [persistColumnFilters, resetPage]
  )

  const setSearch = useCallback(
    (value: string) => {
      persistSearch(value)
      setDebouncedSearch(value)
      resetPage()
    },
    [persistSearch, resetPage, setDebouncedSearch]
  )

  const params = useMemo(
    () =>
      normalizeRemoteTableParams({
        pagination,
        sorting,
        search: debouncedSearch,
        filters: columnFilters,
      }),
    [columnFilters, debouncedSearch, pagination, sorting]
  )

  return {
    pagination,
    sorting,
    columnFilters,
    search,
    params,
    setPagination,
    setSorting,
    setColumnFilters,
    setSearch,
  }
}

export type RemoteTableState = ReturnType<typeof useRemoteTableState>
