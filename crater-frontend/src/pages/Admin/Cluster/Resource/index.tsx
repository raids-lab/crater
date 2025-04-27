import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { type FC, useState } from "react";
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
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { UpdateResourceForm } from "./Form";
import { Badge } from "@/components/ui/badge";
import { RefreshCcwIcon } from "lucide-react";
import {
  Resource,
  apiAdminResourceSync,
  apiResourceList,
  apiAdminResourceDelete,
  apiResourceNetworks,
} from "@/services/api/resource";
import { formatBytes } from "@/utils/formatter";
import { NetworkAssociationForm, UpdateResourceTypeForm } from "./Form";

// New component for Networks cell
const NetworksCell: FC<{ resourceId: number; resourceType?: string }> = ({
  resourceId,
  resourceType,
}) => {
  const networksQuery = useQuery({
    queryKey: ["resource", "networks", resourceId],
    queryFn: () => apiResourceNetworks(resourceId),
    enabled: resourceType === "gpu",
    select: (res) => res.data.data,
  });

  if (resourceType !== "gpu" || !networksQuery.data?.length) {
    return <></>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {networksQuery.data.map((network) => (
        <Badge key={network.ID} variant="outline" className="font-mono">
          {network.name}
        </Badge>
      ))}
    </div>
  );
};

// New component for Actions cell
const ActionsCell: FC<{ resource: Resource }> = ({ resource }) => {
  const [openLabelSheet, setOpenLabelSheet] = useState(false);
  const [openTypeSheet, setOpenTypeSheet] = useState(false);
  const [openNetworkSheet, setOpenNetworkSheet] = useState(false);
  const queryClient = useQueryClient();

  const { mutate: deleteResource } = useMutation({
    mutationFn: () => apiAdminResourceDelete(resource.ID),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["resource", "list"],
      });
      toast.success("资源已删除");
    },
  });

  return (
    <>
      <AlertDialog>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">操作</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-muted-foreground text-xs">
              操作
            </DropdownMenuLabel>
            <DropdownMenuItem onClick={() => setOpenLabelSheet(true)}>
              编辑资源信息
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setOpenTypeSheet(true)}>
              设置资源类型
            </DropdownMenuItem>
            {resource.type === "gpu" && (
              <DropdownMenuItem onClick={() => setOpenNetworkSheet(true)}>
                管理网络关联
              </DropdownMenuItem>
            )}
            <AlertDialogTrigger asChild>
              <DropdownMenuItem className="text-destructive">
                删除资源
              </DropdownMenuItem>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>删除资源</AlertDialogTitle>
                <AlertDialogDescription>
                  确认删除资源 "{resource.name}" 吗？此操作不可撤销。
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => deleteResource()}
                  className="bg-destructive text-destructive-foreground"
                >
                  确认删除
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </DropdownMenuContent>
        </DropdownMenu>
      </AlertDialog>
      <UpdateResourceForm
        current={resource}
        open={openLabelSheet}
        onOpenChange={setOpenLabelSheet}
      />
      <UpdateResourceTypeForm
        current={resource}
        open={openTypeSheet}
        onOpenChange={setOpenTypeSheet}
      />
      {resource.type === "gpu" && (
        <NetworkAssociationForm
          gpuResource={resource}
          open={openNetworkSheet}
          onOpenChange={setOpenNetworkSheet}
        />
      )}
    </>
  );
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索资源名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const columns: ColumnDef<Resource>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"资源"} />
    ),
    cell: ({ row }) => {
      return (
        <Badge className="font-mono font-normal" variant="secondary">
          {row.getValue<string>("name")}
        </Badge>
      );
    },
  },
  {
    accessorKey: "label",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"别名"} />
    ),
    cell: ({ row }) => {
      const text = row.getValue<string>("label");
      if (text === "") {
        return <></>;
      }
      return (
        <Badge className="font-mono font-normal" variant="secondary">
          {text}
        </Badge>
      );
    },
  },
  {
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"类型"} />
    ),
    cell: ({ row }) => {
      const type = row.getValue<string>("type");
      if (!type) {
        return <></>;
      }
      return (
        <Badge variant="secondary">
          {type === "gpu" ? "GPU" : type === "rdma" ? "RDMA" : type}
        </Badge>
      );
    },
  },
  {
    id: "networks",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"关联网络"} />
    ),
    cell: ({ row }) => (
      <NetworksCell
        resourceId={row.original.ID}
        resourceType={row.original.type}
      />
    ),
  },
  {
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"总量"} />
    ),
    cell: ({ row }) => {
      const amount = row.getValue<number>("amount");
      if (amount > 1024 * 1024) {
        return <div className="font-mono">{formatBytes(amount)}</div>;
      }
      return <div className="font-mono">{row.getValue("amount")}</div>;
    },
  },
  {
    accessorKey: "amountSingleMax",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"单节点最大数量"} />
    ),
    cell: ({ row }) => {
      const amount = row.getValue<number>("amountSingleMax");
      if (amount > 1024 * 1024) {
        return <div className="font-mono">{formatBytes(amount)}</div>;
      }
      return <div className="font-mono">{row.getValue("amountSingleMax")}</div>;
    },
  },
  {
    accessorKey: "format",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Format"} />
    ),
    cell: ({ row }) => (
      <Badge className="font-mono font-normal" variant="outline">
        {row.getValue("format")}
      </Badge>
    ),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => <ActionsCell resource={row.original} />,
  },
];

export const Component: FC = () => {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["resource", "list"],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data.data
        .sort((a, b) => a.name.localeCompare(b.name))
        .filter((x) => !x.name.startsWith("hugepages"));
    },
  });

  const { mutate: syncNvidiaLabel } = useMutation({
    mutationFn: apiAdminResourceSync,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["resource", "list"] });
      toast.success("资源列表已同步");
    },
  });

  return (
    <DataTable
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
      info={{
        title: "资源列表",
        description: "资源列表用于定义集群中的资源，如 GPU、CPU、内存等",
      }}
      storageKey="admin_resource_list"
    >
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button>
            <RefreshCcwIcon />
            同步资源列表
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>同步资源列表</AlertDialogTitle>
            <AlertDialogDescription>
              确认更新资源列表信息吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => syncNvidiaLabel()}>
              确认
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DataTable>
  );
};

export default Component;
