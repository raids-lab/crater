import {
  apiAdminGetDataset,
  apiDatasetDelete,
  Dataset,
} from "@/services/api/dataset";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { ColumnDef } from "@tanstack/react-table";
import { TrashIcon } from "lucide-react";
import { format } from "date-fns";
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

  const { mutate: deleteDataset } = useMutation({
    mutationFn: (dataset: Dataset) => apiDatasetDelete(dataset.id),
    onSuccess: async (_, dataset) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "datasets"],
      });
      toast.success(`数据集 ${dataset.name} 已删除`);
    },
  });

  const columns: ColumnDef<Dataset>[] = [
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
      header: "创建者",
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
      cell: ({ row }) => (
        <Button
          variant="ghost"
          size="icon"
          onClick={() => deleteDataset(row.original)}
          title="删除数据集"
        >
          <TrashIcon className="h-4 w-4 text-red-500" />
        </Button>
      ),
    },
  ];

  return (
    <DataTable
      info={{
        title: "数据集管理",
        description: "管理系统中的所有数据集",
      }}
      storageKey="admin_dataset_management"
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
    />
  );
};
