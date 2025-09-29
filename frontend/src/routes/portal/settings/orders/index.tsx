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
// i18n-processed-v1.1.0
// Modified code
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ApprovalOrderDataTable } from '@/components/approval-order/approval-order-data-table'
import {
  ApprovalOrderOperations,
  createViewOnlyConfig,
} from '@/components/approval-order/approval-order-operations'

import { type ApprovalOrder, listMyApprovalOrder } from '@/services/api/approvalorder'

export const Route = createFileRoute('/portal/settings/orders/')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const query = useQuery({
    queryKey: ['portal', 'approvalorders', 'me'],
    queryFn: () => listMyApprovalOrder(),
    select: (res) =>
      [...(res.data ?? [])].sort((a, b) => {
        // 先按状态排序：待审批 > 已批准 > 已拒绝
        const statusOrder = { Pending: 0, Approved: 1, Rejected: 2, Canceled: 3 }
        const statusDiff = (statusOrder[a.status] ?? 4) - (statusOrder[b.status] ?? 4)
        if (statusDiff !== 0) return statusDiff

        // 状态相同时按创建时间倒序排列（最新的在前）
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      }),
  })

  const handleViewOrder = (order: ApprovalOrder) => {
    // 如果是作业类型的工单，跳转到作业详情页面
    if (order.type === 'job') {
      const jobName = order.name
      toast.info(`正在跳转到作业: ${jobName}`)
      navigate({ to: '/portal/jobs/detail/$name', params: { name: jobName } })
    } else {
      // 其他类型工单，显示待实现提示
      toast.success('查看非作业类型工单功能待实现')
    }
  }

  const actionConfig = createViewOnlyConfig(handleViewOrder)

  const getHeader = (key: string): string => {
    switch (key) {
      case 'name':
        return t('ApprovalOrderTable.column.name')
      case 'type':
        return t('ApprovalOrderTable.column.type')
      case 'status':
        return t('ApprovalOrderTable.column.status')
      case 'creator':
        return t('ApprovalOrderTable.column.creator')
      case 'createdAt':
        return t('ApprovalOrderTable.column.createdAt')
      case 'actions':
        return t('ApprovalOrderTable.column.actions')
      default:
        return key
    }
  }

  return (
    <ApprovalOrderDataTable
      query={query}
      storageKey="portal_approvalorder_management"
      info={{
        title: t('ApprovalOrderTable.info.title'),
        description: t('ApprovalOrderTable.info.description'),
      }}
      showExtensionHours={true}
      onNameClick={handleViewOrder}
      getHeader={getHeader}
      renderActions={(order) => <ApprovalOrderOperations order={order} config={actionConfig} />}
    />
  )
}
