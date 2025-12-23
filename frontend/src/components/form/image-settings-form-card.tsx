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
import { CircleHelpIcon, CirclePlus, XIcon } from 'lucide-react'
import { ArrayPath, FieldPath, FieldValues, UseFormReturn, useFieldArray } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import TipBadge from '@/components/badge/tip-badge'
import { FileSelectDialog } from '@/components/file/file-select-dialog'
import { TreeDataItem } from '@/components/file/lazy-file-tree'

import { ImageDefaultArchs } from '@/services/api/imagepack'

import { TagsInput } from './tags-input'
import { VolumeMountType } from '@/utils/form'

export function ImageSettingsFormCard<T extends FieldValues>({
  form,
  imageNamePath,
  imageTagPath,
  imageBuildArchPath,
  volumeMountsPath,
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
          {imageBuildArchPath && (
            <>
              <Separator className="mt-4" />
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
            </>
          )}
          {volumeMountsPath && (
            <VolumeMountsSection form={form} volumeMountsPath={volumeMountsPath as ArrayPath<T>} />
          )}
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  )
}

interface ImageSettingsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>
  imageNamePath: FieldPath<T>
  imageTagPath: FieldPath<T>
  imageBuildArchPath?: FieldPath<T>
  volumeMountsPath?: FieldPath<T>
  className?: string
}

interface VolumeMountsSectionProps<T extends FieldValues> {
  form: UseFormReturn<T>
  volumeMountsPath: ArrayPath<T>
}

function VolumeMountsSection<T extends FieldValues>({
  form,
  volumeMountsPath,
}: VolumeMountsSectionProps<T>) {
  const {
    fields: volumeMountFields,
    append: volumeMountAppend,
    remove: volumeMountRemove,
  } = useFieldArray({
    name: volumeMountsPath,
    control: form.control,
  })

  return (
    <>
      <Separator className="mt-4" />
      <div className="space-y-4 px-4 py-4">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">挂载目录</span>
          <TooltipProvider delayDuration={100}>
            <Tooltip>
              <TooltipTrigger asChild>
                <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
              </TooltipTrigger>
              <TooltipContent>
                <p>挂载用户或公共空间的目录到镜像构建环境中</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        {volumeMountFields.map((field, index) => (
          <div key={field.id} className="relative space-y-3 rounded-md border p-4">
            <button
              type="button"
              onClick={() => volumeMountRemove(index)}
              className="absolute right-2 top-2 rounded-sm opacity-70 transition-opacity hover:opacity-100"
            >
              <XIcon className="size-4" />
            </button>
            <FormField
              control={form.control}
              name={`${volumeMountsPath}.${index}.type` as FieldPath<T>}
              render={({ field }) => (
                <input type="hidden" {...field} value={VolumeMountType.FileType} />
              )}
            />
            <div className="flex items-start gap-3">
              <div className="flex-1">
                <FormField
                  control={form.control}
                  name={`${volumeMountsPath}.${index}.mountPath` as FieldPath<T>}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>挂载点</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder="/workspace/data" className="font-mono" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <div className="flex-1">
                <FormField
                  control={form.control}
                  name={`${volumeMountsPath}.${index}.subPath` as FieldPath<T>}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>选择文件</FormLabel>
                      <FormControl>
                        <FileSelectDialog
                          value={field.value}
                          handleSubmit={(item: TreeDataItem) => {
                            field.onChange(item.id)
                          }}
                          allowSelectFile={false}
                          isrw={false}
                          title="选择挂载目录"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() =>
            volumeMountAppend({
              type: VolumeMountType.FileType,
              subPath: '',
              mountPath: '',
            } as any)
          }
        >
          <CirclePlus className="size-4" />
          添加数据挂载
        </Button>
      </div>
    </>
  )
}
