import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
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
import { ImageCreateForm } from "./CreateForm";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  apiUserDeleteKaniko,
  apiUserListKaniko,
  getHeader,
  imagepackStatuses,
  KanikoInfo,
} from "@/services/api/imagepack";
import { logger } from "@/utils/loglevel";
import { toast } from "sonner";
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
} from "@/components/ui-custom/alert-dialog";
import { Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useRoutes } from "react-router-dom";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "link",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const ImageTable: FC = () => {
  const queryClient = useQueryClient();
  const [openSheet, setOpenSheet] = useState(false);
  const kanikoInfo = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiUserListKaniko(),
    select: (res) => res.data.data.kanikoList,
  });
  const data: KanikoInfo[] = useMemo(() => {
    if (!kanikoInfo.data) {
      return [];
    }
    return kanikoInfo.data.map((item) => ({
      id: item.ID,
      imageLink: item.imageLink,
      status: item.status,
      createdAt: item.createdAt,
    }));
  }, [kanikoInfo.data]);
  const refetchImagePackList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["imagepack", "list"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };
  const { mutate: userDeleteKaniko } = useMutation({
    mutationFn: (id: number) => apiUserDeleteKaniko(id),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });

  const columns: ColumnDef<KanikoInfo>[] = [
    {
      accessorKey: "imageLink",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("imageLink")} />
      ),
      cell: ({ row }) => (
        <Badge className="font-mono font-normal" variant="outline">
          {row.getValue("imageLink")}
        </Badge>
      ),
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("status")} />
      ),
      cell: ({ row }) => {
        const status = imagepackStatuses.find(
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
        return <TimeDistance date={row.getValue("createdAt")}></TimeDistance>;
      },
      sortingFn: "datetime",
    },
    {
      id: "actions",
      enableHiding: false,
      cell: ({ row }) => {
        const kanikoInfo = row.original;
        return (
          <div className="flex flex-row space-x-1">
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-destructive"
                    title="删除镜像"
                  >
                    <Trash2 size={16} strokeWidth={2} />
                  </Button>
                </div>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除镜像</AlertDialogTitle>
                  <AlertDialogDescription>
                    镜像「{kanikoInfo?.imageLink}
                    」将删除
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      userDeleteKaniko(kanikoInfo.id);
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
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      loading={kanikoInfo.isLoading}
      className="col-span-3"
    >
      <Sheet open={openSheet} onOpenChange={setOpenSheet}>
        <SheetTrigger asChild>
          <Button className="h-8 min-w-fit">创建镜像</Button>
        </SheetTrigger>
        <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>创建镜像</SheetTitle>
            <SheetDescription>创建一个新的训练作业镜像</SheetDescription>
          </SheetHeader>
          <Separator className="mt-4" />
          <ImageCreateForm closeSheet={() => setOpenSheet(false)} />
        </SheetContent>
      </Sheet>
    </DataTable>
  );
};

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <ImageTable />,
    },
    // {
    //   path: ":id",
    //   element: <ImageCreateDetail />,
    // },
  ]);

  return <>{routes}</>;
};
