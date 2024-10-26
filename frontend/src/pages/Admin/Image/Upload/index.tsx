import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/OldDataTable";
import { Checkbox } from "@/components/ui/checkbox";
import {
  apiAdminImagePackDelete,
  apiAdminImagePackList,
  apiAdminImagePublicStatusChange,
  UpdateImagePublicStatusRequest,
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
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  ImageDeleteRequest,
  ImagePackInfo,
  ImagePackListType,
  getHeader,
  imagepackPublicPersonalStatus,
  imagepackSourceType,
  imagepackTaskType,
} from "@/services/api/imagepack";
import { Button } from "@/components/ui/button";
import { ToggleLeft, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
import { JobType } from "@/services/api/vcjob";

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
  const imagePackInfo = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiAdminImagePackList(ImagePackListType.Upload),
    select: (res) => res.data.data.imagepacklist,
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
      imagetype: item.imagetype,
      tasktype: item.tasktype,
      ispublic: item.ispublic,
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
    mutationFn: (req: ImageDeleteRequest) => apiAdminImagePackDelete(req),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });
  const { mutate: changeImagePublicStatus } = useMutation({
    mutationFn: (req: UpdateImagePublicStatusRequest) =>
      apiAdminImagePublicStatusChange(req),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像状态更新");
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
      cell: ({ row }) => <div>{row.getValue("nametag")}</div>,
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
      accessorKey: "imagetype",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("imagetype")} />
      ),
      cell: ({ row }) => {
        const imagetype = imagepackSourceType.find(
          (imagetype) => imagetype.value === row.getValue("imagetype"),
        );
        return imagetype?.label;
      },
    },
    {
      accessorKey: "username",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("username")} />
      ),
      cell: ({ row }) => <div>{row.getValue("username")}</div>,
    },
    {
      accessorKey: "tasktype",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("tasktype")} />
      ),
      cell: ({ row }) => {
        const tasktype = imagepackTaskType.find(
          (tasktype) => tasktype.value === row.getValue("tasktype"),
        );
        const type: JobType = tasktype?.label as JobType;
        return <JobTypeLabel jobType={type} />;
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
        const imagepackInfo = row.original;
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
                      deleteAdminImagePack({
                        id: imagepackInfo.id,
                        imagetype: imagepackInfo.imagetype,
                      });
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
                    镜像「{imagepackInfo?.nametag}
                    」状态将更新
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="default"
                    onClick={() => {
                      changeImagePublicStatus({
                        id: imagepackInfo.id,
                        imagetype: imagepackInfo.imagetype,
                      });
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
      loading={imagePackInfo.isLoading}
    ></DataTable>
  );
};
