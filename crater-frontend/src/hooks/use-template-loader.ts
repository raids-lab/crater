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
import { useMutation } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { useEffect, useState } from 'react'
import { FieldValues, UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'

import { PublishSearch } from '@/components/job/publish'

import { apiUserGetImageTemplate } from '@/services/api/imagepack'
import { getJobTemplate } from '@/services/api/jobtemplate'
import { JobTemplate } from '@/services/api/jobtemplate'
import { apiJobTemplate } from '@/services/api/vcjob'

import { importFromJsonString } from '@/utils/form'
import { atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

export interface UIStateUpdater<T> {
  /** Condition to determine if this state should be updated */
  condition: (data: T) => boolean
  /** State setter function */
  setter: (value: boolean) => void
  /** Value to set if condition is true */
  value: boolean
}

interface TemplateLoaderOptions<T extends FieldValues> {
  /** The form object to populate */
  form: UseFormReturn<T>
  /** Metadata configuration for import/export */
  metadata: { version: string; type: string }
  /** Optional UI state updaters for accordions, tabs, etc. */
  uiStateUpdaters?: UIStateUpdater<T>[]
  /** Optional callback when template loaded successfully */
  onSuccess?: (data: T) => void
  /** Optional additional data processing */
  dataProcessor?: (data: T) => T
  /** Search Params */
  searchParams: PublishSearch
}

interface ImageTemplateLoaderOptions<T extends FieldValues> {
  /** The form object to populate */
  form: UseFormReturn<T>
  /** Metadata configuration for import/export */
  metadata: { version: string; type: string }
  /** Optional UI state updaters for accordions, tabs, etc. */
  uiStateUpdaters?: UIStateUpdater<T>[]
  /** Optional callback when template loaded successfully */
  onSuccess?: (data: T) => void
  /** Required ImagePackName */
  imagePackName: string
  /** Required set ImagePackName func */
  setImagePackName: (imagePackName: string) => void
  /** Optional additional data processing */
  dataProcessor?: (data: T) => T
}

/**
 * Custom hook for loading job templates from URL params
 * @example
 * const { fromJob, fromTemplate, templateData } = useTemplateLoader({
 *   form,
 *   metadata: MetadataFormJupyter,
 *   uiStateUpdaters: [
 *     { condition: data => data.envs.length > 0, setter: setEnvOpen, value: "环境变量" },
 *     { condition: data => data.nodeSelector.enable, setter: setOtherOpen, value: "其他设置" },
 *   ]
 * });
 */
export function useTemplateLoader<T extends FieldValues>({
  form,
  metadata,
  uiStateUpdaters = [],
  onSuccess,
  dataProcessor,
  searchParams: { fromJob, fromTemplate },
}: TemplateLoaderOptions<T>) {
  const user = useAtomValue(atomUserInfo)
  const [templateData, setTemplateData] = useState<JobTemplate | null>(null)

  const { mutate: loadJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      try {
        // replace /home/${CRATER_USERNAME} with /home/${username}
        // Import the template data
        let jobInfo = importFromJsonString<T>(metadata, response.data)

        // Apply optional data processing
        if (dataProcessor) {
          jobInfo = dataProcessor(jobInfo)
        }

        // Reset the form with the loaded data
        form.reset(jobInfo)

        // Update UI states based on the loaded data
        uiStateUpdaters.forEach(({ condition, setter, value }) => {
          if (condition(jobInfo)) {
            setter(value)
          }
        })

        // Call the success callback
        if (onSuccess) {
          onSuccess(jobInfo)
        }

        toast.success(`已加载作业 ${fromJob} 的模板配置`)
      } catch (error) {
        showErrorToast(error)
      }
    },
    onError: () => {
      toast.error('获取作业模板失败')
    },
  })

  const { mutate: loadTemplate } = useMutation({
    mutationFn: (templateId: number) => getJobTemplate(templateId),
    onSuccess: (response) => {
      try {
        const template = response.data.template.replace(
          /\/home\/\${CRATER_USERNAME}/g,
          `/home/${user?.name}`
        )
        // Import the template data
        let templateInfo = importFromJsonString<T>(metadata, template)

        // Store template data for display
        setTemplateData(response.data)

        // Apply optional data processing
        if (dataProcessor) {
          templateInfo = dataProcessor(templateInfo)
        }

        // Reset the form with the loaded data
        form.reset(templateInfo)

        // Update UI states based on the loaded data
        uiStateUpdaters.forEach(({ condition, setter, value }) => {
          if (condition(templateInfo)) {
            setter(value)
          }
        })

        // Call the success callback
        if (onSuccess) {
          onSuccess(templateInfo)
        }

        //toast.success(`已加载模板 ${fromTemplate} 的配置`);
      } catch (error) {
        showErrorToast(error)
      }
    },
    onError: () => {
      toast.error('获取模板失败')
    },
  })

  // Load template when fromJob or fromTemplate changes
  useEffect(() => {
    if (fromJob) {
      loadJobTemplate(fromJob)
    } else if (fromTemplate) {
      loadTemplate(Number(fromTemplate))
    }
  }, [loadJobTemplate, loadTemplate, fromJob, fromTemplate])

  return { fromJob, fromTemplate, templateData }
}

export function useImageTemplateLoader<T extends FieldValues>({
  form,
  metadata,
  uiStateUpdaters = [],
  onSuccess,
  imagePackName,
  setImagePackName,
}: ImageTemplateLoaderOptions<T>) {
  const { mutate: loadImageTemplate } = useMutation({
    mutationFn: (imagePackName: string) => apiUserGetImageTemplate(imagePackName),
    onSuccess: (response) => {
      try {
        // Import the template data
        const kanikoInfo = importFromJsonString<T>(metadata, response.data)

        // Reset the form with the loaded data
        form.reset(kanikoInfo)
        // toast.success(JSON.stringify(kanikoInfo));
        // Update UI states based on the loaded data
        uiStateUpdaters.forEach(({ condition, setter, value }) => {
          if (condition(kanikoInfo)) {
            setter(value)
          }
        })

        // Call the success callback
        if (onSuccess) {
          onSuccess(kanikoInfo)
        }

        // toast.success(`已加载镜像 ${imagePackName} 的模板配置`);
      } catch (error) {
        showErrorToast(error)
      }
    },
    onError: () => {
      toast.error('获取镜像模板失败')
    },
  })

  useEffect(() => {
    if (imagePackName !== '') {
      loadImageTemplate(imagePackName)
    }

    return () => {
      setImagePackName('')
    }
  }, [form, imagePackName, loadImageTemplate, setImagePackName])
}
