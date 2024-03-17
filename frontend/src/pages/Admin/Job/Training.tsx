import { apiAdminTaskListByType } from "@/services/api/admin/task";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useEffect, useMemo, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
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
import {
  CircleIcon,
  ClockIcon,
  MinusCircledIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
  StopIcon,
  // CheckIcon,
  //Cross2Icon,
  DotIcon,
  DotFilledIcon,
} from "@radix-ui/react-icons";
import { apiAiTaskDelete } from "@/services/api/aiTask";

type StatusValue =
  | "Queueing"
  | "Created"
  | "Pending"
  | "Running"
  | "Failed"
  | "Succeeded"
  | "Preempted"
  | "Deleted";

const statuses: {
  value: StatusValue;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  {
    value: "Queueing",
    label: "检查配额",
    icon: ClockIcon,
  },
  {
    value: "Created",
    label: "已创建",
    icon: CircleIcon,
  },
  {
    value: "Pending",
    label: "等待中",
    icon: ClockIcon,
  },
  {
    value: "Running",
    label: "运行中",
    icon: StopwatchIcon,
  },
  {
    value: "Failed",
    label: "失败",
    icon: CrossCircledIcon,
  },
  {
    value: "Succeeded",
    label: "成功",
    icon: CheckCircledIcon,
  },
  {
    value: "Preempted",
    label: "被抢占",
    icon: MinusCircledIcon,
  },
  {
    value: "Deleted",
    label: "已停止",
    icon: StopIcon,
  },
];

const deleteStatuses = [
  {
    label: "已删除",
    value: "已删除",
    icon: DotFilledIcon,
  },
  {
    label: "未删除",
    value: "未删除",
    icon: DotIcon,
  },
];

const getHeader = (key: string): string => {
  switch (key) {
    case "taskName":
      return "task 名称";
    case "jobName":
      return "job 名称";
    case "isDeleted":
      return "是否删除";
    case "userName":
      return "创建用户";
    case "status":
      return "状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "finishAt":
      return "删除于";
    default:
      return key;
  }
};
type JTaskInfo = {
  id: number;
  taskName: string;
  jobName: string;
  isDeleted: string;
  userName: string;
  status: string;
  createdAt: string;
  startedAt: string;
  finishAt: string;
  //finishAt: string;
  //gpus: string | number | undefined;
};

const Training: FC = () => {
  const [data, setData] = useState<JTaskInfo[]>([]);
  const { data: taskList, isLoading } = useQuery({
    queryKey: ["admin", "tasklist", "ai"],
    queryFn: () => apiAdminTaskListByType("training"),
    select: (res) => res.data.data.rows,
  });

  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索任务名",
      key: "taskName",
    },
    filterOptions: [
      {
        key: "isDeleted",
        title: "删除状态",
        option: deleteStatuses,
      },
    ],
    getHeader: getHeader,
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: (id: number) => apiAiTaskDelete(id),
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("任务已删除");
    },
  });

  const queryClient = useQueryClient();
  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["admin", "tasklist", "ai"],
        }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      console.error("更新查询失败", error);
    }
  };
  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: JTaskInfo[] = taskList
      //.filter((task) => !task.isDeleted)
      .map((task) => {
        //const task = convertJTask(t);
        return {
          id: task.id,
          taskName: task.taskName,
          jobName: task.jobName,
          isDeleted: task.isDeleted === true ? "已删除" : "未删除",
          userName: task.userName,
          status:
            task.isDeleted === true && task.status === "Running"
              ? "Deleted"
              : task.status,
          createdAt: task.createdAt,
          startedAt: task.startedAt,
          finishAt: task.finishAt,
          //priority: task.slo ? "high" : "low",
          // profileStatus:
          //   // 分析状态：profileStatus = 1 或 2 都显示分析中，不需要等待分析这个选项了
          //   task.profileStatus === 1 ? "2" : task.profileStatus.toString(),
        };
      });
    setData(tableData);
  }, [taskList, isLoading]);

  const columns = useMemo<ColumnDef<JTaskInfo>[]>(
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
      {
        accessorKey: "taskName",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("taskName")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("taskName")}</div>,
      },
      {
        accessorKey: "isDeleted",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("isDeleted")}
          />
        ),
        cell: ({ row }) => {
          const deleteStatus = deleteStatuses.find(
            (deleteStatus) => deleteStatus.value === row.getValue("isDeleted"),
          );
          if (!deleteStatus) {
            return null;
          }
          return (
            <div className="flex items-center">
              {deleteStatus.icon && (
                <deleteStatus.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{deleteStatus.value}</span>
            </div>
          );
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "userName",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("userName")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("userName")}</div>,
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
            <div className="flex flex-row items-center justify-start">
              <div
                className={cn("flex h-3 w-3 rounded-full", {
                  "bg-purple-500 hover:bg-purple-400":
                    status.value === "Queueing",
                  "bg-slate-500 hover:bg-slate-400": status.value === "Created",
                  "bg-pink-500 hover:bg-pink-400": status.value === "Pending",
                  "bg-sky-500 hover:bg-sky-400": status.value === "Running",
                  "bg-red-500 hover:bg-red-400": status.value === "Failed",
                  "bg-emerald-500 hover:bg-emerald-400":
                    status.value === "Succeeded",
                  "bg-orange-500 hover:bg-orange-400":
                    status.value === "Preempted",
                  "bg-rose-500 hover:bg-rose-200": status.value === "Deleted",
                })}
              ></div>
              <div className="ml-1.5">{status.label}</div>
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
          return <TableDate date={row.getValue("createdAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        accessorKey: "startedAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("startedAt")}
          />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("startedAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        accessorKey: "finishAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("finishAt")}
          />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("finishAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const taskInfo = row.original;
          if (row.getValue("isDeleted") === "已删除") {
            return (
              <div>
                <Button
                  variant="outline"
                  className="h-8 w-8 p-0 hover:text-red-700"
                  title="终止 Jupyter Lab"
                  disabled={row.getValue("isDeleted") === "已删除"}
                >
                  <StopIcon />
                </Button>
              </div>
            );
          } else {
            return (
              <div
                className="flex flex-row space-x-1"
                aria-disabled={row.getValue("status") === "Deleted"}
              >
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <div>
                      <Button
                        variant="outline"
                        className="h-8 w-8 p-0 hover:text-red-700"
                        title="终止 Jupyter Lab"
                        disabled={row.getValue("isDeleted") === "已删除"}
                      >
                        <StopIcon />
                      </Button>
                    </div>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除任务</AlertDialogTitle>
                      <AlertDialogDescription>
                        任务「{taskInfo?.taskName}
                        」将停止，请确认已经保存好所需数据。
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
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
              </div>
            );
          }
        },
      },
    ],
    [deleteTask],
  );

  return (
    <div className="space-y-4">
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
      ></DataTable>
    </div>
  );
};

export default Training;
