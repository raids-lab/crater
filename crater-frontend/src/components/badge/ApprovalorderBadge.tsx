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
    label: '作业延时',
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
  },
  {
    value: 'Approved',
    label: '已批准',
  },
  {
    value: 'Rejected',
    label: '已拒绝',
  },
  {
    value: 'Canceled',
    label: '已取消',
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
        label: '作业延时',
        color: 'text-highlight-blue bg-highlight-blue/10',
        description: '作业延时',
      }
    case ApprovalOrderType.Dataset:
      return {
        label: '数据迁移',
        color: 'text-highlight-green bg-highlight-green/10',
        description: '数据迁移',
      }
    default:
      return {
        label: '作业延时',
        color: 'text-highlight-blue bg-highlight-blue/10',
        description: '作业延时',
      }
  }
}

const getApprovalOrderStatusLabel = (
  status: ApprovalOrderStatus
): {
  label: string
  color: string
  description: string
} => {
  switch (status) {
    case ApprovalOrderStatus.Pending:
      return {
        label: '待审批',
        color: 'text-highlight-orange bg-highlight-orange/10',
        description: '待审批',
      }
    case ApprovalOrderStatus.Approved:
      return {
        label: '已批准',
        color: 'text-highlight-green bg-highlight-green/10',
        description: '已批准',
      }
    case ApprovalOrderStatus.Rejected:
      return {
        label: '已拒绝',
        color: 'text-highlight-red bg-highlight-red/10',
        description: '已拒绝',
      }
    case ApprovalOrderStatus.Canceled:
      return {
        label: '已取消',
        color: 'text-highlight-gray bg-highlight-gray/10',
        description: '已取消',
      }
    default:
      return {
        label: '待审批',
        color: 'text-highlight-orange bg-highlight-orange/10',
        description: '待审批',
      }
  }
}

export const ApprovalOrderTypeBadge = ({ type }: { type: ApprovalOrderType }) => {
  return <PhaseBadge phase={type} getPhaseLabel={getApprovalOrderTypeLabel} />
}

export const ApprovalOrderStatusBadge = ({ status }: { status: ApprovalOrderStatus }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getApprovalOrderStatusLabel} />
}

export default ApprovalOrderStatusBadge
