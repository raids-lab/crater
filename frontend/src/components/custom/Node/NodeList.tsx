import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Link } from "react-router-dom";
import NodeTypeLabel from "@/components/label/NodeTypeLabel";
import { NodeType } from "@/services/api/cluster";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { NvidiaGpuInfoCard } from "./GPUInfo";

interface ResourceInfo {
  percent: number;
  description: string;
}

export interface ClusterNodeInfo {
  type: NodeType;
  name: string;
  isReady: string;
  role: string;
  cpu?: ResourceInfo;
  memory?: ResourceInfo;
  gpu?: ResourceInfo;
  labels: Record<string, string>;
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
        className={cn("mb-0.5 font-mono text-sm font-bold", {
          "text-green-600 dark:text-green-400": value.percent <= 20,
          "text-yellow-600 dark:text-yellow-400":
            value.percent > 20 && value.percent <= 50,
          "text-orange-600 dark:text-orange-400":
            value.percent > 50 && value.percent <= 80,
          "text-rose-600 dark:text-red-500": value.percent > 80,
        })}
      >
        {value.percent.toFixed(1)}
        <span className="ml-0.5">%</span>
      </p>
      <p className="font-mono text-xs text-muted-foreground">
        {value.description}
      </p>
    </div>
  );
};

export const nodeColumns: ColumnDef<ClusterNodeInfo>[] = [
  {
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"类型"} />
    ),
    cell: ({ row }) => (
      <NodeTypeLabel nodeType={row.getValue<NodeType>("type")} />
    ),
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"名称"} />
    ),
    cell: ({ row }) => (
      <Link
        className="h-8 px-0 text-left font-normal text-secondary-foreground underline-offset-2 hover:underline"
        to={`${row.original.name}`}
      >
        <p>{row.getValue("name")}</p>
      </Link>
    ),
  },
  {
    accessorKey: "isReady",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"状态"} />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center justify-start">
        <div
          className={cn("flex h-3 w-3 rounded-full", {
            "bg-red-500 hover:bg-red-400": !(
              row.getValue("isReady") === "true"
            ),
            "bg-emerald-500 hover:bg-emerald-400":
              row.getValue("isReady") === "true",
          })}
        />
        <div className="ml-1.5">
          {row.getValue("isReady") === "true" ? "运行中" : "异常"}
        </div>
      </div>
    ),
  },
  {
    accessorKey: "role",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"角色"} />
    ),
    cell: ({ row }) => (
      <Badge
        variant={row.getValue("role") === "master" ? "default" : "secondary"}
        className="font-mono font-normal"
      >
        {row.getValue("role")}
      </Badge>
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
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("gpu")} />,
    sortingFn: (rowA, rowB) => {
      const a = rowA.original.gpu?.percent ?? 0;
      const b = rowB.original.gpu?.percent ?? 0;
      return a - b;
    },
  },
  {
    accessorKey: "labels",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"标签"} />
    ),
    cell: ({ row }) => {
      const labels = row.original.labels;
      return (
        <div className="flex flex-col items-start justify-center gap-1 font-mono">
          {Object.keys(labels)
            .filter((k) => k.includes("nvidia.com/gpu.product"))
            .map((key) => (
              <TooltipProvider delayDuration={100}>
                <Tooltip>
                  <TooltipTrigger>
                    <Badge
                      key={key}
                      className="text-xs font-normal"
                      variant={"outline"}
                    >
                      {labels[key]}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent asChild>
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
];
