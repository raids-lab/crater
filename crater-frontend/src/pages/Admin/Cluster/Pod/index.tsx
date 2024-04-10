import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";
import { apiGetAdminNodeDetail } from "@/services/api/admin/cluster";
import { TableDate } from "@/components/custom/TableDate";
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
import { useSetRecoilState } from "recoil";
import { globalBreadCrumb } from "@/utils/store";

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
            <p className="text-xs text-gray-500">IP地址:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.address}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">节点角色:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.role}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">操作系统类型:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.os}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">操作系统版本:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.osVersion}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">系统架构:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.arch}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">Kubelet版本:</p>
            <p className="ml-2 text-sm font-medium text-black">
              {nodeInfo?.kubeletVersion}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs text-gray-500">容器运行时版本:</p>
            <p className="ml-2 text-sm font-medium text-black">
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

interface ClusterPodInfo {
  name: string;
  status: string;
  time: string;
  address: string;
}

const toolbarConfig: DataTableToolbarConfig<string> = {
  filterInput: {
    placeholder: "搜索容器组名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const columns: ColumnDef<ClusterPodInfo>[] = [
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
    cell: ({ row }) => <div className="font-mono">{row.getValue("name")}</div>,
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

export const PodStatusDetail: FC = () => {
  const { id: nodeName } = useParams();
  const { data: nodeInfo, isLoading } = useQuery({
    queryKey: ["admin", "nodes", nodeName],
    queryFn: () => apiGetAdminNodeDetail(`${nodeName}`),
    select: (res) => res.data.data,
    enabled: !!nodeName,
  });
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);

  // 修改 BreadCrumb
  useEffect(() => {
    if (isLoading) {
      return;
    }
    setBreadcrumb([
      {
        title: "集群管理",
      },
      {
        title: "计算节点",
        path: "/admin/cluster/node",
      },
      {
        title: `${nodeName}`,
      },
    ]);
  }, [setBreadcrumb, nodeName, isLoading]);

  // 处理 podsList 变量
  const podsInfo: ClusterPodInfo[] = useMemo(() => {
    if (isLoading || !nodeInfo) {
      return [];
    }
    return nodeInfo?.pods
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((x) => {
        return {
          name: x.name,
          status: x.status,
          time: x.createTime,
          address: x.IP,
        };
      });
  }, [isLoading, nodeInfo]);

  return (
    <div className="col-span-3 grid gap-8 md:grid-cols-4">
      <div className="flex-none">
        <CardDemo nodeInfo={nodeInfo} />
      </div>
      <div className="md:col-span-3">
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
