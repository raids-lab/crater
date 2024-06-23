import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";
import { apiGetAdminNodeDetail } from "@/services/api/admin/cluster";
import { apiGetAdminNodeGPU } from "@/services/api/admin/cluster";
import { TableDate } from "@/components/custom/TableDate";
import { useState } from "react";
import { cn } from "@/lib/utils";
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
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Link, useParams } from "react-router-dom";
import { Undo2 } from "lucide-react";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";

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

// CardDemo 组件
export function CardDemo({ className, nodeInfo, ...props }: CardDemoProps) {
  return (
    <Card className={className} {...props}>
      <CardHeader>
        <CardTitle>{nodeInfo?.name}</CardTitle>
        <CardDescription>节点详细信息</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4">
        <div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">IP地址:</p>
            <p className="ml-2 text-sm font-medium">{nodeInfo?.address}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">节点角色:</p>
            <p className="ml-2 text-sm font-medium">{nodeInfo?.role}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">操作系统类型:</p>
            <p className="ml-2 text-sm font-medium">{nodeInfo?.os}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">操作系统版本:</p>
            <p className="ml-2 text-sm font-medium">{nodeInfo?.osVersion}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">系统架构:</p>
            <p className="ml-2 text-sm font-medium">{nodeInfo?.arch}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">Kubelet版本:</p>
            <p className="ml-2 text-sm font-medium">
              {nodeInfo?.kubeletVersion}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">容器运行时版本:</p>
            <p className="ml-2 text-sm font-medium">
              {nodeInfo?.containerRuntimeVersion}
            </p>
          </div>
        </div>
      </CardContent>
      <CardFooter className="flex justify-center">
        <div className="w-full">
          <Link to="/admin/cluster/node" className="w-full">
            <Button className="w-full">
              <Undo2 className="mr-2 h-4 w-4" /> 返回集群节点列表
            </Button>
          </Link>
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
    queryFn: () => apiGetAdminNodeGPU(nodeName),
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
          const jobUrl = `http://192.168.5.60:31121/d/R4ZPFfyIz/job-monitor?orgId=1&var-job=${gpuInfo.relateJobs[parseInt(id)]}`;

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
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <>
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      </>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"容器组名称"} />
    ),
    cell: ({ row }) => {
      const podName = encodeURIComponent(row.getValue("name"));
      const link = `http://192.168.5.60:31121/d/MhnFUFLSz/pod_memory?orgId=1&var-node_name=${nodeName}&var-pod_name=${podName}`;
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
        <div
          className={cn("flex h-3 w-3 rounded-full", {
            "bg-red-500 hover:bg-red-400": row.getValue("status") !== "Running",
            "bg-emerald-500 hover:bg-emerald-400":
              row.getValue("status") === "Running",
          })}
        ></div>
        <div className="ml-1.5">
          {row.getValue("status") === "Running" ? "运行中" : "异常"}
        </div>
      </div>
    ),
  },
  {
    accessorKey: "time",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"创建时间"} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("time")}></TableDate>;
    },
    enableSorting: false,
  },
  {
    accessorKey: "address",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"容器组IP地址"} />
    ),
    cell: ({ row }) => (
      <div className="font-mono">{row.getValue("address")}</div>
    ),
    enableSorting: false,
  },
  {
    accessorKey: "cpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"CPU分配"} />
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

// export const PodStatusDetail: FC = () => {
//   const { id: nodeName } = useParams();
//   const { data: nodeInfo, isLoading } = useQuery({
//     queryKey: ["admin", "nodes", nodeName],
//     queryFn: () => apiGetAdminNodeDetail(`${nodeName}`),
//     select: (res) => res.data.data,
//     enabled: !!nodeName,
//   });
//   const safeNodeName = nodeName || "defaultNodeName";
//   const columns = useMemo(() => getColumns(safeNodeName), [safeNodeName]);

//   const setBreadcrumb = useBreadcrumb();

//   // 修改 BreadCrumb
//   useEffect(() => {
//     setBreadcrumb([{ title: nodeName ?? "" }]);
//   }, [setBreadcrumb, nodeName]);

//   // 处理 podsList 变量
//   const podsInfo: ClusterPodInfo[] = useMemo(() => {
//     if (isLoading || !nodeInfo) {
//       return [];
//     }
//     return nodeInfo?.pods
//       .sort((a, b) => a.name.localeCompare(b.name))
//       .map((x) => {
//         return {
//           name: x.name,
//           status: x.status,
//           time: x.createTime,
//           address: x.IP,
//           cpu: x.CPU,
//           memory: x.Mem,
//           isVcjob: x.isVcjob,
//         };
//       });
//   }, [isLoading, nodeInfo]);

//   return (
//     <div className="col-span-3 grid gap-8 md:grid-cols-4">
//       <div className="flex-none">
//         <CardDemo nodeInfo={nodeInfo} />
//         {nodeName && <GPUDetails nodeName={nodeName} />}
//       </div>
//       <div className="md:col-span-3">
//         <DataTable
//           data={podsInfo}
//           columns={columns}
//           toolbarConfig={toolbarConfig}
//           loading={isLoading}
//         />
//       </div>
//     </div>
//   );
// };

// export default PodStatusDetail;

export const PodStatusDetail: FC = () => {
  const { id: nodeName } = useParams();
  const [showAll, setShowAll] = useState(false); // 新增状态用于控制是否显示所有数据

  const { data: nodeInfo, isLoading } = useQuery({
    queryKey: ["admin", "nodes", nodeName],
    queryFn: () => apiGetAdminNodeDetail(`${nodeName}`),
    select: (res) => res.data.data,
    enabled: !!nodeName,
  });
  const safeNodeName = nodeName || "defaultNodeName";
  const columns = useMemo(() => getColumns(safeNodeName), [safeNodeName]);

  const setBreadcrumb = useBreadcrumb();

  // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: nodeName ?? "" }]);
  }, [setBreadcrumb, nodeName]);

  // 处理 podsList 变量
  const podsInfo: ClusterPodInfo[] = useMemo(() => {
    if (isLoading || !nodeInfo) {
      return [];
    }
    // 根据 showAll 状态和 isVcjob 字段筛选数据
    return nodeInfo?.pods
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
      });
  }, [isLoading, nodeInfo, showAll]); // 添加 showAll 作为依赖

  return (
    <div className="col-span-3 grid gap-8 md:grid-cols-4">
      <div className="flex-none">
        <CardDemo nodeInfo={nodeInfo} />
        {nodeName && <GPUDetails nodeName={nodeName} />}
      </div>
      <div className="md:col-span-3">
        <Button onClick={() => setShowAll(!showAll)} className="mb-4 w-full">
          <Undo2 className="mr-2 h-4 w-4" />{" "}
          {showAll ? "显示只包含 Vcjob 的" : "显示所有"}
        </Button>
        <DataTable
          data={podsInfo}
          columns={columns}
          toolbarConfig={toolbarConfig}
          loading={isLoading}
        />
      </div>
    </div>
  );
};

export default PodStatusDetail;
