//import type { FC } from "react";
import { useMemo, useRef, useState } from "react";
import { DataTable } from "@/components/custom/DataTable";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Button } from "@/components/ui/button";
import { TableDate } from "@/components/custom/TableDate";
// import { ArrowLeftIcon } from "@radix-ui/react-icons";
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
  Dataset,
  apiDatasetDelete,
  apiDatasetRename,
} from "@/services/api/dataset";
import { Pencil, Trash2, User, Users } from "lucide-react";
import { toast } from "sonner";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Separator } from "@radix-ui/react-dropdown-menu";
import { DatasetCreateForm } from "@/pages/Portal/Data/CreateForm";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import { OnChangeValue } from "react-select";
import { ShareDatasetToUserDialog } from "@/components/custom/UserNotInSelect";
import { QueueNotInSelect } from "@/components/custom/QueueNotInSelect";
import { UserDataset } from "@/services/api/dataset";
import { QueueDataset } from "@/services/api/dataset";

interface QueueOption {
  value: string;
  id: number;
  label: string;
}

import { IResponse } from "@/services/types";
import { AxiosResponse } from "axios";
interface DatesetTableProps {
  apiGetDataset: () => Promise<AxiosResponse<IResponse<Dataset[]>>>;
  apiShareDatasetwithUser: (
    ud: UserDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiShareDatasetwithQueue: (
    qd: QueueDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
}
export function DatasetTable({
  apiGetDataset,
  apiShareDatasetwithUser,
  apiShareDatasetwithQueue,
}: DatesetTableProps) {
  const refInput = useRef<HTMLInputElement>(null);
  const [datasetNewName, setDatasetNewName] = useState<string>("");
  const data = useQuery({
    queryKey: ["data", "mydataset"],
    queryFn: () => apiGetDataset(),
    select: (res) => res.data.data,
  });

  const { mutate: shareWithQueue } = useMutation({
    mutationFn: (datasetId: number) =>
      apiShareDatasetwithQueue({
        datasetID: datasetId,
        queueIDs: queueIds,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "mydataset"],
      });
      toast.success("共享成功");
      setQueueIds([]);
    },
  });

  const { mutate: RenameDataset } = useMutation({
    mutationFn: (datasetid: number) => rename(datasetid),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "mydataset"],
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
  const { mutate: deleteDataset } = useMutation({
    mutationFn: (req: number) => apiDatasetDelete(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "mydataset"],
      });
      toast.success("数据集已删除");
    },
  });
  const queryClient = useQueryClient();

  const [queueIds, setQueueIds] = useState<number[]>([]);
  const onChangeQueue = (newValue: OnChangeValue<QueueOption, true>) => {
    const queueIds: number[] = newValue.map((queue) => queue.id);
    setQueueIds(queueIds);
  };

  const [openSheet, setOpenSheet] = useState(false);

  const columns = useMemo<ColumnDef<Dataset>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => {
          return (
            <div style={{ display: "flex", alignItems: "center" }}>
              {row.getValue("name")}
            </div>
          );
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
          return <TableDate date={row.getValue("createdAt")}></TableDate>;
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
                      className="h-8 w-8 p-0"
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
                    datasetId={row.original.id}
                    apiShareDatasetwithUser={apiShareDatasetwithUser}
                  />
                </DialogContent>
              </Dialog>
              <Dialog>
                <DialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8"
                      size="icon"
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
                        id={row.original.id}
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
                        onClick={() => shareWithQueue(row.original.id)}
                      >
                        共享
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-destructive"
                      title="删除数据集"
                    >
                      <Trash2 size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除数据集</AlertDialogTitle>
                    <AlertDialogDescription>
                      数据集将删除
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        // check if browser support clipboard
                        deleteDataset(row.original.id);
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
    ],
    [apiShareDatasetwithUser, deleteDataset, shareWithQueue, RenameDataset],
  );

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
  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索名称",
      key: "name",
    },
    filterOptions: [],
    getHeader: getHeader,
  };

  return (
    <DataTable
      query={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      className="col-span-3"
    >
      <Sheet open={openSheet} onOpenChange={setOpenSheet}>
        <SheetTrigger asChild>
          <Button className="h-8 min-w-fit">创建数据集</Button>
        </SheetTrigger>
        <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>创建数据集</SheetTitle>
            <SheetDescription>创建一个新的文件数据集</SheetDescription>
          </SheetHeader>
          <Separator className="mt-4" />
          <DatasetCreateForm closeSheet={() => setOpenSheet(false)} />
        </SheetContent>
      </Sheet>
    </DataTable>
  );
}
