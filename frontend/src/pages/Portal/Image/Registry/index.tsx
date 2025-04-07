import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  apiGetHarborIP,
  apiUserDeleteKanikoList,
  apiUserGetCredential,
  apiUserGetQuota,
  apiUserListKaniko,
  getHeader,
  ImageLinkPair,
  ImagePackStatus,
  imagepackStatuses,
  KanikoInfoResponse,
  ListKanikoResponse,
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
  AlertTriangle,
  RefreshCw,
  ExternalLink,
  ListStart,
  CheckCheck,
  SquareCheckBig,
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
import { IResponse } from "@/services/types";
import { AxiosResponse } from "axios";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { ValidDialog } from "../Image/ValidDialog";
import UserLabel from "@/components/label/UserLabel";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "imageLink",
  },
  filterOptions: [
    {
      key: "status",
      title: "状态",
      option: imagepackStatuses,
    },
  ],
  getHeader: getHeader,
};

interface KanikoListTableProps {
  apiListKaniko: () => Promise<AxiosResponse<IResponse<ListKanikoResponse>>>;
  apiDeleteKanikoList: (
    idList: number[],
  ) => Promise<AxiosResponse<IResponse<string>>>;
  isAdminMode: boolean;
}

export const KanikoListTable: FC<KanikoListTableProps> = ({
  apiListKaniko,
  apiDeleteKanikoList,
  isAdminMode,
}) => {
  const queryClient = useQueryClient();
  const [openPipAptSheet, setOpenPipAptSheet] = useState(false);
  const [openDockerfileSheet, setOpenDockerfileSheet] = useState(false);
  const navigate = useNavigate();
  const [openCheckDialog, setCheckOpenDialog] = useState(false);
  const user = useAtomValue(globalUserInfo);
  const [selectedLinkPairs, setSelectedLinkPairs] = useState<ImageLinkPair[]>(
    [],
  );

  const imageQuery = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiListKaniko(),
    select: (res) => res.data.data.kanikoList,
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

  const { mutate: deleteKanikoList } = useMutation({
    mutationFn: (idList: number[]) => apiDeleteKanikoList(idList),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });

  let columns: ColumnDef<KanikoInfoResponse>[] = [
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
      id: "userInfo",
      accessorKey: "userInfo",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("userInfo")} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
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
                  <DropdownMenuLabel className="text-muted-foreground text-xs">
                    操作
                  </DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => navigate(`${kanikoInfo.ID}`)}
                  >
                    <InfoIcon className="text-highlight-emerald" />
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
                      deleteKanikoList([kanikoInfo.ID]);
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
  columns = columns.filter((column) => column.id !== "nickName" || isAdminMode);
  return (
    <div className="flex flex-col gap-3">
      <div>
        <PageTitle
          title="镜像制作"
          description="支持 Dockerfile 、低代码、快照等方式制作镜像"
          className="mb-2"
        >
          {!isAdminMode ? (
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
          ) : null}
        </PageTitle>
        {!isAdminMode ? (
          <ProjectDetail
            successImageNumber={
              imageQuery.data?.filter((c) => c.status == "Finished").length ?? 0
            }
          />
        ) : null}
      </div>
      <DataTable
        storageKey="image_registry"
        query={imageQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        className="lg:col-span-2"
        multipleHandlers={[
          {
            title: (rows) =>
              `删除 ${rows.length} 个镜像创建任务，以及对应镜像链接`,
            description: (rows) => (
              <div className="border-destructive/20 bg-destructive/5 rounded-md border px-4 py-3">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="text-destructive mt-0.5 h-5 w-5 shrink-0" />
                  <div>
                    <p className="text-destructive font-medium">
                      以下镜像创建任务和对应镜像链接将被删除，确认要继续吗？
                    </p>
                    <p className="text-muted-foreground mt-1 text-sm">
                      {"『" +
                        rows
                          .map((row) => row.original.description)
                          .join("』,『") +
                        "』"}
                    </p>
                  </div>
                </div>
              </div>
            ),
            icon: <Trash2Icon className="text-destructive" />,
            handleSubmit: (rows) => {
              const ids = rows.map((row) => row.original.ID);
              deleteKanikoList(ids);
            },
            isDanger: true,
          },
          {
            title: (rows) => `检测 ${rows.length} 个镜像链接`,
            description: (rows) => (
              <div className="rounded-md border border-green-600/20 bg-green-600/5 px-4 py-3">
                <div className="flex items-start gap-3">
                  <SquareCheckBig className="mt-0.5 h-5 w-5 shrink-0 text-green-600" />
                  <div>
                    <p className="font-medium text-green-600">
                      以下镜像链接将被检测
                    </p>
                    <p className="text-muted-foreground mt-1 text-sm">
                      {"『" +
                        rows
                          .map((row) => row.original.description)
                          .join("』,『") +
                        "』"}
                    </p>
                  </div>
                </div>
              </div>
            ),
            icon: <CheckCheck className="text-green-600" />,
            handleSubmit: (rows) => {
              setSelectedLinkPairs(
                rows.map((row) => ({
                  id: row.original.ID,
                  imageLink: row.original.imageLink,
                  description: row.original.description,
                  creator: row.original.userInfo,
                })),
              );
              setCheckOpenDialog(true);
            },
            isDanger: false,
          },
        ]}
      />
      {!isAdminMode ? (
        <div>
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
        </div>
      ) : null}
      <AlertDialog open={openCheckDialog} onOpenChange={setCheckOpenDialog}>
        <AlertDialogContent>
          <ValidDialog
            linkPairs={selectedLinkPairs}
            onDeleteLinks={(invalidPairs: ImageLinkPair[]) => {
              deleteKanikoList(
                invalidPairs
                  .filter((pair) => pair.creator.username === user.name)
                  .map((pair) => pair.id),
              );
            }}
          />
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

interface UserHarborCredentialsDialogProps {
  isDialogOpen: boolean;
  setIsDialogOpen: (open: boolean) => void;
}

export const UserHarborCredentialsDialog: FC<
  UserHarborCredentialsDialogProps
> = ({ isDialogOpen, setIsDialogOpen }) => {
  const [credentials, setCredentials] =
    useState<ProjectCredentialResponse | null>(null);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const user = useAtomValue(globalUserInfo);
  const { mutate: getProjectCredential } = useMutation({
    mutationFn: () => apiUserGetCredential(),
    onSuccess: async (data) => {
      setCredentials(data.data.data);
      setShowConfirmation(false);
      toast.success("凭据已生成, 请保存您的密码！");
    },
  });
  const harborIP = useQuery({
    queryKey: ["harbor", "ip"],
    queryFn: () => apiGetHarborIP(),
    select: (res) => res.data.data,
  });
  const handleResetClick = () => {
    setShowConfirmation(true);
  };
  return (
    <>
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="h-6 w-6 text-blue-500" />
              Harbor仓库用户凭据
            </DialogTitle>
          </DialogHeader>
          <div>
            <Alert className="my-3 border-yellow-500 bg-yellow-50">
              <AlertTriangle className="h-5 w-5 text-yellow-600" />
              <AlertDescription className="font-medium text-yellow-800">
                请保存好您的用户名和密码，密码只会显示一次！
              </AlertDescription>
            </Alert>
            <div className="flex items-center justify-between rounded-md border border-blue-200 bg-blue-50 px-2 py-3">
              <div className="text-sm text-blue-700">Harbor仓库地址：</div>
              <a
                href={harborIP.data?.ip}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-sm font-medium text-blue-600 hover:text-blue-800"
              >
                {harborIP.data?.ip}
                <ExternalLink className="h-3 w-3" />
              </a>
            </div>
          </div>
          <Separator />
          <div className="grid gap-4 rounded-md border border-gray-100 bg-gray-50 p-3 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="username" className="text-right">
                用户名
              </Label>
              <div className="col-span-3 flex items-center">
                <User className="mr-2 h-4 w-4 text-gray-500" />
                <Input
                  id="username"
                  value={user.name}
                  readOnly
                  className="pr-10"
                />
              </div>
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="project" className="text-right">
                项目名
              </Label>
              <div className="col-span-3 flex items-center">
                <ListStart className="mr-2 h-4 w-4 text-gray-500" />
                <Input
                  id="project"
                  value={"user-" + user.name}
                  readOnly
                  className="pr-10"
                />
              </div>
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="password" className="text-right">
                密码
              </Label>
              <div className="col-span-3 flex items-center">
                <Key className="mr-2 h-4 w-4 text-gray-500" />
                <Input
                  id="password"
                  value={credentials ? credentials.password : "•".repeat(10)}
                  readOnly
                  className="pr-10"
                />
                {credentials ? (
                  <Button
                    type="button"
                    variant="ghost"
                    className="absolute right-12 hover:bg-transparent"
                    onClick={() => copyToClipboard(credentials.password)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                ) : null}
              </div>
            </div>
          </div>
          <DialogFooter className="sm:justify-between">
            <Button
              type="button"
              variant="outline"
              className="flex items-center gap-2"
              onClick={() => handleResetClick()}
            >
              <RefreshCw className="h-4 w-4" />
              重置密码
            </Button>
            <Button
              type="button"
              variant="default"
              onClick={() => setIsDialogOpen(false)}
            >
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <AlertDialog open={showConfirmation} onOpenChange={setShowConfirmation}>
        <AlertDialogContent className="max-w-[150px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认重置密码</AlertDialogTitle>
            <AlertDialogDescription>
              您确定要重置密码吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => getProjectCredential()}
              className="bg-red-500 hover:bg-red-600"
            >
              确认重置
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

interface ProjectDetailProps {
  successImageNumber: number;
}

export const ProjectDetail: FC<ProjectDetailProps> = ({
  successImageNumber,
}) => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const quotaQuery = useQuery({
    queryKey: ["imagepack", "quota"],
    queryFn: () => apiUserGetQuota(),
    select: (res) => res.data.data,
  });
  return (
    <>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <BoxIcon className="text-primary size-5" />
              镜像总数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {quotaQuery.isLoading ? (
                <LoadingCircleIcon />
              ) : (
                <>{successImageNumber}</>
              )}
            </p>
          </CardContent>
        </Card>

        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <HardDriveIcon className="text-primary size-5" />
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

        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <UserRoundIcon className="text-primary size-5" />
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

        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <KeyIcon className="text-primary size-5" />
              访问凭据
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              onClick={() => setIsDialogOpen(true)}
              className="w-full"
            >
              获取初始凭据
            </Button>
          </CardContent>
        </Card>
      </div>

      <UserHarborCredentialsDialog
        isDialogOpen={isDialogOpen}
        setIsDialogOpen={setIsDialogOpen}
      />
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
      element: (
        <KanikoListTable
          apiListKaniko={apiUserListKaniko}
          apiDeleteKanikoList={apiUserDeleteKanikoList}
          isAdminMode={false}
        />
      ),
    },
    {
      path: ":id",
      element: <KanikoDetail />,
    },
  ]);

  return <>{routes}</>;
};
