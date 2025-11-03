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
import { LinkIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { ApprovalOrderDataTable } from '@/components/approval-order/approval-order-data-table'
import {
  ApprovalOrderOperations,
  createViewOnlyConfig,
} from '@/components/approval-order/approval-order-operations'
import { CopyButton } from '@/components/button/copy-button'

import { type ApprovalOrder, listMyApprovalOrder } from '@/services/api/approvalorder'

export const Route = createFileRoute('/portal/settings/orders/')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [urgentApprovalOrder, setUrgentApprovalOrder] = useState<ApprovalOrder | null>(null)

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

  const handleGenerateLink = (order: ApprovalOrder) => {
    setUrgentApprovalOrder(order)
  }

  const actionConfig = createViewOnlyConfig(handleViewOrder)
  if (actionConfig.custom) {
    actionConfig.custom.push({
      key: 'generate-link',
      label: '生成审批链接',
      icon: <LinkIcon className="size-4" />,
      onClick: handleGenerateLink,
      disabled: (order) => order.status !== 'Pending',
      separator: true,
    })
  } else {
    actionConfig.custom = [
      {
        key: 'generate-link',
        label: '生成审批链接',
        icon: <LinkIcon className="size-4" />,
        onClick: handleGenerateLink,
        disabled: (order) => order.status !== 'Pending',
        separator: true,
      },
    ]
  }

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
    <>
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
      {urgentApprovalOrder && (
        <Dialog open={!!urgentApprovalOrder} onOpenChange={() => setUrgentApprovalOrder(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>紧急审批提醒</DialogTitle>
              <DialogDescription>
                您的锁定申请已提交。如需紧急审批，请将以下链接发送给管理员。
              </DialogDescription>
            </DialogHeader>
            <div className="bg-muted/40 my-4 flex items-center justify-between space-x-2 rounded-lg p-3">
              <pre className="text-muted-foreground overflow-auto text-sm">
                {`${window.location.origin}/admin/settings/orders/${urgentApprovalOrder.id}`}
              </pre>
              <CopyButton
                content={`${window.location.origin}/admin/settings/orders/${urgentApprovalOrder.id}`}
              />
            </div>
            <DialogFooter>
              <Button onClick={() => setUrgentApprovalOrder(null)}>关闭</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}
