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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { Database, Hourglass, ListChecks, RefreshCwIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
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

import { ApprovalOrderDataTable } from '@/components/approvalorder/ApprovalOrderDataTable'
import {
  type ApprovalOrderActionConfig,
  ApprovalOrderOperations,
} from '@/components/approvalorder/ApprovalOrderOperations'
import { SectionCards } from '@/components/metrics/section-cards'

import {
  type ApprovalOrder,
  checkPendingApprovalOrder,
  listApprovalOrders,
  updateApprovalOrder,
} from '@/services/api/approvalorder'

import { useApprovalOrderLock } from '@/hooks/useApprovalOrderLock'

import { atomUserInfo } from '@/utils/store'

import { DurationDialog } from '../../jobs/-components/DurationDialog'

export const Route = createFileRoute('/admin/settings/orders/')({
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
  const user = useAtomValue(atomUserInfo)
  const [syncDialogOpen, setSyncDialogOpen] = useState(false)
  const [isSyncing, setIsSyncing] = useState(false)

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
    queryKey: ['admin', 'approvalorders'],
    queryFn: () => listApprovalOrders(),
    select: (res) =>
      [...(res.data ?? [])].sort((a, b) => {
        // 首先按状态排序：Pending > Approved > Rejected
        const statusOrder = { Pending: 0, Approved: 1, Rejected: 2 }
        const aOrder = statusOrder[a.status as keyof typeof statusOrder] ?? 3
        const bOrder = statusOrder[b.status as keyof typeof statusOrder] ?? 3

        if (aOrder !== bOrder) {
          return aOrder - bOrder
        }

        // 在同一状态内，按创建时间降序排列（最新的在前）
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      }),
  })

  const refetchOrders = () => {
    queryClient.invalidateQueries({
      queryKey: ['admin', 'approvalorders'],
    })
  }

  // 批准操作（仅用于无需锁定的场景）
  const { mutate: approveOrder, isPending: isApproving } = useMutation({
    mutationFn: async (order: ApprovalOrder) => {
      await updateApprovalOrder(order.id, {
        name: order.name,
        type: order.type,
        status: 'Approved',
        approvalorderTypeID: Number(order.content.approvalorderTypeID) || 0,
        approvalorderReason: String(order.content.approvalorderReason || ''),
        approvalorderExtensionHours: Number(order.content.approvalorderExtensionHours) || 0,
        reviewerID: user?.id || 0,
      })
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
    mutationFn: async (order: ApprovalOrder) => {
      return updateApprovalOrder(order.id, {
        name: order.name,
        type: order.type,
        status: 'Rejected',
        approvalorderTypeID: Number(order.content.approvalorderTypeID) || 0,
        approvalorderReason: String(order.content.approvalorderReason || ''),
        approvalorderExtensionHours: Number(order.content.approvalorderExtensionHours) || 0,
        reviewerID: user?.id || 0,
      })
    },
    onSuccess: () => {
      toast.success(t('ApprovalOrderTable.toast.rejectSuccess'))
      refetchOrders()
    },
    onError: () => {
      toast.error(t('ApprovalOrderTable.toast.rejectError'))
    },
  })

  // 查看工单详情
  const handleViewOrder = (order: ApprovalOrder) => {
    const orderType = order.type
    if (orderType === 'job') {
      navigate({ to: '/admin/jobs/$name', params: { name: order.name } })
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
            search: { type: order.type },
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
        onClick: rejectOrder,
        disabled: () => isApproving || isRejecting,
      },
    }
  }

  // 同步作业状态函数
  const handleSyncJobStatus = async () => {
    setSyncDialogOpen(true)
  }

  const handleConfirmSync = async () => {
    setIsSyncing(true)
    try {
      const response = await checkPendingApprovalOrder()

      // 显示详细的同步结果
      if (response.data) {
        toast.success(response.data)
      } else {
        toast.success('同步作业状态完成')
      }

      // 刷新工单列表
      queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorders'] })
    } catch (error) {
      toast.error('同步作业状态失败: ' + (error instanceof Error ? error.message : '未知错误'))
    } finally {
      setIsSyncing(false)
      setSyncDialogOpen(false)
    }
  }

  // 统计卡片数据
  const totalPending = useMemo(
    () => (query.data ?? []).filter((o) => o.status === 'Pending').length,
    [query.data]
  )
  const pendingJobDelay = useMemo(
    () =>
      (query.data ?? []).filter(
        (o) =>
          o.status === 'Pending' &&
          o.type === 'job' &&
          Number(o.content?.approvalorderExtensionHours) > 0
      ).length,
    [query.data]
  )
  const pendingDataset = useMemo(
    () => (query.data ?? []).filter((o) => o.status === 'Pending' && o.type === 'dataset').length,
    [query.data]
  )

  return (
    <>
      <SectionCards
        items={[
          {
            title: '待审批工单',
            value: totalPending,
            className: 'text-highlight-blue',
            description: '所有状态为待审批的工单总数',
            icon: ListChecks,
          },
          {
            title: '作业锁定待审批',
            value: pendingJobDelay,
            className: 'text-highlight-purple',
            description: '类型为作业且申请了锁定的待审批工单数',
            icon: Hourglass,
          },
          {
            title: '数据迁移待审批',
            value: pendingDataset,
            className: 'text-highlight-emerald',
            description: '类型为数据集的数据迁移待审批工单数',
            icon: Database,
          },
        ]}
        className="lg:col-span-2"
      />
      <ApprovalOrderDataTable
        query={query}
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
              return t('ApprovalOrderTable.column.creator')
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
      >
        <div className="mb-4">
          <Button
            onClick={handleSyncJobStatus}
            variant="default"
            className="flex items-center gap-2"
            disabled={isSyncing}
          >
            <RefreshCwIcon className="h-4 w-4" />
            {isSyncing ? '同步中...' : '同步作业状态'}
          </Button>
        </div>
      </ApprovalOrderDataTable>

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

      {/* 同步作业状态确认对话框 */}
      <Dialog open={syncDialogOpen} onOpenChange={setSyncDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>同步作业状态</DialogTitle>
            <DialogDescription>
              此操作将检查所有待审批的作业工单，如果对应的作业已不在运行状态，将自动取消相应的工单。
              <br />
              <span className="text-muted-foreground mt-2 block text-sm">
                系统会检查每个待审批工单对应的作业状态，确保工单的有效性。
              </span>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSyncDialogOpen(false)} disabled={isSyncing}>
              取消
            </Button>
            <Button onClick={handleConfirmSync} disabled={isSyncing}>
              {isSyncing ? '同步中...' : '确定同步'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
