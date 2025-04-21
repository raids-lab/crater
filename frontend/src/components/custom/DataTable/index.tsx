import {
  ColumnDef,
  ColumnFiltersState,
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
import { DataTablePagination, MultipleHandler } from "./DataTablePagination";
import { DataTableToolbar, DataTableToolbarConfig } from "./DataTableToolbar";
import { Card, CardContent } from "@/components/ui/card";
import React, { useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import { UseQueryResult } from "@tanstack/react-query";
import LoadingCircleIcon from "@/components/icon/LoadingCircleIcon";
import { Checkbox } from "@/components/ui/checkbox";
import { GridIcon } from "lucide-react";
import PageTitle from "@/components/layout/PageTitle";
import { usePaginationWithStorage } from "@/hooks/usePaginationWithStorage";
import { useLocalStorage } from "usehooks-ts";

interface DataTableProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  info?: {
    title: string;
    description: string;
  };
  storageKey: string;
  query: UseQueryResult<TData[], Error>;
  columns: ColumnDef<TData, TValue>[];
  toolbarConfig?: DataTableToolbarConfig;
  multipleHandlers?: MultipleHandler<TData>[];
  briefChildren?: React.ReactNode;
  className?: string;
}

export function DataTable<TData, TValue>({
  info,
  storageKey,
  query,
  columns,
  toolbarConfig,
  multipleHandlers,
  children,
  briefChildren,
  className,
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = useLocalStorage<ColumnFiltersState>(
    `${storageKey}-column-filters`,
    [],
  );
  const [sorting, setSorting] = useState<SortingState>([]);
  const { data: queryData, isLoading, dataUpdatedAt, refetch } = query;
  const updatedAt = new Date(dataUpdatedAt).toLocaleString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
  const [pagination, setPagination] = usePaginationWithStorage(storageKey);

  const data = useMemo(() => {
    if (!queryData || isLoading) return [];
    return queryData;
  }, [queryData, isLoading]);

  const columnsWithSelection = useMemo(() => {
    if (!multipleHandlers || !columns || multipleHandlers.length === 0) {
      return columns;
    }
    return [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            hidden={table.getRowModel().rows.length === 0}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      ...columns,
    ];
  }, [columns, multipleHandlers]);

  const table = useReactTable({
    data: data,
    columns: columnsWithSelection,
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
    <div className={cn("flex flex-col gap-4", className)}>
      {info && (
        <PageTitle title={info.title} description={info.description}>
          {children}
        </PageTitle>
      )}
      {briefChildren && <>{briefChildren}</>}
      {toolbarConfig && (
        <DataTableToolbar
          table={table}
          config={toolbarConfig}
          isLoading={query.isLoading}
        >
          {!info && <>{children}</>}
        </DataTableToolbar>
      )}
      <Card className="overflow-hidden rounded-md p-0 shadow-xs">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead
                        key={header.id}
                        colSpan={header.colSpan}
                        className="px-4 py-2"
                      >
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
                      <TableCell key={cell.id} className="pl-4">
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
                    <TableCell
                      colSpan={columns.length}
                      className="text-muted-foreground/85 h-60 text-center hover:bg-transparent"
                    >
                      <div className="flex flex-col items-center justify-center py-16">
                        <div className="bg-muted mb-4 rounded-full p-3">
                          <GridIcon className="h-6 w-6" />
                        </div>
                        <p className="select-none">暂无数据</p>
                      </div>
                    </TableCell>
                  )}
                </>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <DataTablePagination
        table={table}
        refetch={() => void refetch()}
        updatedAt={updatedAt}
        multipleHandlers={multipleHandlers}
      />
    </div>
  );
}
