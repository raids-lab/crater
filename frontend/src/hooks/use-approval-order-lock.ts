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
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { getJobPhaseLabel } from '@/components/badge/job-phase-badge'

import { type ApprovalOrder, adminApproveJobOrder } from '@/services/api/approvalorder'
import {
  type IJupyterDetail,
  type JobLockDuration,
  JobPhase,
  apiAdminGetJobDetail,
} from '@/services/api/vcjob'

import { showErrorToast } from '@/utils/toast'

export function useApprovalOrderLock() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selection, setSelection] = useState<{ order: ApprovalOrder; job: IJupyterDetail } | null>(
    null
  )
  const [isFetchingJob, setIsFetchingJob] = useState(false)
  const isDelayDialogOpen = selection !== null

  const handleApproveWithDelay = async (order: ApprovalOrder) => {
    setIsFetchingJob(true)
    try {
      const { data: job } = await apiAdminGetJobDetail(order.name)
      if (job.status !== JobPhase.Running) {
        toast.error(
          t('durationDialog.approval.jobNotRunning', {
            status: getJobPhaseLabel(job.status).label,
          })
        )
        return
      }

      setSelection({ order, job })
    } catch (error) {
      showErrorToast(error)
    } finally {
      setIsFetchingJob(false)
    }
  }

  const submitApprovalLock = async (duration: JobLockDuration) => {
    if (!selection) throw new Error(t('ApprovalOrderTable.toast.approveError'))

    const { order, job } = selection
    await adminApproveJobOrder(order.id, duration)
    toast.success(t('ApprovalOrderTable.toast.approveSuccess'))
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorder', order.id] }),
      queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorders'] }),
      queryClient.invalidateQueries({ queryKey: ['approvalorders'] }),
      queryClient.invalidateQueries({ queryKey: ['portal', 'approvalorders'] }),
      queryClient.invalidateQueries({ queryKey: ['job', 'detail', job.jobName] }),
      queryClient.invalidateQueries({ queryKey: ['remote-list', 'admin-jobs'] }),
    ])
  }

  const onDelayDialogOpenChange = (open: boolean) => {
    if (!open) setSelection(null)
  }

  const extensionHours = Number(selection?.order.content?.approvalorderExtensionHours)
  const selectedExtHours =
    Number.isFinite(extensionHours) && extensionHours > 0 ? Math.floor(extensionHours) : 0

  return {
    selectedOrder: selection?.order,
    selectedJob: selection?.job,
    selectedExtHours,
    isDelayDialogOpen,
    isProcessingLock: isFetchingJob || isDelayDialogOpen,
    handleApproveWithDelay,
    submitApprovalLock,
    onDelayDialogOpenChange,
  }
}
