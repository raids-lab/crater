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
import { TimeDistance } from "@/components/custom/TimeDistance";
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
import SplitLinkButton from "@/components/button/SplitLinkButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import ResourceBadges from "@/components/badge/ResourceBadges";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import { globalJobUrl, globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Badge } from "@/components/ui/badge";
import { ExternalLink, SquareIcon, Trash2Icon } from "lucide-react";

export const priorities = [
  {
    label: "高",
    value: "high",
    className:
      "text-highlight-amber border-highlight-amber bg-highlight-amber/20",
  },
  {
    label: "低",
    value: "low",
    className:
      "text-highlight-slate border-highlight-slate bg-highlight-slate/20",
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
    className:
      "text-highlight-purple border-highlight-purple bg-highlight-purple/20",
  },
  {
    value: "Created",
    label: "等待中",
    className:
      "text-highlight-slate border-highlight-slate bg-highlight-slate/20",
  },
  {
    value: "Running",
    label: "运行中",
    className: "text-highlight-sky border-highlight-sky bg-highlight-sky/20",
  },
  {
    value: "Succeeded",
    label: "已完成",
    className:
      "text-highlight-emerald border-highlight-emerald bg-highlight-emerald/20",
  },
  {
    value: "Preempted",
    label: "被抢占",
    className:
      "text-highlight-orange border-highlight-orange bg-highlight-orange/20",
  },
  {
    value: "Failed",
    label: "失败",
    className: "text-highlight-red border-highlight-red bg-highlight-red/20",
  },
];

export const profilingStatuses = [
  {
    value: "0",
    label: "未分析",
    className:
      "text-highlight-purple border-highlight-purple bg-highlight-purple/20",
  },
  {
    value: "1",
    label: "待分析",
    className:
      "text-highlight-slate border-highlight-slate bg-highlight-slate/20",
  },
  {
    value: "2",
    label: "分析中",
    className: "text-highlight-sky border-highlight-sky bg-highlight-sky/20",
  },
  {
    value: "3",
    label: "已分析",
    className:
      "text-highlight-emerald border-highlight-emerald bg-highlight-emerald/20",
  },
  {
    value: "4",
    label: "失败",
    className: "text-highlight-red border-highlight-red bg-highlight-red/20",
  },
  {
    value: "5",
    label: "跳过",
    className:
      "text-highlight-slate border-highlight-slate bg-highlight-slate/20",
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
          const resources = row.getValue<Record<string, string> | undefined>(
            "resources",
          );
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
          return <TimeDistance date={row.getValue("createdAt")}></TimeDistance>;
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
          return <TimeDistance date={row.getValue("startedAt")}></TimeDistance>;
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
                className="text-primary hover:text-primary/90 h-8 w-8"
                title="打开 Jupyter Lab"
                onClick={() => {
                  toast.info("即将打开 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(`${jobInfo.id}`);
                  }, 500);
                }}
                disabled={
                  jobInfo.status !== "Running" ||
                  userInfo.name !== jobInfo.owner
                }
              >
                <ExternalLink className="size-4" />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  {jobInfo.status == "Deleted" || jobInfo.status == "Freed" ? (
                    <Button
                      variant="outline"
                      size="icon"
                      className="hover:text-destructive/90 h-8 w-8 text-red-600"
                      title="删除 Jupyter Lab"
                    >
                      <Trash2Icon className="size-4" />
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="icon"
                      className="hover:text-destructive/90 h-8 w-8 text-orange-600"
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
                          onClick={() => deleteTask(`${jobInfo.id}`)}
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
                          onClick={() => deleteTask(`${jobInfo.id}`)}
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
      <div className="grid gap-5 lg:grid-cols-2">
        <Card className="row-span-2 flex flex-col justify-between">
          <CardHeader>
            <CardTitle>交互式作业</CardTitle>
            <CardDescription className="pt-2 leading-relaxed text-balance">
              提供开箱即用的 Jupyter Lab 或 Web IDE， 可用于代码编写、调试等。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <SplitLinkButton
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
        multipleHandlers={[
          {
            title: (rows) => `停止或删除 ${rows.length} 个作业`,
            description: (rows) => (
              <>
                作业 {rows.map((row) => row.original.name).join(", ")}{" "}
                将被停止或删除，确认要继续吗？
              </>
            ),
            icon: <Trash2Icon className="text-destructive" />,
            handleSubmit: (rows) => {
              rows.forEach((row) => {
                deleteTask(row.original.jobName);
              });
            },
            isDanger: true,
          },
        ]}
      ></DataTable>
    </>
  );
};

export default ColocateOverview;
