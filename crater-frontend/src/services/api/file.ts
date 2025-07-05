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

import instance from '../axios'
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
  instance.get<IResponse<FileItem[] | undefined>>(`ss/files/${path.replace(/^\//, '')}`)
export const apiGetRWFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/rwfiles/${path.replace(/^\//, '')}`)
export const apiGetAdminFile = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin/files/${path.replace(/^\//, '')}`)

export const apiGetUserFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin/${path.replace(/^\//, '')}`)

export const apiGetQueueFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin/${path.replace(/^\//, '')}`)

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: 'MKCOL',
    url: `ss/${path.replace(/^\//, '')}`,
  })
}

export const apiFileDelete = (path: string) =>
  instance.delete<IResponse<string>>(`/ss/delete/${path.replace(/^\//, '')}`)

export const apiGetUserSpace = () =>
  instance.get<IResponse<UserSpace[] | undefined>>(`ss/userspace`)

export const apiGetQueueSpace = () =>
  instance.get<IResponse<QeueuSpace[] | undefined>>(`ss/queuespace`)

export const apiMoveFile = (req: MoveFile, path: string) =>
  instance.post<IResponse<MoveFile>>(`ss/move/${path.replace(/^\//, '')}`, req)

export const apiGetDatasetFiles = (datasetID: number, path: string) =>
  instance.get<IResponse<FileItem[]>>(
    path === '' ? `ss/dataset/${datasetID}` : `ss/dataset/${datasetID}/${path.replace(/^\//, '')}`
  )
