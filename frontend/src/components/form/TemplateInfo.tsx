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
import { InfoIcon, NotepadTextIcon } from 'lucide-react'
import React from 'react'
import { FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'

import { PublishSearch } from '@/components/job/publish'
import CardTitle from '@/components/label/CardTitle'

import { UIStateUpdater, useTemplateLoader } from '@/hooks/useTemplateLoader'

import { MarkdownRenderer } from './markdown-renderer'

interface TemplateInfoProps<T extends FieldValues> {
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
  /** Default Markdown content */
  defaultMarkdown?: string
  /** Search Params */
  searchParams: PublishSearch
}

export function TemplateInfo<T extends FieldValues>({
  form,
  metadata,
  uiStateUpdaters = [],
  onSuccess,
  searchParams,
  dataProcessor,
  defaultMarkdown,
}: TemplateInfoProps<T>) {
  const { t } = useTranslation()

  // 使用 hook 获取模板信息
  const { fromJob, fromTemplate, templateData } = useTemplateLoader({
    form,
    metadata,
    uiStateUpdaters,
    onSuccess,
    dataProcessor,
    searchParams,
  })

  // 缓存Markdown内容以避免不必要的重渲染
  const markdownContent = React.useMemo(() => {
    return templateData?.document || ''
  }, [templateData?.document])

  if (!fromJob && !fromTemplate) {
    if (defaultMarkdown) {
      return (
        <Card>
          <CardHeader>
            <CardTitle icon={InfoIcon}>{t('templateInfo.jobInstructions')}</CardTitle>
          </CardHeader>
          <CardContent>
            <MarkdownRenderer>{defaultMarkdown}</MarkdownRenderer>
          </CardContent>
        </Card>
      )
    }
    // 如果没有模板，不渲染任何内容
    return null
  }

  return (
    <>
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle icon={NotepadTextIcon} className="flex items-center">
              {fromJob ? t('templateInfo.jobTemplate') : t('templateInfo.presetTemplate')}
              <Badge variant="outline" className="ml-2">
                {fromJob || templateData?.name}
              </Badge>
            </CardTitle>

            {templateData?.userInfo.nickname && (
              <Badge variant="secondary">
                {t('templateInfo.authorLabel', {
                  nickname: templateData.userInfo.nickname,
                })}
              </Badge>
            )}
          </div>

          <CardDescription>
            {fromJob ? t('templateInfo.loadedFromJob', { fromJob }) : templateData?.describe}
          </CardDescription>
        </CardHeader>

        {markdownContent && (
          <CardContent className="pt-0">
            <div className="">
              <MarkdownRenderer>{markdownContent}</MarkdownRenderer>
            </div>
          </CardContent>
        )}
      </Card>
      {defaultMarkdown && (
        <Card>
          <CardHeader>
            <CardTitle icon={InfoIcon}>{t('templateInfo.jobInstructions')}</CardTitle>
          </CardHeader>
          <CardContent>
            <MarkdownRenderer>{defaultMarkdown}</MarkdownRenderer>
          </CardContent>
        </Card>
      )}
    </>
  )
}
