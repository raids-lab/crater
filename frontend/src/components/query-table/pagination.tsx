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
import { Row, Table } from '@tanstack/react-table'
import { ChevronLeftIcon, ChevronRightIcon, RefreshCcw } from 'lucide-react'
import React from 'react'
import { useTranslation } from 'react-i18next'

import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
} from '@/components/ui/pagination'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

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

import TooltipButton from '../button/tooltip-button'

const DOTS = '...'

function usePagination({
  currentPage,
  totalPages,
  siblingCount = 1,
}: {
  currentPage: number
  totalPages: number
  siblingCount?: number
}) {
  return React.useMemo(() => {
    const totalPageNumbers = siblingCount + 5 // Start, end, current, and 2 siblings

    // Case 1: If the number of pages is less than the page numbers we want to show
    if (totalPageNumbers >= totalPages) {
      return Array.from({ length: totalPages }, (_, i) => i + 1)
    }

    const leftSiblingIndex = Math.max(currentPage - siblingCount, 1)
    const rightSiblingIndex = Math.min(currentPage + siblingCount, totalPages)

    const shouldShowLeftDots = leftSiblingIndex > 2
    const shouldShowRightDots = rightSiblingIndex < totalPages - 2

    const firstPageIndex = 1
    const lastPageIndex = totalPages

    // Case 2: No left dots to show, but rights dots to be shown
    if (!shouldShowLeftDots && shouldShowRightDots) {
      const leftItemCount = 3 + 2 * siblingCount
      const leftRange = Array.from({ length: leftItemCount }, (_, i) => i + 1)

      return [...leftRange, DOTS, totalPages]
    }

    // Case 3: No right dots to show, but left dots to be shown
    if (shouldShowLeftDots && !shouldShowRightDots) {
      const rightItemCount = 3 + 2 * siblingCount
      const rightRange = Array.from(
        { length: rightItemCount },
        (_, i) => totalPages - rightItemCount + i + 1
      )

      return [firstPageIndex, DOTS, ...rightRange]
    }

    // Case 4: Both left and right dots to be shown
    if (shouldShowLeftDots && shouldShowRightDots) {
      const middleRange = Array.from(
        { length: rightSiblingIndex - leftSiblingIndex + 1 },
        (_, i) => leftSiblingIndex + i
      )

      return [firstPageIndex, DOTS, ...middleRange, DOTS, lastPageIndex]
    }

    return []
  }, [currentPage, totalPages, siblingCount])
}

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

  const currentPage = table.getState().pagination.pageIndex + 1
  const totalPages = table.getPageCount()

  const paginationRange = usePagination({
    currentPage,
    totalPages,
    siblingCount: 1,
  })

  const onPageChange = (page: number) => {
    table.setPageIndex(page - 1)
  }

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
        <div className="flex items-center space-x-2">
          <Pagination>
            <PaginationContent>
              {/* Previous button */}
              <PaginationItem>
                <PaginationLink
                  aria-label={t('dataTablePagination.previousPage')}
                  size="icon"
                  className={
                    currentPage <= 1
                      ? 'pointer-events-none cursor-not-allowed opacity-50'
                      : 'cursor-pointer'
                  }
                  onClick={() => currentPage > 1 && onPageChange(currentPage - 1)}
                >
                  <ChevronLeftIcon className="size-4" />
                </PaginationLink>
              </PaginationItem>

              {/* Page numbers */}
              {paginationRange.map((pageNumber, index) => {
                if (pageNumber === DOTS) {
                  return (
                    <PaginationItem key={`dots-${index}`}>
                      <PaginationEllipsis />
                    </PaginationItem>
                  )
                }

                return (
                  <PaginationItem key={pageNumber}>
                    <PaginationLink
                      onClick={() => onPageChange(pageNumber as number)}
                      isActive={pageNumber === currentPage}
                      className="cursor-pointer"
                    >
                      {pageNumber}
                    </PaginationLink>
                  </PaginationItem>
                )
              })}

              {/* Next button */}
              <PaginationItem>
                <PaginationLink
                  aria-label={t('dataTablePagination.nextPage')}
                  size="icon"
                  className={
                    currentPage >= totalPages
                      ? 'pointer-events-none cursor-not-allowed opacity-50'
                      : 'cursor-pointer'
                  }
                  onClick={() => currentPage < totalPages && onPageChange(currentPage + 1)}
                >
                  <ChevronRightIcon className="size-4" />
                </PaginationLink>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      </div>
    </div>
  )
}
