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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useIndex } from "../../hooks/useIndex";
import { NewTaskForm } from "./Form";
import { useEffect, useState } from "react";
import { Separator } from "@/components/ui/separator";
import { useQuery } from "@tanstack/react-query";
import { apiTaskList } from "@/services/api/task";
import { logger } from "@/utils/loglevel";
import { showErrorToast } from "@/utils/toast";

type TaskInfo = {
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
      return "开始时间";
    default:
      return "";
  }
};

export const columns: ColumnDef<TaskInfo>[] = [
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
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
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
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
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
    header: "Action",
    enableHiding: false,
    cell: ({ row }) => {
      const payment = row.original;

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
                if (window.isSecureContext && navigator.clipboard) {
                  navigator.clipboard
                    .writeText(payment.id.toString())
                    .catch(() =>
                      alert("Failed to copy payment ID to clipboard."),
                    );
                } else {
                  alert(
                    "Failed to copy payment ID to clipboard, please use HTTPs.",
                  );
                }
              }}
            >
              复制任务 ID
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

export function Component() {
  useIndex("job", "ai");
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = useState({});
  const [openSheet, setOpenSheet] = useState(false);
  const [data, setData] = useState<TaskInfo[]>([]);

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["tasklist"],
    retry: 1,
    queryFn: apiTaskList,
    select: (res) => res.data.data.Tasks,
    onSuccess: (data) => {
      logger.debug("Data is: ", data);
    },
    onError: (err) => showErrorToast("获取任务列表失败", err),
  });

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: TaskInfo[] = taskList.map((task) => ({
      id: task.id,
      name: task.taskName,
      status: task.Status,
      startTime: task.createdAt,
    }));
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
    <div className="w-full space-y-4">
      <div className="flex items-center space-x-2">
        <Sheet open={openSheet} onOpenChange={setOpenSheet}>
          <SheetTrigger asChild>
            <Button className="min-w-fit">新建任务</Button>
          </SheetTrigger>
          {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
          <SheetContent className="max-h-screen overflow-y-auto sm:max-w-2xl">
            <SheetHeader>
              <SheetTitle>新建任务</SheetTitle>
              <SheetDescription>创建一个新的 AI 训练任务</SheetDescription>
            </SheetHeader>
            <Separator className="mt-4" />
            <NewTaskForm closeSheet={() => setOpenSheet(false)} />
          </SheetContent>
        </Sheet>
        <Input
          placeholder="请输入搜索内容"
          value={(table.getColumn("name")?.getFilterValue() as string) ?? ""}
          onChange={(event) =>
            table.getColumn("name")?.setFilterValue(event.target.value)
          }
          className="flex-grow"
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" className="ml-auto min-w-fit">
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
      <div className="rounded-md border">
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
