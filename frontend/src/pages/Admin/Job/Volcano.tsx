import {
  IJobInfo,
  JobType,
  apiAdminTaskListByType,
} from "@/services/api/vcjob";
import { useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";
import { JobPhase } from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { TableDate } from "@/components/custom/TableDate";
import {
  CircleIcon,
  ClockIcon,
  MinusCircledIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
  StopIcon,
} from "@radix-ui/react-icons";
import { useRoutes, Link } from "react-router-dom";

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

const vcjobColumns: ColumnDef<IJobInfo>[] = [
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
    // cell: ({ row }) => <div>{row.getValue("name")}</div>,
    // cell: ({ row }) => (
    //   <Link
    //     to={row.original.jobName}
    //     className="underline-offset-4 hover:underline"
    //   >
    //     {row.getValue("name")}
    //   </Link>
    // ),
    cell: ({ row }) => {
      // 判断状态是否为 'Running'
      return (
        <Link
          to={row.original.jobName}
          className="underline-offset-4 hover:underline"
        >
          {row.getValue("name")}
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
      <DataTableColumnHeader column={column} title={getHeader("resources")} />
    ),
    cell: ({ row }) => {
      const resources = row.getValue<Record<string, string>>("resources");
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
      <DataTableColumnHeader column={column} title={getHeader("createdAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("createdAt")}></TableDate>;
    },
    sortingFn: "datetime",
  },
  {
    accessorKey: "startedAt",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("startedAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("startedAt")} />;
    },
    sortingFn: "datetime",
  },
  {
    accessorKey: "completedAt",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("completedAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("completedAt")}></TableDate>;
    },
    sortingFn: "datetime",
  },
];

const Volcano = () => {
  const vcjobQuery = useQuery({
    queryKey: ["admin", "tasklist", "volcanoJob"],
    queryFn: apiAdminTaskListByType,
    select: (res) => res.data.data,
  });

  return (
    <DataTable
      query={vcjobQuery}
      columns={vcjobColumns}
      toolbarConfig={toolbarConfig}
    />
  );
};

import JobDetail from "./JobDetail";
import ResourceBadges from "@/components/custom/ResourceBadges";
import NodeBadges from "@/components/custom/NodeBadges";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
// import JupyterDetail from "../../Portal/Job/Jupyter/Detail";
// export default Volcano;
export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <Volcano />,
    },
    {
      path: ":id",
      element: <JobDetail />,
      // element: <JupyterDetail />,
    },
  ]);

  return <>{routes}</>;
};
