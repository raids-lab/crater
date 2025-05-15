// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { Row, Table } from "@tanstack/react-table";
import React, { useCallback, useMemo } from "react";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
  RefreshCcw,
} from "lucide-react";
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
} from "@/components/ui-custom/alert-dialog";
import TooltipButton from "../TooltipButton";

export interface MultipleHandler<TData> {
  title: (rows: Row<TData>[]) => string;
  description: (rows: Row<TData>[]) => React.ReactNode;
  handleSubmit: (rows: Row<TData>[]) => void;
  icon: React.ReactNode;
  isDanger?: boolean;
}

interface DataTablePaginationProps<TData> {
  updatedAt: string;
  refetch: () => void;
  table: Table<TData>;
  multipleHandlers?: MultipleHandler<TData>[];
}

// 正确定义多选操作按钮组件的泛型
function MultipleActionButtonsComponent<TData>(props: {
  table: Table<TData>;
  multipleHandlers: MultipleHandler<TData>[];
}) {
  const { table, multipleHandlers } = props;
  const { t } = useTranslation();
  const selectedRows = table.getFilteredSelectedRowModel().rows;

  return (
    <>
      {selectedRows.length > 0 &&
        multipleHandlers?.length > 0 &&
        multipleHandlers.map((multipleHandler, index) => (
          <AlertDialog key={index}>
            <AlertDialogTrigger asChild>
              <TooltipButton
                variant="outline"
                size="icon"
                className="size-9"
                tooltipContent={multipleHandler.title(selectedRows)}
              >
                {multipleHandler.icon}
              </TooltipButton>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {multipleHandler.title(selectedRows)}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {multipleHandler.description(selectedRows)}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>
                    {t("dataTablePagination.cancel")}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    variant={
                      multipleHandler.isDanger ? "destructive" : "default"
                    }
                    onClick={() => {
                      multipleHandler.handleSubmit(selectedRows);
                      // cancel selection
                      table.resetRowSelection();
                    }}
                  >
                    {t("dataTablePagination.confirm")}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </>
            </AlertDialogContent>
          </AlertDialog>
        ))}
    </>
  );
}

// 使用 React.memo 包裹组件，同时保留泛型
const MultipleActionButtons = React.memo(
  MultipleActionButtonsComponent,
) as typeof MultipleActionButtonsComponent;

// 正确定义分页控件组件的泛型
function PaginationControlsComponent<TData>(props: { table: Table<TData> }) {
  const { table } = props;
  const { t } = useTranslation();

  return (
    <div className="flex items-center space-x-1.5">
      <Button
        variant="outline"
        className="hidden size-9 p-0 lg:flex"
        onClick={() => table.setPageIndex(0)}
        disabled={!table.getCanPreviousPage()}
        title={t("dataTablePagination.firstPage")}
      >
        <ChevronsLeftIcon className="size-4" />
      </Button>
      <Button
        variant="outline"
        className="size-9 p-0"
        onClick={() => table.previousPage()}
        disabled={!table.getCanPreviousPage()}
        title={t("dataTablePagination.previousPage")}
      >
        <span className="sr-only">{t("dataTablePagination.previousPage")}</span>
        <ChevronLeftIcon className="size-4" />
      </Button>
      <Button
        variant="outline"
        className="size-9 p-0"
        onClick={() => table.nextPage()}
        disabled={!table.getCanNextPage()}
        title={t("dataTablePagination.nextPage")}
      >
        <span className="sr-only">{t("dataTablePagination.nextPage")}</span>
        <ChevronRightIcon className="size-4" />
      </Button>
      <Button
        variant="outline"
        className="hidden size-9 p-0 lg:flex"
        onClick={() => table.setPageIndex(table.getPageCount() - 1)}
        disabled={!table.getCanNextPage()}
        title={t("dataTablePagination.lastPage")}
      >
        <span className="sr-only">{t("dataTablePagination.lastPage")}</span>
        <ChevronsRightIcon className="size-4" />
      </Button>
    </div>
  );
}

// 使用 React.memo 包裹组件，同时保留泛型
const PaginationControls = React.memo(
  PaginationControlsComponent,
) as typeof PaginationControlsComponent;

// 然后保持主组件 DataTablePagination 的定义不变
function DataTablePaginationComponent<TData>({
  updatedAt,
  refetch,
  table,
  multipleHandlers,
}: DataTablePaginationProps<TData>) {
  const { t } = useTranslation();

  // 使用 useMemo 缓存计算值
  const pageInfo = useMemo(
    () => ({
      currentPage: table.getState().pagination.pageIndex + 1,
      totalPages: table.getPageCount(),
      filteredRowsLength: table.getFilteredRowModel().rows.length,
      selectedRowsLength: table.getFilteredSelectedRowModel().rows.length,
    }),
    [table],
  );

  // 使用 useCallback 缓存事件处理函数
  const handlePageSizeChange = useCallback(
    (value: string) => {
      table.setPageSize(Number(value));
    },
    [table],
  );

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  return (
    <div className="flex w-full items-center justify-between">
      <div className="flex flex-row items-center space-x-1.5 text-xs">
        {multipleHandlers && multipleHandlers.length > 0 && (
          <MultipleActionButtons
            table={table}
            multipleHandlers={multipleHandlers}
          />
        )}
        <TooltipButton
          variant="outline"
          size="icon"
          className="size-9"
          tooltipContent={t("dataTablePagination.refresh")}
          onClick={handleRefresh}
        >
          <RefreshCcw className="h-3.5 w-3.5" />
        </TooltipButton>
        <Select
          value={`${table.getState().pagination.pageSize}`}
          onValueChange={handlePageSizeChange}
        >
          <SelectTrigger className="bg-background h-9 w-[100px] pr-2 pl-3 text-xs">
            <SelectValue placeholder={table.getState().pagination.pageSize} />
          </SelectTrigger>
          <SelectContent side="top">
            {[10, 20, 50, 100, 200].map((pageSize) => (
              <SelectItem key={pageSize} value={`${pageSize}`}>
                {t("dataTablePagination.itemsPerPage", { count: pageSize })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-muted-foreground pl-1.5 font-medium">
          {t("dataTablePagination.updatedAt", { time: updatedAt })}
          {", "}
          {pageInfo.selectedRowsLength === 0 ? (
            <>
              {t("dataTablePagination.totalItems", {
                count: pageInfo.filteredRowsLength,
              })}
            </>
          ) : (
            <>
              {t("dataTablePagination.selectedItems", {
                selected: pageInfo.selectedRowsLength,
                total: pageInfo.filteredRowsLength,
              })}
            </>
          )}
        </p>
      </div>
      <div className="flex items-center space-x-6">
        <p className="text-muted-foreground text-xs font-medium">
          {t("dataTablePagination.currentPage", {
            page: pageInfo.currentPage,
            totalPages: pageInfo.totalPages,
          })}
        </p>
        <PaginationControls table={table} />
      </div>
    </div>
  );
}

// 导出带有泛型的包裹组件
export const DataTablePagination = React.memo(
  DataTablePaginationComponent,
) as typeof DataTablePaginationComponent;
