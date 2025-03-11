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
import { TimeDistance } from "@/components/custom/TimeDistance";
import { toast } from "sonner";
import { getHeader, jobToolbarConfig } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import JobPhaseLabel from "@/components/badge/JobPhaseBadge";
import { Link } from "react-router-dom";
import {
  ExternalLink,
  InfoIcon,
  LockIcon,
  RedoDotIcon,
  SquareIcon,
  Trash2Icon,
} from "lucide-react";
import SplitLinkButton from "@/components/button/SplitLinkButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import { useAtomValue } from "jotai";
import { globalJobUrl } from "@/utils/store";
import NodeBadges from "@/components/badge/NodeBadges";
import ResourceBadges from "@/components/badge/ResourceBadges";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import TooltipButton from "@/components/custom/TooltipButton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Button } from "@/components/ui/button";
import TooltipLink from "@/components/label/TooltipLink";
import DocsButton from "@/components/button/DocsButton";

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
          <TooltipLink
            name={
              <div className="flex flex-row items-center">
                {row.getValue("name")}
                {row.original.keepWhenLowUsage && (
                  <LockIcon className="text-muted-foreground ml-1 size-4" />
                )}
              </div>
            }
            to={row.original.jobName}
            tooltip={
              `查看 ${row.getValue("name")} 作业详情` +
              (row.original.keepWhenLowUsage
                ? "（已锁定，低利用率仍保留）"
                : "")
            }
          />
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
          const jobInfo = row.original;
          const shouldDisable = jobInfo.status !== "Running";
          const shouldStop =
            jobInfo.status !== "Deleted" && jobInfo.status !== "Freed";
          return (
            <div className="flex flex-row space-x-1">
              <TooltipButton
                variant="ghost"
                size="icon"
                className="text-primary hover:bg-primary/10 hover:text-primary/90 h-8 w-8"
                tooltipContent="打开 Jupyter Lab"
                onClick={() => {
                  toast.info("即将打开 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(jobInfo.jobName);
                  }, 500);
                }}
                disabled={shouldDisable}
              >
                <ExternalLink className="size-4" />
              </TooltipButton>
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
                    <Link to={`${jobInfo.jobName}`}>
                      <DropdownMenuItem>
                        <InfoIcon className="text-emerald-600 dark:text-emerald-500" />
                        详情
                      </DropdownMenuItem>
                    </Link>
                    <Link to={`new-jupyter-vcjobs?fromJob=${jobInfo.jobName}`}>
                      <DropdownMenuItem>
                        <RedoDotIcon className="text-purple-600 dark:text-purple-500" />
                        克隆
                      </DropdownMenuItem>
                    </Link>
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem className="group">
                        {shouldStop ? (
                          <SquareIcon className="text-orange-600 dark:text-orange-500" />
                        ) : (
                          <Trash2Icon className="text-destructive" />
                        )}
                        {shouldStop ? "停止" : "删除"}
                      </DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {shouldStop ? "停止作业" : "删除作业"}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {shouldStop
                        ? `作业 ${jobInfo?.name} 将停止，除挂载目录之外的文件将无法恢复，请确认已经保存好所需数据。`
                        : `作业 ${jobInfo?.name} 将被删除，所有元数据都将被清理。`}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => deleteTask(jobInfo.jobName)}
                    >
                      {shouldStop ? "停止" : "删除"}
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
      <DataTable
        info={{
          title: "交互式作业",
          description:
            "提供开箱即用的 Jupyter Lab 或 Web IDE， 可用于测试、调试等",
        }}
        query={interactiveQuery}
        columns={interColumns}
        toolbarConfig={jobToolbarConfig}
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
      >
        <div className="flex flex-row gap-3">
          <DocsButton title="查看文档" url="quick-start/interactive" />
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
        </div>
      </DataTable>
    </>
  );
};

export default InterOverview;
