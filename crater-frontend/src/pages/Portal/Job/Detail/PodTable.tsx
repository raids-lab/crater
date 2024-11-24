import { ColumnDef } from "@tanstack/react-table";
import { apiJobGetPods, PodDetail } from "@/services/api/vcjob";
import ResourceBadges from "@/components/label/ResourceBadges";
import PodPhaseLabel, { podPhases } from "@/components/label/PodPhaseLabel";
import TooltipButton from "@/components/custom/TooltipButton";
import { EthernetPort, LogsIcon, TerminalIcon } from "lucide-react";
import { DataTable } from "@/components/custom/DataTable";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import LogDialog from "@/components/codeblock/LogDialog";
import TerminalDialog from "@/components/codeblock/TerminalDialog";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NodeBadges from "@/components/label/NodeBadges";
import PodIngressDialog from "./Ingress";

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
          <a
            href={`http://10.109.80.1:31121/d/MhnFUFLSz/pod_memory?orgId=1&refresh=5s&var-node_name=${pod.nodename}&var-pod_name=${pod.name}&var-gpu=All&from=now-15m&to=now`}
            className="font-mono underline-offset-4 hover:underline"
            target="_blank"
            rel="noopener noreferrer"
          >
            {pod.name}
          </a>
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
          <div className="flex items-center justify-center space-x-1">
            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8 text-primary hover:bg-primary/10 hover:text-primary/90"
              tooltipContent="打开终端"
              disabled={pod.phase !== "Running"}
              onClick={() => {
                setShowTerminal({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
            >
              <TerminalIcon className="h-4 w-4 text-primary" />
            </TooltipButton>
            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8 text-purple-500 hover:bg-purple-500/10 hover:text-purple-500/90"
              tooltipContent="设置外部访问规则"
              disabled={pod.phase !== "Running"}
              onClick={() => {
                setShowIngress({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
            >
              <EthernetPort className="h-4 w-4" />
            </TooltipButton>
            <TooltipButton
              variant="outline"
              size="icon"
              className="h-8 w-8 text-orange-500 hover:bg-orange-500/10 hover:text-orange-500/90"
              tooltipContent="查看日志"
              onClick={() => {
                setShowLog({
                  namespace: pod.namespace,
                  name: pod.name,
                });
              }}
            >
              <LogsIcon className="h-4 w-4" />
            </TooltipButton>
          </div>
        );
      },
    },
  ];

  if (!query.data || query.data.length === 0) {
    return <></>;
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
