import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card } from '@/components/ui/card'

import WarningAlert from '@/components/custom/warning-alert'

import {
  apiAdminGetGpuAnalysisStatus,
  apiAdminGetLLMConfig,
  apiAdminResetLLMConfig,
  // 1. 确保已引入此 API
  apiAdminSetGpuAnalysisStatus,
  apiAdminUpdateLLMConfig,
} from '@/services/api/system-config'
import { ERROR_BUSINESSLOFGIC_ERROR } from '@/services/error_code'
import { IErrorResponse } from '@/services/types'

import { BasicSettings } from './-components/basic-settings'
import { GpuAnalysis } from './-components/gpu-analysis'
import { LlmFormSchema, LlmSettings, createLlmSettingsSchema } from './-components/llm-settings'

export const Route = createFileRoute('/admin/more/')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.platformSettings') }),
})

function RouteComponent() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  // --- LLM Form Definition ---
  const llmForm = useForm<LlmFormSchema>({
    resolver: zodResolver(createLlmSettingsSchema(t)),
    defaultValues: {
      baseUrl: '',
      modelName: '',
      apiKey: '',
    },
  })

  // --- Queries ---
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

  // Sync Data to Form
  useEffect(() => {
    if (llmConfigData) {
      llmForm.reset({
        baseUrl: llmConfigData.baseUrl,
        modelName: llmConfigData.modelName,
        apiKey: llmConfigData.apiKey || '',
      })
    }
  }, [llmConfigData, llmForm])

  // --- Error Handling ---
  const handleError = (error: unknown) => {
    // 检查 error 是否为非空对象
    if (typeof error === 'object' && error !== null && 'data' in error) {
      const errorData = (error as { data: IErrorResponse }).data
      const errorCode = errorData?.code

      if (errorCode === ERROR_BUSINESSLOFGIC_ERROR) {
        toast.error(t('systemConfig.gpuAnalysis.error.llmCheckFailed'))
      }
    }
  }

  // --- Mutations ---
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

  // 2. 新增：重置配置的 Mutation
  const resetLLMMutation = useMutation({
    mutationFn: apiAdminResetLLMConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'llm'] })

      // 【关键修改】同时刷新 GPU 状态
      // 因为后端重置 LLM 时连带关闭了 GPU 分析，我们需要重新拉取最新的开关状态 (false)
      // 这样界面上的开关就会自动变回关闭状态，不会出现 UI 和后端不一致的情况
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'gpu-status'] })

      toast.success(t('common.resetSuccess'))
    },
    onError: handleError,
  })

  const toggleGpuMutation = useMutation({
    mutationFn: apiAdminSetGpuAnalysisStatus,
    // 【关键修改】将 onSuccess 的签名从 (status) 改为 (_data, newStatus)
    // _data 代表我们不关心 API 的返回值（第一个参数）
    // newStatus 代表我们调用 mutate 时传入的值（第二个参数），即 true 或 false
    onSuccess: (_data, newStatus) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'gpu-status'] })

      // 使用 newStatus 来判断应该显示哪个提示
      const message = newStatus
        ? t('systemConfig.gpuAnalysis.enabledSuccess')
        : t('systemConfig.gpuAnalysis.disabledSuccess')
      toast.success(message)
    },
    onError: handleError,
  })

  // --- Event Handlers ---
  const handleLlmSubmit = (values: LlmFormSchema, validate: boolean) => {
    updateLLMMutation.mutate({ data: values, validate })
  }

  // 3. 新增：处理重置点击
  const handleLlmReset = () => {
    // 建议增加一个简单的确认，防止误触
    // 如果你有封装好的 Confirm Dialog 组件更好，这里使用原生 confirm 演示逻辑
    // 或者直接调用 mutation
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
        {/* 4. 更新：传入 onReset 属性和更新 isPending 状态 */}
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
    </div>
  )
}
