import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Checkbox } from "@/components/ui/checkbox";
import { apiDlDatasetList } from "@/services/api/recommend/dataset";
import { useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { useMemo, type FC } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { useNavigate } from "react-router-dom";

type DatasetInfo = {
  id: string;
  name: string;
  createdAt: string;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "数据集名称";
    case "createdAt":
      return "创建时间";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索数据集名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const Component: FC = () => {
  const navigate = useNavigate();
  const datasetInfo = useQuery({
    queryKey: ["recommend", "dataset", "list"],
    queryFn: () => apiDlDatasetList(),
    select: (res) => res.data.data,
  });

  const columns = useMemo<ColumnDef<DatasetInfo>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(value) => row.toggleSelected(!!value)}
              aria-label="Select row"
              className="ml-2"
            />
          </>
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("createdAt")}
          />
        ),
        cell: ({ row }) => {
          // row format: "2023-10-30T03:21:03.733Z"
          const date = new Date(row.getValue("createdAt"));
          const formatted = date.toLocaleString("zh-CN", {
            timeZone: "Asia/Shanghai",
          });
          return <div>{formatted}</div>;
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const info = row.original;
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">操作</span>
                  <DotsHorizontalIcon className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>操作</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => navigate(`${info.name}`)}>
                  详情
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [navigate],
  );

  const data: DatasetInfo[] = useMemo(() => {
    if (!datasetInfo.data) {
      return [];
    }
    return datasetInfo.data.map((item) => ({
      id: item.name,
      name: item.name,
      createdAt: item.creationTimestamp,
    }));
  }, [datasetInfo.data]);

  return (
    <div className="space-y-1 text-xl">
      <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig} />
    </div>
  );
};
