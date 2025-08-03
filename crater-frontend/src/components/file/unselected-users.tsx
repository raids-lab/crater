/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// i18n-processed-v1.1.0
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { DialogClose, DialogFooter } from '@/components/ui/dialog'

import { UserDataset, apiListUsersNotInDataset } from '@/services/api/dataset'
import { IResponse } from '@/services/types'

import SelectBox from '../custom/SelectBox'

interface UserSelectProps {
  datasetId: number
  apiShareDatasetwithUser: (userDataset: UserDataset) => Promise<IResponse<string>>
}

export default function ShareWithUsers({ datasetId, apiShareDatasetwithUser }: UserSelectProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [users, setUsers] = useState<string[]>([])

  const { data: userList } = useQuery({
    queryKey: ['dataset', 'userOutList', { datasetId }],
    queryFn: () => apiListUsersNotInDataset(datasetId),
    select: (res) => {
      return res.data.map((user) => {
        return {
          value: user.id.toString(),
          label: user.nickname,
          labelNote: user.name,
        }
      })
    },
  })

  const { mutate: shareWithUser } = useMutation({
    mutationFn: (datasetId: number) =>
      apiShareDatasetwithUser({
        datasetID: datasetId,
        userIDs: users.map((user) => parseInt(user)),
      }),
    onSuccess: () => {
      toast.success(t('shareDatasetToUserDialog.toastSuccess'))
      void queryClient.invalidateQueries({
        queryKey: ['data', 'userdataset', datasetId],
      })
    },
  })

  return (
    <>
      <div className="w-full">
        <SelectBox
          options={userList ?? []}
          value={users}
          inputPlaceholder={t('shareDatasetToUserDialog.selectBox.searchPlaceholder')}
          placeholder={t('shareDatasetToUserDialog.selectBox.selectPlaceholder')}
          onChange={setUsers}
        />
      </div>
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">{t('shareDatasetToUserDialog.cancelButton')}</Button>
        </DialogClose>
        <DialogClose asChild>
          <Button variant="default" onClick={() => shareWithUser(datasetId)}>
            {t('shareDatasetToUserDialog.shareButton')}
          </Button>
        </DialogClose>
      </DialogFooter>
    </>
  )
}
