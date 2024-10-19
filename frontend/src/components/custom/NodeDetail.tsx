import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import {
  apiGetNodeDetail,
  apiGetNodePods,
  IClusterPodInfo,
} from "@/services/api/cluster";
import { apiGetNodeGPU } from "@/services/api/cluster";
import { TableDate } from "@/components/custom/TableDate";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import {
  Card,
  CardTitle,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { useNavigate, useParams } from "react-router-dom";
import { ExternalLink, Undo2 } from "lucide-react";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import PodPhaseLabel, { podPhases } from "@/components/label/PodPhaseLabel";
import { Separator } from "@/components/ui/separator";
import { Badge } from "../ui/badge";
import { PodNamespacedName } from "../codeblock/PodContainerDialog";
import LogDialog from "../codeblock/LogDialog";

type CardDemoProps = React.ComponentProps<typeof Card> & {
  nodeInfo?: {
    name: string;
    address: string;
    role: string;
    os: string;
    osVersion: string;
    arch: string;
    kubeletVersion: string;
    containerRuntimeVersion: string;
  };
};

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
const pod_memory = import.meta.env.VITE_GRAFANA_POD_MEMORY;

// CardDemo 组件
export function CardDemo({ className, nodeInfo, ...props }: CardDemoProps) {
  const navigate = useNavigate();
  return (
    <Card className={className} {...props}>
      {/* <CardHeader className="mb-6 border-b bg-muted/50 dark:bg-muted/25">
        <CardTitle>{nodeInfo?.name}</CardTitle>
        <CardDescription>节点属性</CardDescription>
      </CardHeader> */}
      <CardContent className="flex items-center justify-between bg-muted/50 p-6">
        <div className="flex flex-col items-start gap-2">
          <CardTitle>{nodeInfo?.name}</CardTitle>
          <CardDescription>节点属性</CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            onClick={() => {
              window.open(
                `http://8.141.83.224:31120/d/Oxed_c6Wz1/node_monitor?orgId=1&var-node=${nodeInfo?.name}&from=now-30m&to=now`,
              );
            }}
            disabled={nodeInfo?.name == "zjlab-sw"}
          >
            <ExternalLink className="h-3.5 w-3.5" />
          </Button>
        </div>
      </CardContent>
      <Separator />
      <CardContent className="mt-6 grid grid-flow-col grid-rows-7 gap-x-2 gap-y-3 text-xs">
        <p className="text-muted-foreground">IP 地址</p>
        <p className="text-muted-foreground">角色</p>
        <p className="text-muted-foreground">操作系统类型</p>
        <p className="text-muted-foreground">操作系统版本</p>
        <p className="text-muted-foreground">系统架构</p>
        <p className="text-muted-foreground">Kubelet 版本</p>
        <p className="text-muted-foreground">容器运行时</p>
        <p className="font-mono font-medium">{nodeInfo?.address}</p>
        <p className="font-medium capitalize">{nodeInfo?.role}</p>
        <p className="font-medium capitalize">{nodeInfo?.os}</p>
        <p className="font-medium">{nodeInfo?.osVersion}</p>
        <p className="font-medium uppercase">{nodeInfo?.arch}</p>
        <p className="font-mono font-medium">{nodeInfo?.kubeletVersion}</p>
        <p className="font-mono font-medium">
          {nodeInfo?.containerRuntimeVersion}
        </p>
      </CardContent>
      <CardFooter className="flex justify-center">
        <div className="w-full">
          <Button className="w-full" onClick={() => navigate(-1)}>
            <Undo2 className="mr-2 h-4 w-4" /> 返回集群节点列表
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}

export function GPUDetails({ nodeName }: { nodeName: string }) {
  // const nodeName = nodeInfo.name; // Now properly typed as string

  const {
    data: gpuInfo,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["admin", "nodes", nodeName, "gpu"],
    queryFn: () => apiGetNodeGPU(nodeName),
    select: (res) => res.data.data,
    enabled: !!nodeName,
    refetchInterval: 5000,
  });

  if (isLoading || error || !gpuInfo || !gpuInfo.haveGPU) return <></>; // 如果没有 GPU 数据或节点不包含 GPU，不显示组件
  return (
    <Card className="mt-4">
      <CardHeader>
        <CardTitle>节点GPU情况</CardTitle>
        <CardDescription>显示节点的 GPU 数量和利用率</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4">
        <div className="mb-2 flex justify-between">
          <p className="text-xs">GPU 数量:</p>
          <p className="text-sm font-medium">{gpuInfo.gpuCount}</p>
        </div>
        {Object.entries(gpuInfo.gpuUtil).map(([id, util]) => {
          const jobUrl = `${job_monitor}?orgId=1&var-job=${gpuInfo.relateJobs[parseInt(id)]}`;

          return (
            <>
              <div className="mb-2 flex justify-between" key={id}>
                <p className="text-xs">GPU {id} 利用率:</p>
                <p
                  className={`text-sm font-medium ${util > 0 ? "text-green-500" : ""}`}
                >
                  {util}%
                </p>
              </div>
              <div className="mb-2 flex justify-between" key={id}>
                <p className="text-xs">关联Job:</p>
                <p className="text-sm font-medium">
                  <a
                    href={jobUrl}
                    className="hover:underline"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {gpuInfo.relateJobs[parseInt(id)]}
                  </a>
                </p>
              </div>
            </>
          );
        })}
      </CardContent>
    </Card>
  );
}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索 Pod 名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "status",
      title: "状态",
      option: podPhases,
      defaultValues: ["Running"],
    },
    {
      key: "ownerKind",
      title: "类型",
      option: [
        {
          value: "Job",
          label: "BASE",
        },
        {
          value: "AIJob",
          label: "EMIAS",
        },
      ],
      defaultValues: ["Job"],
    },
  ],
  getHeader: (x) => x,
};

const getColumns = (
  nodeName: string,
  handleShowPodLog: (namespacedName: PodNamespacedName) => void,
): ColumnDef<IClusterPodInfo>[] => [
  {
    accessorKey: "ownerKind",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"从属"} />
    ),
    cell: ({ row }) => {
      return (
        <Badge variant="outline" className="font-mono font-normal">
          {row.getValue("ownerKind")}
        </Badge>
      );
    },
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id));
    },
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Pod 名称"} />
    ),
    cell: ({ row }) => {
      const podName = encodeURIComponent(row.getValue("name"));
      const link = `${pod_memory}?orgId=1&var-node_name=${nodeName}&var-pod_name=${podName}&from=now-1h&to=now`;
      return (
        <div className="font-mono">
          <a
            href={link}
            className="hover:underline"
            target="_blank"
            rel="noopener noreferrer"
          >
            {podName}
          </a>
        </div>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"状态"} />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center justify-start">
        <PodPhaseLabel podPhase={row.getValue("status")} />
      </div>
    ),
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id));
    },
  },
  {
    accessorKey: "createTime",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"创建于"} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("createTime")}></TableDate>;
    },
    enableSorting: false,
  },
  {
    accessorKey: "cpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"CPU"} />
    ),
    cell: ({ row }) => {
      const cpuValue = row.getValue("cpu");
      return (
        <div className="font-mono">
          {typeof cpuValue === "number" ? cpuValue.toFixed(3) : "N/A"}
        </div>
      );
    },
  },
  {
    accessorKey: "memory",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Memory"} />
    ),
    cell: ({ row }) => (
      <div className="font-mono">{row.getValue("memory")}</div>
    ),
  },
  // {
  //   accessorKey: "gpu",
  //   header: ({ column }) => (
  //     <DataTableColumnHeader column={column} title={"GPU 分配"} />
  //   ),
  //   cell: () => <div className="font-mono">TODO</div>,
  // },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const taskInfo = row.original;
      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">操作</span>
              <DotsHorizontalIcon className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>操作</DropdownMenuLabel>
            <DropdownMenuItem>监控</DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                handleShowPodLog({
                  namespace: taskInfo.namespace,
                  name: taskInfo.name,
                })
              }
            >
              日志
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
  },
];

export const NodeDetail: FC = () => {
  const { id: nodeName } = useParams();
  const setBreadcrumb = useBreadcrumb();
  const [showLogPod, setShowLogPod] = useState<PodNamespacedName | undefined>();

  const { data: nodeDetail } = useQuery({
    queryKey: ["nodes", nodeName, "detail"],
    queryFn: () => apiGetNodeDetail(`${nodeName}`),
    select: (res) => res.data.data,
    enabled: !!nodeName,
  });

  const podsQuery = useQuery({
    queryKey: ["nodes", nodeName, "pods"],
    queryFn: () => apiGetNodePods(`${nodeName}`),
    select: (res) =>
      res.data.data?.pods.sort((a, b) => a.name.localeCompare(b.name)),
    enabled: !!nodeName,
  });

  const columns = useMemo(
    () => getColumns(nodeName || "defaultNodeName", setShowLogPod),
    [nodeName],
  );

  // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: nodeName ?? "" }]);
  }, [setBreadcrumb, nodeName]);

  return (
    <div className="col-span-3 grid gap-8 md:grid-cols-7">
      <div className="col-span-2 flex-none">
        <CardDemo nodeInfo={nodeDetail} />
        {nodeName && <GPUDetails nodeName={nodeName} />}
      </div>
      <div className="md:col-span-5">
        <DataTable
          query={podsQuery}
          columns={columns}
          toolbarConfig={toolbarConfig}
        />
      </div>
      <LogDialog
        namespacedName={showLogPod}
        setNamespacedName={setShowLogPod}
      />
    </div>
  );
};

export default NodeDetail;
