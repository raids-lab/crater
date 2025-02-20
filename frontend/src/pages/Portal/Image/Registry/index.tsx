import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  apiUserDeleteKaniko,
  apiUserGetCredential,
  apiUserGetQuota,
  apiUserListKaniko,
  getHeader,
  ImagePackStatus,
  KanikoInfoResponse,
  ProjectCredentialResponse,
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
  BoxIcon,
  InfoIcon,
  PackagePlusIcon,
  Trash2Icon,
  HardDriveIcon,
  KeyIcon,
  UserRoundIcon,
  Key,
  Copy,
  User,
} from "lucide-react";
import { useNavigate, useRoutes } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import KanikoDetail from "../Info";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@radix-ui/react-label";
import PageTitle from "@/components/layout/PageTitle";
import DocsButton from "@/components/button/DocsButton";
import { PipAptSheet } from "./PipAptSheet";
import { DockerfileSheet } from "./DockerfileSheet";
import SplitButton from "@/components/button/SplitButton";
import TooltipLink from "@/components/label/TooltipLink";
import ImageLabel from "@/components/label/ImageLabel";
import ImagePhaseBadge from "@/components/badge/ImagePhaseBadge";
import { formatBytes } from "@/utils/formatter";
import LoadingCircleIcon from "@/components/icon/LoadingCircleIcon";

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
  const [openPipAptSheet, setOpenPipAptSheet] = useState(false);
  const [openDockerfileSheet, setOpenDockerfileSheet] = useState(false);
  const navigate = useNavigate();

  const imageQuery = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiUserListKaniko(),
    select: (res) => res.data.data.kanikoList,
  });

  const quotaQuery = useQuery({
    queryKey: ["imagepack", "quota"],
    queryFn: () => apiUserGetQuota(),
    select: (res) => res.data.data,
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
  const [credentials, setCredentials] =
    useState<ProjectCredentialResponse | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const { mutate: getProjectCredential } = useMutation({
    mutationFn: () => apiUserGetCredential(),
    onSuccess: async (data) => {
      setCredentials(data.data.data);
      setIsDialogOpen(true);
      toast.success("凭据已生成, 请保存您的密码！");
    },
  });
  const columns: ColumnDef<KanikoInfoResponse>[] = [
    {
      accessorKey: "imageLink",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("imageLink")} />
      ),
      cell: ({ row }) => (
        <TooltipLink
          name={
            <ImageLabel
              description={row.original.description}
              url={row.getValue<string>("imageLink")}
            />
          }
          to={`${row.original.ID}`}
          tooltip={`查看镜像详情`}
        />
      ),
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
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("status")} />
      ),
      cell: ({ row }) => {
        return (
          <ImagePhaseBadge status={row.getValue<ImagePackStatus>("status")} />
        );
      },
      filterFn: (row, id, value) => {
        return (value as string[]).includes(row.getValue(id));
      },
    },
    {
      accessorKey: "size",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("size")} />
      ),
      cell: ({ row }) => {
        return <span>{formatBytes(row.getValue("size"))}</span>;
      },
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
                    onClick={() => navigate(`${kanikoInfo.ID}`)}
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
                      userDeleteKaniko(kanikoInfo.ID);
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
    <>
      <div>
        <PageTitle
          title="镜像制作"
          description="支持 Dockerfile 、低代码、快照等方式制作镜像"
          className="mb-4"
        >
          <div className="flex flex-row gap-3">
            <DocsButton title="查看文档" url="image/imagebuild" />
            <SplitButton
              icon={<PackagePlusIcon />}
              renderTitle={(title) => `基于${title}构建`}
              itemTitle="构建方式"
              items={[
                {
                  key: "pip-apt",
                  title: "软件包",
                  action: () => {
                    setOpenPipAptSheet(true);
                  },
                },
                {
                  key: "dockerfile",
                  title: " Dockerfile ",
                  action: () => {
                    setOpenDockerfileSheet(true);
                  },
                },
              ]}
              cacheKey="imagepack"
            />
          </div>
        </PageTitle>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                <BoxIcon className="size-5 text-primary" />
                镜像总数
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">
                {quotaQuery.isLoading ? (
                  <LoadingCircleIcon />
                ) : (
                  <>
                    {
                      imageQuery.data?.filter((c) => c.status == "Finished")
                        .length
                    }
                  </>
                )}
              </p>
            </CardContent>
          </Card>

          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                <HardDriveIcon className="size-5 text-primary" />
                存储用量
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">
                {quotaQuery.isLoading ? (
                  <LoadingCircleIcon />
                ) : (
                  <>
                    {Number(quotaQuery.data?.used).toFixed(2)}GiB/
                    {Number(quotaQuery.data?.quota).toFixed(0)}GiB
                  </>
                )}
              </p>
            </CardContent>
          </Card>

          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                <UserRoundIcon className="size-5 text-primary" />
                仓库项目
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">
                {quotaQuery.isLoading ? (
                  <LoadingCircleIcon />
                ) : (
                  <>{quotaQuery.data?.project}</>
                )}
              </p>
            </CardContent>
          </Card>

          <Card className="shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                <KeyIcon className="size-5 text-primary" />
                访问凭据
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Button
                variant="outline"
                onClick={() => getProjectCredential()}
                className="w-full"
              >
                获取初始凭据
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
      <DataTable
        query={imageQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        className="lg:col-span-2"
      />
      <PipAptSheet
        isOpen={openPipAptSheet}
        onOpenChange={setOpenPipAptSheet}
        title="基于软件包构建镜像"
        description="基于平台提供的基础镜像，快速制作自定义镜像"
        className="sm:max-w-3xl"
        closeSheet={() => setOpenPipAptSheet(false)}
      />
      <DockerfileSheet
        isOpen={openDockerfileSheet}
        onOpenChange={setOpenDockerfileSheet}
        title="基于 Dockerfile 构建镜像"
        description="基于 Dockerfile 制作镜像"
        className="sm:max-w-3xl"
        closeSheet={() => setOpenDockerfileSheet(false)}
      />
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="h-6 w-6 text-blue-500" />
              Harbor仓库用户凭据
            </DialogTitle>
            <DialogDescription className="text-yellow-600">
              请保存好您的用户名和密码！
              <br />
              Harbor仓库地址：https://gpu-harbor.act.buaa.edu.cn/
            </DialogDescription>
          </DialogHeader>
          {credentials && (
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="username" className="text-right">
                  Username
                </Label>
                <div className="col-span-3 flex items-center">
                  <User className="mr-2 h-4 w-4 text-gray-500" />
                  <Input
                    id="username"
                    value={credentials.name}
                    readOnly
                    className="pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    className="absolute right-12 hover:bg-transparent"
                    onClick={() => copyToClipboard(credentials.name)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="password" className="text-right">
                  Password
                </Label>
                <div className="col-span-3 flex items-center">
                  <Key className="mr-2 h-4 w-4 text-gray-500" />
                  <Input
                    id="password"
                    value={credentials.password}
                    readOnly
                    className="pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    className="absolute right-12 hover:bg-transparent"
                    onClick={() => copyToClipboard(credentials.password)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          )}
          <DialogFooter className="sm:justify-start">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsDialogOpen(false)}
            >
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};

const copyToClipboard = (text: string) => {
  navigator.clipboard
    .writeText(text)
    .then(() => {
      // 可以添加一个临时的成功提示
      toast.success("复制成功");
    })
    .catch((err) => {
      toast.error("复制失败", err);
    });
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
