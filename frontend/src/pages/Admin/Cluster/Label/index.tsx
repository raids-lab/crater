import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import {
  LabelInfo,
  LabelType,
  apiNodeLabelsList,
  apiNodeLabelsNvidiaSync,
} from "@/services/api/nodelabel";
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
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { UpdateLabelForm } from "./Form";
import { Badge } from "@/components/ui/badge";
import { MyResponsivePieCanvas } from "./LabelPie";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { RefreshCcwIcon } from "lucide-react";

export const getTypeText = (type: LabelType): string => {
  switch (type) {
    case LabelType.NVIDIA:
      return "NVIDIA";
    default:
      return "Unknown";
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索标签名称",
    key: "label",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const columns: ColumnDef<LabelInfo>[] = [
  {
    accessorKey: "label",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"标签"} />
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"别名"} />
    ),
    cell: ({ row }) => {
      const text = row.getValue<string>("name");
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
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"类型"} />
    ),
    cell: ({ row }) => {
      return (
        <Badge
          className="font-mono"
          variant={
            row.getValue("type") === LabelType.NVIDIA
              ? "secondary"
              : "destructive"
          }
        >
          {getTypeText(row.getValue("type"))}
        </Badge>
      );
    },
  },
  {
    accessorKey: "count",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"节点数"} />
    ),
    cell: ({ row }) => <div className="font-mono">{row.getValue("count")}</div>,
  },
  {
    accessorKey: "priority",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"优先级"} />
    ),
    cell: ({ row }) => (
      <div className="font-mono">{row.getValue("priority")}</div>
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
                <DotsHorizontalIcon className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>操作</DropdownMenuLabel>
              <DialogTrigger asChild>
                <DropdownMenuItem>编辑标签</DropdownMenuItem>
              </DialogTrigger>
            </DropdownMenuContent>
          </DropdownMenu>
          <UpdateLabelForm
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
    queryKey: ["label", "list"],
    queryFn: apiNodeLabelsList,
    select: (res) =>
      res.data.data.sort((a, b) => b.label.localeCompare(a.label)),
  });

  const { mutate: syncNvidiaLabel } = useMutation({
    mutationFn: apiNodeLabelsNvidiaSync,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["label", "list"] });
      toast.success("标签已同步");
    },
  });

  return (
    <>
      <div className="grid gap-4 lg:col-span-3 lg:grid-cols-2 lg:gap-6">
        <Card className="col-span-1 flex flex-col justify-between">
          <CardHeader>
            <CardTitle>节点标签</CardTitle>
            <CardDescription className="leading-6">
              部署 Nvidia GPU Operator 后，
              <span className="rounded-md border px-1 py-0.5 font-mono">
                nvidia.com/gpu.product
              </span>{" "}
              标签将被添加到节点上， 用于标识节点的 GPU 类型。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button>
                  <RefreshCcwIcon className="h-4 w-4" />
                  同步 Nvidia 节点
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>同步标签</AlertDialogTitle>
                  <AlertDialogDescription>
                    确认要通过{" "}
                    <span className="rounded-md border px-1 py-0.5 font-mono">
                      nvidia.com/gpu.product
                    </span>{" "}
                    更新类型信息吗？
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
          </CardContent>
        </Card>
        <Card className="h-52 p-6">
          <MyResponsivePieCanvas />
        </Card>
      </div>
      <DataTable
        columns={columns}
        query={query}
        toolbarConfig={toolbarConfig}
      />
    </>
  );
};
