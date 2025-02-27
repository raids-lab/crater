import { ColumnDef } from "@tanstack/react-table";
import { apiJobGetPods, PodDetail } from "@/services/api/vcjob";
import ResourceBadges from "@/components/badge/ResourceBadges";
import PodPhaseLabel, { podPhases } from "@/components/badge/PodPhaseBadge";
import { EthernetPort, GaugeIcon, LogsIcon, TerminalIcon } from "lucide-react";
import { DataTable } from "@/components/custom/DataTable";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import LogDialog from "@/components/codeblock/LogDialog";
import TerminalDialog from "@/components/codeblock/TerminalDialog";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NodeBadges from "@/components/badge/NodeBadges";
import PodIngressDialog from "./Ingress";
import TooltipButton from "@/components/custom/TooltipButton";
import TooltipLink from "@/components/label/TooltipLink";
import Nothing from "@/components/placeholder/Nothing";

const POD_MONITOR = import.meta.env.VITE_GRAFANA_POD_MONITOR;

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "Pod 名称";
    case "namespace":
      return "命名空间";
    case "nodename":
      return "节点";
    case "ip":
      return "内网 IP";
    case "port":
      return "端口";
    case "resource":
      return "申请资源";
    case "phase":
      return "状态";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索 Pod 名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "phase",
      title: "状态",
      option: podPhases,
    },
  ],
  getHeader: getHeader,
};

const getPodMonitorUrl = (pod: PodDetail) => {
  return `${POD_MONITOR}?orgId=1&refresh=5s&var-node_name=${pod.nodename}&var-pod_name=${pod.name}&var-gpu=All&from=now-15m&to=now`;
};

export const PodTable = ({ jobName }: { jobName: string }) => {
  const [showLog, setShowLog] = useState<NamespacedName>();
  const [showTerminal, setShowTerminal] = useState<NamespacedName>();
  const [showIngress, setShowIngress] = useState<NamespacedName>();
  const query = useQuery({
    queryKey: ["job", "detail", jobName, "pods"],
    queryFn: () => apiJobGetPods(jobName),
    select: (res) => res.data.data.sort((a, b) => a.name.localeCompare(b.name)),
    enabled: !!jobName,
  });

  const columns: ColumnDef<PodDetail>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("name")} />
      ),
      cell: ({ row }) => {
        const pod = row.original;
        return pod.name ? (
          <TooltipLink
            name={pod.name}
            to={getPodMonitorUrl(pod)}
            tooltip={`查看 Pod 监控`}
            className="font-mono"
          />
        ) : (
          "---"
        );
      },
    },
    {
      accessorKey: "phase",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("phase")} />
      ),
      cell: ({ row }) => <PodPhaseLabel podPhase={row.getValue("phase")} />,
      filterFn: (row, id, value) => {
        return (value as string[]).includes(row.getValue(id));
      },
    },
    {
      accessorKey: "ip",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("ip")} />
      ),
      cell: ({ row }) => <p className="font-mono">{row.getValue("ip")}</p>,
      enableSorting: false,
    },
    {
      accessorKey: "nodename",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("nodename")} />
      ),
      cell: ({ row }) => <NodeBadges nodes={[row.getValue("nodename")]} />,
    },
    {
      accessorKey: "resource",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("resource")} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row gap-1">
          <ResourceBadges resources={row.getValue("resource")} />
        </div>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const pod = row.original;
        return (
          <div className="flex items-center space-x-1.5">
            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={pod.phase !== "Running"}
              onClick={() => {
                setShowTerminal({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
              tooltipContent="打开网页终端"
            >
              <TerminalIcon className="size-4 text-primary" />
            </TooltipButton>

            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={pod.phase !== "Running"}
              onClick={() => {
                setShowIngress({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
              tooltipContent="配置外部访问"
            >
              <EthernetPort className="size-4 text-purple-600 dark:text-purple-500" />
            </TooltipButton>

            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={pod.phase !== "Running"}
              asChild
              tooltipContent="查看资源监控"
            >
              <a
                target="_blank"
                rel="noopener noreferrer"
                href={getPodMonitorUrl(pod)}
                className="size-9 p-0"
              >
                <GaugeIcon className="size-4 text-green-600 dark:text-green-500" />
              </a>
            </TooltipButton>

            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => {
                setShowLog({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
              tooltipContent="查看容器日志"
            >
              <LogsIcon className="size-4 text-orange-600 dark:text-orange-500" />
            </TooltipButton>
          </div>
        );
      },
    },
  ];

  if (query.isLoading) {
    return <></>;
  }

  if (!query.data || query.data.length === 0) {
    return <Nothing title="作业信息已被清理" />;
  }

  return (
    <>
      <DataTable
        toolbarConfig={toolbarConfig}
        query={query}
        columns={columns}
      />
      <LogDialog namespacedName={showLog} setNamespacedName={setShowLog} />
      <TerminalDialog
        namespacedName={showTerminal}
        setNamespacedName={setShowTerminal}
      />
      <PodIngressDialog
        namespacedName={showIngress}
        setNamespacedName={setShowIngress}
      />
    </>
  );
};
