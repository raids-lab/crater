import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiListUsersNotInDataset, UserDataset } from "@/services/api/dataset";
import SelectBox from "./SelectBox";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { toast } from "sonner";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";

interface UserSelectProps {
  datasetId: number;
  apiShareDatasetwithUser: (
    userDataset: UserDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
}

export function ShareDatasetToUserDialog({
  datasetId,
  apiShareDatasetwithUser,
}: UserSelectProps) {
  const queryClient = useQueryClient();
  const [users, setUsers] = useState<string[]>([]);

  const { data: userList } = useQuery({
    queryKey: ["dataset", "userOutList", { datasetId }],
    queryFn: () => apiListUsersNotInDataset(datasetId),
    select: (res) => {
      return res.data.data.map((user) => {
        return {
          value: user.id.toString(),
          label: user.name,
        };
      });
    },
  });

  const { mutate: shareWithUser } = useMutation({
    mutationFn: (datasetId: number) =>
      apiShareDatasetwithUser({
        datasetID: datasetId,
        userIDs: users.map((user) => parseInt(user)),
      }),
    onSuccess: () => {
      toast.success("共享成功");
      void queryClient.invalidateQueries({
        queryKey: ["data", "userdataset", datasetId],
      });
    },
  });

  return (
    <>
      <div className="grid grid-cols-4 gap-4">
        <SelectBox
          className="col-span-4"
          options={userList ?? []}
          value={users}
          inputPlaceholder="搜索用户"
          placeholder="选择用户"
          onChange={setUsers}
        />
      </div>
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">取消</Button>
        </DialogClose>
        <DialogClose asChild>
          <Button variant="default" onClick={() => shareWithUser(datasetId)}>
            共享
          </Button>
        </DialogClose>
      </DialogFooter>
    </>
  );
}
