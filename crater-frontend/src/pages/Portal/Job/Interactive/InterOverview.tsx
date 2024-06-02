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
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiJobDelete,
  apiJupyterTokenGet,
  JobPhase,
  apiJobInteractiveList,
} from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
import { toast } from "sonner";
import { getHeader } from "@/pages/Portal/Job/Batch/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "./Quota";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";
import { Link } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { PlayIcon, Trash } from "lucide-react";
import SplitButton from "@/components/custom/SplitButton";
import { IVolcanoJobInfo } from "@/services/api/admin/task";

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

export const Component = () => {
  const queryClient = useQueryClient();

  const interactiveQuery = useQuery({
    queryKey: ["job", "interactive"],
    queryFn: apiJobInteractiveList,
    select: (res) => res.data.data.filter((task) => task.jobType === "jupyter"),
  });

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
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

  const interColumns = useMemo<ColumnDef<IVolcanoJobInfo>[]>(
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
                <PlayIcon className="h-4 w-4" />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="终止 Jupyter Lab"
                    >
                      <Trash className="h-4 w-4" />
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
                      onClick={() => deleteTask(taskInfo.jobName)}
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
    [deleteTask, getPortToken],
  );

  return (
    <>
      <div className="col-span-3 grid gap-4 md:grid-cols-2 md:gap-6 xl:grid-cols-4">
        <Card className="flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>交互式作业</CardTitle>
            <CardDescription className="text-balance pt-2 leading-relaxed">
              提供开箱即用的 Jupyter Lab 或 Web IDE， 可用于代码编写、调试等。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <SplitButton
              title="interactive"
              urls={[
                { url: "jupyter", name: " Jupyter Lab" },
                { url: "webide", name: " Web IDE", disabled: true },
              ]}
            />
          </CardContent>
        </Card>
        <Quota />
      </div>
      <DataTable
        query={interactiveQuery}
        columns={interColumns}
        toolbarConfig={toolbarConfig}
      ></DataTable>
    </>
  );
};
