import { PlayIcon, PlusCircledIcon, StopIcon } from "@radix-ui/react-icons";
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
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  IJupyterResp,
  apiJupyterDelete,
  apiJupyterTokenGet,
  apiJupyterList,
  JobPhase,
} from "@/services/api/jupyterTask";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
import Status from "../../Overview/Status";
import { REFETCH_INTERVAL } from "@/config/task";
import { toast } from "sonner";
import { getHeader } from "@/pages/Portal/Job/Ai/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "./Quota";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";
import { Link, useNavigate } from "react-router-dom";

interface JTaskInfo extends IJupyterResp {}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索作业名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "status",
      title: "作业状态",
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
};

export const JupyterOverview = () => {
  const [data, setData] = useState<JTaskInfo[]>([]);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const {
    data: taskList,
    isLoading,
    dataUpdatedAt,
  } = useQuery({
    queryKey: ["jupyter", "list"],
    queryFn: () => apiJupyterList(),
    select: (res) => res.data.data,
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
    mutationFn: (jobName: string) => apiJupyterDelete(jobName),
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("作业已删除");
    },
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (_, jobName) => {
      window.open(`/job/jupyter/${jobName}`);
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
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => (
          <Link
            to={row.original.jobName}
            className="underline-offset-4 hover:underline"
          >
            {row.getValue("name")}
          </Link>
        ),
      },
      {
        accessorKey: "status",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("status")} />
        ),
        cell: ({ row }) => {
          return <JobPhaseLabel jobPhase={row.getValue<JobPhase>("status")} />;
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
                    getPortToken(taskInfo.jobName);
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
                    <AlertDialogTitle>删除作业</AlertDialogTitle>
                    <AlertDialogDescription>
                      作业「{taskInfo?.name}
                      」将停止，请确认已经保存好所需数据。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        // check if browser support clipboard
                        deleteJTask(taskInfo.jobName);
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
    const tableData = taskList;
    setData(tableData);
  }, [taskList, isLoading]);

  const updatedAt = new Date(dataUpdatedAt).toLocaleString();

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
        <Button onClick={() => navigate("new")} className="h-8">
          <PlusCircledIcon className="mr-1.5 h-4 w-4" />
          新建作业
        </Button>
      </DataTable>
      <div className="flex flex-row items-center justify-start space-x-2">
        <div className="pl-2 text-sm text-muted-foreground">
          数据更新于 {updatedAt}
        </div>
      </div>
    </>
  );
};

export default JupyterOverview;
