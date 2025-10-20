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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { CheckCircle, Clock, FileText, Type, User } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import DetailPage from '@/components/layout/detail-page'

import {
  type ApprovalOrder,
  type ApprovalOrderReq,
  adminGetApprovalOrder,
  updateApprovalOrder,
} from '@/services/api/approvalorder'

import { useApprovalOrderLock } from '@/hooks/use-approval-order-lock'

import { atomUserInfo } from '@/utils/store'

import { DurationDialog } from '../../jobs/-components/duration-dialog'

const DETAIL_QUERY_KEY = ['admin', 'approvalorder'] as const

export const Route = createFileRoute('/admin/settings/orders/$id')({
  component: RouteComponent,
  loader: async ({ params }) => ({ crumb: params.id }),
})

function RouteComponent() {
  const queryClient = useQueryClient()
  const user = useAtomValue(atomUserInfo)
  const { id } = Route.useParams()
  const orderId = Number(id) || 0
  const navigate = useNavigate()

  const { data: order, refetch } = useQuery({
    queryKey: [...DETAIL_QUERY_KEY, orderId],
    queryFn: async () => {
      const res = await adminGetApprovalOrder(orderId)
      return res.data
    },
    enabled: orderId > 0,
  })

  const {
    selectedOrder,
    selectedJob,
    selectedExtHours,
    isDelayDialogOpen,
    isFetchingJob,
    handleApproveWithDelay,
    setIsDelayDialogOpen,
    clearSelection,
  } = useApprovalOrderLock()

  const [isRejectDialogOpen, setIsRejectDialogOpen] = useState(false)
  const [rejectionReason, setRejectionReason] = useState('')

  const selectedOrderRef = useRef<ApprovalOrder | null>(null)
  useEffect(() => {
    selectedOrderRef.current = selectedOrder ?? null
  }, [selectedOrder])

  const buildPayload = (target: ApprovalOrder, overrides: Partial<ApprovalOrderReq> = {}) => ({
    name: target.name,
    type: target.type,
    status: overrides.status ?? target.status,
    approvalorderTypeID: Number(target.content?.approvalorderTypeID) || 0,
    approvalorderReason: target.content?.approvalorderReason ?? '',
    approvalorderExtensionHours: Number(target.content?.approvalorderExtensionHours) || 0,
    reviewerID: user?.id || 0,
    reviewNotes:
      overrides.reviewNotes ?? (typeof target.reviewNotes === 'string' ? target.reviewNotes : ''),
  })

  const updateDetailCache = (next: ApprovalOrder) => {
    queryClient.setQueryData([...DETAIL_QUERY_KEY, orderId], next)
  }

  const invalidateOrderLists = () => {
    queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorders'] })
    queryClient.invalidateQueries({ queryKey: ['approvalorders'] })
    queryClient.invalidateQueries({ queryKey: ['portal', 'approvalorders'] })
  }

  const approveMutation = useMutation({
    mutationFn: async (target: ApprovalOrder) => {
      const payload = buildPayload(target, { status: 'Approved' })
      const res = await updateApprovalOrder(target.id, payload)
      return res.data
    },
    onSuccess: async (updated) => {
      toast.success('工单已批准')
      updateDetailCache(updated)
      invalidateOrderLists()
      await refetch()
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : '未知错误'
      toast.error(`批准失败: ${message}`)
    },
  })

  const rejectMutation = useMutation({
    mutationFn: async ({ target, reason }: { target: ApprovalOrder; reason: string }) => {
      const payload = buildPayload(target, { status: 'Rejected', reviewNotes: reason })
      const res = await updateApprovalOrder(target.id, payload)
      return res.data
    },
    onSuccess: async (updated) => {
      toast.success('工单已拒绝')
      setIsRejectDialogOpen(false)
      setRejectionReason('')
      updateDetailCache(updated)
      invalidateOrderLists()
      await refetch()
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : '未知错误'
      toast.error(`拒绝失败: ${message}`)
    },
  })

  const creatorName = useMemo(() => {
    if (!order?.creator) return '-'
    return order.creator.nickname || order.creator.username || '-'
  }, [order])

  const detailButtonText = useMemo(() => {
    if (!order) return ''
    if (order.type === 'job') return '查看作业详情'
    if (order.type === 'dataset') return '查看数据详情'
    return `查看${order.type}详情`
  }, [order])

  const handleApprove = async () => {
    if (!order) return
    if (order.type === 'job') {
      await handleApproveWithDelay(order)
      return
    }
    try {
      await approveMutation.mutateAsync(order)
    } catch {
      // 错误提示已在 mutation onError 中处理
    }
  }

  const handleViewDetail = () => {
    if (!order) return
    if (order.type === 'job') {
      navigate({ to: '/admin/jobs/$name', params: { name: order.name } })
    } else {
      toast.info('查看该类型详情的功能暂未实现')
    }
  }

  const handleRejectSubmit = () => {
    if (!rejectionReason.trim()) {
      toast.error('请输入拒绝理由')
      return
    }
    if (!order) {
      toast.error('工单数据不存在，无法拒绝')
      return
    }
    rejectMutation.mutate({ target: order, reason: rejectionReason.trim() })
  }

  const isPendingStatus = order?.status === 'Pending'
  const isProcessing = approveMutation.isPending || rejectMutation.isPending || isFetchingJob

  if (!order) {
    return <div className="text-muted-foreground p-6 text-center">工单不存在或已被删除</div>
  }

  return (
    <>
      <DetailPage
        header={
          <div className="flex justify-between">
            <div className="flex items-center gap-4">
              <div>
                <h1 className="text-2xl font-bold">{order.name}</h1>
                <p className="text-muted-foreground">查看工单的详细信息</p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              {isPendingStatus && (
                <>
                  <Button
                    variant="outline"
                    onClick={() => setIsRejectDialogOpen(true)}
                    disabled={isProcessing}
                  >
                    拒绝
                  </Button>
                  <Button onClick={handleApprove} disabled={isProcessing}>
                    {order.type === 'job' ? '批准并锁定' : '批准'}
                  </Button>
                </>
              )}
              <Button variant="secondary" onClick={handleViewDetail}>
                {detailButtonText}
              </Button>
            </div>
          </div>
        }
        info={[
          { title: '类型', icon: Type, value: order.type },
          { title: '状态', icon: CheckCircle, value: order.status },
          { title: '创建者', icon: User, value: creatorName },
          {
            title: '创建时间',
            icon: Clock,
            value: new Date(order.createdAt).toLocaleString(),
          },
        ]}
        tabs={[
          {
            key: 'detail',
            label: '详情',
            icon: FileText,
            children: (
              <div className="space-y-4 p-4">
                <div className="rounded-lg border p-4">
                  <h3 className="mb-2 text-lg font-semibold">申请内容</h3>
                  <p>申请原因: {order.content?.approvalorderReason}</p>
                  {order.content?.approvalorderExtensionHours > 0 && (
                    <p>延长时间: {order.content.approvalorderExtensionHours} 小时</p>
                  )}
                </div>
                {order.reviewNotes && order.reviewNotes.trim() && (
                  <div className="rounded-lg border p-4">
                    <h3 className="mb-2 text-lg font-semibold">审核备注</h3>
                    <p>{order.reviewNotes}</p>
                  </div>
                )}
              </div>
            ),
          },
        ]}
      />

      {selectedOrder && selectedJob && (
        <DurationDialog
          key={`${selectedOrder.id}-${selectedExtHours}`}
          jobs={[selectedJob]}
          open={isDelayDialogOpen}
          setOpen={(open) => {
            setIsDelayDialogOpen(open)
            if (!open && !approveMutation.isPending) {
              clearSelection()
            }
          }}
          onSuccess={async () => {
            const target = selectedOrderRef.current
            if (!target) {
              toast.error('未找到工单，无法批准')
              return
            }
            try {
              await approveMutation.mutateAsync(target)
            } catch {
              // 错误提示已在 mutation onError 中处理
            } finally {
              clearSelection()
              await refetch()
            }
          }}
          setExtend={selectedJob.locked}
          defaultDays={Math.floor(selectedExtHours / 24)}
          defaultHours={selectedExtHours % 24}
        />
      )}

      <Dialog open={isRejectDialogOpen} onOpenChange={setIsRejectDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>拒绝工单</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="rejection-reason" className="text-right">
                拒绝理由
              </Label>
              <Input
                id="rejection-reason"
                value={rejectionReason}
                onChange={(e) => setRejectionReason(e.target.value)}
                className="col-span-3"
                placeholder="例如：资源不足"
                disabled={rejectMutation.isPending}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsRejectDialogOpen(false)}
              disabled={rejectMutation.isPending}
            >
              取消
            </Button>
            <Button onClick={handleRejectSubmit} disabled={rejectMutation.isPending}>
              {rejectMutation.isPending ? '提交中...' : '确认拒绝'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
