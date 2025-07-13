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

export enum NodeType {
  Hygon = 'hygon',
  Shenwei = 'shenwei',
  Yitian = 'yitian',
}

interface BriefResource {
  cpu: string
  memory: string
  gpu: string
}

export interface IClusterNodeInfo {
  type: NodeType
  name: string
  role: string
  labels: Record<string, string>
  isReady: string
  taint: string
  capacity: BriefResource
  allocated: BriefResource
  podCount: number
}

export interface IOwnerReference {
  apiVersion: string
  kind: string
  name: string
  uid: string
}

export interface IClusterPodInfo {
  // from backend
  name: string
  namespace: string
  ownerReference: IOwnerReference[]
  ip: string
  createTime: string
  status: string
  resources: Record<string, string>
  locked: boolean
  permanentLocked: boolean
  lockedTimestamp?: string
  // added by frontend
  type?: string
}

export interface IClusterNodeDetail {
  name: string
  role: string
  isReady: string
  time: string
  address: string
  os: string
  osVersion: string
  arch: string
  kubeletVersion: string
  containerRuntimeVersion: string
}

// GPU 信息接口定义
export interface IClusterNodeGPU {
  name: string
  haveGPU: boolean
  gpuCount: number
  gpuUtil: Record<string, number>
  relateJobs: string[]
  gpuMemory: string
  gpuArch: string
  gpuDriver: string
  cudaVersion: string
  gpuProduct: string
}
export interface IClusterNodeTaint {
  name: string
  taint: string
}
export const apiGetNodes = () => instance.get<IResponse<IClusterNodeInfo[]>>(VERSION + '/nodes')

export const apiGetNodeDetail = (name: string) =>
  instance.get<IResponse<IClusterNodeDetail>>(VERSION + `/nodes/${name}`)

export const apiGetNodePods = (name: string) =>
  instance.get<IResponse<IClusterPodInfo[]>>(VERSION + `/nodes/${name}/pods`)

// 获取节点的 GPU 详情
export const apiGetNodeGPU = (name: string) =>
  instance.get<IResponse<IClusterNodeGPU>>(VERSION + `/nodes/${name}/gpu`)
// 改变节点的可调度状态
export const apichangeNodeScheduling = (name: string) =>
  instance.put<IResponse<string>>(VERSION + `/nodes/${name}`)
export const apiAddNodeTaint = (nodetaint: IClusterNodeTaint) =>
  instance.post<IResponse<string>>(VERSION + `/nodes/${nodetaint.name}/taint`, nodetaint)
export const apiDeleteNodeTaint = (nodetaint: IClusterNodeTaint) =>
  instance.delete<IResponse<string>>(VERSION + `/nodes/${nodetaint.name}/taint`, {
    data: nodetaint,
  })
