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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { getJobPhaseLabel } from '@/components/badge/job-phase-badge'

import { type ApprovalOrder, updateApprovalOrder } from '@/services/api/approvalorder'
import { type IJobInfo, JobPhase, apiAdminGetJobList } from '@/services/api/vcjob'

import { atomUserInfo } from '@/utils/store'

// Hook 用于管理工单锁定逻辑
export function useApprovalOrderLock() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAtomValue(atomUserInfo)
  const [selectedOrder, setSelectedOrder] = useState<ApprovalOrder | null>(null)
  const [selectedJob, setSelectedJob] = useState<IJobInfo | null>(null)
  const [isDelayDialogOpen, setIsDelayDialogOpen] = useState(false)
  const [isFetchingJob, setIsFetchingJob] = useState(false)

  // 近 7 天窗口，避免 -1 带来的重负载
  const JOB_DAYS_WINDOW = 7

  // 获取作业列表用于查找对应作业
  const { data: jobList } = useQuery({
    queryKey: ['admin', 'tasklist', 'job', JOB_DAYS_WINDOW],
    queryFn: () => apiAdminGetJobList(JOB_DAYS_WINDOW),
    select: (res) => res.data,
    retry: 1,
  })

  // 选中工单的锁定小时（只用 content.approvalorderExtensionHours，空按 0 处理）
  const selectedExtHours = useMemo(() => {
    const n = Number(selectedOrder?.content?.approvalorderExtensionHours)
    return Number.isFinite(n) && n >= 0 ? Math.floor(n) : 0
  }, [selectedOrder])

  const isRunningPhase = (phase: IJobInfo['status']) => {
    const enumVal = JobPhase[phase as keyof typeof JobPhase]
    if (enumVal !== undefined) return enumVal === JobPhase.Running
    return String(phase).toLowerCase() === 'running'
  }

  const phaseLabel = (phase: IJobInfo['status']) => {
    const enumVal = JobPhase[phase as keyof typeof JobPhase]
    if (enumVal !== undefined) return getJobPhaseLabel(enumVal).label
    return '未知'
  }

  // 批准操作 mutation
  const { mutate: approveOrder } = useMutation({
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
      // 刷新所有工单相关的查询缓存
      queryClient.invalidateQueries({
        queryKey: ['admin', 'approvalorders'],
      })
      queryClient.invalidateQueries({
        queryKey: ['approvalorders'],
      })
      queryClient.invalidateQueries({
        queryKey: ['portal', 'approvalorders'],
      })
      // 清空状态
      setIsDelayDialogOpen(false)
      setSelectedOrder(null)
      setSelectedJob(null)
    },
    onError: () => {
      toast.error(t('ApprovalOrderTable.toast.approveError'))
    },
  })

  // 查找并验证作业状态
  const findAndValidateJob = async (order: ApprovalOrder): Promise<IJobInfo | null> => {
    const findTarget = (list?: IJobInfo[]) =>
      list?.find((j) => j.jobName === order.name || j.name === order.name)

    // 先查缓存
    const cached = findTarget(jobList)
    if (cached) {
      if (!isRunningPhase(cached.status)) {
        toast.error(`该作业当前状态为 ${phaseLabel(cached.status)}，无法锁定（仅运行中可锁定）`)
        return null
      }
      return cached
    }

    // 未命中缓存：优先扩大时间窗口（7 -> 14 -> 30 天）
    setIsFetchingJob(true)
    try {
      const tryDays = [JOB_DAYS_WINDOW, 14, 30]
      let found: IJobInfo | undefined
      let had502 = false

      for (const d of tryDays) {
        try {
          const res = await apiAdminGetJobList(d)
          const list = res.data ?? []

          // 更新缓存，保持与 useQuery 相同的 queryKey
          queryClient.setQueryData(['admin', 'tasklist', 'job', d], res)

          const job = findTarget(list)
          if (job) {
            found = job
            break
          }
        } catch (err) {
          const status = (err as { response?: { status?: number } })?.response?.status
          if (status === 502) {
            had502 = true
            continue
          }
          throw err
        }
      }

      if (!found) {
        if (had502) {
          toast.error('后端网关错误（502），请稍后重试或联系管理员扩大检索窗口')
        } else {
          toast.error('未找到对应作业，可能不在检索时间范围内')
        }
        return null
      }

      if (!isRunningPhase(found.status)) {
        toast.error(`该作业当前状态为 ${phaseLabel(found.status)}，无法锁定（仅运行中可锁定）`)
        return null
      }

      return found
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 502) {
        toast.error('后端网关错误（502），请稍后重试')
      } else {
        toast.error('获取作业列表失败')
      }
      return null
    } finally {
      setIsFetchingJob(false)
    }
  }

  // 处理批准并锁定
  const handleApproveWithDelay = async (order: ApprovalOrder) => {
    const job = await findAndValidateJob(order)
    if (job) {
      setSelectedOrder(order)
      setSelectedJob(job)
      setIsDelayDialogOpen(true)
    }
  }

  // DurationDialog 成功后的回调（作业已经被锁定，现在批准工单）
  const handleDelaySuccess = async () => {
    if (selectedOrder) {
      // DurationDialog的onSuccess已经处理了作业锁定
      // 这里只需要批准工单
      approveOrder(selectedOrder)
    }
  }

  return {
    // 状态
    selectedOrder,
    selectedJob,
    selectedExtHours,
    isDelayDialogOpen,
    isFetchingJob,

    // 操作函数
    handleApproveWithDelay,
    handleDelaySuccess,
    setIsDelayDialogOpen,

    // 清空状态
    clearSelection: () => {
      setSelectedOrder(null)
      setSelectedJob(null)
      setIsDelayDialogOpen(false)
    },
  }
}
