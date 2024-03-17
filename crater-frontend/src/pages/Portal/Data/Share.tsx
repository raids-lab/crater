import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { apiAiTaskShareDirList } from "@/services/api/aiTask";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";

type ShareDirInfo = {
  id: number;
  name: string;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "共享空间名称";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索共享空间名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: getHeader,
};

const columns: ColumnDef<ShareDirInfo>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="ml-2"
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
];

export const Component: FC = () => {
  const shareDirsInfo = useQuery({
    queryKey: ["aitask", "shareDirs"],
    queryFn: () => apiAiTaskShareDirList(),
    select: (res) => res.data.data,
  });

  const data: ShareDirInfo[] = useMemo(() => {
    if (!shareDirsInfo.data) {
      return [];
    }
    return shareDirsInfo.data.map((item, index) => ({
      id: index,
      name: item,
    }));
  }, [shareDirsInfo.data]);

  return (
    <div className="space-y-1 text-xl">
      <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig} />
    </div>
  );
};
