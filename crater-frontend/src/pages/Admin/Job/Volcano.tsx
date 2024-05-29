import { apiAdminTaskListByType } from "@/services/api/admin/task";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";
import { useEffect, useMemo, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";
import { JobPhase } from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
import { Badge } from "@/components/ui/badge";
import { Resource } from "@/pages/Portal/Job/Jupyter/Detail";
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
  CircleIcon,
  ClockIcon,
  MinusCircledIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
  StopIcon,
} from "@radix-ui/react-icons";

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
    case "jobName":
      return "作业名称";
    case "jobType":
      return "作业类型";
    case "queue":
      return "队列";
    case "userName":
      return "创建用户";
    case "status":
      return "状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "nodeName":
      return "节点";
    case "completedAt":
      return "完成于";
    case "resource":
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

export interface VolcanoJobInfo {
  name: string;
  jobName: string;
  userName: string;
  jobType: string;
  queue: string;
  status: string;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodeName: string;
  resource: string;
}
const Volcano: FC = () => {
  const [data, setData] = useState<VolcanoJobInfo[]>([]);
  const { data: taskList, isLoading } = useQuery({
    queryKey: ["admin", "tasklist", "volcanoJob"],
    queryFn: () => apiAdminTaskListByType(),
    select: (res) => res.data.data,
  });

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: VolcanoJobInfo[] = taskList
      //.filter((task) => !task.isDeleted)
      .map((task) => {
        //const task = convertJTask(t);
        return {
          name: task.name,
          jobName: task.jobName,
          jobType: task.jobType,
          userName: task.userName,
          queue: task.queue,
          status: task.status,
          createdAt: task.createdAt,
          startedAt: task.startedAt,
          completedAt: task.completedAt,
          nodeName: task.nodeName,
          resource: task.resource,
        };
      });
    setData(tableData);
  }, [taskList, isLoading]);
  const handleResourceData = (resourceJson: string): Resource => {
    try {
      return JSON.parse(resourceJson) as Resource;
    } catch (error) {
      return {};
    }
  };

  const columns = useMemo<ColumnDef<VolcanoJobInfo>[]>(
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
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("jobName")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
      },
      {
        accessorKey: "jobType",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("jobType")} />
        ),
        cell: ({ row }) => <div>{row.getValue("jobType")}</div>,
      },
      {
        accessorKey: "userName",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("userName")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("userName")}</div>,
      },
      {
        accessorKey: "queue",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("queue")} />
        ),
        cell: ({ row }) => <div>{row.getValue("queue")}</div>,
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
        accessorKey: "nodeName",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("nodeName")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("nodeName")}</div>,
      },
      {
        accessorKey: "resource",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("resource")}
          />
        ),
        cell: ({ row }) => (
          <div className="item-start flex flex-col gap-1">
            {Object.entries(handleResourceData(row.getValue("resource"))).map(
              ([key, value]) => (
                <Badge
                  key={key}
                  variant="secondary"
                  className="gap-2 font-medium"
                >
                  {" "}
                  {key}: {String(value)}{" "}
                </Badge>
              ),
            )}
          </div>
        ),
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
          const taskInfo = row.original;
          if (row.getValue("isDeleted") === "已删除") {
            return (
              <div>
                <Button
                  variant="outline"
                  className="h-8 w-8 p-0 hover:text-red-700"
                  title="终止 Jupyter Lab"
                  disabled={row.getValue("isDeleted") === "已删除"}
                >
                  <StopIcon />
                </Button>
              </div>
            );
          } else {
            return (
              <div
                className="flex flex-row space-x-1"
                aria-disabled={row.getValue("status") === "Deleted"}
              >
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <div>
                      <Button
                        variant="outline"
                        className="h-8 w-8 p-0 hover:text-red-700"
                        title="终止 Jupyter Lab"
                        disabled={row.getValue("isDeleted") === "已删除"}
                      >
                        <StopIcon />
                      </Button>
                    </div>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除作业</AlertDialogTitle>
                      <AlertDialogDescription>
                        作业「{taskInfo?.jobName}
                        」将停止，请确认已经保存好所需数据。
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={() => {
                          // check if browser support clipboard
                        }}
                      >
                        删除
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            );
          }
        },
      },
    ],
    [],
  );

  return (
    <DataTable
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      loading={isLoading}
    />
  );
};

export default Volcano;
