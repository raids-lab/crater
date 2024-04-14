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
} from "@/components/ui-custom/alert-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
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
import {
  apiAiTaskDelete,
  apiAiTaskList,
  convertAiTask,
} from "@/services/api/aiTask";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { useNavigate, useRoutes } from "react-router-dom";
import AiJobDetail from "./Detail";
import { TableDate } from "@/components/custom/TableDate";
import { cn } from "@/lib/utils";
import Status from "../../Overview/Status";
import Quota from "../Jupyter/Quota";
import { REFETCH_INTERVAL } from "@/config/task";
import { toast } from "sonner";
import { getHeader, priorities, profilingStatuses, statuses } from "./statuses";
import { logger } from "@/utils/loglevel";

type TaskInfo = {
  id: number;
  title: string;
  taskType: string;
  status: string;
  priority: string;
  profileStatus: string;
  createdAt: string;
  startedAt: string;
  finishAt: string;
  gpus: string | number | undefined;
};

const toolbarConfig: DataTableToolbarConfig<string> = {
  filterInput: {
    placeholder: "搜索任务名称",
    key: "title",
  },
  filterOptions: [
    {
      key: "status",
      title: "任务状态",
      option: statuses,
    },
    {
      key: "priority",
      title: "优先级",
      option: priorities,
    },
    {
      key: "profileStatus",
      title: "分析状态",
      option: profilingStatuses,
    },
  ],
  getHeader: getHeader,
};

const AiJobHome = () => {
  const [openSheet, setOpenSheet] = useState(false);
  const [data, setData] = useState<TaskInfo[]>([]);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const {
    data: taskList,
    isLoading,
    dataUpdatedAt,
  } = useQuery({
    queryKey: ["aitask", "list"],
    queryFn: apiAiTaskList,
    select: (res) => res.data.data.rows,
    refetchInterval: REFETCH_INTERVAL,
  });

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["aitask", "list"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: (id: number) => apiAiTaskDelete(id),
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("任务已删除");
    },
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
        accessorKey: "id",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("id")} />
        ),
        cell: ({ row }) => <>{row.getValue("id")}</>,
        enableSorting: false,
      },
      {
        accessorKey: "title",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("title")} />
        ),
        cell: ({ row }) => (
          <Button
            onClick={() => navigate(`${row.original.id}`)}
            variant={"link"}
            className="h-8 px-0 text-left font-normal text-secondary-foreground"
          >
            {row.getValue("title")}
          </Button>
        ),
      },
      // {
      //   accessorKey: "taskType",
      //   header: ({ column }) => (
      //     <DataTableColumnHeader
      //       column={column}
      //       title={getHeader("taskType")}
      //     />
      //   ),
      //   cell: ({ row }) => <>{row.getValue("taskType")}</>,
      // },
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
                  "bg-sky-500 hover:bg-sky-400": status.value === "Running",
                  "bg-red-500 hover:bg-red-400": status.value === "Failed",
                  "bg-emerald-500 hover:bg-emerald-400":
                    status.value === "Succeeded",
                  "bg-orange-500 hover:bg-orange-400":
                    status.value === "Preempted",
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
            // <Badge
            //   className={cn("flex w-fit items-center px-2 font-normal", {
            //     "bg-secondary text-secondary-foreground hover:bg-secondary/80":
            //       priority.value === "low",
            //   })}
            // >
            //   {priority.icon && <priority.icon className="mr-1 h-4 w-4" />}
            //   <span>{priority.label}</span>
            // </Badge>
          );
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "profileStatus",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("profileStatus")}
          />
        ),
        cell: ({ row }) => {
          const profiling = profilingStatuses.find(
            (profiling) => profiling.value === row.getValue("profileStatus"),
          );
          if (!profiling) {
            return null;
          }
          return (
            <div className="flex items-center">
              {profiling.icon && (
                <profiling.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{profiling.label}</span>
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
                  <DropdownMenuItem onClick={() => navigate(`${taskInfo.id}`)}>
                    详情
                  </DropdownMenuItem>
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>删除</DropdownMenuItem>
                  </AlertDialogTrigger>
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
          );
        },
      },
    ],
    [deleteTask, navigate],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: TaskInfo[] = taskList
      .filter((task) => !task.isDeleted)
      .map((t) => {
        const task = convertAiTask(t);
        return {
          id: task.id,
          title: task.taskName,
          taskType: task.taskType,
          status: task.status,
          priority: task.slo ? "high" : "low",
          profileStatus:
            // 分析状态：profileStatus = 1 或 2 都显示分析中，不需要等待分析这个选项了
            task.profileStatus === 1 ? "2" : task.profileStatus.toString(),
          createdAt: task.createdAt,
          startedAt: task.startedAt,
          finishAt: task.finishAt,
          gpus: task.resourceRequest["nvidia.com/gpu"],
        };
      });
    setData(tableData);
  }, [taskList, isLoading]);

  const updatedAt = useMemo(() => {
    return new Date(dataUpdatedAt).toLocaleString();
  }, [dataUpdatedAt]);

  return (
    <>
      <div className="col-span-3 grid gap-4 md:grid-cols-2 md:gap-6 xl:grid-cols-4">
        <Status />
        <Quota />
      </div>
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
        loading={isLoading}
        className="col-span-3"
      >
        <Sheet open={openSheet} onOpenChange={setOpenSheet}>
          <SheetTrigger asChild>
            <Button className="h-8 min-w-fit">新建任务</Button>
          </SheetTrigger>
          {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
          <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
            <SheetHeader>
              <SheetTitle>新建任务</SheetTitle>
              <SheetDescription>创建一个新的 AI 训练任务</SheetDescription>
            </SheetHeader>
            <Separator className="mt-4" />
            <NewTaskForm closeSheet={() => setOpenSheet(false)} />
          </SheetContent>
        </Sheet>
      </DataTable>
      <div className="flex flex-row items-center justify-start space-x-2">
        <div className="pl-2 text-sm text-muted-foreground">
          数据更新于 {updatedAt}
        </div>
      </div>
    </>
  );
};

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <AiJobHome />,
    },
    {
      path: ":id",
      element: <AiJobDetail />,
    },
  ]);

  return <>{routes}</>;
};
