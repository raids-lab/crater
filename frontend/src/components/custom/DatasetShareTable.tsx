import { useEffect, useState } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  TableHead,
  TableRow,
  TableHeader,
  TableCell,
  TableBody,
  Table,
} from "@/components/ui/table";
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
} from "@/services/api/dataset";
import { User, Users } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import { QueueNotInSelect } from "@/components/custom/QueueNotInSelect";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
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
}
export function DatasetShareTable({
  apiShareDatasetwithUser,
  apiShareDatasetwithQueue,
}: DatesetShareTableProps) {
  const { id } = useParams<{ id: string; name: string }>();

  const datasetId = id ? parseInt(id, 10) : 0;
  const setBreadcrumb = useBreadcrumb();
  const userDatasetData = useQuery({
    queryKey: ["data", "admin", "userdataset", { datasetId }],
    queryFn: () => apiListUsersInDataset(datasetId),
    select: (res) => res.data.data,
  });
  const queueDatasetData = useQuery({
    queryKey: ["data", "admin", "queuedataset", { datasetId }],
    queryFn: () => apiListQueuesInDataset(datasetId),
    select: (res) => res.data.data,
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
        queryKey: ["data", "mydataset"],
      });
      toast.success("共享成功");
      setQueueIds([]);
    },
  });

  const queryClient = useQueryClient();

  const [queueIds, setQueueIds] = useState<number[]>([]);
  const onChangeQueue = (newValue: OnChangeValue<QueueOption, true>) => {
    const queueIds: number[] = newValue.map((queue) => queue.id);
    setQueueIds(queueIds);
  };

  return (
    <>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              数据集共享用户
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <Dialog>
              <DialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-red-700"
                    title="用户共享"
                  >
                    <User size={32} strokeWidth={1} />
                  </Button>
                </div>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>分享至用户</DialogTitle>
                  <DialogDescription>和指定的用户共享数据集</DialogDescription>
                </DialogHeader>
                <ShareDatasetToUserDialog
                  datasetId={datasetId}
                  apiShareDatasetwithUser={apiShareDatasetwithUser}
                />
              </DialogContent>
            </Dialog>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-7">
          <Table className="w-full">
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs">序号</TableHead>
                <TableHead className="text-xs">用户名称</TableHead>
                <TableHead className="text-xs">是否为创建者</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <a>{}</a>
              {userDatasetData &&
                Array.isArray(userDatasetData.data) &&
                userDatasetData.data.map((userdata, index) => (
                  <TableRow key={userdata.id || index}>
                    <TableCell>{index + 1}</TableCell>
                    <TableCell>{userdata.name}</TableCell>
                    <TableCell>{userdata.isowner ? "是" : "否"}</TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              数据集共享账户
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <Dialog>
              <DialogTrigger asChild>
                <div>
                  <Button
                    variant="outline"
                    className="h-8 w-8 p-0 hover:text-red-700"
                    title="账户共享"
                  >
                    <Users size={32} strokeWidth={1} />
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
                    <QueueNotInSelect id={datasetId} onChange={onChangeQueue} />
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
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-7">
          <Table className="w-full">
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs">序号</TableHead>
                <TableHead className="text-xs">账户名称</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <a>{}</a>
              {queueDatasetData &&
                Array.isArray(queueDatasetData.data) &&
                queueDatasetData.data.map((queuedata, index) => (
                  <TableRow key={queuedata.id || index}>
                    <TableCell>{index + 1}</TableCell>
                    <TableCell>{queuedata.name}</TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </>
  );
}
