import {
  IJobInfo,
  JobType,
  apiAdminGetJobList as apiAdminGetJobList,
  apiJobDeleteForAdmin,
  apiJupyterTokenGet,
} from "@/services/api/vcjob";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import JobPhaseLabel, { jobPhases } from "@/components/label/JobPhaseLabel";
import { JobPhase } from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  CircleIcon,
  ClockIcon,
  MinusCircledIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
  StopIcon,
  DotsHorizontalIcon,
} from "@radix-ui/react-icons";
import { Link } from "react-router-dom";
import ResourceBadges from "@/components/label/ResourceBadges";
import NodeBadges from "@/components/label/NodeBadges";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
} from "@/components/ui-custom/alert-dialog"; // 导入 AlertDialog 组件
import {
  ExternalLink,
  InfoIcon,
  LockIcon,
  RedoDotIcon,
  SquareIcon,
  Trash2Icon,
} from "lucide-react";
import TooltipButton from "@/components/custom/TooltipButton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { logger } from "@/utils/loglevel";
import { useMemo } from "react";

export type StatusValue =
  | "Queueing"
  | "Created"
  | "Pending"
  | "Running"
  | "Failed"
  | "Succeeded"
  | "Preempted"
  | "Deleted";

export const statuses: {
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

export const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "名称";
    case "jobType":
      return "类型";
    case "queue":
      return "账户";
    case "owner":
      return "用户";
    case "status":
      return "状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "completedAt":
      return "完成于";
    case "nodes":
      return "节点";
    case "resources":
      return "资源";
    default:
      return key;
  }
};

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

const AdminJobOverview = () => {
  const queryClient = useQueryClient();

  const vcjobQuery = useQuery({
    queryKey: ["admin", "tasklist", "volcanoJob"],
    queryFn: apiAdminGetJobList,
    select: (res) => res.data.data,
  });

  const refetchTaskList = async () => {
    try {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({
            queryKey: ["admin", "tasklist", "volcanoJob"],
          }),
        ),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDeleteForAdmin,
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

  const vcjobColumns = useMemo<ColumnDef<IJobInfo>[]>(
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
        cell: ({ row }) => {
          // 判断状态是否为 'Running'
          return (
            <Link
              to={row.original.jobName}
              className="flex flex-row items-center hover:text-primary"
            >
              {row.getValue("name")}
              {row.original.keepWhenLowUsage && (
                <LockIcon className="ml-1 h-4 w-4 text-muted-foreground" />
              )}
            </Link>
          );
        },
      },
      {
        accessorKey: "owner",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("owner")} />
        ),
        cell: ({ row }) => <div>{row.getValue("owner")}</div>,
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
          return <TimeDistance date={row.getValue("startedAt")} />;
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
                className="h-8 w-8 text-primary hover:bg-primary/10 hover:text-primary/90"
                tooltipContent="跳转至 Jupyter Lab"
                onClick={() => {
                  toast.info("即将跳转至 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(jobInfo.jobName);
                  }, 500);
                }}
                disabled={shouldDisable}
              >
                <ExternalLink className="h-4 w-4" />
              </TooltipButton>
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
                    <Link to={`${jobInfo.jobName}`}>
                      <DropdownMenuItem>
                        <InfoIcon className="text-emerald-600 dark:text-emerald-500" />
                        详情
                      </DropdownMenuItem>
                    </Link>
                    <Link to={`new-jupyter-vcjobs?fromJob=${jobInfo.jobName}`}>
                      <DropdownMenuItem>
                        <RedoDotIcon className="text-purple-600 dark:text-purple-500" />
                        再建
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
    <DataTable
      query={vcjobQuery}
      columns={vcjobColumns}
      toolbarConfig={toolbarConfig}
    />
  );
};

export default AdminJobOverview;
