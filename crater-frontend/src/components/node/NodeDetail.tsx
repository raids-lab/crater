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
import { TimeDistance } from "@/components/custom/TimeDistance";
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
} from "@/components/ui/card";
import { useParams } from "react-router-dom";
import { ExternalLink } from "lucide-react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import PodPhaseLabel, { podPhases } from "@/components/badge/PodPhaseBadge";
import { Separator } from "@/components/ui/separator";
import { Badge } from "../ui/badge";
import {
  NamespacedName,
  PodNamespacedName,
} from "../codeblock/PodContainerDialog";
import LogDialog from "../codeblock/LogDialog";
import ResourceBadges from "../badge/ResourceBadges";

import yitianURL from "/nodes/yitian.png";
import hygonURL from "/nodes/hygon.png";
import shenweiURL from "/nodes/sw.png";
import TooltipLink from "../label/TooltipLink";

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

const POD_MONITOR = import.meta.env.VITE_GRAFANA_POD_MEMORY;
const GRAFANA_NODE = import.meta.env.VITE_GRAFANA_NODE;

// CardDemo 组件
export function CardDemo({ className, nodeInfo, ...props }: CardDemoProps) {
  const imageURL = useMemo(() => {
    const name = nodeInfo?.name;
    if (!name) return null;
    // if contains ali
    if (name.includes("ali")) {
      return yitianURL;
    } else if (name.includes("hygon")) {
      return hygonURL;
    } else if (name.includes("sw")) {
      return shenweiURL;
    } else {
      return null;
    }
  }, [nodeInfo?.name]);

  return (
    <Card className={className} {...props}>
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
              if (nodeInfo?.name == "ali-2") {
                window.open(
                  `https://ecs.console.aliyun.com/server/i-2zefsgw9xfdks1gjro5l/monitor?regionId=cn-beijing`,
                );
              } else if (nodeInfo?.name == "ali-3") {
                window.open(
                  `https://ecs.console.aliyun.com/server/i-2ze5e0maafu6vn80d2ek/monitor?regionId=cn-beijing`,
                );
              } else {
                window.open(
                  `${GRAFANA_NODE}?orgId=1&var-node=${nodeInfo?.name}&from=now-30m&to=now`,
                );
              }
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
        {imageURL && (
          <img src={imageURL} alt="节点提供方 LOGO" className="h-16 w-auto" />
        )}
      </CardFooter>
    </Card>
  );
}

const getHeader = (name: string): string => {
  switch (name) {
    case "type":
      return "类型";
    case "name":
      return "Pod 名称";
    case "status":
      return "状态";
    case "createTime":
      return "创建于";
    case "resources":
      return "申请资源";
    default:
      return name;
  }
};

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
      key: "type",
      title: "类型",
      option: [
        {
          value: "batch.volcano.sh/v1alpha1/Job",
          label: "BASE",
        },
        {
          value: "aisystem.github.com/v1alpha1/AIJob",
          label: "EMIAS",
        },
      ],
      defaultValues: ["batch.volcano.sh/v1alpha1/Job"],
    },
  ],
  getHeader: getHeader,
};

const getColumns = (
  nodeName: string,
  handleShowPodLog: (namespacedName: PodNamespacedName) => void,
): ColumnDef<IClusterPodInfo>[] => [
  {
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("type")} />
    ),
    cell: ({ row }) => {
      if (!row.getValue("type")) return null;
      const splitValue = row.getValue<string>("type").split("/");
      const apiVersion = splitValue.slice(0, splitValue.length - 1).join("/");
      const kind = splitValue[splitValue.length - 1];
      return (
        <Badge
          variant="outline"
          className="cursor-help font-mono font-normal"
          title={apiVersion}
        >
          {kind}
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
      <DataTableColumnHeader column={column} title={getHeader("name")} />
    ),
    cell: ({ row }) => {
      const podName = row.getValue<string>("name");
      const link = `${POD_MONITOR}?orgId=1&var-node_name=${nodeName}&var-pod_name=${podName}&from=now-1h&to=now`;
      return (
        <TooltipLink
          name={podName}
          to={link}
          tooltip={`查看 Pod 监控`}
          className="font-mono"
        />
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("status")} />
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
    accessorKey: "resources",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("resources")} />
    ),
    cell: ({ row }) => {
      return <ResourceBadges resources={row.getValue("resources")} />;
    },
  },
  {
    accessorKey: "createTime",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("createTime")} />
    ),
    cell: ({ row }) => {
      return <TimeDistance date={row.getValue("createTime")}></TimeDistance>;
    },
    enableSorting: false,
  },
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
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              操作
            </DropdownMenuLabel>
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
  const [showLogPod, setShowLogPod] = useState<NamespacedName>();

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
      res.data.data
        ?.sort((a, b) => a.name.localeCompare(b.name))
        .map((p) => {
          if (p.ownerReference && p.ownerReference.length > 0) {
            p.type = `${p.ownerReference[0].apiVersion}/${p.ownerReference[0].kind}`;
          }
          return p;
        }),
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
