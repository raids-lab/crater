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
import { useTranslation } from 'react-i18next'
import { Row, Table } from '@tanstack/react-table'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
  RefreshCcw,
} from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'
import TooltipButton from '../TooltipButton'
import React from 'react'

export interface MultipleHandler<TData> {
  title: (rows: Row<TData>[]) => string
  description: (rows: Row<TData>[]) => React.ReactNode
  handleSubmit: (rows: Row<TData>[]) => void
  icon: React.ReactNode
  isDanger?: boolean
}

interface DataTablePaginationProps<TData> {
  updatedAt: string
  refetch: () => void
  table: Table<TData>
  multipleHandlers?: MultipleHandler<TData>[]
}

export function DataTablePagination<TData>({
  updatedAt,
  refetch,
  table,
  multipleHandlers,
}: DataTablePaginationProps<TData>) {
  const { t } = useTranslation()

  return (
    <div className="flex w-full items-center justify-between">
      <div className="flex flex-row items-center space-x-1.5 text-xs">
        {table.getFilteredSelectedRowModel().rows.length > 0 &&
          multipleHandlers &&
          multipleHandlers?.length > 0 &&
          multipleHandlers.map((multipleHandler, index) => (
            <AlertDialog key={index}>
              <AlertDialogTrigger asChild>
                <TooltipButton
                  variant="outline"
                  size="icon"
                  className="size-9"
                  tooltipContent={multipleHandler.title(table.getFilteredSelectedRowModel().rows)}
                >
                  {multipleHandler.icon}
                </TooltipButton>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {multipleHandler.title(table.getFilteredSelectedRowModel().rows)}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {multipleHandler.description(table.getFilteredSelectedRowModel().rows)}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('dataTablePagination.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant={multipleHandler.isDanger ? 'destructive' : 'default'}
                      onClick={() => {
                        multipleHandler.handleSubmit(table.getFilteredSelectedRowModel().rows)
                        // cancel selection
                        table.resetRowSelection()
                      }}
                    >
                      {t('dataTablePagination.confirm')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </>
              </AlertDialogContent>
            </AlertDialog>
          ))}
        <TooltipButton
          variant="outline"
          size="icon"
          className="size-9"
          tooltipContent={t('dataTablePagination.refresh')}
          onClick={refetch}
        >
          <RefreshCcw className="h-3.5 w-3.5" />
        </TooltipButton>
        <Select
          value={`${table.getState().pagination.pageSize}`}
          onValueChange={(value) => {
            table.setPageSize(Number(value))
          }}
        >
          <SelectTrigger className="bg-background h-9 w-[100px] pr-2 pl-3 text-xs">
            <SelectValue placeholder={table.getState().pagination.pageSize} />
          </SelectTrigger>
          <SelectContent side="top">
            {[10, 20, 50, 100, 200].map((pageSize) => (
              <SelectItem key={pageSize} value={`${pageSize}`}>
                {t('dataTablePagination.itemsPerPage', { count: pageSize })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-muted-foreground pl-1.5 font-medium">
          {t('dataTablePagination.updatedAt', { time: updatedAt })}
          {', '}
          {table.getFilteredSelectedRowModel().rows.length === 0 ? (
            <>
              {t('dataTablePagination.totalItems', {
                count: table.getFilteredRowModel().rows.length,
              })}
            </>
          ) : (
            <>
              {t('dataTablePagination.selectedItems', {
                selected: table.getFilteredSelectedRowModel().rows.length,
                total: table.getFilteredRowModel().rows.length,
              })}
            </>
          )}
        </p>
      </div>
      <div className="flex items-center space-x-6">
        <p className="text-muted-foreground text-xs font-medium">
          {t('dataTablePagination.currentPage', {
            page: table.getState().pagination.pageIndex + 1,
            totalPages: table.getPageCount(),
          })}
        </p>
        <div className="flex items-center space-x-1.5">
          <Button
            variant="outline"
            className="hidden size-9 p-0 lg:flex"
            onClick={() => table.setPageIndex(0)}
            disabled={!table.getCanPreviousPage()}
            title={t('dataTablePagination.firstPage')}
          >
            <ChevronsLeftIcon className="size-4" />
          </Button>
          <Button
            variant="outline"
            className="size-9 p-0"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            title={t('dataTablePagination.previousPage')}
          >
            <span className="sr-only">{t('dataTablePagination.previousPage')}</span>
            <ChevronLeftIcon className="size-4" />
          </Button>
          <Button
            variant="outline"
            className="size-9 p-0"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            title={t('dataTablePagination.nextPage')}
          >
            <span className="sr-only">{t('dataTablePagination.nextPage')}</span>
            <ChevronRightIcon className="size-4" />
          </Button>
          <Button
            variant="outline"
            className="hidden size-9 p-0 lg:flex"
            onClick={() => table.setPageIndex(table.getPageCount() - 1)}
            disabled={!table.getCanNextPage()}
            title={t('dataTablePagination.lastPage')}
          >
            <span className="sr-only">{t('dataTablePagination.lastPage')}</span>
            <ChevronsRightIcon className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
