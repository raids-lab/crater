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
import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { ChartNoAxesColumn, CircleHelpIcon, Plus, Trash2 } from 'lucide-react'
import { useMemo } from 'react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
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
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import TipBadge from '@/components/badge/TipBadge'
import Combobox, { ComboboxItem } from '@/components/form/Combobox'
import FormLabelMust from '@/components/form/FormLabelMust'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'

import {
  Resource,
  ResourceVGPU,
  apiResourceList,
  apiResourceNetworks,
  apiResourceVGPUList,
} from '@/services/api/resource'

import { configGrafanaOverviewAtom } from '@/utils/store/config'

import { Card, CardContent, CardHeader } from '../ui/card'

interface ResourceFormFieldsProps<T extends FieldValues> {
  form: UseFormReturn<T>
  cpuPath: FieldPath<T>
  memoryPath: FieldPath<T>
  gpuCountPath: FieldPath<T>
  gpuModelPath: FieldPath<T>
  rdmaPath?: {
    rdmaEnabled: FieldPath<T>
    rdmaLabel: FieldPath<T>
  }
  vgpuPath?: {
    vgpuEnabled: FieldPath<T>
    vgpuModels: FieldPath<T>
  }
}

export function ResourceFormFields<T extends FieldValues>({
  form,
  cpuPath,
  memoryPath,
  gpuCountPath,
  gpuModelPath,
  rdmaPath,
  vgpuPath,
}: ResourceFormFieldsProps<T>) {
  const { t } = useTranslation()
  const gpuCount = form.watch(gpuCountPath)
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom)

  // 获取可用资源列表
  const { data: resources } = useQuery({
    queryKey: ['resources', 'list'],
    queryFn: () => apiResourceList(true),
    select: (res) => {
      return res.data
        .sort((a, b) => {
          return b.amountSingleMax - a.amountSingleMax
        })
        .filter((item) => item.amountSingleMax > 0)
        .map(
          (item) =>
            ({
              value: item.name,
              label: item.label,
              detail: item,
            }) as ComboboxItem<Resource>
        )
    },
  })

  return (
    <>
      <div className="grid grid-cols-3 items-start gap-3">
        <FormField
          control={form.control}
          name={cpuPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t('resourceForm.cpuLabel')}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input type="number" {...form.register(cpuPath, { valueAsNumber: true })} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={memoryPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t('resourceForm.memoryLabel')}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input type="number" {...form.register(memoryPath, { valueAsNumber: true })} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={gpuCountPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t('resourceForm.gpuCountLabel')}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input type="number" {...form.register(gpuCountPath, { valueAsNumber: true })} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <FormField
        control={form.control}
        name={gpuModelPath}
        render={({ field }) => (
          <FormItem hidden={gpuCount === 0}>
            <FormLabel>
              {t('resourceForm.gpuModelLabel')}
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <Combobox
                items={resources ?? []}
                renderLabel={(item) => (
                  <div className="flex w-full flex-row items-center justify-between gap-3">
                    <p>{item.label}</p>
                    <TipBadge
                      title={t('resourceForm.gpuTip', {
                        max: item.detail?.amountSingleMax,
                      })}
                      className="bg-highlight-purple/15 text-highlight-purple"
                    />
                  </div>
                )}
                current={field.value ?? ''}
                handleSelect={(value) => {
                  field.onChange(value)
                  if (rdmaPath) {
                    form.resetField(rdmaPath.rdmaEnabled)
                    form.resetField(rdmaPath.rdmaLabel)
                  }
                  if (vgpuPath) {
                    form.resetField(vgpuPath.vgpuEnabled)
                    form.resetField(vgpuPath.vgpuModels)
                  }
                }}
                formTitle={t('resourceForm.gpuComboboxTitle')}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      {rdmaPath && (
        <RDMAFormFields
          form={form}
          resources={resources ?? []}
          gpuModelPath={gpuModelPath}
          rdmaPath={rdmaPath}
        />
      )}
      {vgpuPath && (
        <VGPUFormFields
          form={form}
          resources={resources ?? []}
          gpuModelPath={gpuModelPath}
          vgpuPath={vgpuPath}
        />
      )}
      <div>
        <Sheet>
          <SheetTrigger asChild>
            <Button type="button" variant="secondary" className="cursor-pointer">
              <ChartNoAxesColumn className="size-4" />
              {t('resourceForm.freeResourceButton')}
            </Button>
          </SheetTrigger>
          <SheetContent className="sm:max-w-4xl">
            <SheetHeader>
              <SheetTitle>{t('resourceForm.freeResourceSheetTitle')}</SheetTitle>
            </SheetHeader>
            <div className="h-[calc(100vh-6rem)] w-full px-4">
              <GrafanaIframe baseSrc={grafanaOverview.schedule} />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </>
  )
}

function RDMAFormFields<T extends FieldValues>({
  form,
  resources,
  gpuModelPath,
  rdmaPath,
}: {
  form: UseFormReturn<T>
  resources: ComboboxItem<Resource>[]
  gpuModelPath: FieldPath<T>
  rdmaPath: {
    rdmaEnabled: FieldPath<T>
    rdmaLabel: FieldPath<T>
  }
}) {
  const { t } = useTranslation()
  const gpuModel = form.watch(gpuModelPath)
  const rdmaEnabled = form.watch(rdmaPath.rdmaEnabled)
  const gpuID = useMemo(() => {
    if (gpuModel) {
      const gpu = resources?.find((item) => item.value === gpuModel)
      return gpu?.detail?.ID ?? 0
    }
    return 0
  }, [gpuModel, resources])

  // 获取给定的 GPU 型号对应的网络资源列表
  const { data: networks } = useQuery({
    queryKey: ['resources', 'networks', 'list', gpuID],
    queryFn: () => apiResourceNetworks(gpuID),
    select: (res) =>
      res.data
        .filter((item) => item.amountSingleMax > 0)
        .map(
          (item) =>
            ({
              value: item.name,
              label: item.label,
              detail: item,
            }) as ComboboxItem<Resource>
        ),
  })

  return (
    <>
      {networks && networks.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <FormField
            control={form.control}
            name={rdmaPath.rdmaEnabled}
            render={({ field }) => (
              <FormItem>
                <div className="flex flex-row items-center justify-between space-y-0 space-x-0">
                  <FormLabel>
                    {t('resourceForm.rdmaLabel')}
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <h2 className="mb-0.5 font-semibold">
                            {t('resourceForm.tooltip.title')}
                          </h2>
                          <p>{t('resourceForm.tooltip.line1')}</p>
                          <p>{t('resourceForm.tooltip.line2')}</p>
                          <p>{t('resourceForm.tooltip.line3')}</p>
                          <p>{t('resourceForm.tooltip.line4')}</p>
                          <p>{t('resourceForm.tooltip.line5')}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </FormLabel>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={(value) => field.onChange(value)}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />
          {rdmaEnabled && (
            <FormField
              control={form.control}
              name={rdmaPath.rdmaLabel}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Combobox
                      items={networks}
                      renderLabel={(item) => (
                        <div className="flex w-full flex-row items-center justify-between gap-3">
                          <p>{item.label}</p>
                        </div>
                      )}
                      current={field.value ?? ''}
                      handleSelect={(value) => {
                        field.onChange(value)
                      }}
                      formTitle={t('resourceForm.rdmaNetworkTitle')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('resourceForm.rdmaDescription', {
                      field: 'RDMA',
                    })}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </div>
      )}
    </>
  )
}

function VGPUFormFields<T extends FieldValues>({
  form,
  resources,
  gpuModelPath,
  vgpuPath,
}: {
  form: UseFormReturn<T>
  resources: ComboboxItem<Resource>[]
  gpuModelPath: FieldPath<T>
  vgpuPath: {
    vgpuEnabled: FieldPath<T>
    vgpuModels: FieldPath<T>
  }
}) {
  const gpuModel = form.watch(gpuModelPath)
  const vgpuEnabled = form.watch(vgpuPath.vgpuEnabled)
  const vgpuModels = (form.watch(vgpuPath.vgpuModels) || []) as Array<{
    label: string
    value: number
  }>

  const gpuID = useMemo(() => {
    if (gpuModel) {
      const gpu = resources?.find((item) => item.value === gpuModel)
      return gpu?.detail?.ID ?? 0
    }
    return 0
  }, [gpuModel, resources])

  // 获取给定的 GPU 型号对应的 VGPU 资源列表
  const { data: vgpuResources } = useQuery({
    queryKey: ['resources', 'vgpu', 'list', gpuID],
    queryFn: () => apiResourceVGPUList(gpuID),
    enabled: gpuID > 0,
    select: (res) =>
      res.data
        .filter((item) => item.vgpuResource)
        .map(
          (item) =>
            ({
              value: item.vgpuResource!.name,
              label: item.vgpuResource!.label,
              detail: item,
            }) as ComboboxItem<ResourceVGPU>
        ),
  })

  const addVGPUModel = () => {
    const currentModels = (form.getValues(vgpuPath.vgpuModels) || []) as Array<{
      label: string
      value: number
    }>
    // @ts-expect-error - 忽略类型检查错误
    form.setValue(vgpuPath.vgpuModels, [...currentModels, { label: '', value: 0 }])
  }

  const removeVGPUModel = (index: number) => {
    const currentModels = (form.getValues(vgpuPath.vgpuModels) || []) as Array<{
      label: string
      value: number
    }>
    const newModels = currentModels.filter((_, i: number) => i !== index)
    // @ts-expect-error - 忽略类型检查错误
    form.setValue(vgpuPath.vgpuModels, newModels)
  }

  const updateVGPUModel = (index: number, field: 'label' | 'value', newValue: string | number) => {
    const currentModels = (form.getValues(vgpuPath.vgpuModels) || []) as Array<{
      label: string
      value: number
    }>
    const updatedModels = [...currentModels]
    if (field === 'label') {
      updatedModels[index] = { ...updatedModels[index], label: newValue as string }
    } else {
      updatedModels[index] = { ...updatedModels[index], value: newValue as number }
    }
    // @ts-expect-error - 忽略类型检查错误
    form.setValue(vgpuPath.vgpuModels, updatedModels)
  }
  return (
    <>
      {vgpuResources && vgpuResources.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <FormField
            control={form.control}
            name={vgpuPath.vgpuEnabled}
            render={({ field }) => (
              <FormItem>
                <div className="flex flex-row items-center justify-between space-y-0 space-x-0">
                  <FormLabel>
                    vGPU
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <h2 className="mb-0.5 font-semibold">vGPU 配置</h2>
                          <p>启用 vGPU 支持</p>
                          <p>可以配置多个维度的 vGPU 限制，如显存限制、算力限制等</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </FormLabel>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={(value) => field.onChange(value)}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />
          {vgpuEnabled && (
            <Card>
              <CardHeader className="flex items-center justify-between">
                <FormLabel>vGPU 配置</FormLabel>
                <Button type="button" variant="outline" size="sm" onClick={addVGPUModel}>
                  <Plus className="mr-1 size-4" />
                  添加配置
                </Button>
              </CardHeader>
              {!!vgpuModels.length && (
                <CardContent className="grid grid-cols-1 gap-3">
                  {vgpuModels.map((model, index: number) => (
                    <div key={index} className="flex w-full flex-row items-end gap-2">
                      <FormField
                        control={form.control}
                        // @ts-expect-error Ignore
                        name={`${vgpuPath.vgpuModels}.${index}.label`}
                        render={() => (
                          <FormItem className="flex-1">
                            <FormLabel hidden={index > 0}>vGPU 类型</FormLabel>
                            <FormControl>
                              <Combobox
                                items={vgpuResources}
                                renderLabel={(item) => (
                                  <div className="flex w-full flex-row items-center justify-between gap-3">
                                    <p>{item.label}</p>
                                  </div>
                                )}
                                current={model.label ?? ''}
                                handleSelect={(value) => {
                                  updateVGPUModel(index, 'label', value)
                                }}
                                formTitle="虚拟 GPU 类型"
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        // @ts-expect-error Ignore
                        name={`${vgpuPath.vgpuModels}.${index}.value`}
                        render={() => (
                          <FormItem>
                            <FormLabel hidden={index > 0}>数量</FormLabel>
                            <FormControl>
                              <Input
                                type="number"
                                value={model.value}
                                onChange={(e) =>
                                  updateVGPUModel(index, 'value', parseInt(e.target.value) || 0)
                                }
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => removeVGPUModel(index)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  ))}
                </CardContent>
              )}
            </Card>
          )}
        </div>
      )}
    </>
  )
}
