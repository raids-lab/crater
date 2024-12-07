import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
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
import {
  Database,
  InfoIcon,
  LockIcon,
  PackagePlusIcon,
  Trash2Icon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useNavigate, useRoutes } from "react-router-dom";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import KanikoDetail from "../Info";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";

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
  const navigate = useNavigate();
  const imageQuery = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiUserListKaniko(),
    select: (res) =>
      res.data.data.kanikoList.map(
        (item) =>
          ({
            id: item.ID,
            imageLink: item.imageLink,
            status: item.status,
            createdAt: item.createdAt,
            podName: item.podName,
            podNameSpace: item.podNameSpace,
          }) as KanikoInfo,
      ),
  });
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
        <Button
          onClick={() => navigate(`${row.original.id}`)}
          variant={"link"}
          className="h-8 px-0 text-left font-normal text-secondary-foreground"
        >
          {row.getValue("imageLink")}
        </Button>
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
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">操作</span>
                    <DotsHorizontalIcon className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>操作</DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => navigate(`${kanikoInfo.id}`)}
                  >
                    <InfoIcon className="text-emerald-600 dark:text-emerald-500" />
                    详情
                  </DropdownMenuItem>

                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>
                      <Trash2Icon className="text-destructive" />
                      删除
                    </DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
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
    <div className="grid flex-1 grid-cols-[1fr,300px] gap-6">
      <main>
        <DataTable
          query={imageQuery}
          columns={columns}
          toolbarConfig={toolbarConfig}
        >
          <Sheet open={openSheet} onOpenChange={setOpenSheet}>
            <SheetTrigger asChild>
              <Button className="h-8 min-w-fit">
                <PackagePlusIcon />
                制作镜像
              </Button>
            </SheetTrigger>
            <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
              <SheetHeader>
                <SheetTitle>制作镜像</SheetTitle>
              </SheetHeader>
              <ImageCreateForm closeSheet={() => setOpenSheet(false)} />
            </SheetContent>
          </Sheet>
        </DataTable>
      </main>
      <aside className="space-y-6">
        <Card>
          <CardContent className="p-6">
            <div className="mb-6 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <LockIcon className="h-4 w-4 text-muted-foreground" />
                <h2 className="font-semibold">Access Method</h2>
              </div>
            </div>
          </CardContent>
          <CardFooter className="gap-2">
            <Badge variant="outline" className="bg-transparent">
              Repo
            </Badge>
            <Badge variant="outline" className="bg-transparent">
              Project
            </Badge>
            <Badge variant="outline" className="bg-transparent">
              Password
            </Badge>
          </CardFooter>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="mb-6 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Database className="h-4 w-4 text-muted-foreground" />
                <h2 className="font-semibold">Quota used</h2>
              </div>
            </div>
            <div className="text-2xl font-bold">
              0Byte
              <span className="ml-2 text-sm text-muted-foreground">
                of 20GiB
              </span>
            </div>
          </CardContent>
        </Card>
      </aside>
    </div>
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
      element: <KanikoDetail />,
    },
  ]);

  return <>{routes}</>;
};
