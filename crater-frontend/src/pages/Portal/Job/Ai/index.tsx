import {
  ArrowDownIcon,
  ArrowUpIcon,
  CaretSortIcon,
  ChevronDownIcon,
  DotsHorizontalIcon,
} from "@radix-ui/react-icons";
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
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
import { NewTaskForm } from "./Form";
import { useEffect, useState } from "react";
import { Separator } from "@/components/ui/separator";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiTaskDelete, apiTaskList } from "@/services/api/task";
import { logger } from "@/utils/loglevel";
import { showErrorToast } from "@/utils/toast";
import { useToast } from "@/components/ui/use-toast";
import { DataTable } from "@/components/tasks/components/data-table";
import { DataTableColumnHeader } from "@/components/tasks/components/data-table-column-header";

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
      return "创建时间";
    case "actions":
      return "操作";
    default:
      return "";
  }
};

export const priorities = [
  {
    label: "低",
    value: "low",
    icon: ArrowDownIcon,
  },
  {
    label: "高",
    value: "high",
    icon: ArrowUpIcon,
  },
];

export function Component() {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = useState({});
  const [openSheet, setOpenSheet] = useState(false);
  const [data, setData] = useState<TaskInfo[]>([]);
  const queryClient = useQueryClient();
  const { toast } = useToast();

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

  const columns: ColumnDef<TaskInfo>[] = [
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
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="任务名" />
      ),
      cell: ({ row }) => <div>{row.getValue("name")}</div>,
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="状态" />
      ),
      cell: ({ row }) => (
        <div className="capitalize">{row.getValue("status")}</div>
      ),
    },
    {
      accessorKey: "priority",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="优先级" />
      ),
      cell: ({ row }) => {
        const priority = priorities.find(
          (priority) => priority.value === row.getValue("priority"),
        );

        if (!priority) {
          return null;
        }

        return (
          <div className="flex items-center">
            {priority.icon && (
              <priority.icon className="mr-2 h-4 w-4 text-muted-foreground" />
            )}
            <span>{priority.label}</span>
          </div>
        );
      },
      filterFn: (row, id, value) => {
        return value.includes(row.getValue(id));
      },
    },
    {
      accessorKey: "startTime",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="创建时间" />
      ),
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
    const tableData: TaskInfo[] = taskList.map((task) => ({
      id: task.id,
      name: task.taskName,
      status: task.Status,
      startTime: task.createdAt,
    }));
    setData(tableData);
  }, [taskList, isLoading]);

  return (
    <div className="space-y-4 px-6 py-6">
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
      </div>
      <DataTable data={data} columns={columns} />
    </div>
  );
}
