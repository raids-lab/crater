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
  taints: IClusterNodeTaint[]
  capacity: V1ResourceList
  allocatable: V1ResourceList
  used: V1ResourceList
  workloads: number
  annotations: Record<string, string>
  kernelVersion?: string
  gpuDriver?: string
  address?: string
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
  // 管理员接口返回的字段
  userName?: string
  userID?: number
  userRealName?: string
  accountName?: string
  accountID?: number
  accountRealName?: string
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
  kernelVersion?: string
  capacity?: V1ResourceList
  allocatable?: V1ResourceList
  used?: V1ResourceList
  gpuDriver?: string
}

// GPU 信息接口定义
export interface IGPUDeviceInfo {
  resourceName: string
  label: string
  product: string
  vendorDomain: string
  count: number
  memory: string
  arch: string
  driver: string
  runtimeVersion: string
}

export interface IClusterNodeGPU {
  name: string
  haveGPU: boolean
  gpuCount: number
  gpuDevices: IGPUDeviceInfo[]
  gpuUtil: Record<string, number>
  // GPU ID 到作业名称列表的映射，如 { "0": ["job-1"], "1": ["job-2"] }
  relateJobs: Record<string, string[]>
  // 以下字段保留用于向后兼容（取第一个 GPU 设备的信息）
  gpuMemory: string
  gpuArch: string
  gpuDriver: string
  cudaVersion: string
  gpuProduct: string
}

export interface IClusterNodeLabel {
  key: string
  value: string
}

export interface IClusterNodeAnnotation {
  key: string
  value: string
}

export interface IClusterNodeTaint {
  key: string
  value: string
  effect: string
  reason?: string
  timeAdded?: string
}

export interface IClusterNodeMark {
  labels: IClusterNodeLabel[]
  annotations: IClusterNodeAnnotation[]
  taints: IClusterNodeTaint[]
}

export enum TaintEffect {
  NoSchedule = 'NoSchedule',
  PreferNoSchedule = 'PreferNoSchedule',
  NoExecute = 'NoExecute',
}

export const JoinTaint = (taint: IClusterNodeTaint) => `${taint.key}=${taint.value}:${taint.effect}`

export const apiGetNodes = () => apiV1Get<IResponse<INodeBriefInfo[]>>('nodes')

export const apiGetNodeDetail = (name: string) =>
  apiV1Get<IResponse<IClusterNodeDetail>>(`nodes/${name}`)

export const apiGetNodePods = (name: string) =>
  apiV1Get<IResponse<IClusterPodInfo[]>>(`nodes/${name}/pods`)

export const apiAdminGetNodePods = (name: string) =>
  apiV1Get<IResponse<IClusterPodInfo[]>>(`admin/nodes/${name}/pods`)

// 获取节点的 GPU 详情
export const apiGetNodeGPU = (name: string) =>
  apiV1Get<IResponse<IClusterNodeGPU>>(`nodes/${name}/gpu`)

// 改变节点的可调度状态
export const apichangeNodeScheduling = (name: string, body?: { reason?: string }) =>
  apiV1Put<IResponse<string>>(`nodes/${name}`, body)

export const apiAddNodeTaint = (nodeName: string, taintContent: IClusterNodeTaint) =>
  apiV1Post<IResponse<string>>(`admin/nodes/${nodeName}/taint`, taintContent)

export const apiDeleteNodeTaint = (nodeName: string, taintContent: IClusterNodeTaint) =>
  apiV1Delete<IResponse<string>>(`admin/nodes/${nodeName}/taint`, taintContent)

export const apiAddNodeLabel = (nodeName: string, labelContent: IClusterNodeLabel) =>
  apiV1Post<IResponse<string>>(`admin/nodes/${nodeName}/label`, labelContent)

export const apiDeleteNodeLabel = (nodeName: string, labelContent: IClusterNodeLabel) =>
  apiV1Delete<IResponse<string>>(`admin/nodes/${nodeName}/label`, labelContent)

export const apiAddNodeAnnotation = (nodeName: string, annotationContent: IClusterNodeAnnotation) =>
  apiV1Post<IResponse<string>>(`admin/nodes/${nodeName}/annotation`, annotationContent)

export const apiDeleteNodeAnnotation = (
  nodeName: string,
  annotationContent: IClusterNodeAnnotation
) => apiV1Delete<IResponse<string>>(`admin/nodes/${nodeName}/annotation`, annotationContent)

export const apiGetNodeMark = (name: string) =>
  apiV1Get<IResponse<IClusterNodeMark>>(`admin/nodes/${name}/mark`)
