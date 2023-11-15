import {
  ClockIcon,
  DotsHorizontalIcon,
  MinusCircledIcon,
} from "@radix-ui/react-icons";
import { ColumnDef } from "@tanstack/react-table";
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
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { NewTaskForm } from "./Form";
import { useEffect, useMemo, useState } from "react";
import { Separator } from "@/components/ui/separator";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiAiTaskDelete, apiAiTaskList } from "@/services/api/aiTask";
import { logger } from "@/utils/loglevel";
import { showErrorToast } from "@/utils/toast";
import { useToast } from "@/components/ui/use-toast";
import { DataTable } from "@/components/DataTable";
import { DataTableColumnHeader } from "@/components/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";

type TaskInfo = {
  id: number;
  title: string;
  status: string;
  priority: string;
  createdAt: string;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "title":
      return "任务名称";
    case "status":
      return "状态";
    case "priority":
      return "优先级";
    case "createdAt":
      return "创建时间";
    default:
      return key;
  }
};

const priorities = [
  {
    label: "高优先级",
    value: "high",
    icon: ArrowUpIcon,
  },
  {
    label: "低优先级",
    value: "low",
    icon: ArrowDownIcon,
  },
];

const statuses = [
  {
    value: "Queueing",
    label: "Queueing",
    icon: ClockIcon,
  },
  {
    value: "Pending",
    label: "Pending",
    icon: StopwatchIcon,
  },
  {
    value: "Running",
    label: "Running",
    icon: StopwatchIcon,
  },
  {
    value: "Failed",
    label: "Failed",
    icon: CrossCircledIcon,
  },
  {
    value: "Succeeded",
    label: "Succeeded",
    icon: CheckCircledIcon,
  },
  {
    value: "Suspended",
    label: "Suspended",
    icon: MinusCircledIcon,
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索任务名称",
    key: "title",
  },
  filterOptions: [
    {
      key: "status",
      title: "状态",
      option: statuses,
    },
    {
      key: "priority",
      title: "优先级",
      option: priorities,
    },
  ],
  getHeader: getHeader,
};

export function Component() {
  const [openSheet, setOpenSheet] = useState(false);
  const [data, setData] = useState<TaskInfo[]>([]);
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["aitask", "list"],
    retry: 1,
    queryFn: apiAiTaskList,
    select: (res) => res.data.data.Tasks,
    onSuccess: (data) => {
      logger.debug("Data is: ", data);
    },
    onError: (err) => showErrorToast("获取任务列表失败", err),
  });

  const { mutate: deleteTask } = useMutation({
    mutationFn: (id: number) => apiAiTaskDelete(id),
    onSuccess: () => {
      queryClient
        .invalidateQueries({ queryKey: ["aitask", "list"] })
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

  const columns = useMemo<ColumnDef<TaskInfo>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
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
        accessorKey: "title",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("title")} />
        ),
        cell: ({ row }) => <div>{row.getValue("title")}</div>,
      },
      {
        accessorKey: "status",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("status")} />
        ),
        cell: ({ row }) => {
          const status = statuses.find(
            (status) => status.value === row.getValue("status"),
          );
          if (!status) {
            return null;
          }
          return (
            <div className="flex w-[100px] items-center">
              {status.icon && (
                <status.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{status.label}</span>
            </div>
          );
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "priority",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("priority")}
          />
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
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("createdAt")}
          />
        ),
        cell: ({ row }) => {
          // row format: "2023-10-30T03:21:03.733Z"
          const date = new Date(row.getValue("createdAt"));
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
            <AlertDialog>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">操作</span>
                    <DotsHorizontalIcon className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>操作</DropdownMenuLabel>

                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>删除</DropdownMenuItem>
                  </AlertDialogTrigger>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>View customer</DropdownMenuItem>
                  <DropdownMenuItem>View payment details</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除任务</AlertDialogTitle>
                  <AlertDialogDescription>
                    任务「{taskInfo?.title}」将不再可见，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => {
                      // check if browser support clipboard
                      deleteTask(taskInfo.id);
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          );
        },
      },
    ],
    [deleteTask],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: TaskInfo[] = taskList.map((task) => ({
      id: task.id,
      title: task.taskName,
      status: task.status,
      priority: task.slo ? "high" : "low",
      createdAt: task.createdAt,
    }));
    setData(tableData);
  }, [taskList, isLoading]);

  return (
    <div className="space-y-4 px-6 py-4">
      <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig}>
        <Sheet open={openSheet} onOpenChange={setOpenSheet}>
          <SheetTrigger asChild>
            <Button className="h-8 min-w-fit">新建任务</Button>
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
      </DataTable>
    </div>
  );
}
