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

// i18n-processed-v1.1.0 (no translatable strings)
import { AxiosResponse } from 'axios'
import { IResponse } from '@/services/types'
import {
  UserDataset,
  QueueDataset,
  cancelSharedUserResp,
  cancelSharedQueueResp,
} from '@/services/api/dataset'
import { SharedResourceTable } from './SharedResourceTable'

interface DatesetShareTableProps {
  resourceType: 'model' | 'dataset' | 'sharefile'
  apiShareDatasetwithUser: (ud: UserDataset) => Promise<AxiosResponse<IResponse<string>>>
  apiShareDatasetwithQueue: (qd: QueueDataset) => Promise<AxiosResponse<IResponse<string>>>
  apiCancelDatasetSharewithUser: (
    csu: cancelSharedUserResp
  ) => Promise<AxiosResponse<IResponse<string>>>
  apiCancelDatasetSharewithQueue: (
    csq: cancelSharedQueueResp
  ) => Promise<AxiosResponse<IResponse<string>>>
  apiDatasetDelete: (datasetID: number) => Promise<AxiosResponse<IResponse<string>>>
}

export function ModelShareTable(props: DatesetShareTableProps) {
  return <SharedResourceTable {...props} />
}
