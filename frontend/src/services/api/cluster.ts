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
import { V1OwnerReference } from '@kubernetes/client-node'

import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'
import { IResponse } from '@/services/types'

import { V1ResourceList } from '@/utils/resource'

export enum NodeRole {
  ControlPlane = 'control-plane',
  Worker = 'worker',
  Virtual = 'virtual',
}

export enum NodeStatus {
  Ready = 'Ready',
  NotReady = 'NotReady',
  Unschedulable = 'Unschedulable',
  Occupied = 'Occupied',
}

export interface INodeBriefInfo {
  name: string
  role: NodeRole
  arch: string
  status: NodeStatus
  vendor: string
  taints: string[]
  capacity: V1ResourceList
  allocatable: V1ResourceList
  used: V1ResourceList
  workloads: number
}
export interface IClusterPodInfo {
  // from backend
  name: string
  namespace: string
  ownerReference: V1OwnerReference[]
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
export const apiGetNodes = () => apiV1Get<IResponse<INodeBriefInfo[]>>('nodes')

export const apiGetNodeDetail = (name: string) =>
  apiV1Get<IResponse<IClusterNodeDetail>>(`nodes/${name}`)

export const apiGetNodePods = (name: string) =>
  apiV1Get<IResponse<IClusterPodInfo[]>>(`nodes/${name}/pods`)

// 获取节点的 GPU 详情
export const apiGetNodeGPU = (name: string) =>
  apiV1Get<IResponse<IClusterNodeGPU>>(`nodes/${name}/gpu`)

// 改变节点的可调度状态
export const apichangeNodeScheduling = (name: string) =>
  apiV1Put<IResponse<string>>(`nodes/${name}`)

export const apiAddNodeTaint = (nodetaint: IClusterNodeTaint) =>
  apiV1Post<IResponse<string>>(`nodes/${nodetaint.name}/taint`, nodetaint)

export const apiDeleteNodeTaint = (nodetaint: IClusterNodeTaint) =>
  apiV1Delete<IResponse<string>>(`nodes/${nodetaint.name}/taint`, nodetaint)
