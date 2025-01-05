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
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { RefreshCcwIcon } from "lucide-react";
import {
  Resource,
  apiAdminResourceSync,
  apiResourceList,
} from "@/services/api/resource";
import { formatBytes } from "@/utils/formatter";

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
        <Badge className="font-mono font-normal" variant="outline">
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
        <Badge className="font-mono font-normal" variant="outline">
          {text}
        </Badge>
      );
    },
  },
  {
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"总量"} />
    ),
    cell: ({ row }) => {
      const amount = row.getValue<number>("amount");
      if (amount > 1024 * 1024) {
        return <div>{formatBytes(amount)}</div>;
      }
      return <div>{row.getValue("amount")}</div>;
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
        return <div>{formatBytes(amount)}</div>;
      }
      return <div>{row.getValue("amountSingleMax")}</div>;
    },
  },
  {
    accessorKey: "format",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Format"} />
    ),
    cell: ({ row }) => (
      <div className="font-mono">{row.getValue("format")}</div>
    ),
  },
  // 添加删除键和更新键
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      // eslint-disable-next-line react-hooks/rules-of-hooks
      const [openSheet, setOpenSheet] = useState(false);
      return (
        <Dialog open={openSheet} onOpenChange={setOpenSheet}>
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
              <DialogTrigger asChild>
                <DropdownMenuItem>编辑资源列表</DropdownMenuItem>
              </DialogTrigger>
            </DropdownMenuContent>
          </DropdownMenu>
          <UpdateResourceForm
            current={row.original}
            closeSheet={() => {
              setOpenSheet(false);
            }}
          />
        </Dialog>
      );
    },
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
