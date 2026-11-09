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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { getJobPhaseLabel } from '@/components/badge/job-phase-badge'

import { type ApprovalOrder, reviewApprovalOrder } from '@/services/api/approvalorder'
import { type IJupyterDetail, JobPhase, apiAdminGetJobDetail } from '@/services/api/vcjob'

import { showErrorToast } from '@/utils/toast'

export function useApprovalOrderLock() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selection, setSelection] = useState<{ order: ApprovalOrder; job: IJupyterDetail } | null>(
    null
  )
  const [isDelayDialogOpen, setIsDelayDialogOpen] = useState(false)
  const [isFetchingJob, setIsFetchingJob] = useState(false)

  const { mutate: approveOrder, isPending: isApprovingLock } = useMutation({
    mutationFn: (order: ApprovalOrder) => reviewApprovalOrder(order.id, { status: 'Approved' }),
    onSuccess: (_response, order) => {
      toast.success(t('ApprovalOrderTable.toast.approveSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorder', order.id] })
      void queryClient.invalidateQueries({ queryKey: ['admin', 'approvalorders'] })
      void queryClient.invalidateQueries({ queryKey: ['approvalorders'] })
      void queryClient.invalidateQueries({ queryKey: ['portal', 'approvalorders'] })
    },
    onError: showErrorToast,
  })

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
      setIsDelayDialogOpen(true)
    } catch (error) {
      showErrorToast(error)
    } finally {
      setIsFetchingJob(false)
    }
  }

  const handleDelaySuccess = () => {
    if (selection) {
      void queryClient.invalidateQueries({ queryKey: ['job', 'detail', selection.job.jobName] })
      void queryClient.invalidateQueries({ queryKey: ['remote-list', 'admin-jobs'] })
      approveOrder(selection.order)
    }
  }

  const extensionHours = Number(selection?.order.content?.approvalorderExtensionHours)
  const selectedExtHours =
    Number.isFinite(extensionHours) && extensionHours > 0 ? Math.floor(extensionHours) : 0

  return {
    selectedOrder: selection?.order,
    selectedJob: selection?.job,
    selectedExtHours,
    isDelayDialogOpen,
    isProcessingLock: isFetchingJob || isApprovingLock,
    handleApproveWithDelay,
    handleDelaySuccess,
    setIsDelayDialogOpen,
  }
}
