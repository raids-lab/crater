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
// i18n-processed-v1.1.0
// Modified code
import { UseQueryResult } from '@tanstack/react-query'
import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  SortingState,
  Updater,
  VisibilityState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import React, { useMemo, useState } from 'react'
import { useLocalStorage } from 'usehooks-ts'

import { Checkbox } from '@/components/ui/checkbox'

import usePaginationWithStorage from '@/hooks/use-pagination-with-storage'

import { type MultipleHandler } from './pagination'
import { DataTableShell } from './shell'
import { type DataTableToolbarConfig } from './toolbar'

const resolveUpdater = <TState,>(updater: Updater<TState>, state: TState) => {
  return typeof updater === 'function' ? (updater as (state: TState) => TState)(state) : updater
}

interface DataTableProps<TData, TValue> extends React.HTMLAttributes<HTMLDivElement> {
  info?: {
    title?: string
    description: string
  }
  storageKey: string
  query: UseQueryResult<TData[], Error>
  columns: ColumnDef<TData, TValue>[]
  toolbarConfig?: DataTableToolbarConfig
  multipleHandlers?: MultipleHandler<TData>[]
  briefChildren?: React.ReactNode
  withI18n?: boolean
  className?: string
  initialColumnVisibility?: VisibilityState
}

export function LocalDataTable<TData, TValue>({
  info,
  storageKey,
  query,
  columns,
  toolbarConfig,
  multipleHandlers,
  children,
  briefChildren,
  withI18n = false,
  className,
  initialColumnVisibility = {},
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(initialColumnVisibility)
  const [columnFilters, setColumnFilters] = useLocalStorage<ColumnFiltersState>(
    `${storageKey}-column-filters`,
    []
  )
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')
  const { data: queryData, error, isError, isLoading, dataUpdatedAt, refetch } = query
  const [pagination, setPagination] = usePaginationWithStorage(storageKey)

  const data = useMemo(() => {
    if (!queryData || isLoading) return []
    return queryData
  }, [queryData, isLoading])

  const columnsWithSelection = useMemo(() => {
    if (!multipleHandlers || !columns || multipleHandlers.length === 0) {
      return columns
    }
    return [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            hidden={table.getRowModel().rows.length === 0}
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      ...columns,
    ]
  }, [columns, multipleHandlers])

  const resetPageIndex = React.useCallback(() => {
    setPagination((state) => ({ ...state, pageIndex: 0 }))
  }, [setPagination])

  const handleSortingChange = React.useCallback<OnChangeFn<SortingState>>(
    (updater) => {
      setSorting((state) => resolveUpdater(updater, state))
      resetPageIndex()
    },
    [resetPageIndex]
  )

  const handleColumnFiltersChange = React.useCallback<OnChangeFn<ColumnFiltersState>>(
    (updater) => {
      setColumnFilters((state) => resolveUpdater(updater, state))
      resetPageIndex()
    },
    [resetPageIndex, setColumnFilters]
  )

  const handleGlobalFilterChange = React.useCallback<OnChangeFn<string>>(
    (updater) => {
      setGlobalFilter((state) => resolveUpdater(updater, state))
      resetPageIndex()
    },
    [resetPageIndex]
  )

  const table = useReactTable({
    data: data,
    columns: columnsWithSelection,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      globalFilter,
      pagination,
    },
    enableRowSelection: true,
    autoResetPageIndex: false,
    onRowSelectionChange: setRowSelection,
    onSortingChange: handleSortingChange,
    onColumnFiltersChange: handleColumnFiltersChange,
    onGlobalFilterChange: handleGlobalFilterChange,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  })

  const pageCount = table.getPageCount()
  React.useEffect(() => {
    const lastPageIndex = Math.max(pageCount - 1, 0)
    if (pagination.pageIndex > lastPageIndex) {
      setPagination((state) => ({ ...state, pageIndex: lastPageIndex }))
    }
  }, [pageCount, pagination.pageIndex, setPagination])

  return (
    <DataTableShell
      table={table}
      info={info}
      toolbarConfig={toolbarConfig}
      multipleHandlers={multipleHandlers}
      briefChildren={briefChildren}
      withI18n={withI18n}
      className={className}
      isLoading={isLoading}
      isError={isError}
      errorMessage={error?.message}
      refetch={() => void refetch()}
      dataUpdatedAt={dataUpdatedAt}
    >
      {children}
    </DataTableShell>
  )
}

export const DataTable = LocalDataTable
