import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import { apiGetNodeDetail, apiGetNodePods } from "@/services/api/cluster";
import { apiGetNodeGPU } from "@/services/api/cluster";
import { TableDate } from "@/components/custom/TableDate";
import { useState } from "react";
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
} from "@/components/ui/alert-dialog";
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
import { Undo2 } from "lucide-react";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import PodPhaseLabel from "@/components/custom/PodPhaseLabel";

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
      <CardHeader className="mb-6 border-b bg-muted/50 dark:bg-muted/25">
        <CardTitle>{nodeInfo?.name}</CardTitle>
        <CardDescription>节点属性</CardDescription>
      </CardHeader>
      <CardContent className="grid grid-flow-col grid-rows-7 gap-x-2 gap-y-3 text-xs">
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
    refetchInterval: 1000,
  });

  if (isLoading)
    return (
      <Card>
        <CardContent>Loading GPU details...</CardContent>
      </Card>
    );
  if (error)
    return (
      <Card>
        <CardContent>Error fetching GPU details.</CardContent>
      </Card>
    );
  if (!gpuInfo || !gpuInfo.haveGPU) return null; // 如果没有 GPU 数据或节点不包含 GPU，不显示组件
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

interface ClusterPodInfo {
  name: string;
  status: string;
  time: string;
  address: string;
}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索容器组名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const getColumns = (nodeName: string): ColumnDef<ClusterPodInfo>[] => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"容器组名称"} />
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
  },
  {
    accessorKey: "time",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"创建于"} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("time")}></TableDate>;
    },
    enableSorting: false,
  },
  {
    accessorKey: "cpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"CPU 分配"} />
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
      <DataTableColumnHeader column={column} title={"内存分配"} />
    ),
    cell: ({ row }) => (
      <div className="font-mono">{row.getValue("memory")}</div>
    ),
  },
  {
    accessorKey: "gpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"GPU 分配"} />
    ),
    cell: () => <div className="font-mono">TODO</div>,
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
                <DotsHorizontalIcon className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>操作</DropdownMenuLabel>
              <AlertDialogTrigger asChild>
                <DropdownMenuItem>编辑标签</DropdownMenuItem>
              </AlertDialogTrigger>
            </DropdownMenuContent>
          </DropdownMenu>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>编辑标签</AlertDialogTitle>
              <AlertDialogDescription>
                [WIP] 为节点 {taskInfo.name} 编辑标签
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction>提交</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      );
    },
  },
];

export const NodeDetail: FC = () => {
  const { id: nodeName } = useParams();
  const [showAll, setShowAll] = useState(false); // 新增状态用于控制是否显示所有数据

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
      res.data.data?.pods
        .filter((pod) => showAll || pod.isVcjob === "true")
        .sort((a, b) => a.name.localeCompare(b.name))
        .map((x) => {
          return {
            name: x.name,
            status: x.status,
            time: x.createTime,
            address: x.IP,
            cpu: x.CPU,
            memory: x.Mem,
            isVcjob: x.isVcjob,
          };
        }),
    enabled: !!nodeName,
  });
  const safeNodeName = nodeName || "defaultNodeName";
  const columns = useMemo(() => getColumns(safeNodeName), [safeNodeName]);

  const setBreadcrumb = useBreadcrumb();

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
        <Button onClick={() => setShowAll(!showAll)} className="mb-4 w-full">
          <Undo2 className="mr-2 h-4 w-4" />{" "}
          {showAll ? "显示只包含 Vcjob 的" : "显示所有"}
        </Button>
        <DataTable
          query={podsQuery}
          columns={columns}
          toolbarConfig={toolbarConfig}
        />
      </div>
    </div>
  );
};

export default NodeDetail;
