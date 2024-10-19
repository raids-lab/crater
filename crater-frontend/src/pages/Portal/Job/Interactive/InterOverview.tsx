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
import { TableDate } from "@/components/custom/TableDate";
import { toast } from "sonner";
import { getHeader, jobToolbarConfig } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "./Quota";
import JobPhaseLabel from "@/components/label/JobPhaseLabel";
import { Link } from "react-router-dom";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { ExternalLink, SquareIcon, Trash2Icon } from "lucide-react";
import SplitButton from "@/components/custom/SplitButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import { useAtomValue } from "jotai";
import { globalJobUrl } from "@/utils/store";
import NodeBadges from "@/components/label/NodeBadges";
import ResourceBadges from "@/components/label/ResourceBadges";
import JobTypeLabel from "@/components/custom/JobTypeLabel";

const InterOverview = () => {
  const jobType = useAtomValue(globalJobUrl);
  const queryClient = useQueryClient();

  const interactiveQuery = useQuery({
    queryKey: ["job", "interactive"],
    queryFn: apiJobInteractiveList,
    select: (res) =>
      res.data.data.filter((task) => task.jobType === JobType.Jupyter),
    refetchInterval: REFETCH_INTERVAL,
  });

  const refetchTaskList = async () => {
    try {
      // 隔 200ms 并行发送所有异步请求
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ["job"] }),
        ),
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ["context", "quota"] }),
        ),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("操作成功");
    },
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (_, jobName) => {
      window.open(`/job/jupyter/${jobName}`);
    },
  });

  const interColumns = useMemo<ColumnDef<IJobInfo>[]>(
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
        accessorKey: "jobType",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("jobType")} />
        ),
        cell: ({ row }) => (
          <JobTypeLabel jobType={row.getValue<JobType>("jobType")} />
        ),
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
        accessorKey: "owner",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("owner")} />
        ),
        cell: ({ row }) => <div>{row.getValue("owner")}</div>,
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
        accessorKey: "nodes",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("nodes")} />
        ),
        cell: ({ row }) => {
          const nodes = row.getValue<string[]>("nodes");
          return <NodeBadges nodes={nodes} />;
        },
      },
      {
        accessorKey: "resources",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("resources")}
          />
        ),
        cell: ({ row }) => {
          const resources = row.getValue<Record<string, string>>("resources");
          return <ResourceBadges resources={resources} />;
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
        accessorKey: "completedAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("completedAt")}
          />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("completedAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const jobInfo = row.original;
          return (
            <div className="flex flex-row space-x-1">
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8 text-primary hover:text-primary/90"
                title="跳转至 Jupyter Lab"
                onClick={() => {
                  toast.info("即将跳转至 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(jobInfo.jobName);
                  }, 500);
                }}
                disabled={jobInfo.status !== "Running"}
              >
                <ExternalLink className="h-4 w-4" />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  {jobInfo.status == "Deleted" || jobInfo.status == "Freed" ? (
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8 text-red-600 hover:text-destructive/90"
                      title="删除 Jupyter Lab"
                    >
                      <Trash2Icon className="h-4 w-4" />
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8 text-orange-600 hover:text-destructive/90"
                      title="停止 Jupyter Lab"
                    >
                      <SquareIcon className="h-3.5 w-3.5" strokeWidth={"2.5"} />
                    </Button>
                  )}
                </AlertDialogTrigger>
                <AlertDialogContent>
                  {jobInfo.status == "Deleted" || jobInfo.status == "Freed" ? (
                    <>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除作业</AlertDialogTitle>
                        <AlertDialogDescription>
                          作业「{jobInfo?.name}
                          」将被删除，所有元数据都将被清理。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => deleteTask(jobInfo.jobName)}
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </>
                  ) : (
                    <>
                      <AlertDialogHeader>
                        <AlertDialogTitle>停止作业</AlertDialogTitle>
                        <AlertDialogDescription className="text-balance">
                          作业 {jobInfo?.name} 将停止，除{" "}
                          <span className="font-mono">
                            /home/{jobInfo.owner}
                          </span>{" "}
                          及其他挂载目录之外的文件将无法恢复，请确认已经保存好所需数据。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => deleteTask(jobInfo.jobName)}
                        >
                          停止
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </>
                  )}
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
      <div className="grid gap-5 lg:col-span-3 lg:grid-cols-4">
        <Card className="row-span-2 flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>交互式作业</CardTitle>
            <CardDescription className="text-balance pt-2 leading-relaxed">
              提供开箱即用的 Jupyter Lab 或 Web IDE， 可用于代码编写、调试等。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <SplitButton
              title="interactive"
              urls={[
                {
                  url: `portal/job/inter/new-jupyter-${jobType}`,
                  name: " Jupyter Lab",
                },
                {
                  url: "portal/job/inter/new-webide",
                  name: " Web IDE",
                  disabled: true,
                },
              ]}
            />
          </CardFooter>
        </Card>
        <Quota />
      </div>
      <DataTable
        query={interactiveQuery}
        columns={interColumns}
        toolbarConfig={jobToolbarConfig}
        handleMultipleDelete={(rows) => {
          rows.forEach((row) => {
            deleteTask(row.original.jobName);
          });
        }}
      ></DataTable>
    </>
  );
};

export default InterOverview;
