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

// 后端 GET /nodes 返回的 status 是单值（Occupied > Unschedulable > 首个为 True 的 Condition / Unknown），
// 多个状态会互相覆盖。这里结合 taints / annotations 推导出互相独立的状态维度。

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
  /** 触发该状态的 K8s taint（若有） */
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

export interface NodeExclusiveInfo {
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
  exclusives: NodeExclusiveInfo[]
  /** 按展示顺序排列的所有状态标签，供筛选 / 计数使用 */
  tags: NodeStatusTag[]
}

type NodeStatusSource = Pick<INodeBriefInfo, 'status' | 'taints' | 'annotations'>

const nonEmpty = (value?: string | null): string | undefined => {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

const findTaint = (taints: IClusterNodeTaint[], key: string) => taints.find((t) => t.key === key)

const deriveHealth = (status: string, taints: IClusterNodeTaint[]): NodeHealthInfo => {
  const notReadyTaint = findTaint(taints, NODE_TAINT_NOT_READY)
  if (notReadyTaint || status === NodeStatus.NotReady) {
    return { state: NodeStatus.NotReady, taint: notReadyTaint }
  }
  const unreachableTaint = findTaint(taints, NODE_TAINT_UNREACHABLE)
  if (unreachableTaint || status === NodeStatus.Unknown) {
    return { state: NodeStatus.Unknown, taint: unreachableTaint }
  }
  return { state: NodeStatus.Ready }
}

const derivePressures = (status: string, taints: IClusterNodeTaint[]): NodePressureInfo[] =>
  PRESSURE_TAINTS.flatMap(({ state, taintKey }) => {
    const taint = findTaint(taints, taintKey)
    return taint || status === state ? [{ state, taint }] : []
  })

const deriveUnschedulable = (
  status: string,
  taints: IClusterNodeTaint[],
  annotations: Record<string, string>
): NodeUnschedulableInfo | null => {
  // 保持原有判定：基于 taint 或 status 文本
  const unschedulableTaint = taints.find((t) => t.key.includes(NODE_TAINT_UNSCHEDULABLE))
  const isUnschedulable =
    !!unschedulableTaint ||
    status.toLowerCase().includes('unschedulable') ||
    status.includes('不可调度')
  if (!isUnschedulable) {
    return null
  }
  return {
    reason: nonEmpty(annotations[NODE_ANNOTATION_UNSCHEDULABLE_REASON]),
    operator: nonEmpty(annotations[NODE_ANNOTATION_UNSCHEDULABLE_OPERATOR]),
    timeAdded: unschedulableTaint?.timeAdded,
  }
}

const deriveExclusives = (
  status: string,
  taints: IClusterNodeTaint[],
  annotations: Record<string, string>
): NodeExclusiveInfo[] => {
  const reason = nonEmpty(annotations[NODE_ANNOTATION_OCCUPIED_REASON])
  const operator = nonEmpty(annotations[NODE_ANNOTATION_OCCUPIED_OPERATOR])

  // 与后端 isNodeOccupied 一致：crater.raids.io/account=<name>:NoSchedule
  const seen = new Set<string>()
  const exclusives: NodeExclusiveInfo[] = []
  for (const taint of taints) {
    const account = nonEmpty(taint.value)
    if (taint.key !== NODE_TAINT_ACCOUNT || taint.effect !== 'NoSchedule' || !account) {
      continue
    }
    if (seen.has(account)) {
      continue
    }
    seen.add(account)
    exclusives.push({ account, reason, operator, timeAdded: taint.timeAdded })
  }

  if (exclusives.length === 0 && status === NodeStatus.Occupied) {
    exclusives.push({ reason, operator })
  }
  return exclusives
}

export function deriveNodeStatus(node: NodeStatusSource): DerivedNodeStatus {
  const status = String(node.status ?? '')
  const taints = node.taints ?? []
  const annotations = node.annotations ?? {}

  const health = deriveHealth(status, taints)
  const pressures = derivePressures(status, taints)
  const unschedulable = deriveUnschedulable(status, taints, annotations)
  const exclusives = deriveExclusives(status, taints, annotations)

  const tags: NodeStatusTag[] = [health.state, ...pressures.map((p) => p.state)]
  if (unschedulable) {
    tags.push(NodeStatus.Unschedulable)
  }
  if (exclusives.length > 0) {
    tags.push(NodeStatus.Occupied)
  }

  return { health, pressures, unschedulable, exclusives, tags }
}

export const getNodeStatusTags = (node: NodeStatusSource): NodeStatusTag[] =>
  deriveNodeStatus(node).tags

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
    (derived.exclusives.length > 0 ? 1 : 0)
  )
}
