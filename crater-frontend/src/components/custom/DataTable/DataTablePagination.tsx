import { Table } from "@tanstack/react-table";

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
} from "lucide-react";

interface DataTablePaginationProps<TData> {
  updatedAt: string;
  refetch: () => void;
  table: Table<TData>;
}

export function DataTablePagination<TData>({
  updatedAt,
  refetch,
  table,
}: DataTablePaginationProps<TData>) {
  return (
    <div className="flex w-full items-center justify-between">
      <div className="flex flex-row items-center gap-3 text-xs">
        <Select
          value={`${table.getState().pagination.pageSize}`}
          onValueChange={(value) => {
            table.setPageSize(Number(value));
          }}
        >
          <SelectTrigger className="h-7 w-[100px] bg-background pl-3 pr-2 text-xs">
            <SelectValue placeholder={table.getState().pagination.pageSize} />
          </SelectTrigger>
          <SelectContent side="top">
            {[10, 20, 50, 100].map((pageSize) => (
              <SelectItem key={pageSize} value={`${pageSize}`}>
                {pageSize} 条/页
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-muted-foreground">
          更新于{" "}
          <time
            dateTime="2023-11-23"
            onClick={refetch}
            className="cursor-pointer"
            title="刷新数据"
          >
            {updatedAt}
          </time>
          ，
          {table.getFilteredSelectedRowModel().rows.length === 0 ? (
            <>共 {table.getFilteredRowModel().rows.length} 条</>
          ) : (
            <>
              已选择 {table.getFilteredSelectedRowModel().rows.length} /{" "}
              {table.getFilteredRowModel().rows.length} 条
            </>
          )}
        </p>
      </div>
      <div className="flex items-center space-x-6">
        <p className="text-xs font-medium text-muted-foreground">
          第 {table.getState().pagination.pageIndex + 1} /{" "}
          {table.getPageCount()} 页
        </p>
        <div className="flex items-center space-x-1.5">
          <Button
            variant="outline"
            className="hidden h-7 w-7 p-0 lg:flex"
            onClick={() => table.setPageIndex(0)}
            disabled={!table.getCanPreviousPage()}
            title="前往首页"
          >
            <ChevronsLeftIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-7 w-7 p-0"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            title="前往上一页"
          >
            <span className="sr-only">Go to previous page</span>
            <ChevronLeftIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-7 w-7 p-0"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            title="前往下一页"
          >
            <span className="sr-only">Go to next page</span>
            <ChevronRightIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="hidden h-7 w-7 p-0 lg:flex"
            onClick={() => table.setPageIndex(table.getPageCount() - 1)}
            disabled={!table.getCanNextPage()}
            title="前往尾页"
          >
            <span className="sr-only">Go to last page</span>
            <ChevronsRightIcon className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
