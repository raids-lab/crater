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
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Database, Hourglass, ListChecks } from 'lucide-react'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { ApprovalOrderDataTable } from '@/components/approval-order/approval-order-data-table'
import {
  type ApprovalOrderActionConfig,
  ApprovalOrderOperations,
} from '@/components/approval-order/approval-order-operations'
import { SectionCards } from '@/components/metrics/section-cards'
import { buildRemoteQueryKey } from '@/components/query-table/remote-state'

import {
  type ApprovalOrder,
  apiGetApprovalOrderPage,
  apiGetApprovalOrderSummary,
  reviewApprovalOrder,
} from '@/services/api/approvalorder'
import type { IPage } from '@/services/types'

import { useApprovalOrderLock } from '@/hooks/use-approval-order-lock'
import useRemoteTableState from '@/hooks/use-remote-table-state'

import { DurationDialog } from '../../jobs/-components/duration-dialog'

export const Route = createFileRoute('/admin/more/orders/')({
  component: RouteComponent,
})
export const getHeader = (key: string): string => {
  switch (key) {
    case 'name':
      return '名称'
    case 'type':
      return '类型'
    case 'status':
      return '状态'
    case 'createdAt':
      return '创建于'
    default:
      return key
  }
}

function RouteComponent() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false)

  const [rejectReason, setRejectReason] = useState('')
  const [rejectTarget, setRejectTarget] = useState<ApprovalOrder | null>(null)
  const tableState = useRemoteTableState('admin_approvalorder_management', {
    sorting: [{ id: 'createdAt', desc: true }],
  })

  // 使用锁定管理器 hook
  const {
    selectedOrder,
    selectedJob,
    selectedExtHours,
    isDelayDialogOpen,
    isFetchingJob,
    handleApproveWithDelay,
    handleDelaySuccess,
    setIsDelayDialogOpen,
  } = useApprovalOrderLock()

  const query = useQuery<IPage<ApprovalOrder>, Error>({
    queryKey: buildRemoteQueryKey('approval-orders-admin', tableState.params),
    queryFn: ({ signal }) =>
      apiGetApprovalOrderPage(tableState.params, true, signal).then((res) => res.data),
    placeholderData: keepPreviousData,
  })
  const summaryQuery = useQuery({
    queryKey: ['approval-order-summary'],
    queryFn: ({ signal }) => apiGetApprovalOrderSummary(signal).then((res) => res.data),
  })

  const refetchOrders = () => {
    queryClient.invalidateQueries({
      queryKey: ['remote-list', 'approval-orders-admin'],
    })
    queryClient.invalidateQueries({ queryKey: ['approval-order-summary'] })
  }

  // 批准操作（仅用于无需锁定的场景）
  const { mutate: approveOrder, isPending: isApproving } = useMutation({
    mutationFn: async (order: ApprovalOrder) => {
      await reviewApprovalOrder(order.id, { status: 'Approved' })
      return order
    },
    onSuccess: () => {
      toast.success(t('ApprovalOrderTable.toast.approveSuccess'))
      refetchOrders()
    },
    onError: () => {
      toast.error(t('ApprovalOrderTable.toast.approveError'))
    },
  })

  // 拒绝操作 mutation
  const { mutate: rejectOrder, isPending: isRejecting } = useMutation({
    mutationFn: async ({ order, reason }: { order: ApprovalOrder; reason: string }) => {
      return reviewApprovalOrder(order.id, { status: 'Rejected', reviewNotes: reason })
    },
    onSuccess: () => {
      toast.success(t('ApprovalOrderTable.toast.rejectSuccess'))
      refetchOrders()
      setRejectDialogOpen(false)
      setRejectReason('')
      setRejectTarget(null)
    },
    onError: (error: unknown) => {
      const message =
        error instanceof Error ? error.message : t('ApprovalOrderTable.toast.rejectError')
      toast.error(message)
    },
  })

  // 查看工单详情
  const handleViewOrder = (order: ApprovalOrder) => {
    const orderType = order.type
    if (orderType === 'job') {
      navigate({
        to: `${order.id}`,
        search: (prev) => ({ ...prev, type: 'job' }),
      })
    } else if (orderType === 'dataset') {
      navigate({
        to: `${order.id}`,
        search: (prev) => ({ ...prev, type: 'dataset' }),
      })
    }
  }

  // 创建操作配置
  const createActionConfig = (order: ApprovalOrder): ApprovalOrderActionConfig => {
    const isPending = order.status === 'Pending'

    return {
      view: {
        show: true,
        onClick: () =>
          navigate({
            to: `${order.id}`,
            search: (prev) => ({
              ...prev,
              type: order.type,
            }),
          }),
      },
      approve: {
        show: isPending,
        onClick: (order) => {
          if (order.type === 'job') {
            handleApproveWithDelay(order)
          } else {
            approveOrder(order)
          }
        },
        label: order.type === 'job' ? '批准并锁定' : '批准',
        disabled: () => isApproving || isRejecting || isFetchingJob,
      },
      reject: {
        show: isPending,
        onClick: (current) => {
          setRejectTarget(current)
          setRejectReason('')
          setRejectDialogOpen(true)
        },
        disabled: () => isApproving || isRejecting,
      },
    }
  }

  const handleRejectConfirm = () => {
    if (!rejectTarget) {
      toast.error('没有选中的工单')
      return
    }
    if (!rejectReason.trim()) {
      toast.error('请输入拒绝理由')
      return
    }
    rejectOrder({ order: rejectTarget, reason: rejectReason.trim() })
  }

  // 统计卡片数据

  return (
    <>
      <SectionCards
        items={[
          {
            title: '待审批工单',
            value: summaryQuery.data?.totalPending ?? 0,
            className: 'text-highlight-blue',
            description: '所有状态为待审批的工单总数',
            icon: ListChecks,
          },
          {
            title: '作业锁定待审批',
            value: summaryQuery.data?.pendingJobDelay ?? 0,
            className: 'text-highlight-purple',
            description: '类型为作业且申请了锁定的待审批工单数',
            icon: Hourglass,
          },
          {
            title: '数据迁移待审批',
            value: summaryQuery.data?.pendingDataset ?? 0,
            className: 'text-highlight-emerald',
            description: '类型为数据集的数据迁移待审批工单数',
            icon: Database,
          },
        ]}
        className="lg:col-span-2"
      />
      <ApprovalOrderDataTable
        query={query}
        remoteState={tableState}
        storageKey="admin_approvalorder_management"
        info={{
          title: t('ApprovalOrderTable.info.title'),
          description: t('ApprovalOrderTable.info.description'),
        }}
        showExtensionHours={true}
        onNameClick={handleViewOrder}
        getHeader={(key: string) => {
          switch (key) {
            case 'name':
              return t('ApprovalOrderTable.column.name')
            case 'type':
              return t('ApprovalOrderTable.column.type')
            case 'status':
              return t('ApprovalOrderTable.column.status')
            case 'creator':
            case 'nickname':
              return t('ApprovalOrderTable.column.creator')
            case 'reviewer':
              return t('ApprovalOrderTable.column.reviewer')
            case 'createdAt':
              return t('ApprovalOrderTable.column.createdAt')
            case 'actions':
              return t('ApprovalOrderTable.column.actions')
            default:
              return key
          }
        }}
        renderActions={(order) => (
          <ApprovalOrderOperations order={order} config={createActionConfig(order)} />
        )}
      />

      {/* 锁定锁定对话框 */}
      {selectedJob && selectedOrder && (
        <DurationDialog
          key={`${selectedOrder.id}-${selectedExtHours}`} // 默认值变化时重建
          jobs={[selectedJob]}
          open={isDelayDialogOpen}
          setOpen={setIsDelayDialogOpen}
          onSuccess={handleDelaySuccess}
          setExtend={selectedJob.locked}
          defaultDays={Math.floor(selectedExtHours / 24)}
          defaultHours={selectedExtHours % 24}
        />
      )}

      {/* 拒绝工单对话框 */}
      <Dialog open={rejectDialogOpen} onOpenChange={setRejectDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>拒绝工单</DialogTitle>

            <DialogDescription>
              提供拒绝理由后提交，系统会将该工单标记为拒绝状态。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-4 sm:items-center sm:gap-4">
              <Label htmlFor="reject-reason" className="sm:text-right">
                拒绝理由
              </Label>
              <Input
                id="reject-reason"
                value={rejectReason}
                onChange={(event) => setRejectReason(event.target.value)}
                placeholder="请输入拒绝原因"
                className="sm:col-span-3"
                disabled={isRejecting}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRejectDialogOpen(false)}
              disabled={isRejecting}
            >
              取消
            </Button>
            <Button onClick={handleRejectConfirm} disabled={isRejecting}>
              {isRejecting ? '提交中...' : '确认拒绝'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
