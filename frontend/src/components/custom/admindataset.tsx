import {
  apiAdminGetDataset,
  apiDatasetDelete,
  IDataset,
} from "@/services/api/dataset";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { ColumnDef } from "@tanstack/react-table";
import { Trash2Icon, InfoIcon } from "lucide-react";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { useNavigate } from "react-router-dom";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { format } from "date-fns";
import UserLabel from "@/components/label/UserLabel";
const roles = [
  {
    label: "数据集",
    value: "dataset",
  },
  {
    label: "模型",
    value: "model",
  },
];
export const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    key: "name",
    placeholder: "搜索数据集名称",
  },
  filterOptions: [
    {
      key: "type",
      title: "类型",
      option: roles,
    },
  ],
  getHeader: (key: string): string => {
    switch (key) {
      case "name":
        return "名称";
      case "type":
        return "类型";
      case "username":
        return "创建者";
      case "createdAt":
        return "创建时间";
      default:
        return key;
    }
  },
};
export const AdminDatasetTable = () => {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["admin", "datasets"],
    queryFn: () => apiAdminGetDataset(),
    select: (res) => res.data.data,
  });
  const navigate = useNavigate();
  const { mutate: deleteDataset } = useMutation({
    mutationFn: (dataset: IDataset) => apiDatasetDelete(dataset.id),
    onSuccess: async (_, dataset) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "datasets"],
      });
      toast.success(`数据集 ${dataset.name} 已删除`);
    },
  });

  const columns: ColumnDef<IDataset>[] = [
    {
      accessorKey: "name",
      header: "名称",
    },
    {
      accessorKey: "type",
      header: "类型",
    },
    {
      accessorKey: "username",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={"创建者"} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
    },
    {
      accessorKey: "createdAt",
      header: "创建时间",
      cell: ({ row }) => (
        <div>
          {row.original.createdAt
            ? format(new Date(row.original.createdAt), "yyyy-MM-dd HH:mm:ss")
            : "未知"}
        </div>
      ),
    },
    {
      id: "actions",
      header: "操作",
      cell: ({ row }) => {
        const DatasetInfo = row.original;
        return (
          <div className="flex flex-row space-x-1">
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
                  <DropdownMenuItem
                    onClick={() => navigate(`${DatasetInfo.id}`)}
                  >
                    <InfoIcon className="text-highlight-emerald" />
                    详情
                  </DropdownMenuItem>

                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>
                      <Trash2Icon className="text-destructive" />
                      删除
                    </DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除数据集或模型</AlertDialogTitle>
                  <AlertDialogDescription>
                    数据集{DatasetInfo.name}将被删除
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => {
                      deleteDataset(row.original);
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        );
      },
    },
  ];

  return (
    <DataTable
      info={{
        title: "数据集管理",
        description: "管理系统中的所有数据集/模型",
      }}
      storageKey="admin_dataset_management"
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
    />
  );
};
