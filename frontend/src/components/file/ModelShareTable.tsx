import { useEffect, useMemo } from "react";
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
  FileIcon,
  Folder,
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
import PageTitle from "@/components/layout/PageTitle";
import { DataTable } from "@/components/custom/DataTable";
import { useNavigate } from "react-router-dom";
import { DatasetUpdateForm } from "./updateform";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";

interface DatesetShareTableProps {
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

export function ModelShareTable({
  apiShareDatasetwithUser,
  apiShareDatasetwithQueue,
  apiCancelDatasetSharewithUser,
  apiCancelDatasetSharewithQueue,
  apiDatasetDelete,
}: DatesetShareTableProps) {
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
  const data = useQuery({
    queryKey: ["data", "datasetByID", datasetId],
    queryFn: () => apiGetDatasetByID(datasetId),
    select: (res) => res.data.data,
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
      // 删除成功后刷新数据
      navigate(-1);
      toast.success("模型已删除");
    },
    onError: () => {
      toast.error("删除模型失败");
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
    setBreadcrumb([{ title: "共享情况" }]);
  }, [setBreadcrumb]);

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
                    和指定的用户取消共享模型
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
    [cancelShareWithUser],
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
                    和指定的账户取消共享数据集
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
    [cancelShareWithQueue],
  );
  return (
    <DetailPage
      header={
        <PageTitle title={data.data?.[0]?.name}>
          {user?.name === data.data?.[0]?.attribute.name && (
            <div className="flex flex-row space-x-1">
              <DatasetUpdateForm
                type="dataset"
                initialData={{
                  datasetId: datasetId, // 使用当前数据集ID
                  datasetName: data.data?.[0]?.name || "",
                  describe: data.data?.[0]?.describe || "",
                  url: data.data?.[0]?.url || "",
                  type: "dataset",
                  tags: data.data?.[0]?.extra.tag || [],
                  weburl: data.data?.[0]?.extra.weburl || "",
                  ispublic: true,
                }}
                onSuccess={() => {
                  // 重新查询数据
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
                    <DialogDescription>和指定的用户共享模型</DialogDescription>
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
                    <DialogTitle>共享模型</DialogTitle>
                    <DialogDescription>和账户共享模型</DialogDescription>
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
                      title="删除模型"
                    >
                      <Trash size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>删除模型</DialogTitle>
                    <DialogDescription>
                      模型「{data.data?.[0]?.name}」将被删除
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
        </PageTitle>
      }
      info={[
        { title: "用户", icon: UserRoundIcon, value: data.data?.[0]?.username },
        {
          title: "创建于",
          icon: CalendarIcon,
          value: <TimeDistance date={data.data?.[0]?.createdAt} />,
        },
      ]}
      tabs={[
        {
          key: "datasetinfo",
          icon: FileIcon,
          label: "模型基本信息", // 数据集信息标签
          children: (
            <div className="space-y-1 md:space-y-2 lg:space-y-3">
              {data.data?.[0]?.extra.tag && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Folder className="mr-2 h-5 w-5 text-blue-500" />
                      模型标签
                    </CardTitle>
                    <CardDescription className="text-muted-foreground font-mono text-sm">
                      {data.data?.[0]?.extra.tag.join("、")}
                    </CardDescription>
                  </CardHeader>
                </Card>
              )}
              {data.data?.[0]?.extra.weburl && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Folder className="mr-2 h-5 w-5 text-blue-500" />
                      模型开源仓库地址
                    </CardTitle>
                    <CardDescription className="text-muted-foreground font-mono text-sm">
                      {data.data?.[0]?.extra.weburl}
                    </CardDescription>
                  </CardHeader>
                </Card>
              )}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center text-xl">
                    <Folder className="mr-2 h-5 w-5 text-blue-500" />
                    模型描述
                  </CardTitle>
                  <CardDescription className="text-muted-foreground font-mono text-sm">
                    {data.data?.[0]?.describe}
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
          label: "模型共享用户",
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
          label: "模型共享账户",
          children: (
            <DataTable
              storageKey="file_share_queue"
              query={queueDatasetData}
              columns={queueDatasetColumns}
            />
          ),
          scrollable: true,
        },
      ]}
    />
  );
}
