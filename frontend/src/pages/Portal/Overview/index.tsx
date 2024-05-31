import { useMemo, type FC } from "react";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { DataTable } from "@/components/custom/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useQuery } from "@tanstack/react-query";
import { apiGetNodes } from "@/services/api/node";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { UsageCell } from "@/pages/Admin/Cluster/Node";
import { getAiResource } from "@/utils/resource";
import { getHeader } from "@/pages/Admin/Job/Volcano";
import { TableDate } from "@/components/custom/TableDate";
import { JobPhase } from "@/services/api/vcjob";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";
import { IVolcanoJobInfo, apiTaskListByType } from "@/services/api/admin/task";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { CardTitle } from "@/components/ui-custom/card";

interface ResourceInfo {
  percent: number;
  description: string;
}

interface ClusterNodeInfo {
  name: string;
  cpu: ResourceInfo;
  memory: ResourceInfo;
  gpu: ResourceInfo;
}

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
      defaultValues: ["Running"],
    },
  ],
  getHeader: getHeader,
};

const nodeColumns: ColumnDef<ClusterNodeInfo>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"节点名称"} />
    ),
    cell: ({ row }) => <div>{row.getValue("name")}</div>,
    enableSorting: false,
  },
  {
    accessorKey: "cpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"CPU"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("cpu")} />,
    enableSorting: false,
  },
  {
    accessorKey: "memory",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Memory"} />
    ),
    cell: ({ row }) => (
      <UsageCell value={row.getValue<ResourceInfo>("memory")} />
    ),
    enableSorting: false,
  },
  {
    accessorKey: "gpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"GPU"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("gpu")} />,
    enableSorting: false,
  },
];

export const Component: FC = () => {
  const nodeQuery = useQuery({
    queryKey: ["overview", "nodes"],
    queryFn: () => apiGetNodes(),
    select: (res) =>
      res.data.data.rows
        .sort((a, b) => a.name.localeCompare(b.name))
        .filter((x) => x.role === "worker" && x.isReady)
        .map((x) => {
          const capacity = getAiResource(x.capacity);
          const allocated = getAiResource(x.allocated);
          return {
            name: x.name,
            cpu: {
              percent: (allocated.cpu ?? 0) / (capacity.cpu ?? 1) / 10,
              description: `${((allocated.cpu ?? 0) / 1000).toFixed(1)}/${capacity.cpu ?? 1}`,
            },
            memory: {
              percent:
                ((allocated.memoryNum ?? 0) / (capacity.memoryNum ?? 1)) * 100,
              description: `${(allocated.memoryNum ?? 0).toFixed(1)}/${(capacity.memoryNum ?? 1).toFixed(0)}Gi`,
            },
            gpu: {
              percent: capacity.gpu
                ? ((allocated.gpu ?? 0) / (capacity.gpu ?? 1)) * 100
                : 0,
              description: capacity.gpu
                ? `${(allocated.gpu ?? 0).toFixed(0)}/${capacity.gpu ?? 1}`
                : "",
            },
          };
        }),
  });

  const vcJobColumns = useMemo<ColumnDef<IVolcanoJobInfo>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
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
          return (
            <div className="flex items-start gap-1">
              {nodes?.map((node) => (
                <Badge key={node} variant="secondary">
                  {node}
                </Badge>
              ))}
            </div>
          );
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
          return (
            <div className="flex items-start gap-1">
              {Object.entries(resources).map(([key, value]) => (
                <Badge key={key} variant="secondary">
                  {key.replace("nvidia.com/", "")}: {value}
                </Badge>
              ))}
            </div>
          );
        },
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
    ],
    [],
  );

  const vcjobQuery = useQuery({
    queryKey: ["admin", "tasklist", "volcanoJob"],
    queryFn: () => apiTaskListByType(),
    select: (res) => res.data.data,
  });

  return (
    <>
      <div className="grid items-start gap-4 md:gap-8 lg:col-span-3">
        <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-4 lg:col-span-3 xl:grid-cols-4">
          <Card className="sm:col-span-2">
            <CardHeader className="pb-3">
              <CardTitle>快速开始</CardTitle>
              <CardDescription className="text-balance leading-relaxed">
                在 Crater 启动 Jupyter Lab、打包训练镜像、
                启动深度学习训练作业等。
              </CardDescription>
            </CardHeader>
            <CardFooter>
              <Button>Create New Task</Button>
            </CardFooter>
          </Card>
          <Card className="sm:col-span-2">
            <CardHeader className="pb-3">
              <CardTitle>作业统计</CardTitle>
              <CardDescription className="text-balance leading-relaxed">
                在 Crater 启动 Jupyter Lab、打包训练镜像、
                启动深度学习训练作业等。
              </CardDescription>
            </CardHeader>
            <CardFooter></CardFooter>
          </Card>
        </div>
        <DataTable
          info={{
            title: "作业信息",
            description: "查看集群作业的运行情况",
          }}
          query={vcjobQuery}
          columns={vcJobColumns}
          toolbarConfig={toolbarConfig}
        ></DataTable>
        <DataTable
          info={{
            title: "节点信息",
            description: "查看集群节点的资源使用情况",
          }}
          query={nodeQuery}
          columns={nodeColumns}
        ></DataTable>
      </div>
    </>
  );
};
