import { Row, Table } from "@tanstack/react-table";

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
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
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

interface DataTablePaginationProps<TData> {
  updatedAt: string;
  refetch: () => void;
  table: Table<TData>;
  handleDelete?: (rows: Row<TData>[]) => void;
}

export function DataTablePagination<TData>({
  updatedAt,
  refetch,
  table,
  handleDelete,
}: DataTablePaginationProps<TData>) {
  return (
    <div className="flex w-full items-center justify-between">
      <div className="flex flex-row items-center space-x-1.5 text-xs">
        {table.getFilteredSelectedRowModel().rows.length > 0 && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                className="h-7 w-7"
                title="删除"
              >
                <Trash2 className="h-3.5 w-3.5 text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <>
                <AlertDialogHeader>
                  <AlertDialogTitle>批量删除作业</AlertDialogTitle>
                  <AlertDialogDescription>
                    选中的作业将被删除，此操作不可撤销。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      if (handleDelete) {
                        handleDelete(table.getFilteredSelectedRowModel().rows);
                        // cancel selection
                        table.resetRowSelection();
                      } else {
                        toast.warning("批量删除功能开发中");
                      }
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </>
            </AlertDialogContent>
          </AlertDialog>
        )}
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          title="刷新"
          onClick={refetch}
        >
          <RefreshCcw className="h-3.5 w-3.5" />
        </Button>
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
        <p className="pl-1.5 text-muted-foreground">
          更新于 <time dateTime="2023-11-23">{updatedAt}</time>，
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
