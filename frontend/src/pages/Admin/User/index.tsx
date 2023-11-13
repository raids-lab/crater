import {
  CaretSortIcon,
  ChevronDownIcon,
  DotsHorizontalIcon,
} from "@radix-ui/react-icons";
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiTaskDelete } from "@/services/api/task";
import { logger } from "@/utils/loglevel";
import { showErrorToast } from "@/utils/toast";
import { useToast } from "@/components/ui/use-toast";

type UserInfo = {
  id: number;
  name: string;
  status: string;
  startTime: string;
};

const getTaskInfoTitle = (key: string) => {
  switch (key) {
    case "name":
      return "任务名";
    case "status":
      return "状态";
    case "startTime":
      return "创建时间";
    case "actions":
      return "操作";
    default:
      return "";
  }
};

async function getData(): Promise<UserInfo[]> {
  // Fetch data from your API here.
  // For now, we'll just return some fake data.
  await new Promise((resolve) => setTimeout(resolve, 1000));
  return [
    {
      id: 1,
      name: "100",
      status: "pending",
      startTime: "m@example.com",
    },
    // ...
  ];
}

export function Component() {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = useState({});
  const [data, setData] = useState<UserInfo[]>([]);
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["userlist"],
    retry: 1,
    queryFn: getData,
    select: (res) => res,
    onSuccess: (data) => {
      logger.debug("Data is: ", data);
    },
    onError: (err) => showErrorToast("获取任务列表失败", err),
  });

  const { mutate: deleteTask } = useMutation({
    mutationFn: (id: number) => apiTaskDelete(id),
    onSuccess: () => {
      queryClient
        .invalidateQueries({ queryKey: ["tasklist"] })
        .then(() => {
          toast({
            title: `删除成功`,
            description: `任务已删除`,
          });
        })
        .catch((err) => {
          showErrorToast("刷新任务列表失败", err);
        });
    },
    onError: (err) => showErrorToast("无法删除任务", err),
  });

  const columns: ColumnDef<UserInfo>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
          className="ml-2"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          className="ml-2"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "name",
      header: ({ column }) => {
        return (
          <div className="flex flex-row items-center space-x-1">
            <p>{getTaskInfoTitle("name")}</p>
            <Button
              variant="ghost"
              size="icon"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              <CaretSortIcon className="h-4 w-4" />
            </Button>
          </div>
        );
      },
      cell: ({ row }) => <div>{row.getValue("name")}</div>,
    },
    {
      accessorKey: "status",
      header: getTaskInfoTitle("status"),
      cell: ({ row }) => (
        <div className="capitalize">{row.getValue("status")}</div>
      ),
    },
    {
      accessorKey: "startTime",
      header: ({ column }) => {
        return (
          <div className="flex flex-row items-center space-x-1">
            <p>{getTaskInfoTitle("startTime")}</p>
            <Button
              variant="ghost"
              size="icon"
              title="amount sort"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              <CaretSortIcon className="h-4 w-4" />
            </Button>
          </div>
        );
      },
      cell: ({ row }) => {
        // row format: "2023-10-30T03:21:03.733Z"
        const date = new Date(row.getValue("startTime"));
        const formatted = date.toLocaleString("zh-CN", {
          timeZone: "Asia/Shanghai",
        });
        return <div>{formatted}</div>;
      },
    },
    {
      id: "actions",
      header: getTaskInfoTitle("actions"),
      enableHiding: false,
      cell: ({ row }) => {
        const taskInfo = row.original;

        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <DotsHorizontalIcon className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => {
                  // check if browser support clipboard
                  deleteTask(taskInfo.id);
                }}
              >
                删除
              </DropdownMenuItem>
              {/* <DropdownMenuSeparator /> */}
              {/* <DropdownMenuItem>View customer</DropdownMenuItem>
                <DropdownMenuItem>View payment details</DropdownMenuItem> */}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: UserInfo[] = taskList;
    setData(tableData);
  }, [taskList, isLoading]);

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
  });

  return (
    <div className="space-y-4 px-6 py-6">
      <div className="flex items-center space-x-2">
        <Input
          placeholder="请输入搜索内容"
          value={(table.getColumn("name")?.getFilterValue() as string) ?? ""}
          onChange={(event) =>
            table.getColumn("name")?.setFilterValue(event.target.value)
          }
          className="flex-grow bg-background"
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              className="ml-auto min-w-fit bg-background"
            >
              显示内容 <ChevronDownIcon className="ml-2 h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {table
              .getAllColumns()
              .filter((column) => column.getCanHide())
              .map((column) => {
                return (
                  <DropdownMenuCheckboxItem
                    key={column.id}
                    className="capitalize"
                    checked={column.getIsVisible()}
                    onCheckedChange={(value) =>
                      column.toggleVisibility(!!value)
                    }
                  >
                    {getTaskInfoTitle(column.id) ?? column.id}
                  </DropdownMenuCheckboxItem>
                );
              })}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="rounded-md border bg-background shadow-sm">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id}>
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
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-48 text-center text-gray-400"
                >
                  暂无数据
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-end space-x-2 py-4">
        <div className="flex-1 text-sm text-muted-foreground">
          {table.getFilteredSelectedRowModel().rows.length} of{" "}
          {table.getFilteredRowModel().rows.length} row(s) selected.
        </div>
        <div className="space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
