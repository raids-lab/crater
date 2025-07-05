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

import { showErrorToast } from './toast'

export type K8sResources = Record<string, string> | undefined

export type Resources = {
  cpu?: number
  memory?: number
  others?: Record<string, string>
}

export const convertKResourceToResource = (key: string, value?: string): number | undefined => {
  // key is cpu or memory or others
  if (!value) {
    return undefined
  }
  switch (key) {
    case 'cpu':
      // 100m -> 0.1
      if (value.includes('m')) {
        return parseFloat(value.replace('m', '')) / 1000
      } else {
        return parseFloat(value)
      }
    case 'memory':
      // "Ki", "Mi", "Gi" "Ti" => "Gi"
      if (value.includes('Ki')) {
        return parseInt(value.replace('Ki', '')) / 1024 / 1024
      } else if (value.includes('Mi')) {
        return parseInt(value.replace('Mi', '')) / 1024
      } else if (value.includes('Gi')) {
        return parseInt(value.replace('Gi', ''))
      } else if (value.includes('Ti')) {
        return parseInt(value.replace('Ti', '')) * 1024
      } else {
        showErrorToast(`Invalid memory ${value} with unrecognized unit`)
        return 0
      }
    default:
      return parseInt(value)
  }
}

export const convertResourceToKResource = (key: string, value?: number): string => {
  // key is cpu or memory or others
  switch (key) {
    case 'cpu':
      return `${value}`
    case 'memory':
      // "Ki", "Mi", "Gi" "Ti" => "Gi"
      return `${value}Gi`
    default:
      return `${value}`
  }
}

export const convertToResources = (resources?: K8sResources): Resources => {
  // if resource.cpu is string, parse to int
  // if resource.cpu is number, keep it
  if (!resources) {
    return {
      cpu: 0,
      memory: 0,
      others: {},
    }
  }
  const cpu = convertKResourceToResource('cpu', resources.cpu)
  const memory = convertKResourceToResource('memory', resources.memory)

  const others = resources
  delete others.cpu
  delete others.memory

  return {
    cpu,
    memory,
    others,
  }
}

export const convertToK8sResources = (resources: Resources): K8sResources => {
  const cpu = resources.cpu !== undefined && resources.cpu > -1 ? `${resources.cpu}` : undefined
  const memory =
    resources.memory !== undefined && resources.memory > -1 ? `${resources.memory}Gi` : undefined

  const k8sResource: K8sResources = resources.others ?? {}

  if (cpu !== undefined) {
    k8sResource['cpu'] = cpu
  }

  if (memory !== undefined) {
    k8sResource['memory'] = memory
  }

  return k8sResource
}

export const hasNvidiaGPU = (resources: K8sResources): boolean => {
  // key is start with nvidia.com/, means it's a gpu resource
  return Object.keys(resources ?? {}).some((key) => key.startsWith('nvidia.com/'))
}
