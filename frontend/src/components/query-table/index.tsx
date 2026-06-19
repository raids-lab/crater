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
  PaginationState,
  SortingState,
  Updater,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { GridIcon } from 'lucide-react'
import React, { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocalStorage } from 'usehooks-ts'

import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import PageTitle from '@/components/layout/page-title'

import usePaginationWithStorage from '@/hooks/use-pagination-with-storage'

import { cn } from '@/lib/utils'

import { DataTablePagination, MultipleHandler } from './pagination'
import { DataTableToolbar, DataTableToolbarConfig } from './toolbar'

const resolveUpdater = <TState,>(updater: Updater<TState>, state: TState) => {
  return typeof updater === 'function' ? (updater as (state: TState) => TState)(state) : updater
}

/**
 * ServerDrivenConfig switches the table to manual pagination/sorting/filtering
 * mode and lets a parent component own the request shape. Pass the current
 * total row count (so the pager can compute pageCount) and optional facets to
 * preserve the "Status (12)" badges that the toolbar shows in client mode.
 *
 * Facets are keyed by column id, then by row value, e.g.
 * `{ status: { Running: 12 }, jobType: { jupyter: 30 } }`. Missing keys
 * gracefully degrade to "no count" instead of breaking layout.
 */
export interface ServerDrivenConfig {
  total: number
  facets?: Record<string, Record<string, number>>
  pagination: PaginationState
  setPagination: OnChangeFn<PaginationState>
  sorting: SortingState
  setSorting: OnChangeFn<SortingState>
  columnFilters: ColumnFiltersState
  setColumnFilters: OnChangeFn<ColumnFiltersState>
  globalFilter: string
  setGlobalFilter: OnChangeFn<string>
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
  /**
   * When provided, the table runs in server-driven mode: pagination, sorting,
   * filtering and global search become controlled inputs the parent owns, and
   * the parent is expected to refetch data when those inputs change.
   */
  serverDriven?: ServerDrivenConfig
}

export function DataTable<TData, TValue>({
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
  serverDriven,
}: DataTableProps<TData, TValue>) {
  const { t } = useTranslation()
  const [rowSelection, setRowSelection] = useState({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(initialColumnVisibility)
  const [localColumnFilters, setLocalColumnFilters] = useLocalStorage<ColumnFiltersState>(
    `${storageKey}-column-filters`,
    []
  )
  const [localSorting, setLocalSorting] = useState<SortingState>([])
  const [localGlobalFilter, setLocalGlobalFilter] = useState('')
  const { data: queryData, isLoading, dataUpdatedAt, refetch } = query
  const updatedAt = new Date(dataUpdatedAt).toLocaleString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
  const [localPagination, setLocalPagination] = usePaginationWithStorage(storageKey)

  // When serverDriven is provided, the parent owns state and the table runs in
  // manual mode. Otherwise we fall back to the existing client-side pipeline.
  const columnFilters = serverDriven ? serverDriven.columnFilters : localColumnFilters
  const setColumnFilters = serverDriven ? serverDriven.setColumnFilters : setLocalColumnFilters
  const sorting = serverDriven ? serverDriven.sorting : localSorting
  const setSorting = serverDriven ? serverDriven.setSorting : setLocalSorting
  const globalFilter = serverDriven ? serverDriven.globalFilter : localGlobalFilter
  const setGlobalFilter = serverDriven ? serverDriven.setGlobalFilter : setLocalGlobalFilter
  const pagination = serverDriven ? serverDriven.pagination : localPagination
  const setPagination = serverDriven ? serverDriven.setPagination : setLocalPagination

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
    [resetPageIndex, setSorting]
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
    [resetPageIndex, setGlobalFilter]
  )

  // Build a custom faceted-unique-values resolver from server-supplied counts
  // so the toolbar continues to render "Running (12)" style badges. When the
  // server doesn't report a column's facets, fall back to an empty map (the
  // dropdown still works, only the trailing count disappears).
  const serverFacetsResolver = useMemo(() => {
    if (!serverDriven?.facets) return undefined
    const facets = serverDriven.facets
    return <T,>(_table: import('@tanstack/react-table').Table<T>, columnId: string) =>
      () => {
        const map = new Map<string, number>()
        const bucket = facets[columnId]
        if (!bucket) return map
        for (const [value, count] of Object.entries(bucket)) {
          map.set(value, count)
        }
        return map
      }
  }, [serverDriven])

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
    manualPagination: !!serverDriven,
    manualSorting: !!serverDriven,
    manualFiltering: !!serverDriven,
    pageCount: serverDriven
      ? Math.max(1, Math.ceil(serverDriven.total / Math.max(pagination.pageSize, 1)))
      : undefined,
    rowCount: serverDriven ? serverDriven.total : undefined,
    onRowSelectionChange: setRowSelection,
    onSortingChange: handleSortingChange,
    onColumnFiltersChange: handleColumnFiltersChange,
    onGlobalFilterChange: handleGlobalFilterChange,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    // Filtering / pagination / sorting row models are unused in manual mode but
    // keeping them around lets the same component stay drop-in for client pages.
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: serverFacetsResolver ?? getFacetedUniqueValues(),
  })

  const pageCount = table.getPageCount()
  React.useEffect(() => {
    if (serverDriven) return
    const lastPageIndex = Math.max(pageCount - 1, 0)
    if (pagination.pageIndex > lastPageIndex) {
      setPagination((state) => ({ ...state, pageIndex: lastPageIndex }))
    }
  }, [pageCount, pagination.pageIndex, setPagination, serverDriven])

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {info && (
        <PageTitle title={info.title} description={info.description}>
          {children}
        </PageTitle>
      )}
      {briefChildren && <>{briefChildren}</>}
      {toolbarConfig && (
        <DataTableToolbar table={table} config={toolbarConfig} isLoading={query.isLoading}>
          {!info && <>{children}</>}
        </DataTableToolbar>
      )}
      <Card className="overflow-hidden rounded-md p-0 shadow-xs">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead
                        key={header.id}
                        colSpan={header.colSpan}
                        className="text-muted-foreground px-4"
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    )
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id} data-state={row.getIsSelected() && 'selected'}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className="pl-4">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={table.getAllColumns().length} className="h-60">
                        <LoadingCircleIcon />
                      </TableCell>
                    </TableRow>
                  ) : (
                    <TableRow>
                      <TableCell
                        colSpan={table.getAllColumns().length}
                        className="text-muted-foreground/85 h-60 text-center hover:bg-transparent"
                      >
                        <div className="flex flex-col items-center justify-center py-16">
                          <div className="bg-muted mb-4 rounded-full p-3">
                            <GridIcon className="h-6 w-6" />
                          </div>
                          <p className="select-none">
                            {withI18n ? t('dataTable.noData') : '暂无数据'}
                          </p>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <DataTablePagination
        table={table}
        refetch={() => void refetch()}
        updatedAt={updatedAt}
        multipleHandlers={multipleHandlers}
      />
    </div>
  )
}
