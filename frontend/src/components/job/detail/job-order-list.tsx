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
import { DurationDialog } from '@/routes/admin/jobs/-components/duration-dialog'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
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

import { ApprovalOrderDataTable } from '@/components/approval-order/approval-order-data-table'
import {
  type ApprovalOrderActionConfig,
  ApprovalOrderOperations,
  createViewOnlyConfig,
} from '@/components/approval-order/approval-order-operations'

import {
  type ApprovalOrder,
  listApprovalOrdersbyName,
  reviewApprovalOrder,
} from '@/services/api/approvalorder'

import useAdmin from '@/hooks/use-admin'
import { useApprovalOrderLock } from '@/hooks/use-approval-order-lock'

interface JobOrderListProps {
  jobName: string
}

export default function JobOrderList({ jobName }: JobOrderListProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isAdmin = useAdmin()
  const navigate = useNavigate()
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false)
  const [rejectReason, setRejectReason] = useState('')
  const [rejectTarget, setRejectTarget] = useState<ApprovalOrder | null>(null)

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

  const query = useQuery({
    queryKey: ['approvalorders', 'byName', jobName],
    queryFn: () => listApprovalOrdersbyName(jobName),
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

  // 刷新工单列表
  const refetchOrders = () => {
    queryClient.invalidateQueries({
      queryKey: ['approvalorders', 'byName', jobName],
    })
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
    onError: () => {
      toast.error(t('ApprovalOrderTable.toast.rejectError'))
    },
  })

  const handleRejectConfirm = () => {
    if (!rejectTarget) return
    const reason = rejectReason.trim()
    if (!reason) {
      toast.error('请输入拒绝理由')
      return
    }
    rejectOrder({ order: rejectTarget, reason })
  }

  const handleViewOrder = (order: ApprovalOrder) => {
    // 首先显示一个提示，确认点击被触发
    toast.info(`正在跳转到工单: ${order.name}`)

    try {
      if (isAdmin) {
        // 管理员跳转到管理员作业详情页面，默认落在详情页
        navigate({
          to: '/admin/more/orders/$id',
          params: { id: String(order.id) },
          search: { tab: 'detail' },
        })
      } else {
        // 普通用户跳转到门户作业详情页面（当前仅有详情视图，无需附加 tab 参数）
        navigate({
          to: '/portal/more/orders/$id',
          params: { id: String(order.id) },
        })
      }
    } catch (error) {
      toast.error(`导航错误: ${error}`)
    }
  }

  // 创建操作配置 - 根据是否是管理员显示不同的操作
  const createActionConfig = (order: ApprovalOrder): ApprovalOrderActionConfig => {
    if (!isAdmin) {
      const config = createViewOnlyConfig(handleViewOrder)
      // 临时调试：确认配置正确
      // eslint-disable-next-line no-console
      console.log('View-only config created:', config)
      return config
    }

    const isPending = order.status === 'Pending'
    return {
      view: {
        show: true,
        onClick: handleViewOrder,
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
        onClick: (order) => {
          setRejectTarget(order)
          setRejectReason('')
          setRejectDialogOpen(true)
        },
        disabled: () => isApproving || isRejecting,
      },
    }
  }

  if (query.isLoading) {
    return <div className="p-4 text-center">加载中...</div>
  }

  if (query.isError) {
    return <div className="text-destructive p-4 text-center">加载失败</div>
  }

  const orders = query.data ?? []

  if (orders.length === 0) {
    return null
  }

  return (
    <>
      <ApprovalOrderDataTable
        query={query}
        storageKey={`job_orders_${jobName}`}
        info={{
          title: '相关工单',
          description: `与作业 ${jobName} 相关的工单`,
        }}
        showExtensionHours={isAdmin}
        renderActions={(order) => (
          <ApprovalOrderOperations order={order} config={createActionConfig(order)} />
        )}
      />

      {/* 工单延期对话框 */}
      {isDelayDialogOpen && selectedOrder && selectedJob && (
        <DurationDialog
          key={`${selectedOrder.id}-${selectedExtHours}`} // 默认值变化时重建
          jobs={[selectedJob]}
          open={isDelayDialogOpen}
          setOpen={setIsDelayDialogOpen}
          onSuccess={handleDelaySuccess}
          setExtend={selectedJob.locked}
          defaultDays={Math.floor((selectedExtHours || 8) / 24)}
          defaultHours={(selectedExtHours || 8) % 24}
        />
      )}

      <Dialog open={rejectDialogOpen} onOpenChange={setRejectDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>拒绝工单</DialogTitle>
            <DialogDescription>
              提供拒绝理由后提交，系统会将该工单标记为拒绝状态。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Input
              value={rejectReason}
              onChange={(event) => setRejectReason(event.target.value)}
              placeholder="请输入拒绝原因"
              disabled={isRejecting}
            />
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
