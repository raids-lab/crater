import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import NodeTypeBadge from "@/components/badge/NodeTypeBadge";
import { NodeType } from "@/services/api/cluster";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { NvidiaGpuInfoCard } from "./GPUInfo";
import NodeStatusBadge from "@/components/badge/NodeStatusBadge";
import TooltipLink from "../label/TooltipLink";

interface ResourceInfo {
  percent: number;
  description: string;
}

export interface ClusterNodeInfo {
  type: NodeType;
  name: string;
  isReady: string;
  taint: string;
  role: string;
  cpu?: ResourceInfo;
  memory?: ResourceInfo;
  gpu?: ResourceInfo;
  labels: Record<string, string>;
  podCount?: number;
}

export const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索节点名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

export const UsageCell: FC<{ value?: ResourceInfo }> = ({ value }) => {
  if (!value || isNaN(value.percent)) {
    return <></>;
  }

  return (
    <div>
      <p
        className={cn(
          "text-highlight-emerald mb-0.5 font-mono text-sm font-bold",
          {
            "text-highlight-yellow": value.percent > 25 && value.percent <= 50,
            "text-highlight-orange": value.percent > 50 && value.percent <= 75,
            "text-highlight-red": value.percent > 75,
          },
        )}
      >
        {value.percent.toFixed(1)}
        <span className="ml-0.5">%</span>
      </p>
      <p className="text-muted-foreground font-mono text-xs">
        {value.description}
      </p>
    </div>
  );
};

export const getNodeColumns = (
  getNicknameByName?: (name: string) => string | undefined,
): ColumnDef<ClusterNodeInfo>[] => {
  return [
    {
      accessorKey: "type",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"类型"} />
      ),
      cell: ({ row }) => (
        <NodeTypeBadge nodeType={row.getValue<NodeType>("type")} />
      ),
    },
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"名称"} />
      ),
      cell: ({ row }) => (
        <TooltipLink
          name={row.getValue("name")}
          to={row.getValue("name")}
          tooltip={`查看 ${row.original.name} 节点详情`}
        />
      ),
    },
    {
      accessorKey: "isReady",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"状态"} />
      ),
      cell: ({ row }) => {
        const taints = row.original.taint.split(",");
        const status = row.getValue<string>("isReady");

        // 如果状态为"occupied"，提取占用的账户名
        let accountInfo = null;
        if (status === "occupied" && getNicknameByName) {
          const occupiedAccount = taints
            .find((t) => t.startsWith("crater.raids.io/account"))
            ?.split("=")[1]
            ?.split(":")[0];

          if (occupiedAccount) {
            // 获取账户昵称
            const nickname = getNicknameByName(occupiedAccount);
            accountInfo = nickname
              ? `${nickname} (${occupiedAccount})`
              : occupiedAccount;
          }
        }

        // 过滤taints，如果状态是"occupied"
        const displayTaints =
          status === "occupied"
            ? taints.filter((taint) =>
                taint.includes("crater.raids.io/account"),
              )
            : taints;

        return (
          <div className="flex flex-row items-center justify-start gap-1">
            <NodeStatusBadge status={status} />

            {/* 如果有账户信息，显示一个单独的提示 */}
            {status === "occupied" && accountInfo && (
              <Tooltip>
                <TooltipTrigger className="flex size-4 items-center justify-center rounded-full bg-blue-500 text-xs text-white">
                  A
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs font-medium">占用账户: {accountInfo}</p>
                </TooltipContent>
              </Tooltip>
            )}

            {/* 原有的taints提示 */}
            {row.original.taint &&
              displayTaints.length > 0 &&
              status !== "occupied" && (
                <Tooltip>
                  <TooltipTrigger className="flex size-4 items-center justify-center rounded-full bg-slate-600 text-xs text-white">
                    {displayTaints.length}
                  </TooltipTrigger>
                  <TooltipContent className="font-mono">
                    {displayTaints.map((taint, index) => (
                      <p key={index} className="text-xs">
                        {taint}
                      </p>
                    ))}
                  </TooltipContent>
                </Tooltip>
              )}
          </div>
        );
      },
    },
    {
      accessorKey: "role",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"角色"} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center justify-start gap-1">
          <Badge
            variant={
              row.getValue("role") === "master" ? "default" : "secondary"
            }
            className="font-mono font-normal"
          >
            {row.getValue("role")}
          </Badge>
          <Tooltip>
            <TooltipTrigger
              className={cn(
                "bg-secondary text-secondary-foreground flex size-4 items-center justify-center rounded-full text-xs hover:cursor-help",
                {
                  "bg-primary text-primary-foreground":
                    row.original.podCount && row.original.podCount > 0,
                },
              )}
            >
              {row.original.podCount}
            </TooltipTrigger>
            <TooltipContent>
              {row.original.podCount} 个作业正在运行
            </TooltipContent>
          </Tooltip>
        </div>
      ),
    },
    {
      accessorKey: "cpu",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"CPU"} />
      ),
      cell: ({ row }) => {
        return <UsageCell value={row.original.cpu} />;
      },
      sortingFn: (rowA, rowB) => {
        const a = rowA.original.cpu?.percent ?? 0;
        const b = rowB.original.cpu?.percent ?? 0;
        return a - b;
      },
    },
    {
      accessorKey: "memory",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"Memory"} />
      ),
      cell: ({ row }) => (
        <UsageCell value={row.getValue<ResourceInfo | undefined>("memory")} />
      ),
      sortingFn: (rowA, rowB) => {
        const a = rowA.original.memory?.percent ?? 0;
        const b = rowB.original.memory?.percent ?? 0;
        return a - b;
      },
    },
    {
      accessorKey: "gpu",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"GPU"} />
      ),
      cell: ({ row }) => (
        <UsageCell value={row.getValue<ResourceInfo>("gpu")} />
      ),
      sortingFn: (rowA, rowB) => {
        const a = rowA.original.gpu?.percent ?? 0;
        const b = rowB.original.gpu?.percent ?? 0;
        return a - b;
      },
    },
    {
      accessorKey: "labels",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"GPU 标签"} />
      ),
      cell: ({ row }) => {
        const labels = row.original.labels;
        return (
          <div className="flex flex-col items-start justify-center gap-1 font-mono">
            {Object.keys(labels)
              .filter((k) => k.includes("nvidia.com/gpu.product"))
              .map((key) => (
                <TooltipProvider delayDuration={100} key={key}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Badge
                        className="cursor-help text-xs font-normal"
                        variant={"outline"}
                      >
                        {labels[key]}
                      </Badge>
                    </TooltipTrigger>
                    <TooltipContent className="p-0">
                      <NvidiaGpuInfoCard labels={labels} />
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ))}
          </div>
        );
      },
      enableSorting: false,
    },
  ] as ColumnDef<ClusterNodeInfo>[];
};
