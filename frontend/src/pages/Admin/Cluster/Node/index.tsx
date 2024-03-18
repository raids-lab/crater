import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";
import { apiGetAdminNodes } from "@/services/api/admin/cluster";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { getAiResource } from "@/utils/resource";
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

interface ResourceInfo {
  percent: number;
  description: string;
}

interface ClusterNodeInfo {
  name: string;
  isReady: boolean;
  role: string;
  cpu: ResourceInfo;
  memory: ResourceInfo;
  gpu: ResourceInfo;
  labels: Record<string, string>;
}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索节点名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const UsageCell: FC<{ value: ResourceInfo }> = ({ value }) => {
  if (value.description === "") {
    return <div />;
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

const columns: ColumnDef<ClusterNodeInfo>[] = [
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
      <DataTableColumnHeader column={column} title={"节点名称"} />
    ),
    cell: ({ row }) => <div className="font-mono">{row.getValue("name")}</div>,
    enableSorting: false,
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
            "bg-red-500 hover:bg-red-400": !row.getValue("isReady"),
            "bg-emerald-500 hover:bg-emerald-400":
              row.getValue("isReady") === true,
          })}
        ></div>
        <div className="ml-1.5">
          {row.getValue("isReady") === true ? "运行中" : "异常"}
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
      <DataTableColumnHeader column={column} title={"CPUs"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("cpu")} />,
    sortingFn: (rowA, rowB) => {
      const a = rowA.original.cpu.percent;
      const b = rowB.original.cpu.percent;
      return a - b;
    },
  },
  {
    accessorKey: "memory",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Memory"} />
    ),
    cell: ({ row }) => (
      <UsageCell value={row.getValue<ResourceInfo>("memory")} />
    ),
    sortingFn: (rowA, rowB) => {
      const a = rowA.original.memory.percent;
      const b = rowB.original.memory.percent;
      return a - b;
    },
  },
  {
    accessorKey: "gpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"GPUs"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("gpu")} />,
    sortingFn: (rowA, rowB) => {
      if (!rowA.original.gpu.description && rowB.original.gpu.description) {
        return 1;
      } else if (
        rowA.original.gpu.description &&
        !rowB.original.gpu.description
      ) {
        return -1;
      }

      const a = rowA.original.gpu.percent;
      const b = rowB.original.gpu.percent;
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
        <div className="flex gap-1 font-mono">
          {Object.keys(labels)
            .filter((k) => k.includes("gpu"))
            .map((key) => (
              <Badge
                key={key}
                className="text-xs font-normal"
                variant={"outline"}
              >
                {key}:{labels[key]}
              </Badge>
            ))}
        </div>
      );
    },
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

export const Component: FC = () => {
  const query = useQuery({
    queryKey: ["admin", "nodes"],
    queryFn: () => apiGetAdminNodes(),
    select: (res) => res.data.data.rows,
  });

  const data: ClusterNodeInfo[] = useMemo(() => {
    if (!query.data) {
      return [];
    }
    // sort by name
    return query.data
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((x) => {
        const capacity = getAiResource(x.capacity);
        const allocated = getAiResource(x.allocated);
        return {
          name: x.name,
          isReady: x.isReady,
          role: x.role,
          cpu: {
            percent: (allocated.cpu ?? 0) / (capacity.cpu ?? 1) / 10,
            description: `${((allocated.cpu ?? 0) / 1000).toFixed(1)}/${capacity.cpu ?? 1}`,
          },
          memory: {
            percent:
              ((allocated.memoryNum ?? 0) / (capacity.memoryNum ?? 1)) * 100,
            description: `${(allocated.memoryNum ?? 0).toFixed(1)}/${(capacity.memoryNum ?? 1).toFixed(0)}Gi`,
          },
          gpu: {
            percent: capacity.gpu
              ? ((allocated.gpu ?? 0) / (capacity.gpu ?? 1)) * 100
              : 0,
            description: capacity.gpu
              ? `${(allocated.gpu ?? 0).toFixed(0)}/${capacity.gpu ?? 1}`
              : "",
          },
          labels: x.labels,
        };
      });
  }, [query.data]);

  return (
    <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig} />
  );
};
