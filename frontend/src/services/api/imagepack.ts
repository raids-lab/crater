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

import instance, { VERSION } from '@/services/axios'
import { IResponse } from '@/services/types'
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from '@radix-ui/react-icons'
import { IUserInfo, JobType } from './vcjob'
import { IUserAttributes } from './admin/user'
import { Visibility } from '@/components/badge/VisibilityBadge'

export type ListKanikoResponse = {
  kanikoList: KanikoInfoResponse[]
}

export type KanikoInfoResponse = {
  ID: number
  imageLink: string
  status: ImagePackStatus
  buildSource: ImagePackSource
  createdAt: string
  size: number
  dockerfile: string
  description: string
  podName: string
  podNameSpace: string
  userInfo: IUserInfo
  tags: string[]
  imagepackName: string
}

export type ListImageResponse = {
  imageList: ImageInfoResponse[]
}

export type ImageInfoResponse = {
  ID: number
  imageLink: string
  description: string
  status: string
  createdAt: string
  isPublic: boolean
  taskType: JobType
  userInfo: IUserInfo
  tags: string[]
  imageBuildSource: imagepackSourceTypeValue
  imagepackName: string
  imageShareStatus: Visibility
}

export type ProjectCredentialResponse = {
  name: string
  password: string
  exist: boolean
}

export type ProjectDetailResponse = {
  quota: number
  used: number
  total: number
  project: string
}

export type ImageUsers = {
  nickname: string
  name: string
  id: number
}
export type ImageAccounts = {
  name: string
  id: number
}
export type ImageShareObjectsResponse = {
  userList: ImageUsers[]
  accountList: ImageAccounts[]
}

export type SearchUngrantedUserResponse = {
  userList: IUserAttributes[]
}

export type GetUngrantedAccountsResponse = {
  accountList: ImageAccounts[]
}

export const getHeader = (key: string): string => {
  switch (key) {
    case 'image':
      return '镜像'
    case 'userInfo':
      return '提交者'
    case 'status':
      return '状态'
    case 'createdAt':
      return '创建时间'
    case 'taskType':
      return '类型'
    case 'imageType':
      return '镜像类型'
    case 'imageShareStatus':
      return '可见性'
    case 'size':
      return '大小'
    default:
      return key
  }
}

export type ImagePackStatus = 'Initial' | 'Pending' | 'Running' | 'Finished' | 'Failed' | ''

export enum ImagePackSource {
  Dockerfile = 'Dockerfile',
  PipApt = 'PipApt',
  Snapshot = 'Snapshot',
  EnvdAdvanced = 'EnvdAdvanced',
  EnvdRaw = 'EnvdRaw',
}

export const imagepackStatuses: {
  value: ImagePackStatus
  label: string
  icon: React.ComponentType<{ className?: string }>
}[] = [
  {
    value: 'Initial',
    label: '检查中',
    icon: CircleIcon,
  },
  {
    value: 'Pending',
    label: '等待中',
    icon: ClockIcon,
  },
  {
    value: 'Running',
    label: '运行中',
    icon: StopwatchIcon,
  },
  {
    value: 'Finished',
    label: '成功',
    icon: CheckCircledIcon,
  },
  {
    value: 'Failed',
    label: '失败',
    icon: CrossCircledIcon,
  },
]

export type imagepackSourceTypeValue = 1 | 2

export const imagepackSourceType: {
  value: imagepackSourceTypeValue
  label: string
}[] = [
  {
    value: 1,
    label: '镜像制作',
  },
  {
    value: 2,
    label: '镜像上传',
  },
]

export interface KanikoCreate {
  description: string
  image: string
  packages: string
  requirements: string
  name: string
  tag: string
  tags: string[]
  template: string
  buildSource: ImagePackSource
  archs: string[]
}

export interface DockerfileCreate {
  description: string
  dockerfile: string
  name: string
  tag: string
  tags: string[]
  template: string
  buildSource: ImagePackSource
  archs: string[]
}

export interface EnvdCreate {
  description: string
  envd: string
  name: string
  tag: string
  python: string
  base: string
  tags: string[]
  template: string
  buildSource: ImagePackSource
}

export interface ImageUpload {
  imageLink: string
  imageName: string
  imageTag: string
  description: string
  taskType: JobType
  tags: string[]
}

export interface UpdateDescription {
  id: number
  description: string
}

export interface UpdateImageTag {
  id: number
  tags: string[]
}

export interface UpdateTaskType {
  id: number
  taskType: JobType
}

export interface ImageLinkPair {
  id: number
  imageLink: string
  description: string
  creator: IUserInfo
}

export interface ImageLinkPairs {
  linkPairs: ImageLinkPair[]
}

export interface ImageIDList {
  idList: number[]
}

export interface HarborIPData {
  ip: string
}

export interface GetImageShare {
  imageID: number
}

export interface AddImageShare {
  idList: number[]
  imageID: number
  type: string // user: share with user, account: share with account
}

export interface DeleteImageShare {
  id: number
  imageID: number
  type: string // user: cancel share with user, account: cancel share with account
}

export interface SearchUngrantedUsers {
  imageID: number
  // name: string;
}

export interface GetUngrantedAccounts {
  imageID: number
}

export const ImageTaskType = {
  JupyterTask: 1, // Jupyter交互式任务
  WebIDETask: 2, // Web IDE任务
  TensorflowTask: 3, // Tensorflow任务
  PytorchTask: 4, // Pytorch任务
  RayTask: 5, // Ray任务
  DeepSpeedTask: 6, // DeepSpeed任务
  OpenMPITask: 7, // OpenMPI任务
  UserDefineTask: 8, // 用户自定义任务
}

export const ImageDefaultTags = [
  { value: 'CUDA' },
  { value: 'Pytorch' },
  { value: 'Jupyter' },
  { value: 'Python' },
  { value: 'vLLM' },
]

export const ImageDefaultArchs = [
  { value: 'linux/amd64' },
  { value: 'linux/amd64/v2' },
  { value: 'linux/amd64/v3' },
  { value: 'linux/386' },
  { value: 'linux/arm64' },
  { value: 'linux/arm/v7' },
  { value: 'linux/arm/v6' },
]

export async function FetchAllUniqueImageTagObjects(): Promise<{ value: string }[]> {
  const res = await apiUserListImage()
  const imageList = res.data.data.imageList ?? []
  // 用 ImageDefaultTags 初始化 Set
  const tagSet = new Set<string>(ImageDefaultTags.map((item) => item.value))
  const lowerCaseTagSet = new Set<string>(ImageDefaultTags.map((item) => item.value.toLowerCase()))

  // 统计所有镜像tags出现频次（忽略大小写）
  const tagCount: Record<string, { count: number; raw: string }> = {}
  imageList.forEach((image) => {
    if (Array.isArray(image.tags)) {
      image.tags.forEach((tag) => {
        const lower = tag.toLowerCase()
        if (lowerCaseTagSet.has(lower)) return // 跳过已在默认标签中的
        if (!tagCount[lower]) {
          tagCount[lower] = { count: 1, raw: tag }
        } else {
          tagCount[lower].count += 1
        }
      })
    }
  })

  // 取出现频次最高的5个tag
  const topTags = Object.values(tagCount)
    .sort((a, b) => b.count - a.count)
    .slice(0, 5)
    .map((item) => item.raw)

  // 合并到set
  topTags.forEach((tag) => {
    tagSet.add(tag)
  })

  // 返回与 ImageDefaultTags 相同格式
  return Array.from(tagSet).map((tag) => ({ value: tag }))
}

export const imageLinkRegex =
  /^[a-zA-Z0-9.-]+(\/[a-zA-Z0-9_.-]+)+:[a-zA-Z0-9_](?:[a-zA-Z0-9_.-]*[a-zA-Z0-9_])?$/

export const imageNameRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*$/
export const imageTagRegex = /^(?=.{1,128}$)(?![-.])([a-zA-Z0-9_.+-]*)(?<![-.])$/

export const dockerfileImageLinkRegex = /^FROM\s+(?:--platform=[^\s]+\s+)?([^\s#]+)/gim

export function parseImageLink(imageLink: string) {
  const parts = imageLink.split(':')
  if (parts.length === 2) {
    const nameParts = parts[0].split('/')
    const imageName = nameParts.length > 1 ? nameParts[nameParts.length - 1] : nameParts[0]
    const imageTag = parts[1]
    return { imageName, imageTag }
  }
  return { imageName: '', imageTag: '' }
}

export const apiUserListKaniko = () =>
  instance.get<IResponse<ListKanikoResponse>>(`${VERSION}/images/kaniko`)

export const apiUserCreateKaniko = async (imagepack: KanikoCreate) => {
  const response = await instance.post<IResponse<string>>(VERSION + '/images/kaniko', imagepack)
  return response.data
}

export const apiUserCreateByDockerfile = async (imageDockerfile: DockerfileCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + '/images/dockerfile',
    imageDockerfile
  )
  return response.data
}

export const apiUserCreateByEnvd = async (envdInfo: EnvdCreate) => {
  const response = await instance.post<IResponse<string>>(VERSION + '/images/envd', envdInfo)
  return response.data
}

export const apiUserDeleteKaniko = (id: number) =>
  instance.delete<IResponse<string>>(VERSION + `/images/kaniko/${id}`)

export const apiUserDeleteKanikoList = (idList: number[]) =>
  instance.post<IResponse<string>>(`${VERSION}/images/deletekaniko`, {
    idList,
  })

export const apiUserGetKaniko = (name: string) =>
  instance.get<IResponse<KanikoInfoResponse>>(`${VERSION}/images/getbyname?name=${name}`)

export const apiUserListImage = () =>
  instance.get<IResponse<ListImageResponse>>(`${VERSION}/images/image`)

export const apiUserChangeImagePublicStatus = (id: number) =>
  instance.post<IResponse<string>>(VERSION + `/images/change/${id}`)

export const apiUserUploadImage = async (imageupload: ImageUpload) => {
  const response = await instance.post<IResponse<string>>(VERSION + '/images/image', imageupload)
  return response.data
}

export const apiUserDeleteImage = (id: number) =>
  instance.delete<IResponse<string>>(VERSION + `/images/image/${id}`)

export const apiUserDeleteImageList = (idList: number[]) =>
  instance.post<IResponse<string>>(`${VERSION}/images/deleteimage`, { idList })

export const apiUserGetCredential = () =>
  instance.post<IResponse<ProjectCredentialResponse>>(`${VERSION}/images/credential`)

export const apiUserGetQuota = () =>
  instance.get<IResponse<ProjectDetailResponse>>(`${VERSION}/images/quota`)

export const apiUserChangeImageDescription = (data: UpdateDescription) =>
  instance.post<IResponse<string>>(`${VERSION}/images/description`, data)

export const apiUserChangeImageTaskType = (data: UpdateTaskType) =>
  instance.post<IResponse<string>>(`${VERSION}/images/type`, data)

export const apiUserCheckImageValid = (linkPairs: ImageLinkPairs) =>
  instance.post<IResponse<ImageLinkPairs>>(`${VERSION}/images/valid`, linkPairs)

export const apiGetHarborIP = () =>
  instance.get<IResponse<HarborIPData>>(`${VERSION}/images/harbor`)

export const apiUserUpdateImageTags = (data: UpdateImageTag) =>
  instance.post<IResponse<string>>(`${VERSION}/images/tags`, data)

export const apiUserGetImageTemplate = (name: string) =>
  instance.get<IResponse<string>>(`${VERSION}/images/template?name=${name}`)

export const apiUserGetImageShareObjects = (data: GetImageShare) =>
  instance.get<IResponse<ImageShareObjectsResponse>>(`${VERSION}/images/share`, {
    params: data,
  })

export const apiUserAddImageShare = (data: AddImageShare) =>
  instance.post<IResponse<string>>(`${VERSION}/images/share`, data)

export const apiUserDeleteImageShare = (data: DeleteImageShare) =>
  instance.delete<IResponse<string>>(`${VERSION}/images/share`, { data })

export const apiUserSearchUser = (data: SearchUngrantedUsers) =>
  instance.get<IResponse<SearchUngrantedUserResponse>>(`${VERSION}/images/user`, {
    params: data,
  })

export const apiUserGetUngrantedAccounts = (data: GetUngrantedAccounts) =>
  instance.get<IResponse<GetUngrantedAccountsResponse>>(`${VERSION}/images/account`, {
    params: data,
  })
