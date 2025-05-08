// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
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
import useIsAdmin from "@/hooks/useAdmin";

interface SharedResourceTableProps {
  resourceType: "model" | "dataset" | "sharefile";
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
  const { t } = useTranslation();
  const { id } = useParams<{ id: string; name: string }>();
  const navigate = useNavigate();
  const datasetId = id ? parseInt(id, 10) : 0;
  const setBreadcrumb = useBreadcrumb();
  const user = useAtomValue(globalUserInfo);
  const isAdminMode = useIsAdmin();
  const dataTypeLabel = (() => {
    switch (resourceType) {
      case "dataset":
        return t("sharedResource.dataset");
      case "model":
        return t("sharedResource.model");
      case "sharefile":
        return t("sharedResource.sharefile");
      default:
        return t("sharedResource.dataset");
    }
  })();
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
      toast.success(t("sharedResource.shareCanceled"));
    },
  });

  const { mutate: deleteDataset } = useMutation({
    mutationFn: (datasetID: number) => apiDatasetDelete(datasetID),
    onSuccess: () => {
      navigate(-1);
      toast.success(
        t("sharedResource.deletedSuccess", { type: dataTypeLabel }),
      );
    },
    onError: () => {
      toast.error(t("sharedResource.deleteFailed", { type: dataTypeLabel }));
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
      toast.success(t("sharedResource.shareCanceled"));
    },
  });

  useEffect(() => {
    setBreadcrumb([{ title: t("sharedResource.detail") }]);
  }, [setBreadcrumb, t]);

  const formattedTags = useMemo(() => {
    const tags = query.data?.extra.tag;
    if (!tags || !Array.isArray(tags)) return [];
    return tags.map((tag) => ({ value: tag }));
  }, [query.data?.extra.tag]);

  const userDatasetColumns = useMemo<ColumnDef<UserDatasetResp>[]>(() => {
    return [
      {
        accessorKey: "index",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t("sharedResource.serialNumber")}
          />
        ),
        cell: ({ row }) => <div>{row.index + 1}</div>,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t("sharedResource.userName")}
          />
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
                    title={t("sharedResource.cancelShare")}
                  >
                    <X size={24} strokeWidth={2} />
                  </Button>
                </div>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>
                    {t("sharedResource.cancelUserShare")}
                  </DialogTitle>
                  <DialogDescription>
                    {t("sharedResource.cancelUserShareDescription", {
                      type: dataTypeLabel,
                    })}
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose>{t("sharedResource.cancel")}</DialogClose>
                  <DialogClose>
                    <Button
                      type="submit"
                      variant="default"
                      onClick={() => cancelShareWithUser(row.original.id)}
                    >
                      {t("sharedResource.confirm")}
                    </Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        ),
      },
    ];
  }, [cancelShareWithUser, dataTypeLabel, t]);

  const queueDatasetColumns = useMemo<ColumnDef<QueueDatasetGetResp>[]>(
    () => [
      {
        accessorKey: "index",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t("sharedResource.serialNumber")}
          />
        ),
        cell: ({ row }) => <div>{row.index + 1}</div>,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t("sharedResource.accountName")}
          />
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
                    title={t("sharedResource.cancelShare")}
                  >
                    <X size={24} strokeWidth={2} />
                  </Button>
                </div>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>
                    {t("sharedResource.cancelAccountShare")}
                  </DialogTitle>
                  <DialogDescription>
                    {t("sharedResource.cancelAccountShareDescription", {
                      type: dataTypeLabel,
                    })}
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose>{t("sharedResource.cancel")}</DialogClose>
                  <DialogClose>
                    <Button
                      type="submit"
                      variant="default"
                      onClick={() => cancelShareWithQueue(row.original.id)}
                    >
                      {t("sharedResource.confirm")}
                    </Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        ),
      },
    ],
    [cancelShareWithQueue, dataTypeLabel, t],
  );

  const datasetFilescolumns = useMemo<ColumnDef<FileItem>[]>(() => {
    const getHeader = (key: string): string => {
      switch (key) {
        case "name":
          return t("sharedResource.name");
        case "modifytime":
          return t("sharedResource.lastModified");
        case "size":
          return t("sharedResource.size");
        default:
          return key;
      }
    };
    return [
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
    ];
  }, [t]);

  return (
    <DetailPage
      header={
        <DetailTitle
          icon={resourceType === "model" ? BotIcon : DatabaseIcon}
          title={query.data?.name}
          description={
            query.data?.extra.editable
              ? t("sharedResource.editable")
              : t("sharedResource.readOnly")
          }
        >
          {(user?.name === query.data?.userInfo.username || isAdminMode) && (
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
                      title={t("sharedResource.userShare")}
                    >
                      <User size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>{t("sharedResource.shareToUser")}</DialogTitle>
                    <DialogDescription>
                      {t("sharedResource.shareToUserDescription", {
                        type: dataTypeLabel,
                      })}
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
                      title={t("sharedResource.accountShare")}
                    >
                      <Users size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>
                      {t("sharedResource.shareToAccount", {
                        type: dataTypeLabel,
                      })}
                    </DialogTitle>
                    <DialogDescription>
                      {t("sharedResource.shareToAccountDescription", {
                        type: dataTypeLabel,
                      })}
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
                      title={t("sharedResource.delete", {
                        type: dataTypeLabel,
                      })}
                    >
                      <Trash size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>
                      {t("sharedResource.deleteTitle", { type: dataTypeLabel })}
                    </DialogTitle>
                    <DialogDescription>
                      {t("sharedResource.deleteDescription", {
                        type: dataTypeLabel,
                        name: query.data?.name,
                      })}
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <DialogClose>
                      <Button variant="outline">
                        {t("sharedResource.cancel")}
                      </Button>
                    </DialogClose>
                    <DialogClose>
                      <Button
                        type="submit"
                        variant="default"
                        onClick={() => deleteDataset(datasetId)}
                      >
                        {t("sharedResource.delete")}
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
          title: t("sharedResource.user"),
          icon: UserRoundIcon,
          value: query.data?.userInfo.username,
        },
        {
          title: t("sharedResource.createdAt"),
          icon: CalendarIcon,
          value: <TimeDistance date={query.data?.createdAt} />,
        },
      ]}
      tabs={[
        {
          key: "datasetinfo",
          icon: FileIcon,
          label: t("sharedResource.datasetInfo", { type: dataTypeLabel }),
          children: (
            <div className="space-y-1 md:space-y-2 lg:space-y-3">
              {query.data?.extra.tag && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Folder className="mr-2 h-5 w-5 text-blue-500" />
                      {t("sharedResource.datasetTags", { type: dataTypeLabel })}
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
                      {t("sharedResource.datasetRepo", { type: dataTypeLabel })}
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
                    {t("sharedResource.datasetDescription", {
                      type: dataTypeLabel,
                    })}
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
          label: t("sharedResource.userShares", { type: dataTypeLabel }),
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
          label: t("sharedResource.accountShares", { type: dataTypeLabel }),
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
          label: t("sharedResource.datasetFiles", { type: dataTypeLabel }),
          children: (
            <>
              <TooltipButton
                variant="outline"
                size="icon"
                onClick={handleBackClick}
                className="mb-2 h-8 w-8"
                tooltipContent={t("sharedResource.goBack")}
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
          scrollable: true,
        },
      ]}
    />
  );
}
