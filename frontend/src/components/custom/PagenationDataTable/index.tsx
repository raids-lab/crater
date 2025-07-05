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

// i18n-processed-v1.1.0 (no translatable strings)
import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  SortingState,
  VisibilityState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { DataTableToolbar, DataTableToolbarConfig } from './DataTableToolbar'
import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'
import DataTableCard from './DataTableCard'

interface DataTableProps<TData, TValue> extends React.HTMLAttributes<HTMLDivElement> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  toolbarConfig: DataTableToolbarConfig
  loading: boolean
  onPaginationChange: OnChangeFn<PaginationState>
  rowCount?: number
  pagination: PaginationState
  className?: string
}

export function DataTable<TData, TValue>({
  columns,
  data,
  toolbarConfig,
  loading,
  onPaginationChange,
  rowCount,
  pagination,
  children,
  className,
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [sorting, setSorting] = useState<SortingState>([
    // { id: "createdAt", desc: true },
  ])

  const [cachedRowCount, setCachedRowCount] = useState(rowCount)

  useEffect(() => {
    if (rowCount && rowCount !== cachedRowCount) setCachedRowCount(rowCount)
  }, [rowCount, cachedRowCount])

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
    },
    // pagination
    manualPagination: true,
    onPaginationChange,
    rowCount: cachedRowCount,
    // shadcn example props
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  })

  return (
    <div className={cn('col-span-3 flex flex-col gap-2', className)}>
      <DataTableToolbar table={table} config={toolbarConfig}>
        {children}
      </DataTableToolbar>
      <DataTableCard table={table} loading={loading} columns={columns} />
    </div>
  )
}
