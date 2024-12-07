import { ColumnDef } from "@tanstack/react-table";
import { apiJobGetPods, PodDetail } from "@/services/api/vcjob";
import ResourceBadges from "@/components/label/ResourceBadges";
import PodPhaseLabel, { podPhases } from "@/components/label/PodPhaseLabel";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";

const POD_MONITOR = import.meta.env.VITE_GRAFANA_POD_MEMORY;

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
            href={`${POD_MONITOR}?orgId=1&refresh=5s&var-node_name=${pod.nodename}&var-pod_name=${pod.name}&var-gpu=All&from=now-15m&to=now`}
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
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">更多操作</span>
                <DotsHorizontalIcon className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel className="text-xs text-muted-foreground">
                操作
              </DropdownMenuLabel>
              <DropdownMenuItem
                disabled={pod.phase !== "Running"}
                onClick={() => {
                  setShowTerminal({
                    namespace: pod.namespace,
                    name: pod.name,
                  });
                }}
              >
                <TerminalIcon className="text-primary" />
                Terminal
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={pod.phase !== "Running"}
                onClick={() => {
                  setShowIngress({
                    namespace: pod.namespace,
                    name: pod.name,
                  });
                }}
              >
                <EthernetPort className="text-purple-600 dark:text-purple-500" />
                外部访问
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => {
                  setShowLog({
                    namespace: pod.namespace,
                    name: pod.name,
                  });
                }}
              >
                <LogsIcon className="text-orange-600 dark:text-orange-500" />
                日志与诊断
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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
