import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import {
  apiNodeLabelsDelete,
  apiNodeLabelsList,
} from "@/services/api/nodelabel";
import { useMutation, useQuery } from "@tanstack/react-query";
import { IResponse } from "@/services/types";
import { ColumnDef } from "@tanstack/react-table";
import { useMemo, type FC, useState } from "react";
import { Separator } from "@/components/ui/separator";
import { NewLabelForm, UpdateLabelForm } from "./Form.tsx";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

interface LabelInfo {
  ID: number;
  Name: string;
  Priority: number;
  Type: string;
}

export enum LabelType {
  GPU = 1,
}
// const handleUpdateLabel = (
//   id: number,
//   name: string,
//   priority: number,
//   type: string,
// ) => {
//   void apiNodeLabelsUpdate(id, name, priority, type).then((response) => {
//     logger.debug("update label response", response);
//   });
// };

export const getTypeText = (type: LabelType): string => {
  switch (type) {
    case LabelType.GPU:
      return "GPU";
    default:
      return "Unknown";
  }
};

const toolbarConfig: DataTableToolbarConfig<string> = {
  filterInput: {
    placeholder: "搜索标签名称",
    key: "Name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

export const Component: FC = () => {
  const queryClient = useQueryClient();
  const [openSheet, setOpenSheet] = useState(false);
  const query = useQuery({
    queryKey: ["label", "list"],
    queryFn: apiNodeLabelsList,
    select: (res: IResponse<LabelInfo[]>) => res.data,
  });

  const { mutate: handleDeleteLabel } = useMutation({
    mutationFn: (id: number) => apiNodeLabelsDelete(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["label", "list"] });
      toast.success("标签已删除");
    },
  });

  const data: LabelInfo[] = useMemo(() => {
    if (!query.data) {
      return [];
    }
    return query.data;
  }, [query.data]);

  const columns = useMemo<ColumnDef<LabelInfo>[]>(
    () => [
      {
        accessorKey: "Name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"名称"} />
        ),
        cell: ({ row }) => <div>{row.getValue("Name")}</div>,
      },
      {
        accessorKey: "Priority",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"优先级"} />
        ),
        cell: ({ row }) => <div>{row.getValue("Priority")}</div>,
      },
      {
        accessorKey: "Type",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"类型"} />
        ),
        cell: ({ row }) => <div>{getTypeText(row.getValue("Type"))}</div>,
      },
      // 添加删除键和更新键
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const labelId = row.original.ID;
          const labelName = row.getValue<string>("Name");
          const nowLabelPriority = row.getValue<number>("Priority");
          const nowLabelType = row.getValue<LabelType>("Type");
          // eslint-disable-next-line react-hooks/rules-of-hooks
          const [openSheet, setOpenSheet] = useState(false);
          return (
            <AlertDialog open={openSheet} onOpenChange={setOpenSheet}>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">操作</span>
                    <DotsHorizontalIcon className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>编辑标签</DropdownMenuItem>
                  </AlertDialogTrigger>
                  <DropdownMenuItem
                    onClick={() => {
                      void handleDeleteLabel(labelId);
                    }}
                  >
                    删除标签
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>编辑标签</AlertDialogTitle>
                  <UpdateLabelForm
                    labelId={labelId}
                    nowLabelName={labelName}
                    nowLabelPriority={nowLabelPriority}
                    nowLabelType={nowLabelType}
                    closeSheet={() => {
                      setOpenSheet(false);
                    }}
                  />
                </AlertDialogHeader>
              </AlertDialogContent>
            </AlertDialog>
          );
        },
      },
    ],
    [handleDeleteLabel],
  );

  return (
    <>
      {query.isLoading && <p>Loading...</p>}
      {!query.isLoading && data.length === 0 && <p>No data available</p>}
      {!query.isLoading && data.length > 0 && (
        <DataTable
          columns={columns}
          data={data}
          loading={query.isLoading}
          toolbarConfig={toolbarConfig}
        >
          <Sheet open={openSheet} onOpenChange={setOpenSheet}>
            <SheetTrigger asChild>
              <Button className="h-8 min-w-fit">新建标签</Button>
            </SheetTrigger>
            {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
            <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
              <SheetHeader>
                <SheetTitle>新建标签</SheetTitle>
                <SheetDescription>创建一个新的标签</SheetDescription>
              </SheetHeader>
              <Separator className="mt-4" />
              <NewLabelForm closeSheet={() => setOpenSheet(false)} />
            </SheetContent>
          </Sheet>
        </DataTable>
      )}
    </>
  );
};
