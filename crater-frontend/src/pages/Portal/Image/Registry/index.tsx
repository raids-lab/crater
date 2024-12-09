import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
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
  BookOpenIcon,
  BoxIcon,
  InfoIcon,
  PackagePlusIcon,
  Trash2Icon,
  HardDriveIcon,
  KeyIcon,
  UserRoundIcon,
  ChartColumnIcon,
} from "lucide-react";
import { useNavigate, useRoutes } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import KanikoDetail from "../Info";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { DockerfileSheet } from "./DockerfileSheet";
import { shortestImageName } from "@/utils/formatter";

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
          {shortestImageName(row.getValue("imageLink"))}
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
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="text-xs text-muted-foreground">
                    操作
                  </DropdownMenuLabel>
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
    <div className="grid gap-x-4 gap-y-6 lg:grid-cols-2">
      <Card className="flex flex-col justify-between">
        <CardHeader>
          <CardTitle className="flex flex-row items-center justify-start gap-2">
            <BoxIcon className="text-primary" /> 镜像仓库
          </CardTitle>
          <CardDescription className="text-balance pt-2 leading-relaxed">
            通过 Kaniko 制作镜像，支持 Dockerfile 和低代码方式制作镜像
          </CardDescription>
        </CardHeader>
        <CardFooter className="flex flex-row gap-3">
          <Button className="h-8 min-w-fit" onClick={() => setOpenSheet(true)}>
            <PackagePlusIcon />
            镜像制作
          </Button>
          <Button
            variant="secondary"
            className="h-8 min-w-fit"
            onClick={() => toast.warning("TODO(huangsy): 补充镜像文档")}
          >
            <BookOpenIcon />
            查看文档
          </Button>
        </CardFooter>
      </Card>
      <Card className="flex flex-col justify-between">
        <CardHeader>
          <CardTitle className="flex flex-row items-center gap-2">
            <ChartColumnIcon className="text-primary" />
            使用情况
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="flex items-center">
              <BoxIcon className="mr-2 size-4 text-muted-foreground" />
              <h4 className="text-sm font-medium text-muted-foreground">
                镜像总数:
              </h4>
              <p className="ml-2 font-bold">3</p>
            </div>
            <div className="flex items-center">
              <HardDriveIcon className="mr-2 size-4 text-muted-foreground" />
              <h4 className="text-sm font-medium text-muted-foreground">
                存储用量:
              </h4>
              <p className="ml-2 font-bold">20GiB / 50GiB</p>
            </div>
            <div className="flex items-center">
              <UserRoundIcon className="mr-2 size-4 text-muted-foreground" />
              <h4 className="text-sm font-medium text-muted-foreground">
                仓库项目:
              </h4>
              <p className="ml-2 font-bold">user-admin</p>
            </div>
            <div className="flex items-center">
              <KeyIcon className="mr-2 size-4 text-muted-foreground" />
              <h4 className="text-sm font-medium text-muted-foreground">
                访问凭据:
              </h4>
              <Button
                variant="link"
                onClick={() =>
                  toast.warning("TODO(huangsy): 提供 Harbor Project 账户密码")
                }
              >
                获取初始凭据
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
      <DataTable
        query={imageQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        className="lg:col-span-2"
      />
      <DockerfileSheet
        isOpen={openSheet}
        onOpenChange={setOpenSheet}
        title="镜像制作 (当前不可用)"
        description="基于平台提供的基础镜像，快速制作自定义镜像"
        className="sm:max-w-3xl"
        closeSheet={() => setOpenSheet(false)}
      />
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
