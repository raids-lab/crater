import {
  IJobInfo,
  JobType,
  apiAdminGetJobList as apiAdminGetJobList,
  apiJobDeleteForAdmin,
  apiJobToggleKeepForAdmin,
} from "@/services/api/vcjob";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import JobPhaseLabel, { jobPhases } from "@/components/badge/JobPhaseBadge";
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
import ResourceBadges from "@/components/badge/ResourceBadges";
import NodeBadges from "@/components/badge/NodeBadges";
import JobTypeLabel, { jobTypes } from "@/components/badge/JobTypeBadge";
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
} from "@/components/ui-custom/alert-dialog";
import {
  InfoIcon,
  LockIcon,
  SquareIcon,
  Trash2Icon,
  UnlockIcon,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { logger } from "@/utils/loglevel";
import { useMemo, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { JobNameCell } from "@/components/label/JobNameLabel";

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
      key: "jobType",
      title: "类型",
      option: jobTypes,
    },
    {
      key: "status",
      title: "状态",
      option: jobPhases,
      defaultValues: ["Running"],
    },
  ],
  getHeader: getHeader,
};

const AdminJobOverview = () => {
  const queryClient = useQueryClient();
  const [days, setDays] = useState(7);

  const vcjobQuery = useQuery({
    queryKey: ["admin", "tasklist", "job", days],
    queryFn: () => apiAdminGetJobList(days),
    select: (res) => res.data.data,
  });

  const refetchTaskList = async () => {
    try {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({
            queryKey: ["admin", "tasklist", "job", days],
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

  const { mutate: toggleKeepTask } = useMutation({
    mutationFn: apiJobToggleKeepForAdmin,
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("操作成功");
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
        cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
      },
      {
        accessorKey: "owner",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("owner")} />
        ),
        cell: ({ row }) => <div>{row.getValue("owner")}</div>,
      },
      {
        accessorKey: "queue",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("queue")} />
        ),
        cell: ({ row }) => <div>{row.getValue("queue")}</div>,
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
          const shouldStop =
            jobInfo.status !== "Deleted" && jobInfo.status !== "Freed";
          return (
            <div className="flex flex-row space-x-1">
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
                        <InfoIcon className="text-highlight-emerald" />
                        详情
                      </DropdownMenuItem>
                    </Link>
                    <DropdownMenuItem
                      onClick={() => toggleKeepTask(jobInfo.jobName)}
                      title="设置作业自动清除策略"
                    >
                      {row.original.keepWhenLowUsage ? (
                        <UnlockIcon className="text-highlight-purple" />
                      ) : (
                        <LockIcon className="text-highlight-purple" />
                      )}
                      {row.original.keepWhenLowUsage ? "解锁" : "锁定"}
                    </DropdownMenuItem>
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem className="group">
                        {shouldStop ? (
                          <SquareIcon className="text-highlight-orange" />
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
    [deleteTask, toggleKeepTask],
  );

  return (
    <DataTable
      info={{
        title: "作业管理",
        description:
          "管理员可对作业进行锁定以避免被定时策略清理，或手动停止或删除用户的作业",
      }}
      query={vcjobQuery}
      columns={vcjobColumns}
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
        {
          title: (rows) => `锁定或解锁 ${rows.length} 个作业`,
          description: (rows) => (
            <>
              作业 {rows.map((row) => row.original.name).join(", ")}{" "}
              将被锁定或解锁，确认要继续吗？
            </>
          ),
          icon: <LockIcon className="text-highlight-purple" />,
          handleSubmit: (rows) => {
            rows.forEach((row) => {
              toggleKeepTask(row.original.jobName);
            });
          },
          isDanger: false,
        },
      ]}
    >
      <Select
        value={days.toString()}
        onValueChange={(value) => {
          setDays(parseInt(value));
        }}
      >
        <SelectTrigger className="bg-background h-9 pr-2 pl-3">
          <SelectValue placeholder={days.toString()} />
        </SelectTrigger>
        <SelectContent side="top">
          {[7, 14, 30, 90, -1].map((pageSize) => (
            <SelectItem key={pageSize} value={`${pageSize}`}>
              {pageSize === -1 ? "全部" : `近 ${pageSize} 天`}数据
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </DataTable>
  );
};

export default AdminJobOverview;
