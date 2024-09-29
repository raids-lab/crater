import { useMemo, type FC } from "react";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { DataTable } from "@/components/custom/DataTable";
import { useQuery } from "@tanstack/react-query";
import { apiGetNodes } from "@/services/api/node";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { nodeColumns, ClusterNodeInfo } from "@/components/custom/NodeList";
import { getAiResource } from "@/utils/resource";
import { getHeader } from "@/pages/Admin/Job/Volcano";
import { TableDate } from "@/components/custom/TableDate";
import { JobPhase } from "@/services/api/vcjob";
import JobPhaseLabel, {
  aijobPhases,
  jobPhases,
} from "@/components/custom/JobPhaseLabel";
import { IJobInfo, JobType, apiJobAllList } from "@/services/api/vcjob";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { CardTitle } from "@/components/ui-custom/card";
import NivoPie from "@/components/chart/NivoPie";
import SplitButton from "@/components/custom/SplitButton";
import { Button } from "@/components/ui/button";
import { BoxIcon, FileTextIcon } from "lucide-react";
import { useNavigate, useRoutes } from "react-router-dom";
import LoadingCircleIcon from "@/components/icon/LoadingCircleIcon";
import ResourceBadges from "@/components/custom/ResourceBadges";
import NodeBadges from "@/components/custom/NodeBadges";
import JobTypeLabel, { jobTypes } from "@/components/custom/JobTypeLabel";
import { REFETCH_INTERVAL } from "@/config/task";
import { useAtomValue } from "jotai";
import { globalJobUrl, globalSettings } from "@/utils/store";
import NodeDetail from "@/components/custom/NodeDetail";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索账户名称",
    key: "queue",
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

interface PieCardProps extends React.HTMLAttributes<HTMLDivElement> {
  cardTitle: string;
  isLoading?: boolean;
}

const PieCard = ({ children, cardTitle, isLoading }: PieCardProps) => {
  return (
    <Card className="relative">
      {isLoading && (
        <div className="absolute bottom-0 left-0 right-0 top-0 z-10 flex items-center justify-center">
          <LoadingCircleIcon />
        </div>
      )}
      <CardHeader className="pb-3">
        <CardTitle>{cardTitle}</CardTitle>
      </CardHeader>
      <div className="flex h-44 items-center justify-center px-2">
        {children}
      </div>
    </Card>
  );
};

export const Component: FC = () => {
  const jobType = useAtomValue(globalJobUrl);
  const { scheduler } = useAtomValue(globalSettings);

  const nodeQuery = useQuery({
    queryKey: ["overview", "nodes"],
    queryFn: apiGetNodes,
    select: (res) =>
      res.data.data.rows
        .sort((a, b) => a.name.localeCompare(b.name))
        .sort((a, b) => b.name.localeCompare(a.name))
        .map((x) => {
          const capacity = getAiResource(x.capacity);
          const allocated = getAiResource(x.allocated);
          return {
            type: x.type,
            name: x.name,
            isReady: x.isReady,
            role: x.role,
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
            labels: x.labels,
          } as ClusterNodeInfo;
        }),
  });

  const vcJobColumns = useMemo<ColumnDef<IJobInfo>[]>(
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
        accessorKey: "queue",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("queue")} />
        ),
        cell: ({ row }) => <div>{row.getValue("queue")}</div>,
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
    ],
    [],
  );

  const jobQuery = useQuery({
    queryKey: ["overview", "joblist"],
    queryFn: apiJobAllList,
    select: (res) => res.data.data,
    refetchInterval: REFETCH_INTERVAL,
  });

  const jobStatus = useMemo(() => {
    if (!jobQuery.data) {
      return [];
    }
    const data = jobQuery.data;
    const counts = data.reduce(
      (acc, item) => {
        const phase = item.status;
        if (!acc[phase]) {
          acc[phase] = 0;
        }
        acc[phase] += 1;
        return acc;
      },
      {} as Record<string, number>,
    );
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: phase,
      value: count,
    }));
  }, [jobQuery.data]);

  const queueStatus = useMemo(() => {
    if (!jobQuery.data) {
      return [];
    }
    const data = jobQuery.data;
    const counts = data
      .filter((job) => job.status == "Running")
      .reduce(
        (acc, item) => {
          const queue = item.queue;
          if (!acc[queue]) {
            acc[queue] = 0;
          }
          acc[queue] += 1;
          return acc;
        },
        {} as Record<string, number>,
      );
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: phase,
      value: count,
    }));
  }, [jobQuery.data]);

  const typeStatus = useMemo(() => {
    if (!jobQuery.data) {
      return [];
    }
    const data = jobQuery.data;
    const counts = data
      .filter((job) => job.status == "Running")
      .reduce(
        (acc, item) => {
          const type = item.jobType;
          if (!acc[type]) {
            acc[type] = 0;
          }
          acc[type] += 1;
          return acc;
        },
        {} as Record<string, number>,
      );
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: phase,
      value: count,
    }));
  }, [jobQuery.data]);

  const gpuStatus = useMemo(() => {
    if (!jobQuery.data) {
      return [];
    }
    const data = jobQuery.data;
    const counts = data
      .filter((job) => job.status == "Running")
      .reduce(
        (acc, item) => {
          const resources = item.resources;
          for (const [k, value] of Object.entries(resources)) {
            if (k.startsWith("nvidia.com")) {
              const key = k.replace("nvidia.com/", "");
              if (!acc[key]) {
                acc[key] = 0;
              }
              acc[key] += parseInt(value);
            }
          }
          return acc;
        },
        {} as Record<string, number>,
      );
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: phase,
      value: count,
    }));
  }, [jobQuery.data]);

  const navigate = useNavigate();

  const mainElement = (
    <>
      <div className="grid gap-4 md:gap-6 lg:col-span-3 lg:grid-cols-3">
        <Card className="col-span-2 flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>快速开始</CardTitle>
            <CardDescription className="text-balance pt-2 leading-relaxed">
              在 Crater 启动批处理或交互式计算作业、定制镜像、启动微服务或
              Serverless 等。
            </CardDescription>
          </CardHeader>
          <CardFooter className="flex flex-row gap-3">
            <SplitButton
              title="overview"
              urls={[
                {
                  url: "portal/job/batch/new-tensorflow",
                  name: " Tensorflow 作业",
                },
                {
                  url: "portal/job/batch/new-pytorch",
                  name: " Pytorch 作业",
                },
                {
                  url: `portal/job/batch/new-${jobType}`,
                  name: "自定义作业（单机）",
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
                { url: "portal/job/inter/new-jupyter", name: " Jupyter Lab" },
              ]}
            />
            <Button
              variant="secondary"
              onClick={() => navigate("/portal/image/createimage")}
              className="hidden xl:flex"
            >
              <BoxIcon className="mr-2 h-4 w-4" />
              镜像制作
            </Button>
            <Button
              variant="secondary"
              onClick={() =>
                window.open("https://crater.act.buaa.edu.cn/website/docs/intro")
              }
              className="hidden lg:flex"
            >
              <FileTextIcon className="mr-2 h-4 w-4" />
              使用说明
            </Button>
          </CardFooter>
        </Card>
        <PieCard cardTitle="作业统计" isLoading={jobQuery.isLoading}>
          <NivoPie
            data={jobStatus}
            margin={{ top: 25, bottom: 30 }}
            colors={({ id }) => {
              const phases = scheduler === "volcano" ? jobPhases : aijobPhases;
              return phases.find((x) => x.value === id)?.color ?? "#000";
            }}
            arcLabelsTextColor="#ffffff"
          />
        </PieCard>
        <PieCard cardTitle="账户统计" isLoading={jobQuery.isLoading}>
          <NivoPie data={queueStatus} margin={{ top: 25, bottom: 30 }} />
        </PieCard>
        <PieCard cardTitle="类型统计" isLoading={jobQuery.isLoading}>
          <NivoPie
            data={typeStatus}
            margin={{ top: 25, bottom: 30 }}
            colors={{ scheme: "set2" }}
          />
        </PieCard>
        <PieCard cardTitle="已申请 GPU" isLoading={jobQuery.isLoading}>
          <NivoPie
            data={gpuStatus}
            margin={{ top: 25, bottom: 30 }}
            colors={{ scheme: "accent" }}
          />
        </PieCard>
      </div>
      <DataTable
        info={{
          title: "作业信息",
          description: "查看集群作业的运行情况",
        }}
        query={jobQuery}
        columns={vcJobColumns}
        toolbarConfig={toolbarConfig}
      />
      <DataTable
        info={{
          title: "节点信息",
          description: "集群节点维度的资源分配情况",
        }}
        query={nodeQuery}
        columns={nodeColumns}
      ></DataTable>
    </>
  );

  const routes = useRoutes([
    {
      index: true,
      element: mainElement,
    },
    {
      path: ":id",
      element: <NodeDetail />,
    },
  ]);

  return <>{routes}</>;

  return mainElement;
};
