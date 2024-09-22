import {
  ColumnDef,
  ColumnFiltersState,
  PaginationState,
  Row,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DataTablePagination } from "./DataTablePagination";
import { DataTableToolbar, DataTableToolbarConfig } from "./DataTableToolbar";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import { UseQueryResult } from "@tanstack/react-query";
import { CardTitle } from "@/components/ui-custom/card";
import LoadingCircleIcon from "@/components/icon/LoadingCircleIcon";

interface DataTableProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  info?: {
    title: string;
    description: string;
    children?: React.ReactNode;
  };
  query: UseQueryResult<TData[], Error>;
  columns: ColumnDef<TData, TValue>[];
  toolbarConfig?: DataTableToolbarConfig;
  handleMultipleDelete?: (rows: Row<TData>[]) => void;
  className?: string;
}

export function DataTable<TData, TValue>({
  info,
  query,
  columns,
  toolbarConfig,
  handleMultipleDelete,
  children,
  className,
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [sorting, setSorting] = useState<SortingState>([]);
  const { data: queryData, isLoading, dataUpdatedAt, refetch } = query;
  const updatedAt = new Date(dataUpdatedAt).toLocaleString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });

  const data = useMemo(() => {
    if (!queryData || isLoading) return [];
    return queryData;
  }, [queryData, isLoading]);

  const table = useReactTable({
    data: data,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  return (
    <div className={cn("flex flex-col gap-2 lg:col-span-3", className)}>
      {toolbarConfig && (
        <DataTableToolbar
          table={table}
          config={toolbarConfig}
          isLoading={query.isLoading}
        >
          {children}
        </DataTableToolbar>
      )}
      <Card className="overflow-hidden">
        {info ? (
          <CardHeader className="mb-6 flex flex-row items-start border-b bg-muted/50 dark:bg-muted/25">
            <div className="grid gap-2">
              <CardTitle>{info.title}</CardTitle>
              <CardDescription>{info.description}</CardDescription>
            </div>
            <div className="ml-auto flex items-center gap-1">
              {info.children}
            </div>
          </CardHeader>
        ) : (
          <CardHeader className="py-3" />
        )}
        <CardContent>
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow
                  key={headerGroup.id}
                  className="bg-accent hover:bg-accent"
                >
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead key={header.id} colSpan={header.colSpan}>
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                      </TableHead>
                    );
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() && "selected"}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={columns.length} className="h-60">
                        <LoadingCircleIcon />
                      </TableCell>
                    </TableRow>
                  ) : (
                    <TableRow>
                      <TableCell
                        colSpan={columns.length}
                        className="h-60 text-center text-muted-foreground"
                      >
                        暂无数据
                      </TableCell>
                    </TableRow>
                  )}
                </>
              )}
            </TableBody>
          </Table>
        </CardContent>
        <CardFooter
          className={cn(
            "flex flex-row space-x-2 border-t bg-muted/50 px-6 py-4 dark:bg-muted/25",
          )}
        >
          {table.getRowModel().rows?.length > 0 && (
            <DataTablePagination
              table={table}
              refetch={() => void refetch()}
              updatedAt={updatedAt}
              handleDelete={handleMultipleDelete}
            />
          )}
        </CardFooter>
      </Card>
    </div>
  );
}
