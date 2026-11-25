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
import { IClusterNodeTaint, INodeBriefInfo, NodeStatus } from '@/services/api/cluster'

export const NODE_TAINT_NOT_READY = 'node.kubernetes.io/not-ready'
export const NODE_TAINT_UNREACHABLE = 'node.kubernetes.io/unreachable'
export const NODE_TAINT_UNSCHEDULABLE = 'node.kubernetes.io/unschedulable'
export const NODE_TAINT_ACCOUNT = 'crater.raids.io/account'

export const NODE_ANNOTATION_UNSCHEDULABLE_REASON = 'crater.raids.io/unschedulable-reason'
export const NODE_ANNOTATION_UNSCHEDULABLE_OPERATOR = 'crater.raids.io/unschedulable-operator'
export const NODE_ANNOTATION_OCCUPIED_REASON = 'crater.raids.io/taint-reason-occupied'
export const NODE_ANNOTATION_OCCUPIED_OPERATOR = 'crater.raids.io/taint-operator-occupied'

export type NodeHealthState = NodeStatus.Ready | NodeStatus.NotReady | NodeStatus.Unknown

export type NodePressureState =
  | NodeStatus.MemoryPressure
  | NodeStatus.DiskPressure
  | NodeStatus.PIDPressure
  | NodeStatus.NetworkUnavailable

/** 可用于筛选的独立状态标签，值与 nodeStatuses 中的 value 一一对应 */
export type NodeStatusTag =
  | NodeHealthState
  | NodePressureState
  | NodeStatus.Unschedulable
  | NodeStatus.Occupied

const PRESSURE_TAINTS: ReadonlyArray<{ state: NodePressureState; taintKey: string }> = [
  { state: NodeStatus.MemoryPressure, taintKey: 'node.kubernetes.io/memory-pressure' },
  { state: NodeStatus.DiskPressure, taintKey: 'node.kubernetes.io/disk-pressure' },
  { state: NodeStatus.PIDPressure, taintKey: 'node.kubernetes.io/pid-pressure' },
  { state: NodeStatus.NetworkUnavailable, taintKey: 'node.kubernetes.io/network-unavailable' },
]

export interface NodeHealthInfo {
  state: NodeHealthState
  /** 与该状态对应的 K8s 污点（若有） */
  taint?: IClusterNodeTaint
}

export interface NodePressureInfo {
  state: NodePressureState
  taint?: IClusterNodeTaint
}

export interface NodeUnschedulableInfo {
  reason?: string
  operator?: string
  timeAdded?: string
}

export interface NodeOccupiedInfo {
  /** 独占账户名；为空表示后端标记了 Occupied 但未能从 taint 中解析出账户 */
  account?: string
  reason?: string
  operator?: string
  timeAdded?: string
}

export interface DerivedNodeStatus {
  health: NodeHealthInfo
  pressures: NodePressureInfo[]
  unschedulable: NodeUnschedulableInfo | null
  occupied: NodeOccupiedInfo | null
  /** 按展示顺序排列的所有状态标签，供筛选 / 计数使用 */
  tags: NodeStatusTag[]
}

type NodeStatusSource = Pick<
  INodeBriefInfo,
  'status' | 'readyStatus' | 'unschedulable' | 'pressures' | 'taints' | 'annotations'
>

const nonEmpty = (value?: string | null): string | undefined => {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

const findTaint = (taints: IClusterNodeTaint[], key: string) => taints.find((t) => t.key === key)

const deriveHealth = (node: NodeStatusSource, taints: IClusterNodeTaint[]): NodeHealthInfo => {
  const notReadyTaint = findTaint(taints, NODE_TAINT_NOT_READY)
  const unreachableTaint = findTaint(taints, NODE_TAINT_UNREACHABLE)
  switch (node.readyStatus) {
    case 'True':
      return { state: NodeStatus.Ready }
    case 'False':
      return { state: NodeStatus.NotReady, taint: notReadyTaint }
    case 'Unknown':
      return { state: NodeStatus.Unknown, taint: unreachableTaint }
  }
  if (notReadyTaint || node.status === NodeStatus.NotReady) {
    return { state: NodeStatus.NotReady, taint: notReadyTaint }
  }
  if (unreachableTaint || node.status === NodeStatus.Unknown) {
    return { state: NodeStatus.Unknown, taint: unreachableTaint }
  }
  return { state: NodeStatus.Ready }
}

const derivePressures = (node: NodeStatusSource, taints: IClusterNodeTaint[]): NodePressureInfo[] =>
  PRESSURE_TAINTS.flatMap(({ state, taintKey }) => {
    const taint = findTaint(taints, taintKey)
    return node.pressures?.includes(state) || node.status === state || taint
      ? [{ state, taint }]
      : []
  })

const deriveUnschedulable = (
  node: NodeStatusSource,
  taints: IClusterNodeTaint[],
  annotations: Record<string, string>
): NodeUnschedulableInfo | null => {
  const unschedulableTaint = taints.find(
    (t) => t.key === NODE_TAINT_UNSCHEDULABLE && t.effect === 'NoSchedule'
  )
  const isUnschedulable =
    typeof node.unschedulable === 'boolean'
      ? node.unschedulable
      : !!unschedulableTaint || node.status === NodeStatus.Unschedulable
  if (!isUnschedulable) {
    return null
  }
  return {
    reason: nonEmpty(annotations[NODE_ANNOTATION_UNSCHEDULABLE_REASON]),
    operator: nonEmpty(annotations[NODE_ANNOTATION_UNSCHEDULABLE_OPERATOR]),
    timeAdded: unschedulableTaint?.timeAdded,
  }
}

const deriveOccupied = (
  status: NodeStatus,
  taints: IClusterNodeTaint[],
  annotations: Record<string, string>
): NodeOccupiedInfo | null => {
  const taint = taints.find((t) => t.key === NODE_TAINT_ACCOUNT && t.effect === 'NoSchedule')
  const account = nonEmpty(taint?.value)
  if (!account && status !== NodeStatus.Occupied) {
    return null
  }
  return {
    account,
    reason: nonEmpty(annotations[NODE_ANNOTATION_OCCUPIED_REASON]),
    operator: nonEmpty(annotations[NODE_ANNOTATION_OCCUPIED_OPERATOR]),
    timeAdded: taint?.timeAdded,
  }
}

export function deriveNodeStatus(node: NodeStatusSource): DerivedNodeStatus {
  const taints = node.taints ?? []
  const annotations = node.annotations ?? {}

  const health = deriveHealth(node, taints)
  const pressures = derivePressures(node, taints)
  const unschedulable = deriveUnschedulable(node, taints, annotations)
  const occupied = deriveOccupied(node.status, taints, annotations)

  const tags: NodeStatusTag[] = [health.state, ...pressures.map((p) => p.state)]
  if (unschedulable) {
    tags.push(NodeStatus.Unschedulable)
  }
  if (occupied) {
    tags.push(NodeStatus.Occupied)
  }

  return { health, pressures, unschedulable, occupied, tags }
}

export const getNodeStatusTags = (node: NodeStatusSource): NodeStatusTag[] =>
  deriveNodeStatus(node).tags

export const matchesNodeStatusFilter = (node: NodeStatusSource, filterValue: unknown): boolean => {
  if (!Array.isArray(filterValue) || filterValue.length === 0) {
    return true
  }
  const tags: string[] = getNodeStatusTags(node)
  return filterValue.some((value) => tags.includes(String(value)))
}

const HEALTH_SEVERITY: Record<NodeHealthState, number> = {
  [NodeStatus.NotReady]: 3,
  [NodeStatus.Unknown]: 2,
  [NodeStatus.Ready]: 0,
}

/** 排序用严重程度：健康状态权重最高，其次告警数量，再其次调度限制 */
export const getNodeStatusSeverity = (node: NodeStatusSource): number => {
  const derived = deriveNodeStatus(node)
  return (
    HEALTH_SEVERITY[derived.health.state] * 100 +
    derived.pressures.length * 10 +
    (derived.unschedulable ? 2 : 0) +
    (derived.occupied ? 1 : 0)
  )
}
