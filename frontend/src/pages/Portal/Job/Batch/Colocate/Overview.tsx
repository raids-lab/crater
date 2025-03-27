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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiJobDelete, apiJobBatchList } from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Link, useNavigate } from "react-router-dom";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { toast } from "sonner";
import { getHeader } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "../../Interactive/Quota";
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
import { globalJobUrl } from "@/utils/store";
import { useAtomValue } from "jotai";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Badge } from "@/components/ui/badge";
import { Trash2Icon } from "lucide-react";

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
    {
      key: "profileStatus",
      title: "分析状态",
      option: profilingStatuses,
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
  const navigate = useNavigate();
  const jobType = useAtomValue(globalJobUrl);

  const batchQuery = useQuery({
    queryKey: ["job", "batch"],
    queryFn: apiJobBatchList,
    select: (res) =>
      res.data.data
        .filter((task) => task.jobType !== JobType.Jupyter)
        .sort((a, b) =>
          b.createdAt.localeCompare(a.createdAt),
        ) as unknown as ColocateJobInfo[],
    refetchInterval: REFETCH_INTERVAL,
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
        accessorKey: "profileStatus",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("profileStatus")}
          />
        ),
        cell: ({ row }) => {
          let profiling = profilingStatuses.find(
            (profiling) => profiling.value === row.getValue("profileStatus"),
          );
          if (!profiling) {
            return null;
          }
          if (row.getValue<string>("status") === "Succeeded") {
            profiling = {
              value: "3",
              label: "已分析",
              className:
                "text-highlight-emerald border-highlight-emerald bg-highlight-emerald/20",
            };
          }
          return (
            <Badge className={profiling.className} variant="outline">
              {profiling.label}
            </Badge>
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
        accessorKey: "completedAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("completedAt")}
          />
        ),
        cell: ({ row }) => {
          return (
            <TimeDistance date={row.getValue("completedAt")}></TimeDistance>
          );
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
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="text-muted-foreground text-xs">
                    操作
                  </DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => navigate(`${taskInfo.jobName}`)}
                  >
                    详情
                  </DropdownMenuItem>
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>删除</DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除作业</AlertDialogTitle>
                  <AlertDialogDescription>
                    作业「{taskInfo?.name}」将不再可见，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => deleteTask(taskInfo.id.toString())}
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

  return (
    <>
      <div className="grid gap-5 lg:col-span-3 lg:grid-cols-4">
        <Card className="row-span-2 flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>批处理作业</CardTitle>
            <CardDescription className="pt-2 leading-relaxed text-balance">
              指无须人工干预而执行系列程序的作业，包含单机作业、Pytorch
              分布式训练作业、 Tensorflow 分布式训练作业、Ray 分布式训练作业、
              OpenMPI 分布式计算作业等。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <SplitLinkButton
              title="batch"
              urls={[
                {
                  url: "portal/job/batch/new-tensorflow",
                  name: " Tensorflow 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-pytorch",
                  name: " Pytorch 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-ray",
                  name: " Ray 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-deepspeed",
                  name: " DeepSpeed 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-openmpi",
                  name: " OpenMPI 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-" + jobType,
                  name: "自定义作业（单机）",
                },
              ]}
            />
          </CardFooter>
        </Card>
        <Quota />
      </div>
      <DataTable
        info={{
          title: "批处理作业",
          description: "批处理作业是一种无需人工干预的作业",
        }}
        storageKey="portal_aijob_batch"
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
