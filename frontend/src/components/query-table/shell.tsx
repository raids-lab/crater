import type { Table as TableInstance } from '@tanstack/react-table'
import { flexRender } from '@tanstack/react-table'
import { GridIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'
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

import { cn } from '@/lib/utils'

import { DataTablePagination, type MultipleHandler } from './pagination'
import { DataTableToolbar, type DataTableToolbarConfig } from './toolbar'

interface DataTableShellProps<TData> {
  table: TableInstance<TData>
  info?: {
    title?: string
    description: string
  }
  toolbarConfig?: DataTableToolbarConfig
  multipleHandlers?: MultipleHandler<TData>[]
  children?: ReactNode
  briefChildren?: ReactNode
  withI18n: boolean
  className?: string
  isLoading: boolean
  isError: boolean
  errorMessage?: string
  refetch: () => void
  dataUpdatedAt: number
  totalItems?: number
}

export function DataTableShell<TData>({
  table,
  info,
  toolbarConfig,
  multipleHandlers,
  children,
  briefChildren,
  withI18n,
  className,
  isLoading,
  isError,
  errorMessage,
  refetch,
  dataUpdatedAt,
  totalItems,
}: DataTableShellProps<TData>) {
  const { t } = useTranslation()
  const updatedAt = dataUpdatedAt
    ? new Date(dataUpdatedAt).toLocaleString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    : '--'

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {info && (
        <PageTitle title={info.title} description={info.description}>
          {children}
        </PageTitle>
      )}
      {briefChildren && <>{briefChildren}</>}
      {toolbarConfig && (
        <DataTableToolbar table={table} config={toolbarConfig} isLoading={isLoading}>
          {!info && <>{children}</>}
        </DataTableToolbar>
      )}
      <Card className="overflow-hidden rounded-md p-0 shadow-xs">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      className="text-muted-foreground px-4"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length > 0 ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id} data-state={row.getIsSelected() && 'selected'}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className="pl-4">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : isLoading ? (
                <TableRow>
                  <TableCell colSpan={table.getAllColumns().length} className="h-60">
                    <LoadingCircleIcon />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell
                    colSpan={table.getAllColumns().length}
                    className="text-destructive h-60 text-center hover:bg-transparent"
                  >
                    {errorMessage || t('common.error')}
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
                      <p className="select-none">{withI18n ? t('dataTable.noData') : '暂无数据'}</p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <DataTablePagination
        table={table}
        refetch={refetch}
        updatedAt={updatedAt}
        multipleHandlers={multipleHandlers}
        totalItems={totalItems}
      />
    </div>
  )
}
