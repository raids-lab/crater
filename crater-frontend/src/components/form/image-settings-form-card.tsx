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
import { CircleHelpIcon } from 'lucide-react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import TipBadge from '@/components/badge/tip-badge'

import { ImageDefaultArchs } from '@/services/api/imagepack'

import { TagsInput } from './tags-input'

export function ImageSettingsFormCard<T extends FieldValues>({
  form,
  imageNamePath,
  imageTagPath,
  imageBuildArchPath,
  className,
}: ImageSettingsFormCardProps<T>) {
  const { t } = useTranslation()

  return (
    <Accordion type="single" collapsible className={`w-full rounded-lg border ${className}`}>
      <AccordionItem value="image-settings" className="border-none">
        <AccordionTrigger className="px-4 py-3 hover:no-underline">
          <div className="flex flex-row items-center gap-1.5">
            {t('imageSettingsCard')}
            <TipBadge title={t('imageSettingsForm.tipBadgeTitle')} />
          </div>
        </AccordionTrigger>
        <AccordionContent>
          <Separator />
          <div className="flex items-start gap-4 px-4 pt-4">
            <div className="flex-1">
              <FormField
                control={form.control}
                name={imageNamePath}
                render={({ field }) => (
                  <FormItem className="flex h-full flex-col">
                    <FormLabel>
                      {t('imageSettingsForm.nameLabel')}
                      <TooltipProvider delayDuration={100}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>输入用户自定义的镜像名，若为空，则由系统自动生成</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('imageSettingsForm.namePlaceholder')} />
                    </FormControl>
                    <FormMessage className="min-h-[20px] leading-none" />
                  </FormItem>
                )}
              />
            </div>
            <div className="flex-1">
              <FormField
                control={form.control}
                name={imageTagPath}
                render={({ field }) => (
                  <FormItem className="flex h-full flex-col">
                    <FormLabel>
                      {t('imageSettingsForm.tagLabel')}
                      <TooltipProvider delayDuration={100}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>输入用户自定义的镜像标签，若为空，则由系统自动生成</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('imageSettingsForm.tagPlaceholder')} />
                    </FormControl>
                    <FormMessage className="min-h-[20px] leading-none" />
                  </FormItem>
                )}
              />
            </div>
          </div>
        </AccordionContent>
      </AccordionItem>
      {imageBuildArchPath && (
        <AccordionItem value="image-settings" className="border-none">
          <AccordionContent>
            <Separator />
            <div className="flex items-start gap-4 px-4 pt-4">
              <div className="flex-1">
                <TagsInput
                  form={form}
                  tagsPath={imageBuildArchPath}
                  label={`镜像架构`}
                  description={`选择自定义的镜像构建架构，若为空则默认为 linux/amd64 架构，若多选则同时构建多架构版本的镜像`}
                  customTags={ImageDefaultArchs}
                  imageBuildArch={true}
                />
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>
      )}
    </Accordion>
  )
}

interface ImageSettingsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>
  imageNamePath: FieldPath<T>
  imageTagPath: FieldPath<T>
  imageBuildArchPath?: FieldPath<T>
  className?: string
}
