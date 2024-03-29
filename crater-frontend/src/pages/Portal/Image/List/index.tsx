import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";
import { NewTaskForm } from "./Form";
import { TableDate } from "@/components/custom/TableDate";
import { apiUserImagePackList } from "@/services/api/imagepack";

type ImagePackInfo = {
  id: number;
  nametag: string;
  link: string;
  status: string;
  createdAt: string;
};

type ImagePackStatusValue =
  | "Initial"
  | "Pending"
  | "Running"
  | "Finished"
  | "Failed";

const imagepack_statuses: {
  value: ImagePackStatusValue;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  {
    value: "Initial",
    label: "检查中",
    icon: CircleIcon,
  },
  {
    value: "Pending",
    label: "等待中",
    icon: ClockIcon,
  },
  {
    value: "Running",
    label: "运行中",
    icon: StopwatchIcon,
  },
  {
    value: "Finished",
    label: "成功",
    icon: CheckCircledIcon,
  },
  {
    value: "Failed",
    label: "失败",
    icon: CrossCircledIcon,
  },
];

const getHeader = (key: string): string => {
  switch (key) {
    case "nametag":
      return "名称";
    case "link":
      return "链接";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "link",
  },
  filterOptions: [],
  getHeader: getHeader,
};

const columns: ColumnDef<ImagePackInfo>[] = [
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
    accessorKey: "nametag",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("nametag")} />
    ),
    cell: ({ row }) => <div>{row.getValue("nametag")}</div>,
  },
  {
    accessorKey: "link",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("link")} />
    ),
    cell: ({ row }) => <div>{row.getValue("link")}</div>,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("status")} />
    ),
    cell: ({ row }) => {
      const status = imagepack_statuses.find(
        (status) => status.value === row.getValue("status"),
      );
      if (!status) {
        return null;
      }
      return (
        <div className="flex flex-row items-center justify-start">
          <div
            className={cn("flex h-3 w-3 rounded-full", {
              "bg-purple-500 hover:bg-purple-400": status.value === "Initial",
              "bg-slate-500 hover:bg-slate-400": status.value === "Pending",
              "bg-sky-500 hover:bg-sky-400": status.value === "Running",
              "bg-red-500 hover:bg-red-400": status.value === "Failed",
              "bg-emerald-500 hover:bg-emerald-400":
                status.value === "Finished",
            })}
          ></div>
          <div className="ml-1.5">{status.label}</div>
        </div>
      );
    },
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id));
    },
  },
  {
    accessorKey: "createdAt",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("createdAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("createdAt")}></TableDate>;
    },
    sortingFn: "datetime",
  },
];

export const Component: FC = () => {
  const [openSheet, setOpenSheet] = useState(false);
  const imagePackInfo = useQuery({
    queryKey: ["imagelink", "status"],
    queryFn: () => apiUserImagePackList(),
    select: (res) => res.data.data,
  });
  const data: ImagePackInfo[] = useMemo(() => {
    if (!imagePackInfo.data) {
      return [];
    }
    return imagePackInfo.data.map((item) => ({
      id: item.ID,
      link: item.imagelink,
      status: item.status,
      createdAt: item.createdAt,
      nametag: item.nametag,
    }));
  }, [imagePackInfo.data]);

  return (
    <div className="space-y-1 text-xl">
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
        loading={imagePackInfo.isLoading}
      >
        <Sheet open={openSheet} onOpenChange={setOpenSheet}>
          <SheetTrigger asChild>
            <Button className="h-8 min-w-fit">创建镜像</Button>
          </SheetTrigger>
          {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
          <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
            <SheetHeader>
              <SheetTitle>创建镜像</SheetTitle>
              <SheetDescription>创建一个新的训练任务镜像</SheetDescription>
            </SheetHeader>
            <Separator className="mt-4" />
            <NewTaskForm closeSheet={() => setOpenSheet(false)} />
          </SheetContent>
        </Sheet>
      </DataTable>
    </div>
  );
};
