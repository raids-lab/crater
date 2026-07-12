import type { UseQueryResult } from '@tanstack/react-query'
import type { ColumnDef, OnChangeFn, VisibilityState } from '@tanstack/react-table'
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import type { HTMLAttributes } from 'react'
import { useEffect, useMemo, useRef, useState } from 'react'

import { Checkbox } from '@/components/ui/checkbox'

import type { IPage } from '@/services/types'

import type { RemoteTableState } from '@/hooks/use-remote-table-state'

import { type MultipleHandler } from './pagination'
import { getLastPageIndex, shouldClearRemoteSelection } from './remote-state'
import { DataTableShell } from './shell'
import { type DataTableToolbarConfig } from './toolbar'

interface RemoteDataTableProps<TData, TValue> extends HTMLAttributes<HTMLDivElement> {
  info?: {
    title?: string
    description: string
  }
  query: UseQueryResult<IPage<TData>, Error>
  state: RemoteTableState
  columns: ColumnDef<TData, TValue>[]
  getRowId: (row: TData) => string
  toolbarConfig?: DataTableToolbarConfig
  multipleHandlers?: MultipleHandler<TData>[]
  briefChildren?: React.ReactNode
  withI18n?: boolean
  initialColumnVisibility?: VisibilityState
}

export function RemoteDataTable<TData, TValue>({
  info,
  query,
  state,
  columns,
  getRowId,
  toolbarConfig,
  multipleHandlers,
  children,
  briefChildren,
  withI18n = false,
  className,
  initialColumnVisibility = {},
}: RemoteDataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(initialColumnVisibility)
  const data = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const pageIndex = state.pagination.pageIndex
  const pageSize = state.pagination.pageSize
  const setPagination = state.setPagination
  const previousParams = useRef(state.params)

  const columnsWithSelection = useMemo(() => {
    if (!multipleHandlers?.length) return columns
    return [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(Boolean(value))}
            hidden={table.getRowModel().rows.length === 0}
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(Boolean(value))}
          />
        ),
        enableSorting: false,
        enableHiding: false,
      } satisfies ColumnDef<TData>,
      ...columns,
    ]
  }, [columns, multipleHandlers])

  const handleGlobalFilterChange: OnChangeFn<string> = (updater) => {
    const next = typeof updater === 'function' ? updater(state.search) : updater
    state.setSearch(next)
  }

  const table = useReactTable({
    data,
    columns: columnsWithSelection,
    getRowId,
    pageCount: Math.ceil(total / state.pagination.pageSize),
    state: {
      pagination: state.pagination,
      sorting: state.sorting,
      columnFilters: state.columnFilters,
      globalFilter: state.search,
      columnVisibility,
      rowSelection,
    },
    enableRowSelection: Boolean(multipleHandlers?.length),
    enableMultiSort: true,
    maxMultiSortColCount: 3,
    autoResetPageIndex: false,
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    onPaginationChange: state.setPagination,
    onSortingChange: state.setSorting,
    onColumnFiltersChange: state.setColumnFilters,
    onGlobalFilterChange: handleGlobalFilterChange,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    getCoreRowModel: getCoreRowModel(),
  })

  useEffect(() => {
    if (shouldClearRemoteSelection(previousParams.current, state.params)) {
      setRowSelection({})
    }
    previousParams.current = state.params
  }, [state.params])

  useEffect(() => {
    if (!query.data || query.isError || query.isFetching || query.isPlaceholderData) return
    const lastPageIndex = getLastPageIndex(query.data.total, pageSize)
    if (pageIndex > lastPageIndex) {
      setPagination((current) => ({ ...current, pageIndex: lastPageIndex }))
    }
  }, [
    pageIndex,
    pageSize,
    query.data,
    query.isError,
    query.isFetching,
    query.isPlaceholderData,
    setPagination,
  ])

  return (
    <DataTableShell
      table={table}
      info={info}
      toolbarConfig={toolbarConfig}
      multipleHandlers={multipleHandlers}
      briefChildren={briefChildren}
      withI18n={withI18n}
      className={className}
      isLoading={query.isLoading}
      isError={query.isError}
      errorMessage={query.error?.message}
      refetch={() => void query.refetch()}
      dataUpdatedAt={query.dataUpdatedAt}
      totalItems={total}
    >
      {children}
    </DataTableShell>
  )
}
