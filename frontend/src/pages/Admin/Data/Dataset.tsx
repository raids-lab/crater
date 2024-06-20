import type { FC } from "react";

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
  apiAdminGetDataset,
  apiAdminShareDatasetwithQueue,
  apiAdminShareDatasetwithUser,
  apiDatasetDelete,
  apiDatasetRename,
} from "@/services/api/dataset";
import { Pencil, Trash2, User, Users } from "lucide-react";
import { logger } from "@/utils/loglevel";
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
import { DatasetCreateForm } from "./CreateForm";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";

export const Component: FC = () => {
  const [openSheet, setOpenSheet] = useState(false);
  const refInput = useRef<HTMLInputElement>(null);
  const refInput2 = useRef<HTMLInputElement>(null);
  const refInput3 = useRef<HTMLInputElement>(null);
  const [sharedUserID, setSharedUserID] = useState<string>("");
  const [sharedQueueID, setSharedQueueID] = useState<string>("");
  const [datasetNewName, setDatasetNewName] = useState<string>("");
  const data = useQuery({
    queryKey: ["data", "alldataset"],
    queryFn: () => apiAdminGetDataset(),
    select: (res) => res.data.data,
  });

  const { mutate: ShareDatasetWithUser } = useMutation({
    mutationFn: (datasetid: number) => shareDatasetwithUser(datasetid),
    onSuccess: async () => {
      await refetchDatasetList();
    },
  });
  const shareDatasetwithUser = async (datasetid: number) => {
    if (sharedUserID != "") {
      const userID = parseInt(sharedUserID, 10);
      if (!isNaN(userID)) {
        await apiAdminShareDatasetwithUser({
          datasetID: datasetid,
          userID: userID,
        }).then(() => {
          toast.success("共享成功");
        });
      } else {
        toast.error("请输入数字ID");
      }
    }
  };
  const { mutate: ShareDatasetWithQueue } = useMutation({
    mutationFn: (datasetid: number) => shareDatasetwithQueue(datasetid),
    onSuccess: async () => {
      await refetchDatasetList();
    },
  });
  const shareDatasetwithQueue = async (datasetid: number) => {
    if (sharedQueueID != "") {
      const queueID = parseInt(sharedQueueID, 10);
      if (!isNaN(queueID)) {
        await apiAdminShareDatasetwithQueue({
          datasetID: datasetid,
          queueID: queueID,
        }).then(() => {
          toast.success("共享成功");
        });
      } else {
        toast.error("请输入数字ID");
      }
    }
  };

  const { mutate: RenameDataset } = useMutation({
    mutationFn: (datasetid: number) => rename(datasetid),
    onSuccess: async () => {
      await refetchDatasetList();
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
    setSharedUserID(e.target.value);
  };

  const handleChange2 = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setSharedQueueID(e.target.value);
  };
  const handleChange3 = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setDatasetNewName(e.target.value);
  };
  const { mutate: deleteDataset } = useMutation({
    mutationFn: (req: number) => apiDatasetDelete(req),
    onSuccess: async () => {
      await refetchDatasetList();
      toast.success("数据集已删除");
    },
  });
  const queryClient = useQueryClient();

  const refetchDatasetList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["data", "alldataset"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

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
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
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
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="用户共享"
                    >
                      <User size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>共享数据集</AlertDialogTitle>
                    <AlertDialogDescription>
                      和用户共享数据集
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="user" className="text-right">
                        用户ID
                      </Label>
                      <Input
                        id="user"
                        type="number"
                        defaultValue=""
                        className="col-span-3"
                        ref={refInput}
                        onChange={handleChange}
                        required
                      />
                    </div>
                  </div>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="default"
                      onClick={() => ShareDatasetWithUser(row.original.id)}
                    >
                      共享
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="队列共享"
                    >
                      <Users size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>共享数据集</AlertDialogTitle>
                    <AlertDialogDescription>
                      和队列共享数据集
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="queue" className="text-right">
                        队列ID
                      </Label>
                      <Input
                        id="queue"
                        type="number"
                        defaultValue=""
                        className="col-span-3"
                        ref={refInput2}
                        onChange={handleChange2}
                        required
                      />
                    </div>
                  </div>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="default"
                      onClick={() => ShareDatasetWithQueue(row.original.id)}
                    >
                      共享
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="重命名"
                    >
                      <Pencil size={16} strokeWidth={2} />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>重命名</AlertDialogTitle>
                    <AlertDialogDescription>
                      重命名数据集
                    </AlertDialogDescription>
                  </AlertDialogHeader>
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
                        ref={refInput3}
                        onChange={handleChange3}
                        required
                      />
                    </div>
                  </div>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="default"
                      onClick={() => RenameDataset(row.original.id)}
                    >
                      确认
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          );
        },
      },
    ],
    [deleteDataset, ShareDatasetWithUser, ShareDatasetWithQueue, RenameDataset],
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
};
