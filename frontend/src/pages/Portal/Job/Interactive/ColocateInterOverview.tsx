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
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiJobDelete,
  apiJobBatchList,
  apiJupyterTokenGet,
} from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { Link } from "react-router-dom";
import { TableDate } from "@/components/custom/TableDate";
import { toast } from "sonner";
import { getHeader } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "./Quota";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import SplitButton from "@/components/custom/SplitButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import ResourceBadges from "@/components/custom/ResourceBadges";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
import { globalJobUrl, globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { ExternalLink, SquareIcon, Trash2Icon } from "lucide-react";

export const priorities = [
  {
    label: "高",
    value: "high",
    className: "text-amber-500 border-amber-500 bg-amber-500/10",
  },
  {
    label: "低",
    value: "low",
    className: "text-slate-500 border-slate-500 bg-slate-500/10",
  },
];

export type StatusValue =
  | "Queueing"
  | "Created"
  | "Running"
  | "Failed"
  | "Succeeded"
  | "Preempted";

export const statuses: {
  value: StatusValue;
  label: string;
  className?: string;
}[] = [
  {
    value: "Queueing",
    label: "检查中",
    className: "text-purple-500 border-purple-500 bg-purple-500/10",
  },
  {
    value: "Created",
    label: "等待中",
    className: "text-slate-500 border-slate-500 bg-slate-500/10",
  },
  {
    value: "Running",
    label: "运行中",
    className: "text-sky-500 border-sky-500 bg-sky-500/10",
  },
  {
    value: "Succeeded",
    label: "已完成",
    className: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
  },
  {
    value: "Preempted",
    label: "被抢占",
    className: "text-orange-500 border-orange-500 bg-orange-500/10",
  },
  {
    value: "Failed",
    label: "失败",
    className: "text-red-500 border-red-500 bg-red-500/10",
  },
];

export const profilingStatuses = [
  {
    value: "0",
    label: "未分析",
    className: "text-purple-500 border-purple-500 bg-purple-500/10",
  },
  {
    value: "1",
    label: "待分析",
    className: "text-slate-500 border-slate-500 bg-slate-500/10",
  },
  {
    value: "2",
    label: "分析中",
    className: "text-sky-500 border-sky-500 bg-sky-500/10",
  },
  {
    value: "3",
    label: "已分析",
    className: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
  },
  {
    value: "4",
    label: "失败",
    className: "text-red-500 border-red-500 bg-red-500/10",
  },
  {
    value: "5",
    label: "跳过",
    className: "text-slate-500 border-slate-500 bg-slate-500/10",
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索名称",
    key: "title",
  },
  filterOptions: [
    {
      key: "status",
      title: "作业状态",
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

interface ColocateJobInfo extends IJobInfo {
  id: number;
  profileStatus: string;
  priority: string;
}

const ColocateOverview = () => {
  const queryClient = useQueryClient();
  const jobType = useAtomValue(globalJobUrl);
  const userInfo = useAtomValue(globalUserInfo);

  const batchQuery = useQuery({
    queryKey: ["job", "interactive"],
    queryFn: apiJobBatchList,
    select: (res) =>
      res.data.data
        .filter((task) => task.jobType === JobType.Jupyter)
        .sort((a, b) =>
          b.createdAt.localeCompare(a.createdAt),
        ) as unknown as ColocateJobInfo[],
    refetchInterval: REFETCH_INTERVAL,
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (res) => {
      const data = res.data.data;
      window.open(`${data.baseURL}?token=${data.token}`, "_blank");
    },
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

  const batchColumns = useMemo<ColumnDef<ColocateJobInfo>[]>(
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
            to={`${row.original.id}`}
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
          const status = statuses.find(
            (status) => status.value === row.getValue("status"),
          );
          if (!status) {
            return null;
          }
          return (
            <Badge className={status.className} variant="outline">
              {status.label}
            </Badge>
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
            <Badge className={priority.className} variant="outline">
              {priority.label}
            </Badge>
          );
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
        enableSorting: false,
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
                    getPortToken(`${jobInfo.id}`);
                  }, 500);
                }}
                disabled={
                  jobInfo.status !== "Running" ||
                  userInfo.name !== jobInfo.owner
                }
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
    [deleteTask, getPortToken, userInfo.name],
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
        query={batchQuery}
        columns={batchColumns}
        toolbarConfig={toolbarConfig}
        handleMultipleDelete={(rows) => {
          rows.forEach((row) => {
            deleteTask(row.original.id.toString());
          });
          void refetchTaskList();
        }}
      ></DataTable>
    </>
  );
};

export default ColocateOverview;
