// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
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
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [users, setUsers] = useState<string[]>([]);

  const { data: userList } = useQuery({
    queryKey: ["dataset", "userOutList", { datasetId }],
    queryFn: () => apiListUsersNotInDataset(datasetId),
    select: (res) => {
      return res.data.data.map((user) => {
        return {
          value: user.id.toString(),
          label: user.nickname,
          labelNote: user.name,
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
      toast.success(t("shareDatasetToUserDialog.toastSuccess"));
      void queryClient.invalidateQueries({
        queryKey: ["data", "userdataset", datasetId],
      });
    },
  });

  return (
    <>
      <div className="w-full">
        <SelectBox
          options={userList ?? []}
          value={users}
          inputPlaceholder={t(
            "shareDatasetToUserDialog.selectBox.searchPlaceholder",
          )}
          placeholder={t(
            "shareDatasetToUserDialog.selectBox.selectPlaceholder",
          )}
          onChange={setUsers}
        />
      </div>
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">
            {t("shareDatasetToUserDialog.cancelButton")}
          </Button>
        </DialogClose>
        <DialogClose asChild>
          <Button variant="default" onClick={() => shareWithUser(datasetId)}>
            {t("shareDatasetToUserDialog.shareButton")}
          </Button>
        </DialogClose>
      </DialogFooter>
    </>
  );
}
