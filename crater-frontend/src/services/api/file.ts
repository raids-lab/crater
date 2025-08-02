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
import { apiClient, apiDelete, apiGet, apiPost } from '@/services/client'
import { IResponse } from '@/services/types'

export interface FileItem {
  isdir: boolean
  modifytime: string
  name: string
  size: number
  sys?: never
}

export interface UserSpace {
  username: string
  space: string
}

export interface QeueuSpace {
  queuename: string
  space: string
}

export interface MoveFile {
  fileName: string
  dst: string
}

export const apiGetFiles = (path: string) =>
  apiGet<IResponse<FileItem[] | undefined>>(
    `ss/files/${encodeURIComponent(path.replace(/^\//, ''))}`
  )
export const apiGetRWFiles = (path: string) =>
  apiGet<IResponse<FileItem[] | undefined>>(`ss/rwfiles/${path.replace(/^\//, '')}`)
export const apiGetAdminFile = (path: string) =>
  apiGet<IResponse<FileItem[] | undefined>>(`ss/admin/files/${path.replace(/^\//, '')}`)

export const apiGetUserFiles = (path: string) =>
  apiGet<IResponse<FileItem[] | undefined>>(`ss/admin/${path.replace(/^\//, '')}`)

export const apiGetQueueFiles = (path: string) =>
  apiGet<IResponse<FileItem[] | undefined>>(`ss/admin/${path.replace(/^\//, '')}`)

export const apiMkdir = async (path: string) => {
  await apiClient('ss/' + path.replace(/^\//, ''), {
    method: 'MKCOL',
  })
}

export const apiFileDelete = (path: string) =>
  apiDelete<IResponse<string>>(`ss/delete/${path.replace(/^\//, '')}`)

export const apiGetUserSpace = () => apiGet<IResponse<UserSpace[] | undefined>>(`ss/userspace`)

export const apiGetQueueSpace = () => apiGet<IResponse<QeueuSpace[] | undefined>>(`ss/queuespace`)

export const apiMoveFile = (req: MoveFile, path: string) =>
  apiPost<IResponse<MoveFile>>(`ss/move/${path.replace(/^\//, '')}`, req)

export const apiGetDatasetFiles = (datasetID: number, path: string) =>
  apiGet<IResponse<FileItem[]>>(
    path === '' ? `ss/dataset/${datasetID}` : `ss/dataset/${datasetID}/${path.replace(/^\//, '')}`
  )
