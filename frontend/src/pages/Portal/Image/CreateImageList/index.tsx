import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
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
import { ImageCreateForm } from "./CreateForm";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  ImagePackInfo,
  apiUserImagePackDelete,
  apiUserImagePackList,
  getHeader,
  imagepackStatuses,
  ImageDeleteRequest,
  imagepackTaskType,
  ImagePackListType,
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
import { useNavigate, useRoutes } from "react-router-dom";
import ImageCreateDetail from "../Info";
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

export const ImageTable: FC = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [openSheet, setOpenSheet] = useState(false);
  const imagePackInfo = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiUserImagePackList(ImagePackListType.Create),
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
  const { mutate: deleteUserImagePack } = useMutation({
    mutationFn: (req: ImageDeleteRequest) => apiUserImagePackDelete(req),
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
        <Button
          onClick={() => navigate(`${row.original.id}`)}
          variant={"link"}
          className="h-8 px-0 text-left font-normal text-secondary-foreground"
        >
          {row.getValue("nametag")}
        </Button>
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
                      deleteUserImagePack({
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
    {
      path: ":id",
      element: <ImageCreateDetail />,
    },
  ]);

  return <>{routes}</>;
};
