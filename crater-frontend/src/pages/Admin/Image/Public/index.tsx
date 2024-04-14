import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { NewTaskForm } from "./Form";
import { TableDate } from "@/components/custom/TableDate";
import {
  ImagePackInfo,
  getHeader,
  imagepackStatuses,
} from "@/services/api/imagepack";
import {
  apiAdminImagePackDelete,
  apiAdminImagePackList,
} from "@/services/api/admin/imagepack";
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
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";
import { Trash2, UserRoundMinus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "link",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const Component: FC = () => {
  const [openSheet, setOpenSheet] = useState(false);
  const queryClient = useQueryClient();
  const imagePackInfo = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiAdminImagePackList(1),
    select: (res) => res.data.data,
  });
  const data: ImagePackInfo[] = useMemo(() => {
    if (!imagePackInfo.data) {
      return [];
    }
    return imagePackInfo.data.map((item) => ({
      id: item.ID,
      link: item.imagelink,
      username: item.creatername,
      status: item.status,
      createdAt: item.createdAt,
      nametag: item.nametag,
      params: item.params,
    }));
  }, [imagePackInfo.data]);

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
  const { mutate: deleteAdminImagePack } = useMutation({
    mutationFn: (id: number) => apiAdminImagePackDelete(id),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });

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
      cell: ({ row }) => (
        //     params: {
        //       Convs: 0,
        //       Activations: 0,
        //       Denses: 0,
        //       Others: 0,
        //       GFLOPs: 0,
        //       BatchSize: 0,
        //       Params: 0,
        //       ModelSize: 0,
        //     },
        <TooltipProvider delayDuration={0}>
          <Tooltip>
            <TooltipTrigger>{row.getValue("nametag")}</TooltipTrigger>
            <TooltipContent className="grid grid-cols-2 gap-1 p-4 font-mono">
              <div className="col-span-2 pb-2 text-sm font-bold">
                Profile 信息
              </div>
              <div>Convs: </div>
              <div>{row.original.params.Convs}</div>
              <div>Activations: </div>
              <div>{row.original.params.Activations}</div>
              <div>Denses: </div>
              <div>{row.original.params.Denses}</div>
              <div>Others: </div>
              <div>{row.original.params.Others}</div>
              <div>GFLOPs: </div>
              <div>{row.original.params.GFLOPs.toFixed(2)}</div>
              <div>BatchSize: </div>
              <div>{row.original.params.BatchSize}</div>
              <div>Params: </div>
              <div>{row.original.params.Params}</div>
              <div>ModelSize: </div>
              <div>{row.original.params.ModelSize.toFixed(2)}</div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ),
    },
    {
      accessorKey: "link",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("link")} />
      ),
      cell: ({ row }) => (
        <Badge className="font-mono font-normal" variant="outline">
          {row.getValue("link")}
        </Badge>
      ),
    },
    {
      accessorKey: "username",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("username")} />
      ),
      cell: ({ row }) => <div>{row.getValue("username")}</div>,
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
        return <TableDate date={row.getValue("createdAt")}></TableDate>;
      },
      sortingFn: "datetime",
    },
    {
      id: "actions",
      enableHiding: false,
      cell: ({ row }) => {
        const imagepackInfo = row.original;
        return (
          <div className="flex flex-row space-x-1">
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-red-700"
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
                    镜像「{imagepackInfo?.nametag}
                    」将删除
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      // check if browser support clipboard
                      deleteAdminImagePack(imagepackInfo.id);
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-blue-700"
                    title="profile数据"
                  >
                    <UserRoundMinus size={16} strokeWidth={2} />
                  </Button>
                </div>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Profile参数</AlertDialogTitle>
                  <AlertDialogDescription>
                    <div className="grid grid-cols-2 gap-1 p-4 font-mono text-lg">
                      <div>Convs: {imagepackInfo?.params.Convs}</div>
                      <div>
                        Activations: {imagepackInfo?.params.Activations}
                      </div>
                      <div>Denses: {imagepackInfo?.params.Denses}</div>
                      <div>Others: {imagepackInfo?.params.Others}</div>
                      <div>
                        GFLOPs: {imagepackInfo?.params.GFLOPs.toFixed(2)}
                      </div>
                      <div>BatchSize: {imagepackInfo?.params.BatchSize}</div>
                      <div>Params: {imagepackInfo?.params.Params}</div>
                      <div>
                        ModelSize: {imagepackInfo?.params.ModelSize.toFixed(2)}
                      </div>
                    </div>
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>关闭</AlertDialogCancel>
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
  );
};
