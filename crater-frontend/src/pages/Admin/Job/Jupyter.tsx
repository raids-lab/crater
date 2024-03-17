import { apiAdminTaskListByType } from "@/services/api/admin/task";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useEffect, useMemo, useState } from "react";
import { ColumnDef, PaginationState } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
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
} from "@radix-ui/react-icons";
import { apiJTaskDelete, convertJTask } from "@/services/api/jupyterTask";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

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

const getHeader = (key: string): string => {
  switch (key) {
    case "taskName":
      return "任务名称";
    case "userName":
      return "用户";
    case "status":
      return "状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "gpus":
      return "GPU";
    default:
      return key;
  }
};

type JTaskInfo = {
  id: number;
  userName: string;
  taskName: string;
  gpus: string | number | undefined;
  status: string;
  createdAt: string;
  startedAt: string;
  finishAt: string;
};

const Jupyter: FC = () => {
  const [pagination, setPagination] = useState<PaginationState>({
    pageSize: 10,
    pageIndex: 0,
  });
  const [data, setData] = useState<JTaskInfo[]>([]);
  const { data: tasks, isLoading } = useQuery({
    queryKey: [
      "admin",
      "tasklist",
      "jupyter",
      pagination.pageSize,
      pagination.pageIndex,
    ],
    queryFn: () =>
      apiAdminTaskListByType(
        "jupyter",
        pagination.pageSize,
        pagination.pageIndex,
      ),
    select: (res) => res.data.data,
  });

  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索用户名",
      key: "userName",
    },
    filterOptions: [
      {
        key: "status",
        title: "状态",
        option: statuses,
      },
    ],
    getHeader: getHeader,
  };

  const { mutate: deleteJTask } = useMutation({
    mutationFn: (id: number) => apiJTaskDelete(id),
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
        queryClient.invalidateQueries({ queryKey: ["admin", "tasklist"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      console.error("更新查询失败", error);
    }
  };
  useEffect(() => {
    if (isLoading) return;
    if (!tasks?.rows) return;
    const tableData: JTaskInfo[] = tasks.rows
      //.filter((task) => !task.isDeleted)
      .map((t) => {
        const task = convertJTask(t);
        return {
          id: task.id,
          userName: task.userName,
          taskName: task.taskName,
          gpus: task.resourceRequest["nvidia.com/gpu"],
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
  }, [tasks, isLoading]);

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
        accessorKey: "gpus",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("gpus")} />
        ),
        cell: ({ row }) => <>{row.getValue("gpus")}</>,
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
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const taskInfo = row.original;
          return (
            <div className="flex flex-row space-x-1">
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 text-destructive hover:text-destructive disabled:text-muted-foreground"
                    title="终止 Jupyter Lab"
                    disabled={taskInfo.status !== "Running"}
                  >
                    <StopIcon />
                  </Button>
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
                        deleteJTask(taskInfo.id);
                      }}
                    >
                      删除
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          );
        },
      },
    ],
    [deleteJTask],
  );

  return (
    <div className="space-y-6">
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
        loading={isLoading}
        pagination={pagination}
        onPaginationChange={setPagination}
        rowCount={tasks?.rowCount || 0}
      ></DataTable>

      <Card className=" bg-destructive text-destructive-foreground">
        <CardHeader className="py-3" />
        <CardContent>
          分页功能开发中，搜索、排序、筛选功能暂时不可用。
        </CardContent>
      </Card>
    </div>
  );
};

export default Jupyter;
