import { useMemo, type FC } from "react";
import {
  Card,
  CardTitle,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { DataTable } from "@/components/custom/DataTable";
import { useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { nodeColumns } from "@/components/node/NodeList";
import { getHeader } from "@/pages/Admin/Job/Overview";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { JobPhase } from "@/services/api/vcjob";
import JobPhaseLabel, {
  aijobPhases,
  jobPhases,
} from "@/components/badge/JobPhaseBadge";
import { IJobInfo, JobType, apiJobAllList } from "@/services/api/vcjob";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NivoPie from "@/components/chart/NivoPie";
import SplitButton from "@/components/custom/SplitButton";
import { Button } from "@/components/ui/button";
import {
  BookOpenIcon,
  BoxIcon,
  FlaskConicalIcon,
  LayoutGridIcon,
  UserRoundIcon,
  ZapIcon,
} from "lucide-react";
import { useNavigate, useRoutes } from "react-router-dom";
import ResourceBadges from "@/components/badge/ResourceBadges";
import NodeBadges from "@/components/badge/NodeBadges";
import JobTypeLabel, { jobTypes } from "@/components/badge/JobTypeBadge";
import { REFETCH_INTERVAL } from "@/config/task";
import { useAtomValue } from "jotai";
import { globalJobUrl, globalSettings } from "@/utils/store";
import NodeDetail from "@/components/node/NodeDetail";
import useNodeQuery from "@/hooks/query/useNodeQuery";
import GpuIcon from "@/components/icon/GpuIcon";
import PieCard from "@/components/chart/PieCard";

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

export const Component: FC = () => {
  const jobType = useAtomValue(globalJobUrl);
  const { scheduler } = useAtomValue(globalSettings);

  const nodeQuery = useNodeQuery(true);

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

  const userStatus = useMemo(() => {
    if (!jobQuery.data) {
      return [];
    }
    const data = jobQuery.data;
    const counts = data
      .filter((job) => job.status == "Running")
      .reduce(
        (acc, item) => {
          const owner = item.owner;
          if (!acc[owner]) {
            acc[owner] = 0;
          }
          acc[owner] += 1;
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
          for (const [k, value] of Object.entries(resources ?? {})) {
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
            <CardTitle className="flex flex-row items-center">
              <ZapIcon className="mr-1 text-primary" />
              快速开始
            </CardTitle>
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
                  url: `portal/job/inter/new-jupyter-${jobType}`,
                  name: " Jupyter Lab",
                },
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
              ]}
            />
            <Button
              variant="secondary"
              onClick={() => navigate("/portal/image/createimage")}
              className="hidden xl:flex"
            >
              <BoxIcon className="size-4" />
              镜像仓库
            </Button>
            <Button
              variant="secondary"
              onClick={() =>
                window.open(
                  `https://${import.meta.env.VITE_HOST}/website/docs/intro`,
                )
              }
              className="hidden lg:flex"
            >
              <BookOpenIcon className="size-4" />
              平台文档
            </Button>
          </CardFooter>
        </Card>
        <PieCard
          icon={FlaskConicalIcon}
          cardTitle="作业状态"
          cardDescription="查看集群所有作业的状态统计"
          isLoading={jobQuery.isLoading}
        >
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
        <PieCard
          icon={UserRoundIcon}
          cardTitle="用户统计"
          cardDescription="当前正在运行作业所属的用户"
          isLoading={jobQuery.isLoading}
        >
          <NivoPie data={userStatus} margin={{ top: 25, bottom: 30 }} />
        </PieCard>
        <PieCard
          icon={LayoutGridIcon}
          cardTitle="作业类型"
          cardDescription="当前正在运行作业所属的类型"
          isLoading={jobQuery.isLoading}
        >
          <NivoPie
            data={typeStatus}
            margin={{ top: 25, bottom: 30 }}
            colors={{ scheme: "set2" }}
          />
        </PieCard>
        <PieCard
          icon={GpuIcon}
          cardTitle="使用中 GPU"
          cardDescription="正在使用的 GPU 资源"
          isLoading={jobQuery.isLoading}
        >
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
      />
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
};
