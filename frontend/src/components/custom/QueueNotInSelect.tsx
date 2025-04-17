import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiListQueuesNotInDataset,
  QueueDataset,
} from "@/services/api/dataset";
import SelectBox from "./SelectBox";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { toast } from "sonner";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";

interface QueueSelectProps {
  datasetId: number;
  apiShareDatasetwithQueue: (
    qd: QueueDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
}

export function ShareDatasetToQueueDialog({
  datasetId,
  apiShareDatasetwithQueue,
}: QueueSelectProps) {
  const queryClient = useQueryClient();
  const [queueIds, setQueueIds] = useState<string[]>([]);

  const { data: queueList } = useQuery({
    queryKey: ["dataset", "queueOutList", { datasetId }],
    queryFn: () => apiListQueuesNotInDataset(datasetId),
    select: (res) => {
      return res.data.data.map((queue) => {
        return {
          value: queue.id.toString(),
          label: queue.name,
        };
      });
    },
  });

  const { mutate: shareWithQueue } = useMutation({
    mutationFn: (datasetId: number) =>
      apiShareDatasetwithQueue({
        datasetID: datasetId,
        queueIDs: queueIds.map((queue) => parseInt(queue)),
      }),
    onSuccess: () => {
      toast.success("共享成功");
      void queryClient.invalidateQueries({
        queryKey: ["data", "queuedataset", datasetId],
      });
      setQueueIds([]);
    },
  });

  return (
    <>
      <div className="w-full">
        <SelectBox
          options={queueList ?? []}
          value={queueIds}
          inputPlaceholder="搜索账户"
          placeholder="选择账户"
          onChange={setQueueIds}
        />
      </div>
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">取消</Button>
        </DialogClose>
        <DialogClose asChild>
          <Button variant="default" onClick={() => shareWithQueue(datasetId)}>
            共享
          </Button>
        </DialogClose>
      </DialogFooter>
    </>
  );
}
