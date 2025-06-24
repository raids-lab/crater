import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import { Button } from "@/components/ui/button";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  apiUserDeleteKanikoList,
  apiUserListKaniko,
  getHeader,
  ImageLinkPair,
  ImagePackSource,
  ImagePackStatus,
  imagepackStatuses,
  KanikoInfoResponse,
  ListKanikoResponse,
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
  InfoIcon,
  PackagePlusIcon,
  Trash2Icon,
  AlertTriangle,
  CheckCheck,
  SquareCheckBig,
  RedoDotIcon,
} from "lucide-react";
import { useNavigate, useRoutes } from "react-router-dom";
import KanikoDetail from "../Info";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import DocsButton from "@/components/button/DocsButton";
import { PipAptSheet } from "./PipAptSheet";
import { DockerfileSheet } from "./DockerfileSheet";
import SplitButton from "@/components/button/SplitButton";
import TooltipLink from "@/components/label/TooltipLink";
import ImageLabel from "@/components/label/ImageLabel";
import ImagePhaseBadge from "@/components/badge/ImagePhaseBadge";
import { formatBytes } from "@/utils/formatter";
import { IResponse } from "@/services/types";
import { AxiosResponse } from "axios";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { ValidDialog } from "../Image/ValidDialog";
import UserLabel from "@/components/label/UserLabel";
import { EnvdSheet } from "./EnvdSheet";
import { EnvdRawSheet } from "./EnvdRawSheet";
import { ProjectDetail } from "./ProjectDetail";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "image",
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
  const [openEnvdSheet, setOpenEnvdSheet] = useState(false);
  const [openEnvdRawSheet, setOpenEnvdRawSheet] = useState(false);
  const [imagePackName, setImagePackName] = useState<string>("");
  const navigate = useNavigate();
  const [openCheckDialog, setCheckOpenDialog] = useState(false);
  const user = useAtomValue(globalUserInfo);
  const [selectedLinkPairs, setSelectedLinkPairs] = useState<ImageLinkPair[]>(
    [],
  );

  const imageQuery = useQuery({
    queryKey: ["imagepack", "list"],
    queryFn: () => apiListKaniko(),
    select: (res) =>
      res.data.data.kanikoList.map((i) => ({
        ...i,
        image: `${i.imageLink} (${i.description})`,
      })),
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
      accessorKey: "image",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("image")} />
      ),
      cell: ({ row }) => (
        <TooltipLink
          name={
            <ImageLabel
              description={row.original.description}
              url={row.original.imageLink}
              tags={row.original.tags}
            />
          }
          to={`${row.original.imagepackName}`}
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
        return (
          <>
            {row.getValue<number>("size") > 0 && (
              <span>{formatBytes(row.getValue("size"))}</span>
            )}
          </>
        );
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
                    onClick={() => navigate(`${kanikoInfo.imagepackName}`)}
                  >
                    <InfoIcon className="text-highlight-emerald" />
                    详情
                  </DropdownMenuItem>

                  <DropdownMenuItem
                    onClick={() => {
                      setImagePackName(kanikoInfo.imagepackName);
                      if (
                        kanikoInfo.buildSource === ImagePackSource.EnvdAdvanced
                      ) {
                        setOpenEnvdSheet(true);
                      } else if (
                        kanikoInfo.buildSource === ImagePackSource.EnvdRaw
                      ) {
                        setOpenEnvdRawSheet(true);
                      } else if (
                        kanikoInfo.buildSource === ImagePackSource.Dockerfile
                      ) {
                        setOpenDockerfileSheet(true);
                      } else if (
                        kanikoInfo.buildSource === ImagePackSource.PipApt
                      ) {
                        setOpenPipAptSheet(true);
                      } else {
                        toast.error("该镜像不支持克隆");
                      }
                    }}
                  >
                    <RedoDotIcon className="text-highlight-purple size-4" />
                    克隆
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
    <>
      <DataTable
        info={
          isAdminMode
            ? {
                title: "镜像制作管理",
                description: "支持 Dockerfile 、低代码、快照等方式制作镜像",
              }
            : {
                title: "镜像制作",
                description: "支持 Dockerfile 、低代码、快照等方式制作镜像",
              }
        }
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
                  <div className="w-full overflow-hidden">
                    <p className="flex items-center gap-2 font-medium">
                      <AlertTriangle className="text-destructive h-4 w-4" />
                      <span className="text-destructive text-sm leading-4">
                        以下镜像创建任务和对应镜像链接将被删除，确认要继续吗？
                      </span>
                    </p>
                    <Separator className="my-2" />
                    <ScrollArea className="h-[200px] w-full overflow-hidden">
                      <div className="p-4">
                        {rows.map((row, index) => (
                          <p
                            key={index}
                            className="text-muted-foreground text-sm"
                          >
                            {`『${row.original.description}』`}
                          </p>
                        ))}
                      </div>
                    </ScrollArea>
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
                  <div className="w-full overflow-hidden">
                    <p className="flex items-center gap-2 font-medium">
                      <SquareCheckBig className="h-4 w-4 text-green-600" />
                      <span className="text-sm leading-4 text-green-600">
                        以下镜像链接将被检测
                      </span>
                    </p>
                    <Separator className="my-2" />
                    <ScrollArea className="h-[200px] w-full overflow-hidden">
                      <div className="p-4">
                        {rows.map((row, index) => (
                          <p
                            key={index}
                            className="text-muted-foreground text-sm"
                          >
                            {`『${row.original.description}』`}
                          </p>
                        ))}
                      </div>
                    </ScrollArea>
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
        briefChildren={
          !isAdminMode ? (
            <ProjectDetail
              successImageNumber={
                imageQuery.data?.filter((c) => c.status == "Finished").length ??
                0
              }
            />
          ) : null
        }
      >
        {!isAdminMode ? (
          <div className="flex flex-row gap-3">
            <DocsButton title="查看文档" url="image/imagebuild" />
            <SplitButton
              icon={<PackagePlusIcon />}
              renderTitle={(title) => title}
              itemTitle="构建方式"
              items={[
                {
                  key: "envd",
                  title: "Python + CUDA 自定义构建",
                  action: () => {
                    setOpenEnvdSheet(true);
                  },
                },
                {
                  key: "pip-apt",
                  title: "基于现有镜像构建",
                  action: () => {
                    setOpenPipAptSheet(true);
                  },
                },
                {
                  key: "dockerfile",
                  title: "基于 Dockerfile 构建",
                  action: () => {
                    setOpenDockerfileSheet(true);
                  },
                },
                {
                  key: "envd-raw",
                  title: "基于 Envd 构建",
                  action: () => {
                    setOpenEnvdRawSheet(true);
                  },
                },
              ]}
              cacheKey="imagepack"
            />
          </div>
        ) : null}
      </DataTable>
      {!isAdminMode ? (
        <div>
          <EnvdSheet
            isOpen={openEnvdSheet}
            onOpenChange={setOpenEnvdSheet}
            title="Python + CUDA 自定义构建"
            description="Python+CUDA指定版本构建"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenEnvdSheet(false);
              setImagePackName("");
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <EnvdRawSheet
            isOpen={openEnvdRawSheet}
            onOpenChange={setOpenEnvdRawSheet}
            title="高级 Envd 构建脚本"
            description="直接编写 Envd 构建脚本，实现更复杂的定制化"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenEnvdRawSheet(false);
              setImagePackName("");
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <PipAptSheet
            isOpen={openPipAptSheet}
            onOpenChange={setOpenPipAptSheet}
            title="基于现有镜像构建"
            description="基于平台提供的基础镜像，快速制作自定义镜像"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenPipAptSheet(false);
              setImagePackName("");
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <DockerfileSheet
            isOpen={openDockerfileSheet}
            onOpenChange={setOpenDockerfileSheet}
            title="基于 Dockerfile 构建镜像"
            description="基于 Dockerfile 制作镜像"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenDockerfileSheet(false);
              setImagePackName("");
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
        </div>
      ) : null}
      <Dialog open={openCheckDialog} onOpenChange={setCheckOpenDialog}>
        <DialogContent>
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
        </DialogContent>
      </Dialog>
    </>
  );
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
      path: ":name",
      element: <KanikoDetail />,
    },
  ]);

  return <>{routes}</>;
};
