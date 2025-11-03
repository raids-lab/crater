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
import { apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { IResponse } from '../types'

export enum LabelType {
  NVIDIA = 1,
}

export interface LabelInfo {
  id: number
  label: string
  name: string
  resource: string
  type: LabelType
  count: number
  priority: number
}

export const apiNodeLabelsList = () => {
  return apiV1Get<IResponse<LabelInfo[]>>('labels')
}

export const apiNodeLabelsUpdate = (id: number, name: string, priority: number) => {
  return apiV1Put<IResponse<LabelInfo>>(`admin/labels/${id}`, {
    name,
    priority,
  })
}

export const apiNodeLabelsNvidiaSync = () => {
  return apiV1Post<IResponse<never>>('admin/labels/sync/nvidia')
}
