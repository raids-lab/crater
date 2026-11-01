import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  KthenaInferenceTemplate,
  KthenaInferenceTemplateConfig,
  apiCreateKthenaInferenceTemplate,
  apiDeleteKthenaInferenceTemplate,
  apiListKthenaInferenceTemplates,
  apiUpdateKthenaInferenceTemplate,
} from '@/services/api/inference'

import { showErrorToast } from '@/utils/toast'

import { DeploymentPreset, FormSchema } from './serving-form-schema'
import { fromRequest, toRequest } from './serving-serialization'

export function useServingTemplates(form: UseFormReturn<FormSchema>, scope: readonly unknown[]) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userTemplateQueryKey = ['kthena', 'inference-templates', ...scope]
  const [activePresetKey, setActivePresetKey] = useState<string | null>(null)
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<KthenaInferenceTemplate | null>(null)
  const [templateToDelete, setTemplateToDelete] = useState<KthenaInferenceTemplate | null>(null)
  const [templateName, setTemplateName] = useState('')
  const [templateDescription, setTemplateDescription] = useState('')

  const { data: userTemplates = [], isLoading: isUserTemplatesLoading } = useQuery({
    queryKey: userTemplateQueryKey,
    queryFn: () => apiListKthenaInferenceTemplates().then((r) => r.data),
  })
  const { mutate: saveUserTemplate, isPending: isSavingUserTemplate } = useMutation({
    mutationFn: ({
      templateID,
      data,
    }: {
      templateID?: number
      data: { name: string; description: string; config: KthenaInferenceTemplateConfig }
    }) => {
      const payload = data
      return templateID
        ? apiUpdateKthenaInferenceTemplate(templateID, payload).then((res) => res.data)
        : apiCreateKthenaInferenceTemplate(payload).then((res) => res.data)
    },
    onSuccess: async (template) => {
      await queryClient.invalidateQueries({ queryKey: userTemplateQueryKey })
      setActivePresetKey(`user:${template.id}`)
      setTemplateDialogOpen(false)
      setEditingTemplate(null)
      toast.success(t('kthena.form.template.saved'))
    },
    onError: showErrorToast,
  })
  const { mutate: deleteUserTemplate, isPending: isDeletingUserTemplate } = useMutation({
    mutationFn: (templateID: number) => apiDeleteKthenaInferenceTemplate(templateID),
    onSuccess: async (_, templateID) => {
      await queryClient.invalidateQueries({ queryKey: userTemplateQueryKey })
      if (activePresetKey === `user:${templateID}`) {
        setActivePresetKey(null)
      }
      setTemplateToDelete(null)
      toast.success(t('kthena.form.template.deleted'))
    },
    onError: showErrorToast,
  })

  const applyPreset = (preset: DeploymentPreset) => {
    form.reset({ ...form.getValues(), ...preset.values })
    setActivePresetKey(preset.key)
  }
  const applyUserTemplate = (template: KthenaInferenceTemplate) => {
    if (template.config.schemaVersion !== 2) {
      toast.error(t('kthena.topology.oldTemplate'))
      return
    }
    form.reset(fromRequest({ ...template.config, name: form.getValues('name') }))
    setActivePresetKey(`user:${template.id}`)
  }
  const buildUserTemplateConfig = (): KthenaInferenceTemplateConfig => {
    const { name, ...config } = toRequest(form.getValues())
    void name
    return { ...config, schemaVersion: 2 }
  }
  const openTemplateDialog = (template?: KthenaInferenceTemplate) => {
    setEditingTemplate(template ?? null)
    setTemplateName(template?.name ?? '')
    setTemplateDescription(template?.description ?? '')
    setTemplateDialogOpen(true)
  }
  const submitUserTemplate = () => {
    const name = templateName.trim()
    if (!name) {
      toast.error(t('kthena.form.template.nameRequired'))
      return
    }
    saveUserTemplate({
      templateID: editingTemplate?.id,
      data: {
        name,
        description: templateDescription.trim(),
        config: buildUserTemplateConfig(),
      },
    })
  }
  const activeUserTemplate = userTemplates.find(
    (template) => activePresetKey === `user:${template.id}`
  )
  return {
    activePresetKey,
    setActivePresetKey,
    templateDialogOpen,
    setTemplateDialogOpen,
    editingTemplate,
    setEditingTemplate,
    templateToDelete,
    setTemplateToDelete,
    templateName,
    setTemplateName,
    templateDescription,
    setTemplateDescription,
    userTemplates,
    isUserTemplatesLoading,
    isSavingUserTemplate,
    deleteUserTemplate,
    isDeletingUserTemplate,
    applyPreset,
    applyUserTemplate,
    openTemplateDialog,
    submitUserTemplate,
    activeUserTemplate,
  }
}
