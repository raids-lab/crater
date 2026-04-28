import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card } from '@/components/ui/card'

import WarningAlert from '@/components/custom/warning-alert'

import {
  apiAdminGetBillingStatus,
  apiAdminGetGpuAnalysisStatus,
  apiAdminGetLLMConfig,
  apiAdminGetPrequeueConfig,
  apiAdminGrantAllUsersExtraBalance,
  apiAdminResetAllBillingBalances,
  apiAdminResetLLMConfig,
  apiAdminSetBillingStatus,
  apiAdminSetGpuAnalysisStatus,
  apiAdminUpdateLLMConfig,
  apiAdminUpdatePrequeueConfig,
} from '@/services/api/system-config'
import { ERROR_BUSINESS_LOGIC_ERROR } from '@/services/error_code'
import { IErrorResponse } from '@/services/types'

import { showErrorToast } from '@/utils/toast'

import { BasicSettings } from './-components/basic-settings'
import { BillingSettings } from './-components/billing-settings'
import { GpuAnalysis } from './-components/gpu-analysis'
import { LlmFormSchema, LlmSettings, createLlmSettingsSchema } from './-components/llm-settings'
import { PrequeueSettings } from './-components/prequeue-settings'

export const Route = createFileRoute('/admin/more/')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.platformSettings') }),
})

function RouteComponent() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const llmForm = useForm<LlmFormSchema>({
    resolver: zodResolver(createLlmSettingsSchema(t)),
    defaultValues: {
      baseUrl: '',
      modelName: '',
      apiKey: '',
    },
  })

  const { data: llmConfigData } = useQuery({
    queryKey: ['admin', 'system-config', 'llm'],
    queryFn: async () => {
      const res = await apiAdminGetLLMConfig()
      return res.data
    },
  })

  const { data: gpuStatusData } = useQuery({
    queryKey: ['admin', 'system-config', 'gpu-status'],
    queryFn: () => apiAdminGetGpuAnalysisStatus().then((res) => res.data),
  })

  const { data: prequeueConfigData } = useQuery({
    queryKey: ['admin', 'system-config', 'prequeue'],
    queryFn: () => apiAdminGetPrequeueConfig().then((res) => res.data),
  })

  const [backfillEnabled, setBackfillEnabled] = useState(false)
  const [queueQuotaEnabled, setQueueQuotaEnabled] = useState(false)
  const [prequeueWaitingToleranceSeconds, setPrequeueWaitingToleranceSeconds] = useState('')
  const [activateTickerIntervalSeconds, setActivateTickerIntervalSeconds] = useState('')
  const [maxTotalActivationsPerRound, setMaxTotalActivationsPerRound] = useState('')
  const [prequeueCandidateSize, setPrequeueCandidateSize] = useState('')

  const { data: billingStatusData } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })

  useEffect(() => {
    if (llmConfigData) {
      llmForm.reset({
        baseUrl: llmConfigData.baseUrl,
        modelName: llmConfigData.modelName,
        apiKey: llmConfigData.apiKey || '',
      })
    }
  }, [llmConfigData, llmForm])

  useEffect(() => {
    if (prequeueConfigData) {
      setBackfillEnabled(prequeueConfigData.backfillEnabled)
      setQueueQuotaEnabled(prequeueConfigData.queueQuotaEnabled)
      setPrequeueWaitingToleranceSeconds(
        String(prequeueConfigData.normalJobWaitingToleranceSeconds ?? '')
      )
      setActivateTickerIntervalSeconds(
        String(prequeueConfigData.activateTickerIntervalSeconds ?? '')
      )
      setMaxTotalActivationsPerRound(String(prequeueConfigData.maxTotalActivationsPerRound ?? ''))
      setPrequeueCandidateSize(String(prequeueConfigData.prequeueCandidateSize ?? ''))
    }
  }, [prequeueConfigData])

  const handleError = (error: unknown) => {
    if (typeof error === 'object' && error !== null && 'data' in error) {
      const errorData = (error as { data: IErrorResponse }).data
      const errorCode = errorData?.code

      if (errorCode === ERROR_BUSINESS_LOGIC_ERROR) {
        toast.error(t('systemConfig.gpuAnalysis.error.llmCheckFailed'))
      }
    }
  }

  const updateLLMMutation = useMutation({
    mutationFn: (vars: { data: LlmFormSchema; validate: boolean }) =>
      apiAdminUpdateLLMConfig({
        ...vars.data,
        apiKey: vars.data.apiKey ?? '',
        validate: vars.validate,
      }),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'llm'] })
      if (vars.validate) {
        toast.success(t('systemConfig.llm.testAndSaveSuccess'))
      } else {
        toast.success(t('systemConfig.llm.saveSuccess'))
      }
    },
    onError: handleError,
  })

  const resetLLMMutation = useMutation({
    mutationFn: apiAdminResetLLMConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'llm'] })

      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'gpu-status'] })
      toast.success(t('common.resetSuccess'))
    },
    onError: handleError,
  })

  const toggleGpuMutation = useMutation({
    mutationFn: apiAdminSetGpuAnalysisStatus,
    onSuccess: (_data, newStatus) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'gpu-status'] })

      const message = newStatus
        ? t('systemConfig.gpuAnalysis.enabledSuccess')
        : t('systemConfig.gpuAnalysis.disabledSuccess')
      toast.success(message)
    },
    onError: handleError,
  })

  const updateBillingMutation = useMutation({
    mutationFn: apiAdminSetBillingStatus,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'billing-status'] })
      toast.success(
        t('systemConfig.billing.saveSuccess', {
          defaultValue: 'Billing 配置已更新',
        })
      )
    },
    onError: showErrorToast,
  })

  const resetAllBillingMutation = useMutation({
    mutationFn: apiAdminResetAllBillingBalances,
    onSuccess: async (res) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'billing-status'] }),
        queryClient.invalidateQueries({ queryKey: ['account'] }),
        queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] }),
        queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
        queryClient.invalidateQueries({ queryKey: ['context', 'billing-summary'] }),
      ])
      toast.success(
        `已重置 ${res.data.accountsAffected} 个账户、${res.data.userAccountsAffected} 条成员免费额度`
      )
    },
    onError: showErrorToast,
  })

  const grantAllUsersExtraMutation = useMutation({
    mutationFn: apiAdminGrantAllUsersExtraBalance,
    onSuccess: async (res) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] }),
        queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
        queryClient.invalidateQueries({ queryKey: ['context', 'billing-summary'] }),
      ])
      toast.success(`已为 ${res.data.usersAffected} 个用户发放 ${res.data.delta} 点 extra 额度`)
    },
    onError: showErrorToast,
  })

  const handleLlmSubmit = (values: LlmFormSchema, validate: boolean) => {
    updateLLMMutation.mutate({ data: values, validate })
  }

  const handleLlmReset = () => {
    resetLLMMutation.mutate()
  }

  const handleGpuToggle = async (checked: boolean) => {
    if (checked) {
      const isValid = await llmForm.trigger()
      if (!isValid) {
        toast.error(t('systemConfig.llm.validation.formInvalid'))
        return
      }

      toast.info(t('systemConfig.gpuAnalysis.verifyingLLM'))
      const currentLlmValues = llmForm.getValues()

      updateLLMMutation.mutate(
        { data: currentLlmValues, validate: true },
        {
          onSuccess: () => {
            toggleGpuMutation.mutate(true)
          },
        }
      )
    } else {
      toggleGpuMutation.mutate(false)
    }
  }

  const buildPrequeuePayload = () => ({
    backfillEnabled,
    queueQuotaEnabled,
    normalJobWaitingToleranceSeconds: Number(prequeueWaitingToleranceSeconds),
    activateTickerIntervalSeconds: Number(activateTickerIntervalSeconds),
    maxTotalActivationsPerRound: Number(maxTotalActivationsPerRound),
    prequeueCandidateSize: Number(prequeueCandidateSize),
  })

  const validatePrequeuePositiveIntegers = () => {
    const positiveIntegerValues = [
      prequeueWaitingToleranceSeconds,
      activateTickerIntervalSeconds,
      maxTotalActivationsPerRound,
      prequeueCandidateSize,
    ]
    for (const item of positiveIntegerValues) {
      const value = Number(item)
      if (!Number.isInteger(value) || value <= 0) {
        toast.error(t('systemConfig.prequeue.invalidPositiveInteger'))
        return false
      }
    }
    return true
  }

  const invalidatePrequeueConfig = () => {
    queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'prequeue'] })
    queryClient.invalidateQueries({ queryKey: ['context', 'prequeue'] })
    queryClient.invalidateQueries({ queryKey: ['context', 'job-resource-summary'] })
  }

  const updatePrequeueMutation = useMutation({
    mutationFn: () => apiAdminUpdatePrequeueConfig(buildPrequeuePayload()),
    onSuccess: () => {
      invalidatePrequeueConfig()
      toast.success(t('systemConfig.prequeue.saveSuccess'))
    },
    onError: handleError,
  })

  const handlePrequeueSubmit = () => {
    if (!validatePrequeuePositiveIntegers()) {
      return
    }
    updatePrequeueMutation.mutate()
  }

  const isPrequeueConfigPending = updatePrequeueMutation.isPending

  return (
    <div className="space-y-6">
      <WarningAlert
        title={t('systemSetting.warning.title')}
        description={t('systemSetting.warning.description')}
      />

      <Card>
        <BasicSettings />
      </Card>

      <Card>
        <LlmSettings
          form={llmForm}
          isPending={updateLLMMutation.isPending || resetLLMMutation.isPending}
          onSubmit={handleLlmSubmit}
          onReset={handleLlmReset}
        />

        <GpuAnalysis
          enabled={gpuStatusData?.enabled || false}
          isPending={toggleGpuMutation.isPending || updateLLMMutation.isPending}
          onToggle={handleGpuToggle}
        />
      </Card>

      <Card>
        <PrequeueSettings
          backfillEnabled={backfillEnabled}
          queueQuotaEnabled={queueQuotaEnabled}
          isPending={isPrequeueConfigPending}
          waitingToleranceSeconds={prequeueWaitingToleranceSeconds}
          activateTickerIntervalSeconds={activateTickerIntervalSeconds}
          maxTotalActivationsPerRound={maxTotalActivationsPerRound}
          prequeueCandidateSize={prequeueCandidateSize}
          onBackfillEnabledChange={setBackfillEnabled}
          onQueueQuotaEnabledChange={setQueueQuotaEnabled}
          onWaitingToleranceSecondsChange={setPrequeueWaitingToleranceSeconds}
          onActivateTickerIntervalSecondsChange={setActivateTickerIntervalSeconds}
          onMaxTotalActivationsPerRoundChange={setMaxTotalActivationsPerRound}
          onPrequeueCandidateSizeChange={setPrequeueCandidateSize}
          onSubmit={handlePrequeueSubmit}
        />
      </Card>

      <Card>
        <BillingSettings
          status={billingStatusData}
          isSaving={updateBillingMutation.isPending}
          isResettingAll={resetAllBillingMutation.isPending}
          isGrantingAllExtra={grantAllUsersExtraMutation.isPending}
          onSave={(payload) => updateBillingMutation.mutate(payload)}
          onResetAll={() => resetAllBillingMutation.mutate()}
          onGrantAllExtra={(payload) => grantAllUsersExtraMutation.mutate(payload)}
        />
      </Card>
    </div>
  )
}
