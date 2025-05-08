// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DoubleArrowLeftIcon,
  DoubleArrowRightIcon,
} from "@radix-ui/react-icons";
import { Table } from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface DataTablePaginationProps<TData> {
  table: Table<TData>;
  hidden?: boolean;
}

export function DataTablePagination<TData>({
  table,
  hidden,
}: DataTablePaginationProps<TData>) {
  const { t } = useTranslation();

  return (
    <>
      {!hidden && (
        <div className="mt-4 flex items-center justify-between px-2">
          <div className="text-muted-foreground flex-1 text-sm">
            {table.getFilteredSelectedRowModel().rows.length === 0 ? (
              <>{t("pagination.totalRows", { count: table.getRowCount() })}</>
            ) : (
              <>
                {t("pagination.selectedRows", {
                  count: table.getFilteredSelectedRowModel().rows.length,
                })}
              </>
            )}
          </div>
          <div className="flex items-center space-x-6 lg:space-x-8">
            <div className="flex items-center space-x-2">
              <Select
                value={`${table.getState().pagination.pageSize}`}
                onValueChange={(value) => {
                  table.setPageSize(Number(value));
                }}
              >
                <SelectTrigger className="h-8 w-[110px]">
                  <SelectValue
                    placeholder={table.getState().pagination.pageSize}
                  />
                </SelectTrigger>
                <SelectContent side="top">
                  {[10, 20, 50, 100].map((pageSize) => (
                    <SelectItem key={pageSize} value={`${pageSize}`}>
                      {t("pagination.rowsPerPage", { count: pageSize })}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex w-fit items-center justify-center text-sm font-medium">
              {t("pagination.currentPage", {
                currentPage: table.getState().pagination.pageIndex + 1,
                totalPages: table.getPageCount(),
              })}
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                className="hidden h-8 w-8 p-0 lg:flex"
                onClick={() => table.setPageIndex(0)}
                disabled={!table.getCanPreviousPage()}
                title={t("pagination.firstPage")}
              >
                <DoubleArrowLeftIcon className="size-4" />
              </Button>
              <Button
                variant="outline"
                className="h-8 w-8 p-0"
                onClick={() => table.previousPage()}
                disabled={!table.getCanPreviousPage()}
                title={t("pagination.previousPage")}
              >
                <span className="sr-only">Go to previous page</span>
                <ChevronLeftIcon className="size-4" />
              </Button>
              <Button
                variant="outline"
                className="h-8 w-8 p-0"
                onClick={() => table.nextPage()}
                disabled={!table.getCanNextPage()}
                title={t("pagination.nextPage")}
              >
                <span className="sr-only">Go to next page</span>
                <ChevronRightIcon className="size-4" />
              </Button>
              <Button
                variant="outline"
                className="hidden h-8 w-8 p-0 lg:flex"
                onClick={() => table.setPageIndex(table.getPageCount() - 1)}
                disabled={!table.getCanNextPage()}
                title={t("pagination.lastPage")}
              >
                <span className="sr-only">Go to last page</span>
                <DoubleArrowRightIcon className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
