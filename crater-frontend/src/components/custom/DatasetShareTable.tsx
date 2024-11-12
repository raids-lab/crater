import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { Label } from "@/components/ui/label";
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
import { OnChangeValue } from "react-select";
import { ShareDatasetToUserDialog } from "@/components/custom/UserNotInSelect";
import {
  apiListUsersInDataset,
  apiListQueuesInDataset,
  UserDataset,
  QueueDataset,
  UserDatasetResp,
  QueueDatasetGetResp,
  Dataset,
  apiDatasetRename,
  apiGetDatasetByID,
  cancelSharedUserResp,
  cancelSharedQueueResp,
} from "@/services/api/dataset";
import { Pencil, User, Users, X } from "lucide-react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { QueueNotInSelect } from "@/components/custom/QueueNotInSelect";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { DataTable } from "./DataTable";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "./DataTable/DataTableColumnHeader";
import { TimeDistance } from "./TimeDistance";
import { Input } from "../ui/input";
interface QueueOption {
  value: string;
  id: number;
  label: string;
}
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
}
export function DatasetShareTable({
  apiShareDatasetwithUser,
  apiShareDatasetwithQueue,
  apiCancelDatasetSharewithUser,
  apiCancelDatasetSharewithQueue,
}: DatesetShareTableProps) {
  const { id } = useParams<{ id: string; name: string }>();

  const datasetId = id ? parseInt(id, 10) : 0;
  const setBreadcrumb = useBreadcrumb();
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
  const refInput = useRef<HTMLInputElement>(null);
  const [datasetNewName, setDatasetNewName] = useState<string>("");
  const data = useQuery({
    queryKey: ["data", "datasetByID", datasetId],
    queryFn: () => apiGetDatasetByID(datasetId),
    select: (res) => res.data.data,
  });

  const { mutate: RenameDataset } = useMutation({
    mutationFn: (datasetid: number) => rename(datasetid),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "datasetByID", datasetId],
      });
    },
  });

  const rename = async (datasetid: number) => {
    if (datasetNewName != "") {
      await apiDatasetRename({
        datasetID: datasetid,
        name: datasetNewName,
      }).then(() => {
        toast.success("已修改名称");
      });
    } else {
      toast.error("名称不能为空");
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setDatasetNewName(e.target.value);
  };

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

  const { mutate: shareWithQueue } = useMutation({
    mutationFn: (datasetId: number) =>
      apiShareDatasetwithQueue({
        datasetID: datasetId,
        queueIDs: queueIds,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "queuedataset", datasetId],
      });
      toast.success("共享成功");
      setQueueIds([]);
    },
  });
  const getHeader = (key: string): string => {
    switch (key) {
      case "name":
        return "名称";
      case "createdAt":
        return "创建于";
      case "username":
        return "创建者";
      case "describe":
        return "描述";
      case "url":
        return "数据集位置";
      default:
        return key;
    }
  };

  const [queueIds, setQueueIds] = useState<number[]>([]);
  const onChangeQueue = (newValue: OnChangeValue<QueueOption, true>) => {
    const queueIds: number[] = newValue.map((queue) => queue.id);
    setQueueIds(queueIds);
  };

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
                    和指定的用户取消共享数据集
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
  const datasetColumns = useMemo<ColumnDef<Dataset>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => {
          return <div>{row.getValue("name")}</div>;
        },
        enableSorting: false,
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("createdAt")}
          />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue("createdAt")}></TimeDistance>;
        },
        //sortingFn: "datetime",
        enableSorting: false,
      },
      {
        accessorKey: "username",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("username")}
          />
        ),
        cell: ({ row }) => {
          return <div>{row.getValue("username")}</div>;
        },
      },
      {
        accessorKey: "describe",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("describe")}
          />
        ),
        cell: ({ row }) => {
          return (
            <div style={{ display: "flex", alignItems: "center" }}>
              {row.getValue("describe")}
            </div>
          );
        },
        //sortingFn: "datetime",
        enableSorting: false,
      },
      {
        accessorKey: "url",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("url")} />
        ),
        cell: ({ row }) => {
          return <div>{row.getValue("url")}</div>;
        },
        //sortingFn: "datetime",
        enableSorting: false,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          return (
            <div className="flex flex-row space-x-1">
              <Dialog>
                <DialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8"
                      size="icon"
                      title="重命名"
                    >
                      <Pencil size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>重命名</DialogTitle>
                    <DialogDescription>重命名数据集</DialogDescription>
                  </DialogHeader>
                  <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="rename" className="text-right">
                        数据集新名称
                      </Label>
                      <Input
                        id="rename"
                        type="text"
                        defaultValue=""
                        className="col-span-3"
                        ref={refInput}
                        onChange={handleChange}
                        required
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <DialogClose>取消</DialogClose>
                    <DialogClose>
                      <Button
                        type="submit"
                        variant="default"
                        onClick={() => RenameDataset(row.original.id)}
                      >
                        确认
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
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
                      和指定的用户共享数据集
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
                    <DialogTitle>共享数据集</DialogTitle>
                    <DialogDescription>和账户共享数据集</DialogDescription>
                  </DialogHeader>
                  <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="queue" className="text-right">
                        账户名称
                      </Label>
                      <QueueNotInSelect
                        id={datasetId}
                        onChange={onChangeQueue}
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <DialogClose>
                      <Button variant="outline">取消</Button>
                    </DialogClose>
                    <DialogClose>
                      <Button
                        type="submit"
                        variant="default"
                        onClick={() => shareWithQueue(datasetId)}
                      >
                        共享
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          );
        },
      },
    ],
    [RenameDataset, apiShareDatasetwithUser, datasetId, shareWithQueue],
  );
  return (
    <>
      <DataTable
        info={{
          title: "数据集信息",
          description: "",
        }}
        query={data}
        columns={datasetColumns}
      ></DataTable>
      <DataTable
        info={{
          title: "数据集共享用户",
          description: "",
        }}
        query={userDatasetData}
        columns={userDatasetColumns}
      ></DataTable>
      <DataTable
        info={{
          title: "数据集共享账户",
          description: "",
        }}
        query={queueDatasetData}
        columns={queueDatasetColumns}
      ></DataTable>
    </>
  );
}
