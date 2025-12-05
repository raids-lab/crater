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
import { PhaseBadge } from './phase-badge'

// 审批单类型枚举
export enum ApprovalOrderType {
  Job = 'job',
  Dataset = 'dataset',
}

// 审批单状态枚举
export enum ApprovalOrderStatus {
  Pending = 'Pending',
  Approved = 'Approved',
  Rejected = 'Rejected',
  Canceled = 'Canceled',
}

export const approvalOrderTypes = [
  {
    value: 'job',
    label: '作业锁定',
  },
  {
    value: 'dataset',
    label: '数据迁移',
  },
]

export const approvalOrderStatuses = [
  {
    value: 'Pending',
    label: '待审批',
    color: 'text-highlight-orange bg-highlight-orange/10',
    description: '工单等待审批中',
  },
  {
    value: 'Approved',
    label: '已批准',
    color: 'text-highlight-green bg-highlight-green/10',
    description: '工单已获得批准',
  },
  {
    value: 'Rejected',
    label: '已拒绝',
    color: 'text-highlight-red bg-highlight-red/10',
    description: '工单已被拒绝',
  },
  {
    value: 'Canceled',
    label: '已取消',
    color: 'text-highlight-gray bg-highlight-gray/10',
    description: '工单已取消',
  },
]

const getApprovalOrderTypeLabel = (
  type: ApprovalOrderType
): {
  label: string
  color: string
  description: string
} => {
  switch (type) {
    case ApprovalOrderType.Job:
      return {
        label: '作业锁定',
        color: 'text-highlight-blue bg-highlight-blue/10',
        description: '作业锁定',
      }
    case ApprovalOrderType.Dataset:
      return {
        label: '数据迁移',
        color: 'text-highlight-green bg-highlight-green/10',
        description: '数据迁移',
      }
    default:
      return {
        label: '作业锁定',
        color: 'text-highlight-blue bg-highlight-blue/10',
        description: '作业锁定',
      }
  }
}

const getApprovalOrderStatusLabel = (
  status: string
): {
  label: string
  color: string
  description: string
} => {
  const foundStatus = approvalOrderStatuses.find((s) => s.value === status)
  if (foundStatus) {
    return foundStatus
  }
  return {
    label: '未知',
    color: 'text-highlight-slate bg-highlight-slate/20',
    description: '未知状态',
  }
}

export const ApprovalOrderTypeBadge = ({ type }: { type: ApprovalOrderType }) => {
  return <PhaseBadge phase={type} getPhaseLabel={getApprovalOrderTypeLabel} />
}

export const ApprovalOrderStatusBadge = ({ status }: { status: string }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getApprovalOrderStatusLabel} />
}

export default ApprovalOrderStatusBadge
