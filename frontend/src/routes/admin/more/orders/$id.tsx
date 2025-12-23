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
import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { CheckCircle, Clock, FileText, Type, User } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { ApprovalOrderStatusBadge } from '@/components/badge/approvalorder-badge'
import NodeStatusBadge from '@/components/badge/node-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import DetailPage from '@/components/layout/detail-page'

import {
  type ApprovalOrder,
  type ApprovalOrderReq,
  adminGetApprovalOrder,
  updateApprovalOrder,
} from '@/services/api/approvalorder'
import { apiJobGetPods } from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'

import { useApprovalOrderLock } from '@/hooks/use-approval-order-lock'

import { atomUserInfo } from '@/utils/store'

import { DurationDialog } from '../../jobs/-components/duration-dialog'

const DETAIL_QUERY_KEY = ['admin', 'approvalorder'] as const

export const Route = createFileRoute('/admin/more/orders/$id')({
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

  // Fetch nodes
  const { data: allNodes } = useQuery(queryNodes())

  // Fetch pods if it's a job and pending
  const { data: pods, isLoading: isPodsLoading } = useQuery({
    queryKey: ['job', 'detail', order?.name, 'pods'],
    queryFn: () => apiJobGetPods(order!.name),
    select: (res) => res.data,
    enabled: !!order && order.type === 'job' && order.status === 'Pending',
    staleTime: 1000 * 60,
  })

  const resources = pods?.[0]?.resource
  const nodeNames = useMemo(
    () => Array.from(new Set(pods?.map((p) => p.nodename).filter(Boolean) || [])),
    [pods]
  )

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

  const reviewerName = useMemo(() => {
    if (!order) return '-'
    const hasReviewer = order.reviewer && order.reviewer.username

    if (order.status !== 'Pending' && !hasReviewer) {
      return '系统'
    }

    return hasReviewer ? order.reviewer.nickname || order.reviewer.username : '-'
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
          {
            title: '状态',
            icon: CheckCircle,
            value: <ApprovalOrderStatusBadge status={order.status} />,
          },
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
            scrollable: true,
            children: (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                {order.type === 'job' && order.status === 'Pending' && (
                  <Card className="md:col-span-2">
                    <CardHeader>
                      <CardTitle className="text-lg">作业资源详情</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="flex flex-col gap-2">
                        <span className="text-muted-foreground text-sm">申请资源</span>
                        {isPodsLoading ? (
                          <div className="bg-muted h-6 w-24 animate-pulse rounded"></div>
                        ) : (
                          <ResourceBadges resources={resources} />
                        )}
                      </div>
                      <div className="flex flex-col gap-2">
                        <span className="text-muted-foreground text-sm">所在节点</span>
                        {isPodsLoading ? (
                          <div className="bg-muted h-6 w-24 animate-pulse rounded"></div>
                        ) : (
                          <div className="flex flex-col gap-2">
                            {nodeNames.map((nodeName) => {
                              const nodeInfo = allNodes?.find((n) => n.name === nodeName)
                              return (
                                <div key={nodeName} className="flex items-center gap-2">
                                  <Link
                                    to="/admin/cluster/nodes/$node"
                                    params={{ node: nodeName }}
                                    className="hover:underline"
                                  >
                                    {nodeName}
                                  </Link>
                                  {nodeInfo && <NodeStatusBadge status={nodeInfo.status} />}
                                </div>
                              )
                            })}
                            {nodeNames.length === 0 && <span>-</span>}
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                )}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">申请内容</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="flex flex-col gap-1">
                      <span className="text-muted-foreground text-sm">申请原因</span>
                      <p className="font-medium">{order.content?.approvalorderReason || '-'}</p>
                    </div>
                    {order.content?.approvalorderExtensionHours > 0 && (
                      <div className="flex justify-between border-t pt-2">
                        <span className="text-muted-foreground text-sm">延长时间</span>
                        <span className="font-medium">
                          {order.content.approvalorderExtensionHours} 小时
                        </span>
                      </div>
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">审核进度</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground text-sm">当前状态</span>
                      <span className="font-medium">
                        <ApprovalOrderStatusBadge status={order.status} />
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground text-sm">审核人</span>
                      <span className="font-medium">{reviewerName}</span>
                    </div>
                    <div className="flex flex-col gap-1 border-t pt-2">
                      <span className="text-muted-foreground text-sm">审核备注</span>
                      <p className="text-sm">{order.reviewNotes || '暂无'}</p>
                    </div>
                  </CardContent>
                </Card>
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
