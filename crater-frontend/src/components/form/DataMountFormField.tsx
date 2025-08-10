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
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { CirclePlus, HardDriveIcon, XIcon } from 'lucide-react'
import { useState } from 'react'
import { ArrayPath, Path, UseFormReturn, useFieldArray } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { FileSelectDialog } from '@/components/file/FileSelectDialog'
import AccordionCard from '@/components/form/AccordionCard'
import Combobox from '@/components/form/Combobox'
import { ComboboxItem } from '@/components/form/Combobox'
import DatasetItem from '@/components/form/DatasetItem'
import FormLabelMust from '@/components/form/FormLabelMust'

import { IDataset, apiGetDataset } from '@/services/api/dataset'

import { VolumeMountType, VolumeMountsSchema } from '@/utils/form'
import { atomUserInfo } from '@/utils/store'

import { cn } from '@/lib/utils'

interface VolumeMountsCardProps<
  T extends {
    volumeMounts: VolumeMountsSchema
  },
> {
  form: UseFormReturn<T>
  className?: string
}

export function VolumeMountsCard<
  T extends {
    volumeMounts: VolumeMountsSchema
  },
>({ form, className }: VolumeMountsCardProps<T>) {
  const { t } = useTranslation()
  const [dataMountOpen, setDataMountOpen] = useState<boolean>(true)
  const user = useAtomValue(atomUserInfo)

  // Get dataset information
  const datasetInfo = useQuery({
    queryKey: ['datsets'],
    queryFn: () => apiGetDataset(),
    select: (res) => {
      // 去重，根据 Label
      // TODO(zhengxl): 不允许重复的 Label，需要在上传数据集时进行校验
      const uniqueLabels = new Set()
      const uniqueData = res.data.filter((item) => {
        if (uniqueLabels.has(item.name)) {
          return false
        }
        uniqueLabels.add(item.name)
        return true
      })
      return uniqueData.map(
        (item) =>
          ({
            value: item.id.toString(),
            label: `${item.name} 
            (${item.extra.tag?.join(', ')})
            [${item.type === 'model' ? '模型' : '数据集'}]
            [${item.userInfo.nickname} ${item.userInfo.username}]`,
            selectedLabel: item.name,
            detail: item,
          }) as ComboboxItem<IDataset>
      )
    },
  })

  // Field array for volume mounts
  const {
    fields: volumeMountFields,
    append: volumeMountAppend,
    remove: volumeMountRemove,
  } = useFieldArray({
    name: 'volumeMounts' as ArrayPath<T>,
    control: form.control,
  })

  const resetVolumeMountsFields = (index: number, type: number) => {
    form.setValue(`volumeMounts.${index}` as Path<T>, {
      type: type,
      subPath: '',
      mountPath: '',
    })
  }

  const currentValues = form.watch()

  return (
    <AccordionCard
      cardTitle={t('volumeMounts.cardTitle')}
      icon={HardDriveIcon}
      open={dataMountOpen}
      setOpen={setDataMountOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {volumeMountFields.map((field, index) => (
          <div key={field.id}>
            <Separator className={cn('mb-5', index === 0 && 'hidden')} />
            <div className="space-y-5">
              <FormField
                control={form.control}
                name={`volumeMounts.${index}.subPath`}
                render={({ field }) => {
                  const disabled =
                    form.getValues(`volumeMounts.${index}.mountPath`) === `/home/${user.name}`
                  return (
                    <FormItem className="relative">
                      <FormLabel>
                        {t('volumeMounts.mountSource', { index: index + 1 })}
                        <FormLabelMust />
                      </FormLabel>
                      <button
                        type="button"
                        onClick={() => {
                          volumeMountRemove(index)
                        }}
                        className="data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute -top-1.5 right-0 cursor-pointer rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-hidden disabled:pointer-events-none"
                      >
                        <XIcon className="size-4" />
                        <span className="sr-only">Close</span>
                      </button>
                      <FormControl>
                        <Tabs
                          value={
                            currentValues.volumeMounts[index].type === VolumeMountType.FileType
                              ? 'file'
                              : 'dataset'
                          }
                          onValueChange={(value) => {
                            form.setValue(
                              `volumeMounts.${index}.type`,
                              value === 'file' ? VolumeMountType.FileType : VolumeMountType.DataType
                            )
                          }}
                          className="w-full"
                        >
                          <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger
                              value="file"
                              onClick={() =>
                                resetVolumeMountsFields(index, VolumeMountType.FileType)
                              }
                              className="cursor-pointer"
                              disabled={disabled}
                            >
                              {t('volumeMounts.fileTab')}
                            </TabsTrigger>
                            <TabsTrigger
                              value="dataset"
                              onClick={() =>
                                resetVolumeMountsFields(index, VolumeMountType.DataType)
                              }
                              className="cursor-pointer"
                              disabled={disabled}
                            >
                              {t('volumeMounts.dataTab')}
                            </TabsTrigger>
                          </TabsList>
                          <TabsContent value="file">
                            <FileSelectDialog
                              value={field.value.split('/').pop()}
                              handleSubmit={(item) => {
                                field.onChange(item.id)
                                form.setValue(
                                  `volumeMounts.${index}.type`,
                                  VolumeMountType.FileType
                                )
                                let mountName = `/data/${item.name}`
                                switch (item.id) {
                                  case 'user':
                                    mountName = `/home/${user.name}`
                                    break
                                  case 'account':
                                    mountName = '/data/account'
                                    break
                                  case 'public':
                                    mountName = '/data/public'
                                    break
                                }
                                form.setValue(`volumeMounts.${index}.mountPath`, mountName)
                              }}
                              disabled={disabled}
                            />
                          </TabsContent>
                          <TabsContent value="dataset">
                            <Combobox
                              items={datasetInfo.data ?? []}
                              current={field.value}
                              disabled={disabled}
                              useDialog={true}
                              renderLabel={(item) => <DatasetItem item={item} />}
                              handleSelect={(value) => {
                                field.onChange(value)
                                form.setValue(
                                  `volumeMounts.${index}.type`,
                                  VolumeMountType.DataType
                                )
                                form.setValue(`volumeMounts.${index}.datasetID`, Number(value))
                                form.setValue(
                                  `volumeMounts.${index}.mountPath`,
                                  `/data/${datasetInfo.data?.find((item) => item.value === value)?.detail?.name}`
                                )
                              }}
                              formTitle="数据"
                            />
                          </TabsContent>
                        </Tabs>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )
                }}
              />
              <FormField
                control={form.control}
                name={`volumeMounts.${index}.mountPath`}
                render={({ field }) => {
                  const disabled =
                    form.getValues(`volumeMounts.${index}.mountPath`) === `/home/${user.name}`
                  return (
                    <FormItem>
                      <FormLabel>
                        {t('volumeMounts.mountPoint', { index: index + 1 })}
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} className="font-mono" />
                      </FormControl>
                      <FormDescription>
                        {disabled
                          ? t('volumeMounts.defaultMountDescription')
                          : t('volumeMounts.mountPathDescription')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )
                }}
              />
            </div>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full cursor-pointer"
          onClick={() =>
            volumeMountAppend({
              type: VolumeMountType.FileType,
              subPath: '',
              mountPath: '',
            })
          }
        >
          <CirclePlus className="size-4" />
          {t('volumeMounts.addButton', {
            mountType: t('volumeMounts.cardTitle'),
          })}
        </Button>
      </div>
    </AccordionCard>
  )
}
