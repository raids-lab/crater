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
import { apiGet } from '@/services/client'
import { IResponse } from '@/services/types'

// 集群存储服务的基础URL
const clusterStorageBaseURL = 'https://crater.act.buaa.edu.cn/api'

// 创建集群存储服务的客户端
const clusterStorageClient = {
  get: async <T>(path: string) => {
    const response = await fetch(`${clusterStorageBaseURL}/${path}`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`,
        'Content-Type': 'application/json',
      },
    })
    return response.json() as Promise<T>
  },
  delete: async <T>(path: string) => {
    const response = await fetch(`${clusterStorageBaseURL}/${path}`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`,
        'Content-Type': 'application/json',
      },
    })
    return response.json() as Promise<T>
  },
  post: async <T>(path: string, data: unknown) => {
    const response = await fetch(`${clusterStorageBaseURL}/${path}`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    })
    return response.json() as Promise<T>
  },
  mkcol: async (path: string) => {
    await fetch(`${clusterStorageBaseURL}/${path}`, {
      method: 'MKCOL',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`,
        'Content-Type': 'application/json',
      },
    })
  },
}

export interface FileItem {
  isdir: boolean
  modifytime: string
  name: string
  size: number
  sys?: never
}

export interface MoveFile {
  fileName: string
  dst: string
}

export const apiGetFiles = (path: string) =>
  clusterStorageClient.get<IResponse<FileItem[] | undefined>>(
    `ss/files/${encodeURIComponent(path.replace(/^\//, ''))}`
  )

export const apiGetRWFiles = (path: string) =>
  clusterStorageClient.get<IResponse<FileItem[] | undefined>>(
    `ss/rwfiles/${path.replace(/^\//, '')}`
  )

export const apiGetAdminFiles = (path: string) =>
  clusterStorageClient.get<IResponse<FileItem[] | undefined>>(
    `ss/admin/files/${path.replace(/^\//, '')}`
  )

export const apiMkdir = async (path: string) => {
  await clusterStorageClient.mkcol('ss/' + path.replace(/^\//, ''))
}

export const apiFileDelete = (path: string) =>
  clusterStorageClient.delete<IResponse<string>>(`ss/delete/${path.replace(/^\//, '')}`)

export const apiMoveFile = (req: MoveFile, path: string) =>
  clusterStorageClient.post<IResponse<MoveFile>>(`ss/move/${path.replace(/^\//, '')}`, req)

export const apiGetDatasetFiles = (datasetID: number, path: string) =>
  clusterStorageClient.get<IResponse<FileItem[]>>(
    path === '' ? `ss/dataset/${datasetID}` : `ss/dataset/${datasetID}/${path.replace(/^\//, '')}`
  )

export interface DirectorySize {
  path: string
  size: number
  unit: string
  formatted: string
}

export const apiGetDirectorySize = (path: string) =>
  apiGet<IResponse<DirectorySize>>(`v1/storage/dirsize/${path.replace(/^\//, '')}`)

export interface MyQuota {
  space_quota: number
  space_quota_formatted: string
}

export const apiGetMyQuota = () => apiGet<IResponse<MyQuota>>('v1/storage/my-quota')
