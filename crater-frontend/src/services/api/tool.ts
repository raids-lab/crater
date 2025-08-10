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
import { Event as KubernetesEvent } from 'kubernetes-types/core/v1'
import { ContainerState, ContainerStateTerminated } from 'kubernetes-types/core/v1'

import { apiV1Delete, apiV1Get, apiV1Post } from '@/services/client'

import { IResponse } from '../types'

export type TerminatedState = ContainerStateTerminated

export interface ContainerInfo {
  name: string
  image: string
  state: ContainerState
  resources?: Record<string, string>
  restartCount: number
  isInitContainer: boolean
}

export interface ContainerStatusResponse {
  containers: ContainerInfo[]
}

export const apiGetPodContainers = (namespace?: string, podName?: string) =>
  apiV1Get<IResponse<ContainerStatusResponse>>(`namespaces/${namespace}/pods/${podName}/containers`)

export const apiGetPodEvents = (namespace?: string, podName?: string) =>
  apiV1Get<IResponse<KubernetesEvent[]>>(`namespaces/${namespace}/pods/${podName}/events`)

export interface PodContainerLogQueryReq {
  timestamps: boolean
  follow: boolean
  previous: boolean
  tailLines?: number
}

export const apiGetPodContainerLog = (
  namespace?: string,
  podName?: string,
  containerName?: string,
  query?: PodContainerLogQueryReq
) => {
  return apiV1Get<IResponse<string>>(
    `namespaces/${namespace}/pods/${podName}/containers/${containerName}/log`,
    {
      searchParams: { ...query },
    }
  )
}

export interface PodIngress {
  name: string
  port: number
  prefix: string
}

export interface PodIngressMgr {
  name: string
  port: number
}

export interface PodIngressesList {
  ingresses: PodIngress[]
}

export const apiGetPodIngresses = (namespace: string, podName: string) =>
  apiV1Get<IResponse<PodIngressesList>>(`namespaces/${namespace}/pods/${podName}/ingresses`)

export const apiCreatePodIngress = (
  namespace: string,
  podName: string,
  ingressMgr: PodIngressMgr
) =>
  apiV1Post<IResponse<PodIngress>>(`namespaces/${namespace}/pods/${podName}/ingresses`, ingressMgr)

export const apiDeletePodIngress = (
  namespace: string,
  podName: string,
  ingressMgr: PodIngressMgr
) =>
  apiV1Delete<IResponse<string>>(`namespaces/${namespace}/pods/${podName}/ingresses`, {
    data: ingressMgr,
  })

export interface PodNodeport {
  name: string
  containerPort: number
  address: string
  nodePort: number
}

export interface PodNodeportMgr {
  name: string
  containerPort: number
}

export interface PodNodeportsList {
  nodeports: PodNodeport[]
}

export const apiGetPodNodeports = (namespace: string, podName: string) =>
  apiV1Get<IResponse<PodNodeportsList>>(`namespaces/${namespace}/pods/${podName}/nodeports`)

export const apiCreatePodNodeport = (
  namespace: string,
  podName: string,
  nodeportMgr: PodNodeportMgr
) =>
  apiV1Post<IResponse<PodIngress>>(`namespaces/${namespace}/pods/${podName}/nodeports`, nodeportMgr)

export const apiDeletePodNodeport = (
  namespace: string,
  podName: string,
  nodeportMgr: PodNodeportMgr
) =>
  apiV1Delete<IResponse<string>>(`namespaces/${namespace}/pods/${podName}/nodeports`, {
    data: nodeportMgr,
  })
