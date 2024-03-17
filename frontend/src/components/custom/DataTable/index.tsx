import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
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
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useEffect, useState } from "react";

interface DataTableProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  toolbarConfig: DataTableToolbarConfig;
  loading: boolean;
  onPaginationChange: OnChangeFn<PaginationState>;
  rowCount?: number;
  pagination: PaginationState;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  toolbarConfig,
  loading,
  onPaginationChange,
  rowCount,
  pagination,
  children,
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [sorting, setSorting] = useState<SortingState>([
    // { id: "createdAt", desc: true },
  ]);

  const [cachedRowCount, setCachedRowCount] = useState(rowCount);

  useEffect(() => {
    if (rowCount && rowCount !== cachedRowCount) setCachedRowCount(rowCount);
  }, [rowCount, cachedRowCount]);

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
    },
    // pagination
    manualPagination: true,
    onPaginationChange,
    rowCount: cachedRowCount,
    // shadcn example props
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  return (
    <Card>
      <CardHeader>
        <DataTableToolbar table={table} config={toolbarConfig}>
          {children}
        </DataTableToolbar>
      </CardHeader>
      <CardContent>
        <>
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
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
              {table.getRowModel().rows?.length || loading ? (
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
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-32 text-center"
                  >
                    暂无数据
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          <DataTablePagination
            table={table}
            hidden={table.getRowModel().rows?.length === 0}
          />
        </>
      </CardContent>
    </Card>
  );
}
