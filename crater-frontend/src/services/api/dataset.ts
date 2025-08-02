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
import { apiV1Delete, apiV1Get, apiV1Post } from '@/services/client'
import { IResponse } from '@/services/types'

import { IUserAttributes } from './admin/user'
import { IUserInfo } from './vcjob'

export interface Extra {
  tag: string[]
  weburl: string
  editable: boolean
}

export interface IDataset {
  id: number
  name: string
  url: string
  describe: string
  createdAt: string
  type: 'model' | 'dataset' | 'sharefile'
  extra: Extra
  userInfo: IUserInfo
}

export interface UserDataset {
  datasetID: number
  userIDs: number[]
}

export interface QueueDataset {
  queueIDs: number[]
  datasetID: number
}

export interface DatasetReq {
  describe: string
  name: string
  url: string
  type: string
  tags: string[]
  weburl: string
  ispublic: boolean
  editable: boolean
}

export interface DatasetRenameReq {
  datasetID: number
  name: string
}

export interface UserDatasetResp {
  id: number
  name: string
  isowner: boolean
}
export interface QueueDatasetGetResp {
  id: number
  name: string
}

export interface cancelSharedUserResp {
  datasetID: number
  userID: number
}
export interface cancelSharedQueueResp {
  datasetID: number
  queueID: number
}
export const apiGetDataset = () => apiV1Get<IResponse<IDataset[]>>(`/dataset/mydataset`)

//因为table表单的query必须要返回数组，实际上数组里只有一个数据集的数据
export const apiGetDatasetByID = (datasetID: number) =>
  apiV1Get<IResponse<IDataset[]>>(`/dataset/detail/${datasetID}`)

export const apiAdminGetDataset = () => apiV1Get<IResponse<IDataset[]>>(`/admin/dataset/alldataset`)

export const apiShareDatasetwithUser = (ud: UserDataset) =>
  apiV1Post<IResponse<string>>('/dataset/share/user', ud)

export const apiShareDatasetwithQueue = (qd: QueueDataset) =>
  apiV1Post<IResponse<string>>('/dataset/share/queue', qd)

export const apiDatasetCreate = (dataset: DatasetReq) =>
  apiV1Post<IResponse<string>>('/dataset/create', dataset)

export const apiDatasetDelete = (datasetID: number) =>
  apiV1Delete<IResponse<string>>(`/dataset/delete/${datasetID}`)

export const apiDatasetRename = (drr: DatasetRenameReq) =>
  apiV1Post<IResponse<string>>('/dataset/rename', drr)

export const apiAdminShareDatasetwithUser = (ud: UserDataset) =>
  apiV1Post<IResponse<string>>('/admin/dataset/share/user', ud)

export const apiAdminShareDatasetwithQueue = (qd: QueueDataset) =>
  apiV1Post<IResponse<string>>('/admin/dataset/share/queue', qd)

export const apiListUsersNotInDataset = (datasetID: number) =>
  apiV1Get<IResponse<IUserAttributes[]>>(`/dataset/${datasetID}/usersNotIn`)

export const apiListQueuesNotInDataset = (datasetID: number) =>
  apiV1Get<IResponse<QueueDatasetGetResp[]>>(`/dataset/${datasetID}/queuesNotIn`)
export const apiListUsersInDataset = (datasetID: number) =>
  apiV1Get<IResponse<UserDatasetResp[]>>(`/dataset/${datasetID}/usersIn`)

export const apiListQueuesInDataset = (datasetID: number) =>
  apiV1Get<IResponse<QueueDatasetGetResp[]>>(`/dataset/${datasetID}/queuesIn`)

export const apiCancelShareWithUser = (CSU: cancelSharedUserResp) =>
  apiV1Post<IResponse<string>>(`/dataset/cancelshare/user`, CSU)

export const apiCancelShareWithQueue = (CSQ: cancelSharedQueueResp) =>
  apiV1Post<IResponse<string>>(`/dataset/cancelshare/queue`, CSQ)

export const apiAdminCancelShareWithUser = (CSU: cancelSharedUserResp) =>
  apiV1Post<IResponse<string>>(`/admin/dataset/cancelshare/user`, CSU)

export const apiAdminCancelShareWithQueue = (CSQ: cancelSharedQueueResp) =>
  apiV1Post<IResponse<string>>(`/admin/dataset/cancelshare/queue`, CSQ)
export const apiDatasetUpdate = (dataset: DatasetReq & { datasetId: number }) =>
  apiV1Post<IResponse<string>>('/dataset/update', dataset)
