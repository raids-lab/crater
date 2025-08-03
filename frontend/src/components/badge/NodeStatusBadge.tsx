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
import { PhaseBadge } from './PhaseBadge'

// These are valid but not exhaustive conditions of node. A cloud provider may set a condition not listed here.
// Relevant events contain "NodeReady", "NodeNotReady", "NodeSchedulable", and "NodeNotSchedulable".
// const (
// 	// NodeReady means kubelet is healthy and ready to accept pods.
// 	NodeReady NodeConditionType = "Ready"
// 	// NodeMemoryPressure means the kubelet is under pressure due to insufficient available memory.
// 	NodeMemoryPressure NodeConditionType = "MemoryPressure"
// 	// NodeDiskPressure means the kubelet is under pressure due to insufficient available disk.
// 	NodeDiskPressure NodeConditionType = "DiskPressure"
// 	// NodePIDPressure means the kubelet is under pressure due to insufficient available PID.
// 	NodePIDPressure NodeConditionType = "PIDPressure"
// 	// NodeNetworkUnavailable means that network for the node is not correctly configured.
// 	NodeNetworkUnavailable NodeConditionType = "NetworkUnavailable"
// )
export const nodeStatuses = [
  {
    value: 'Ready',
    label: '运行中',
    color: 'text-highlight-blue bg-highlight-blue/20',
    description: '节点正常运行',
  },
  {
    value: 'MemoryPressure',
    label: '内存压力',
    color: 'text-highlight-red bg-highlight-red/20',
    description: '节点内存压力过大',
  },
  {
    value: 'DiskPressure',
    label: '磁盘压力',
    color: 'text-highlight-red bg-highlight-red/20',
    description: '节点磁盘压力过大',
  },
  {
    value: 'PIDPressure',
    label: 'PID压力',
    color: 'text-highlight-red bg-highlight-red/20',
    description: '节点PID压力过大',
  },
  {
    value: 'NetworkUnavailable',
    label: '网络不可用',
    color: 'text-highlight-red bg-highlight-red/20',
    description: '节点网络不可用',
  },
  {
    value: 'Unschedulable',
    label: '不可调度',
    color: 'text-highlight-orange bg-highlight-orange/20',
    description: '节点不可调度',
  },
  {
    value: 'Occupied',
    label: '已占用',
    color: 'text-highlight-yellow bg-highlight-yellow/20',
    description: '节点已被占用',
  },
]

const getNodeStatusLabel = (
  status: string
): {
  label: string
  color: string
  description: string
} => {
  const foundStatus = nodeStatuses.find((s) => s.value === status)
  if (foundStatus) {
    return foundStatus
  } else {
    return {
      label: '未知',
      color: 'text-highlight-slate bg-highlight-slate/20',
      description: '无法获取节点状态信息',
    }
  }
}

const NodeStatusBadge = ({ status }: { status: string }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getNodeStatusLabel} />
}

export default NodeStatusBadge
