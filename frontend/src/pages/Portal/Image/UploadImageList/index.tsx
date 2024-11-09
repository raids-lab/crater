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
import { ImageUploadForm } from "./UploadForm";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  getHeader,
  imagepackPublicPersonalStatus,
  apiUserListImage,
  ImageInfo,
  apiUserChangeImagePublicStatus,
  apiUserDeleteImage,
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
import { ToggleLeft, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "link",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const Component: FC = () => {
  const queryClient = useQueryClient();
  const [openSheet, setOpenSheet] = useState(false);
  const user = useAtomValue(globalUserInfo);
  const imageInfo = useQuery({
    queryKey: ["imagelink", "list"],
    queryFn: () => apiUserListImage(),
    select: (res) => res.data.data.imagelist,
  });
  const data: ImageInfo[] = useMemo(() => {
    if (!imageInfo.data) {
      return [];
    }
    return imageInfo.data.map((item) => ({
      id: item.ID,
      link: item.imagelink,
      status: item.status,
      createdAt: item.createdAt,
      tasktype: item.tasktype,
      ispublic: item.ispublic,
      creatorname: item.creatorname,
    }));
  }, [imageInfo.data]);
  const refetchImagePackList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["imagelink", "list"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };
  const { mutate: deleteUserImage } = useMutation({
    mutationFn: (id: number) => apiUserDeleteImage(id),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });
  const { mutate: changeImagePublicStatus } = useMutation({
    mutationFn: (id: number) => apiUserChangeImagePublicStatus(id),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像状态更新");
    },
  });
  const columns: ColumnDef<ImageInfo>[] = [
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
      accessorKey: "creatorname",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={getHeader("creatorname")}
        />
      ),
      cell: ({ row }) => <div>{row.getValue("creatorname")}</div>,
    },
    {
      accessorKey: "tasktype",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("tasktype")} />
      ),
      cell: ({ row }) => {
        return <JobTypeLabel jobType={row.getValue("tasktype")} />;
      },
    },
    {
      accessorKey: "ispublic",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("ispublic")} />
      ),
      cell: ({ row }) => {
        const imagePublicPersonalStatus = imagepackPublicPersonalStatus.find(
          (imagePublicPersonalStatus) =>
            imagePublicPersonalStatus.value === row.getValue("ispublic"),
        );
        return imagePublicPersonalStatus?.label;
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
        const imageInfo = row.original;
        return (
          <div className="flex flex-row space-x-1">
            <AlertDialog>
              <AlertDialogTrigger
                disabled={imageInfo.creatorname === user.name}
                asChild
              >
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-destructive"
                    title="删除镜像"
                    disabled={imageInfo.creatorname === user.name}
                  >
                    <Trash2 size={16} strokeWidth={2} />
                  </Button>
                </div>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除镜像</AlertDialogTitle>
                  <AlertDialogDescription>
                    镜像「{imageInfo?.link}
                    」将删除
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      deleteUserImage(imageInfo.id);
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
                    className="hover:text-default h-8 w-8 p-0"
                    title="更新公私状态"
                  >
                    <ToggleLeft size={16} strokeWidth={2} />
                  </Button>
                </div>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>更新公私状态</AlertDialogTitle>
                  <AlertDialogDescription>
                    镜像「{imageInfo?.link}
                    」状态将更新
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="default"
                    onClick={() => {
                      changeImagePublicStatus(imageInfo.id);
                    }}
                  >
                    确认
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
      loading={imageInfo.isLoading}
      className="col-span-3"
    >
      <Sheet open={openSheet} onOpenChange={setOpenSheet}>
        <SheetTrigger asChild>
          <Button className="h-8 min-w-fit">导入镜像</Button>
        </SheetTrigger>
        <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>导入镜像</SheetTitle>
            <SheetDescription>导入一个新的训练作业镜像</SheetDescription>
          </SheetHeader>
          <Separator className="mt-4" />
          <ImageUploadForm closeSheet={() => setOpenSheet(false)} />
        </SheetContent>
      </Sheet>
    </DataTable>
  );
};
