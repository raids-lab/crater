import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ShareDatasetToUserDialog } from "@/components/custom/UserNotInSelect";
import {
  apiListUsersInDataset,
  apiListQueuesInDataset,
  UserDataset,
  QueueDataset,
  UserDatasetResp,
  QueueDatasetGetResp,
  apiGetDatasetByID,
  cancelSharedUserResp,
  cancelSharedQueueResp,
} from "@/services/api/dataset";
import {
  User,
  Users,
  X,
  UserRoundIcon,
  CalendarIcon,
  Trash,
  File,
  FileIcon,
  Folder,
  BotIcon,
  DatabaseIcon,
  FilesIcon,
  ArrowLeftIcon,
} from "lucide-react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { ShareDatasetToQueueDialog } from "@/components/custom/QueueNotInSelect";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "../custom/DataTable/DataTableColumnHeader";
import { TimeDistance } from "../custom/TimeDistance";
import { DetailPage } from "@/components/layout/DetailPage";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { DataTable } from "@/components/custom/DataTable";
import { useNavigate } from "react-router-dom";
import { DatasetUpdateForm } from "./updateform";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import DetailTitle from "../layout/DetailTitle";
import { FileItem, apiGetDatasetFiles } from "@/services/api/file";
import { FileSizeComponent } from "./FileSize";
import TooltipButton from "../custom/TooltipButton";

interface SharedResourceTableProps {
  resourceType: "model" | "dataset";
  apiShareDatasetwithUser: (
    ud: UserDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiShareDatasetwithQueue: (
    qd: QueueDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiCancelDatasetSharewithUser: (
    csu: cancelSharedUserResp,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiCancelDatasetSharewithQueue: (
    csq: cancelSharedQueueResp,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiDatasetDelete: (
    datasetID: number,
  ) => Promise<AxiosResponse<IResponse<string>>>;
}

export function SharedResourceTable({
  resourceType,
  apiShareDatasetwithUser,
  apiShareDatasetwithQueue,
  apiCancelDatasetSharewithUser,
  apiCancelDatasetSharewithQueue,
  apiDatasetDelete,
}: SharedResourceTableProps) {
  const { id } = useParams<{ id: string; name: string }>();
  const navigate = useNavigate();
  const datasetId = id ? parseInt(id, 10) : 0;
  const setBreadcrumb = useBreadcrumb();
  const user = useAtomValue(globalUserInfo);
  const userDatasetData = useQuery({
    queryKey: ["data", "userdataset", datasetId],
    queryFn: () => apiListUsersInDataset(datasetId),
    select: (res) => res.data.data,
  });
  const queueDatasetData = useQuery({
    queryKey: ["data", "queuedataset", datasetId],
    queryFn: () => apiListQueuesInDataset(datasetId),
    select: (res) => res.data.data,
  });
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["data", "datasetByID", datasetId],
    queryFn: () => apiGetDatasetByID(datasetId),
    select: (res) => res.data.data[0],
  });
  const [pathname, setPathname] = useState<string>("");

  const handleBackClick = () => {
    if (pathname) {
      const pathParts = pathname.split("/").filter(Boolean); // 分割路径并过滤空字符串

      if (pathParts.length > 1) {
        pathParts.pop(); // 移除最后一部分
        const newPath = "/" + pathParts.join("/"); // 重新拼接路径
        setPathname(newPath); // 更新当前路径
      } else {
        setPathname(""); // 如果路径是根目录，清空路径
      }
    }
  };

  const queryDataset = useQuery({
    queryKey: ["datasetfiles", pathname, datasetId],
    queryFn: () => apiGetDatasetFiles(datasetId, pathname),
    select: (res) => {
      return (
        res.data.data
          ?.map((r) => {
            return {
              name: r.name,
              modifytime: r.modifytime,
              isdir: r.isdir,
              size: r.size,
              sys: r.sys,
            };
          })
          .sort((a, b) => {
            if (a.isdir && !b.isdir) {
              return -1; // a在b之前
            } else if (!a.isdir && b.isdir) {
              return 1; // a在b之后
            } else {
              return a.name.localeCompare(b.name);
            }
          }) ?? []
      );
    },
  });

  const { mutate: cancelShareWithUser } = useMutation({
    mutationFn: (userId: number) =>
      apiCancelDatasetSharewithUser({ datasetID: datasetId, userID: userId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "userdataset", datasetId],
      });
      toast.success("已取消共享");
    },
  });

  const { mutate: deleteDataset } = useMutation({
    mutationFn: (datasetID: number) => apiDatasetDelete(datasetID),
    onSuccess: () => {
      navigate(-1);
      toast.success(`${resourceType === "model" ? "模型" : "数据集"}已删除`);
    },
    onError: () => {
      toast.error(`删除${resourceType === "model" ? "模型" : "数据集"}失败`);
    },
  });

  const { mutate: cancelShareWithQueue } = useMutation({
    mutationFn: (queueId: number) =>
      apiCancelDatasetSharewithQueue({
        datasetID: datasetId,
        queueID: queueId,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "queuedataset", datasetId],
      });
      toast.success("已取消共享");
    },
  });

  useEffect(() => {
    setBreadcrumb([{ title: "详情" }]);
  }, [setBreadcrumb]);

  const formattedTags = useMemo(() => {
    const tags = query.data?.extra.tag;
    if (!tags || !Array.isArray(tags)) return [];
    return tags.map((tag) => ({ value: tag }));
  }, [query.data?.extra.tag]);

  const userDatasetColumns = useMemo<ColumnDef<UserDatasetResp>[]>(
    () => [
      {
        accessorKey: "index",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"序号"} />
        ),
        cell: ({ row }) => <div>{row.index + 1}</div>,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"用户名称"} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => (
          <div className="flex flex-row space-x-1">
            <Dialog>
              <DialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-red-700"
                    title="取消共享"
                  >
                    <X size={24} strokeWidth={2} />
                  </Button>
                </div>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>取消用户共享</DialogTitle>
                  <DialogDescription>
                    和指定的用户取消共享
                    {resourceType === "model" ? "模型" : "数据集"}
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose>取消</DialogClose>
                  <DialogClose>
                    <Button
                      type="submit"
                      variant="default"
                      onClick={() => cancelShareWithUser(row.original.id)}
                    >
                      确认
                    </Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        ),
      },
    ],
    [cancelShareWithUser, resourceType],
  );

  const queueDatasetColumns = useMemo<ColumnDef<QueueDatasetGetResp>[]>(
    () => [
      {
        accessorKey: "index",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"序号"} />
        ),
        cell: ({ row }) => <div>{row.index + 1}</div>,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"账户名称"} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => (
          <div className="flex flex-row space-x-1">
            <Dialog>
              <DialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-red-700"
                    title="取消共享"
                  >
                    <X size={24} strokeWidth={2} />
                  </Button>
                </div>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>取消账户共享</DialogTitle>
                  <DialogDescription>
                    和指定的账户取消共享
                    {resourceType === "model" ? "模型" : "数据集"}
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose>取消</DialogClose>
                  <DialogClose>
                    <Button
                      type="submit"
                      variant="default"
                      onClick={() => cancelShareWithQueue(row.original.id)}
                    >
                      确认
                    </Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        ),
      },
    ],
    [cancelShareWithQueue, resourceType],
  );
  const getHeader = (key: string): string => {
    switch (key) {
      case "name":
        return "名称";
      case "modifytime":
        return "更新于";
      case "size":
        return "大小";
      default:
        return key;
    }
  };
  const datasetFilescolumns = useMemo<ColumnDef<FileItem>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return (
              <div style={{ display: "flex", alignItems: "center" }}>
                <Folder className="mr-2 size-5 text-yellow-600 dark:text-yellow-400" />
                <Button
                  onClick={() => {
                    setPathname((prevPath) => {
                      const cleanPrev = prevPath.replace(/\/+$/, "");
                      return `${cleanPrev}/${row.original.name}`;
                    });
                  }}
                  variant={"link"}
                  className="text-secondary-foreground h-8 px-0 text-left font-normal"
                >
                  {row.getValue("name")}
                </Button>
              </div>
            );
          } else {
            return (
              <div style={{ display: "flex", alignItems: "center" }}>
                <File className="text-muted-foreground mr-2 size-5" />
                <span className="text-secondary-foreground">
                  {row.getValue("name")}
                </span>
              </div>
            );
          }
        },
      },
      {
        accessorKey: "modifytime",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("modifytime")}
          />
        ),
        cell: ({ row }) => {
          return (
            <TimeDistance date={row.getValue("modifytime")}></TimeDistance>
          );
        },
      },
      {
        accessorKey: "size",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("size")} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return;
          } else {
            return <FileSizeComponent size={row.getValue("size")} />;
          }
        },
      },
    ],
    [],
  );

  return (
    <DetailPage
      header={
        <DetailTitle
          icon={resourceType === "model" ? BotIcon : DatabaseIcon}
          title={query.data?.name}
          description={query.data?.extra.editable ? "可编辑" : "只读"}
        >
          {user?.name === query.data?.userInfo.username && (
            <div className="flex flex-row space-x-1">
              <DatasetUpdateForm
                type={resourceType}
                initialData={{
                  datasetId: datasetId,
                  datasetName: query.data?.name || "",
                  describe: query.data?.describe || "",
                  url: query.data?.url || "",
                  type: resourceType,
                  tags: formattedTags,
                  weburl: query.data?.extra.weburl || "",
                  ispublic: true,
                  readOnly: !query.data?.extra.editable,
                }}
                onSuccess={() => {
                  queryClient.invalidateQueries({
                    queryKey: ["data", "datasetByID", datasetId],
                  });
                }}
              />
              <Dialog>
                <DialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="用户共享"
                    >
                      <User size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>分享至用户</DialogTitle>
                    <DialogDescription>
                      和指定的用户共享
                      {resourceType === "model" ? "模型" : "数据集"}
                    </DialogDescription>
                  </DialogHeader>
                  <ShareDatasetToUserDialog
                    datasetId={datasetId}
                    apiShareDatasetwithUser={apiShareDatasetwithUser}
                  />
                </DialogContent>
              </Dialog>
              <Dialog>
                <DialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="账户共享"
                    >
                      <Users size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>
                      共享{resourceType === "model" ? "模型" : "数据集"}
                    </DialogTitle>
                    <DialogDescription>
                      和账户共享{resourceType === "model" ? "模型" : "数据集"}
                    </DialogDescription>
                  </DialogHeader>
                  <ShareDatasetToQueueDialog
                    datasetId={datasetId}
                    apiShareDatasetwithQueue={apiShareDatasetwithQueue}
                  />
                </DialogContent>
              </Dialog>
              <Dialog>
                <DialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title={`删除${resourceType === "model" ? "模型" : "数据集"}`}
                    >
                      <Trash size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>
                      删除{resourceType === "model" ? "模型" : "数据集"}
                    </DialogTitle>
                    <DialogDescription>
                      {resourceType === "model" ? "模型" : "数据集"}「
                      {query.data?.name}」将被删除
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <DialogClose>
                      <Button variant="outline">取消</Button>
                    </DialogClose>
                    <DialogClose>
                      <Button
                        type="submit"
                        variant="default"
                        onClick={() => deleteDataset(datasetId)}
                      >
                        删除
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          )}
        </DetailTitle>
      }
      info={[
        {
          title: "用户",
          icon: UserRoundIcon,
          value: query.data?.userInfo.username,
        },
        {
          title: "创建于",
          icon: CalendarIcon,
          value: <TimeDistance date={query.data?.createdAt} />,
        },
      ]}
      tabs={[
        {
          key: "datasetinfo",
          icon: FileIcon,
          label: `${resourceType === "model" ? "模型" : "数据集"}基本信息`,
          children: (
            <div className="space-y-1 md:space-y-2 lg:space-y-3">
              {query.data?.extra.tag && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Folder className="mr-2 h-5 w-5 text-blue-500" />
                      {resourceType === "model" ? "模型" : "数据集"}标签
                    </CardTitle>
                    <CardDescription className="text-muted-foreground font-mono text-sm">
                      {query.data?.extra.tag.join("、")}
                    </CardDescription>
                  </CardHeader>
                </Card>
              )}
              {query.data?.extra.weburl && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Folder className="mr-2 h-5 w-5 text-blue-500" />
                      {resourceType === "model" ? "模型" : "数据集"}开源仓库地址
                    </CardTitle>
                    <CardDescription className="text-muted-foreground font-mono text-sm">
                      {query.data?.extra.weburl}
                    </CardDescription>
                  </CardHeader>
                </Card>
              )}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center text-xl">
                    <Folder className="mr-2 h-5 w-5 text-blue-500" />
                    {resourceType === "model" ? "模型" : "数据集"}描述
                  </CardTitle>
                  <CardDescription className="text-muted-foreground font-mono text-sm">
                    {query.data?.describe}
                  </CardDescription>
                </CardHeader>
              </Card>
            </div>
          ),
          scrollable: true,
        },
        {
          key: "usershare",
          icon: User,
          label: `${resourceType === "model" ? "模型" : "数据集"}共享用户`,
          children: (
            <DataTable
              storageKey="file_share_user"
              query={userDatasetData}
              columns={userDatasetColumns}
            />
          ),
          scrollable: true,
        },
        {
          key: "accountshare",
          icon: Users,
          label: `${resourceType === "model" ? "模型" : "数据集"}共享账户`,
          children: (
            <DataTable
              storageKey="file_share_queue"
              query={queueDatasetData}
              columns={queueDatasetColumns}
            />
          ),
          scrollable: true,
        },
        {
          key: "datasetFiles",
          icon: FilesIcon,
          label: `${resourceType === "model" ? "模型" : "数据集"}文件`,
          children: (
            <>
              <TooltipButton
                variant="outline"
                size="icon"
                onClick={handleBackClick}
                className="h-8 w-8"
                tooltipContent="返回上一级"
              >
                <ArrowLeftIcon className="size-4" />
              </TooltipButton>

              <DataTable
                storageKey="dataset_files"
                query={queryDataset}
                columns={datasetFilescolumns}
              />
            </>
          ),
        },
      ]}
    />
  );
}
