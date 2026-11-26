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
import { NodeStatus } from '@/services/api/cluster'

import { PhaseBadge, type PhaseBadgeData } from './phase-badge'

interface NodeStatusMeta extends Omit<PhaseBadgeData, 'icon'> {
  value: NodeStatus
}

const PRESSURE_COLOR = 'text-highlight-red bg-highlight-red/20'

// 顺序即筛选器中的展示顺序：健康状态 → 压力告警 → 调度限制
export const nodeStatuses: NodeStatusMeta[] = [
  {
    value: NodeStatus.Ready,
    label: '运行中',
    color: 'text-highlight-blue bg-highlight-blue/20',
    description: '节点正常运行',
  },
  {
    value: NodeStatus.NotReady,
    label: '未就绪',
    color: 'text-highlight-red bg-highlight-red/20',
    description: '节点未就绪，kubelet 或容器运行时异常',
  },
  {
    value: NodeStatus.Unknown,
    label: '未知',
    color: 'text-highlight-slate bg-highlight-slate/20',
    description: '节点失联或无法获取节点状态',
  },
  {
    value: NodeStatus.MemoryPressure,
    label: '内存压力',
    color: PRESSURE_COLOR,
    description: '节点内存压力过大',
  },
  {
    value: NodeStatus.DiskPressure,
    label: '磁盘压力',
    color: PRESSURE_COLOR,
    description: '节点磁盘压力过大',
  },
  {
    value: NodeStatus.PIDPressure,
    label: 'PID压力',
    color: PRESSURE_COLOR,
    description: '节点PID压力过大',
  },
  {
    value: NodeStatus.NetworkUnavailable,
    label: '网络不可用',
    color: PRESSURE_COLOR,
    description: '节点网络不可用',
  },
  {
    value: NodeStatus.Unschedulable,
    label: '不可调度',
    color: 'text-highlight-orange bg-highlight-orange/20',
    description: '节点不可调度',
  },
  {
    value: NodeStatus.Occupied,
    label: '已占用',
    color: 'text-highlight-yellow bg-highlight-yellow/20',
    description: '节点已被占用',
  },
]

const unknownNodeStatus: PhaseBadgeData = {
  label: '未知',
  color: 'text-highlight-slate bg-highlight-slate/20',
  description: '无法获取节点状态信息',
}

export const getNodeStatusMeta = (status: string): PhaseBadgeData =>
  nodeStatuses.find((s) => s.value === status) ?? unknownNodeStatus

const NodeStatusBadge = ({
  status,
  reason,
  disableDefaultTooltip = false,
}: {
  status: string
  reason?: string
  disableDefaultTooltip?: boolean
}) => {
  const getPhaseLabel = (s: string) => {
    const base = getNodeStatusMeta(s)
    if (reason && typeof reason === 'string' && reason.trim() !== '') {
      return {
        ...base,
        description: reason.trim(),
      }
    }
    return base
  }

  return (
    <PhaseBadge
      phase={status}
      getPhaseLabel={getPhaseLabel}
      disableDefaultTooltip={disableDefaultTooltip}
    />
  )
}

export default NodeStatusBadge
