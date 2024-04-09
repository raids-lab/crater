import { PlayIcon, StopIcon } from "@radix-ui/react-icons";
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
  apiJTaskDelete,
  apiJTaskList,
  convertJTask,
  apiJTaskGetPortToken,
} from "@/services/api/jupyterTask";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
import { cn } from "@/lib/utils";
import Status from "../../Overview/Status";
import Quota from "../../Overview/Quota";
import { REFETCH_INTERVAL } from "@/config/task";
import { toast } from "sonner";
import { getHeader, statuses } from "@/pages/Portal/Job/Ai/statuses";
import { logger } from "@/utils/loglevel";

type JTaskInfo = {
  id: number;
  title: string;
  taskType: string;
  status: string;
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
  ],
  getHeader: getHeader,
};

export const Component = () => {
  const [openSheet, setOpenSheet] = useState(false);
  const [data, setData] = useState<JTaskInfo[]>([]);
  const queryClient = useQueryClient();

  const {
    data: taskList,
    isLoading,
    dataUpdatedAt,
  } = useQuery({
    queryKey: ["jupyter", "list"],
    queryFn: apiJTaskList,
    select: (res) => res.data.data.rows,
    refetchInterval: REFETCH_INTERVAL,
  });

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jupyter", "list"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteJTask } = useMutation({
    mutationFn: (id: number) => apiJTaskDelete(id),
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("任务已删除");
    },
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (id: number) => apiJTaskGetPortToken(id),
    onSuccess: (res, id) => {
      const jupyterPort = res.data.data.port;
      const jupyterToken = res.data.data.token;
      if (jupyterPort) {
        window.open(`http://192.168.5.60:${jupyterPort}?token=${jupyterToken}`);
      } else {
        window.open(`/job/jupyter/${id}`);
      }
    },
  });

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
          <div>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(value) => row.toggleSelected(!!value)}
              aria-label="Select row"
            />
          </div>
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
        cell: ({ row }) => <>{row.getValue("title")}</>,
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
              <Button
                variant="outline"
                className="h-8 w-8 p-0 hover:text-sky-700"
                title="运行 Jupyter Lab"
                onClick={() => {
                  toast.info("即将跳转至 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(taskInfo.id);
                  }, 500);
                }}
                disabled={row.getValue("status") !== "Running"}
              >
                <PlayIcon />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="终止 Jupyter Lab"
                    >
                      <StopIcon />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除任务</AlertDialogTitle>
                    <AlertDialogDescription>
                      任务「{taskInfo?.title}
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
    [deleteJTask, getPortToken],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: JTaskInfo[] = taskList
      .filter((task) => !task.isDeleted)
      .map((t) => {
        const task = convertJTask(t);
        return {
          id: task.id,
          title: task.taskName,
          taskType: task.taskType,
          status: task.status,
          //priority: task.slo ? "high" : "low",
          // profileStatus:
          //   // 分析状态：profileStatus = 1 或 2 都显示分析中，不需要等待分析这个选项了
          //   task.profileStatus === 1 ? "2" : task.profileStatus.toString(),
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
      <div className="col-span-3 grid grid-cols-2 gap-6">
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
              <SheetDescription>创建一个新的 Jupyter 训练任务</SheetDescription>
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
