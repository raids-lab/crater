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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { isValidCron } from 'cron-validator'
import { t } from 'i18next'
import { AlarmClockIcon } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormField, FormItem } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'

import CronJobStatusBadge from '@/components/badge/cronjob-status-badge'
import TipBadge from '@/components/badge/tip-badge'
import LoadableButton from '@/components/button/loadable-button'

import {
  CronJobConfigStatus,
  apiAdminCronJobConfigStatus,
  apiAdminLongTimeRunningJobsCleanup,
  apiAdminLowGPUUsageJobsCleanup,
  apiAdminWaitingJupyterJobCancel,
  apiJobScheduleAdmin,
  apiJobScheduleChangeAdmin,
} from '@/services/api/vcjob'

import { cn } from '@/lib/utils'

import CronJobRecordsTable from './-components/cronjob-records-table'

export const Route = createFileRoute('/admin/cronjobs/')({
  component: CronPolicy,
  loader: () => ({ crumb: t('navigation.cronPolicy') }),
})

// Moved Zod schema to component
const getCronErrorMessage = (t: (key: string) => string) => t('cronPolicy.invalidCron')

const getCleanLongTimeSchema = (t: (key: string) => string) =>
  z.object({
    status: z.nativeEnum(CronJobConfigStatus),
    spec: z.string().refine((value) => isValidCron(value), {
      message: getCronErrorMessage(t),
    }),
    configs: z.object({
      batchDays: z.coerce.number().int().positive(),
      interactiveDays: z.coerce.number().int().positive(),
    }),
  })

const getCleanLowGpuSchema = (t: (key: string) => string) =>
  z.object({
    status: z.nativeEnum(CronJobConfigStatus),
    spec: z.string().refine((value) => isValidCron(value), {
      message: getCronErrorMessage(t),
    }),
    configs: z.object({
      timeRange: z.coerce.number().int().positive(),
      util: z.coerce.number(),
      waitTime: z.coerce.number().int().positive(),
    }),
  })

const getCleanWaitingJupyterSchema = (t: (key: string) => string) =>
  z.object({
    status: z.nativeEnum(CronJobConfigStatus),
    spec: z.string().refine((value) => isValidCron(value), {
      message: getCronErrorMessage(t),
    }),
    configs: z.object({
      waitMinitues: z.coerce.number().int().positive(),
    }),
  })

const getFormSchema = (t: (key: string) => string) =>
  z.object({
    cleanLongTime: getCleanLongTimeSchema(t),
    cleanLowGpu: getCleanLowGpuSchema(t),
    cleanWaitingJupyter: getCleanWaitingJupyterSchema(t),
  })

type FormValues = z.infer<ReturnType<typeof getFormSchema>>

function CronPolicy({ className }: { className?: string }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState<boolean>(false)
  const [jobStatuses, setJobStatuses] = useState<{
    cleanLongTime?: string
    cleanLowGpu?: string
    cleanWaitingJupyter?: string
  }>({})

  const formSchema = getFormSchema(t)
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
  })

  // 定期获取任务状态(每3秒)
  const statusQuery = useQuery({
    queryKey: ['admin', 'cronjob', 'status'],
    queryFn: async () => {
      const res = await apiAdminCronJobConfigStatus({
        name: ['clean-long-time-job', 'clean-low-gpu-util-job', 'clean-waiting-jupyter'],
      })
      if (res.code === 0 && res.data) {
        return res.data
      }
      return {}
    },
    refetchInterval: 5000,
    enabled: !loading,
  })

  // 更新状态到组件状态
  useEffect(() => {
    if (statusQuery.data) {
      setJobStatuses({
        cleanLongTime: statusQuery.data['clean-long-time-job']?.status,
        cleanLowGpu: statusQuery.data['clean-low-gpu-util-job']?.status,
        cleanWaitingJupyter: statusQuery.data['clean-waiting-jupyter']?.status,
      })
    }
  }, [statusQuery.data])

  const loadJobSchedule = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apiJobScheduleAdmin()
      const detail = res.data
      if (detail) {
        const jobs = detail.configs || []
        // 将 CronJobConfig 转换为表单格式
        const cleanLongTime = jobs.find((job) => job.name === 'clean-long-time-job')
        const cleanLowGpu = jobs.find((job) => job.name === 'clean-low-gpu-util-job')
        const cleanWaitingJupyter = jobs.find((job) => job.name === 'clean-waiting-jupyter')

        const formData: FormValues = {
          cleanLongTime: cleanLongTime
            ? {
                status: cleanLongTime.status as CronJobConfigStatus,
                spec: cleanLongTime.spec,
                configs: cleanLongTime.config as {
                  batchDays: number
                  interactiveDays: number
                },
              }
            : form.getValues('cleanLongTime'),
          cleanLowGpu: cleanLowGpu
            ? {
                status: cleanLowGpu.status as CronJobConfigStatus,
                spec: cleanLowGpu.spec,
                configs: cleanLowGpu.config as {
                  timeRange: number
                  util: number
                  waitTime: number
                },
              }
            : form.getValues('cleanLowGpu'),
          cleanWaitingJupyter: cleanWaitingJupyter
            ? {
                status: cleanWaitingJupyter.status as CronJobConfigStatus,
                spec: cleanWaitingJupyter.spec,
                configs: cleanWaitingJupyter.config as {
                  waitMinitues: number
                },
              }
            : form.getValues('cleanWaitingJupyter'),
        }
        form.reset(formData)
      } else {
        toast.error(t('cronPolicy.loadError') + res.msg)
      }
    } catch (error) {
      toast.error(t('cronPolicy.loadError') + error)
    } finally {
      setLoading(false)
    }
  }, [form, t])

  useEffect(() => {
    loadJobSchedule()
  }, [loadJobSchedule])

  // Mutation for updating clean long-time job
  const { mutate: updateLongTimeJob, isPending: isLongTimeUpdating } = useMutation({
    mutationFn: async () => {
      const data = form.getValues('cleanLongTime')
      const res = await apiJobScheduleChangeAdmin({
        name: 'clean-long-time-job',
        status: data.status,
        spec: data.spec,
        config: data.configs,
      })
      if (res.code !== 0) {
        throw new Error(res.msg)
      }
      return { res, status: data.status }
    },
    onSuccess: () => {
      toast.success(t('cronPolicy.longTimeSuccess'))
      statusQuery.refetch()
    },
    onError: (error: Error) => {
      toast.error(t('cronPolicy.longTimeError') + error.message)
    },
  })

  // Mutation for updating clean low GPU job
  const { mutate: updateLowGpuJob, isPending: isLowGpuUpdating } = useMutation({
    mutationFn: async () => {
      const data = form.getValues('cleanLowGpu')
      const res = await apiJobScheduleChangeAdmin({
        name: 'clean-low-gpu-util-job',
        status: data.status,
        spec: data.spec,
        config: data.configs,
      })
      if (res.code !== 0) {
        throw new Error(res.msg)
      }
      return { res, status: data.status }
    },
    onSuccess: () => {
      toast.success(t('cronPolicy.lowGpuSuccess'))
      statusQuery.refetch()
    },
    onError: (error: Error) => {
      toast.error(t('cronPolicy.lowGpuError') + error.message)
    },
  })

  // Mutation for updating clean waiting jupyter job
  const { mutate: updateWaitingJupyterJob, isPending: isWaitingJupyterUpdating } = useMutation({
    mutationFn: async () => {
      const data = form.getValues('cleanWaitingJupyter')
      const res = await apiJobScheduleChangeAdmin({
        name: 'clean-waiting-jupyter',
        status: data.status,
        spec: data.spec,
        config: data.configs,
      })
      if (res.code !== 0) {
        throw new Error(res.msg)
      }
      return { res, status: data.status }
    },
    onSuccess: () => {
      toast.success(t('cronPolicy.jupyterSuccess'))
      statusQuery.refetch()
    },
    onError: (error: Error) => {
      toast.error(t('cronPolicy.jupyterError') + error.message)
    },
  })

  const runJob = async () => {}

  const confirmJobRun = async () => {
    try {
      const longTimeStatus = form.getValues('cleanLongTime.status')
      const lowGpuStatus = form.getValues('cleanLowGpu.status')
      const waitingJupyterStatus = form.getValues('cleanWaitingJupyter.status')

      const longTimeData = form.getValues('cleanLongTime.configs')
      const lowGpuData = form.getValues('cleanLowGpu.configs')
      const waitingJupyterData = form.getValues('cleanWaitingJupyter.configs')

      const promises = []

      if (longTimeStatus !== CronJobConfigStatus.Suspended) {
        promises.push(
          apiAdminLongTimeRunningJobsCleanup({
            batchDays: Number(longTimeData.batchDays),
            interactiveDays: Number(longTimeData.interactiveDays),
          })
        )
      } else {
        promises.push(Promise.resolve(null))
      }

      if (lowGpuStatus !== CronJobConfigStatus.Suspended) {
        promises.push(
          apiAdminLowGPUUsageJobsCleanup({
            timeRange: Number(lowGpuData.timeRange),
            util: Number(lowGpuData.util),
            waitTime: Number(lowGpuData.waitTime),
          })
        )
      } else {
        promises.push(Promise.resolve(null))
      }

      if (waitingJupyterStatus !== CronJobConfigStatus.Suspended) {
        promises.push(
          apiAdminWaitingJupyterJobCancel({
            waitMinutes: Number(waitingJupyterData.waitMinitues),
          })
        )
      } else {
        promises.push(Promise.resolve(null))
      }

      const [longTimeRes, lowGpuRes, waitingJupyterRes] = await Promise.all(promises)

      let deletedCount = 0
      let remindedCount = 0

      if (longTimeRes && longTimeRes.code === 0 && longTimeRes.data) {
        if (Array.isArray(longTimeRes.data)) {
          deletedCount += longTimeRes.data.length
        } else {
          const reminded = longTimeRes.data.reminded || []
          const deleted = longTimeRes.data.deleted || []
          remindedCount += reminded.length
          deletedCount += deleted.length
        }
      }

      if (lowGpuRes && lowGpuRes.code === 0 && lowGpuRes.data) {
        if (Array.isArray(lowGpuRes.data)) {
          deletedCount += lowGpuRes.data.length
        } else {
          const reminded = lowGpuRes.data.reminded || []
          const deleted = lowGpuRes.data.deleted || []
          remindedCount += reminded.length
          deletedCount += deleted.length
        }
      }

      if (waitingJupyterRes && waitingJupyterRes.code === 0 && waitingJupyterRes.data) {
        if (Array.isArray(waitingJupyterRes.data)) {
          deletedCount += waitingJupyterRes.data.length
        } else {
          const reminded = waitingJupyterRes.data.reminded || []
          const deleted = waitingJupyterRes.data.deleted || []
          remindedCount += reminded.length
          deletedCount += deleted.length
        }
      }

      const totalCount = deletedCount + remindedCount

      if (totalCount === 0) {
        toast.info(t('cronPolicy.noJobs'))
      } else {
        toast.success(
          t('cronPolicy.cleanupSummary', {
            total: totalCount,
            deleted: deletedCount,
            reminded: remindedCount,
          })
        )
      }
    } catch (error) {
      toast.error(t('cronPolicy.runJobError') + error)
    }
  }

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Card className="flex flex-col">
        <CardHeader>
          <CardTitle className="flex items-center gap-1.5">
            <AlarmClockIcon className="text-primary" />
            {t('cronPolicy.title')}
            <TipBadge />
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <Skeleton className="h-20 w-full" />
          ) : (
            <Form {...form}>
              <div className="space-y-8 p-4">
                <div className="rounded-md border p-4">
                  <div className="mb-4 flex items-center gap-2">
                    <h3 className="font-semibold">{t('cronPolicy.longTimeTitle')}</h3>
                    {jobStatuses.cleanLongTime && (
                      <CronJobStatusBadge status={jobStatuses.cleanLongTime} />
                    )}
                  </div>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanLongTime.status"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Switch
                              checked={field.value !== CronJobConfigStatus.Suspended}
                              onCheckedChange={(checked) =>
                                field.onChange(
                                  checked ? CronJobConfigStatus.Idle : CronJobConfigStatus.Suspended
                                )
                              }
                            />
                          </FormControl>
                          <span>{t('cronPolicy.enable')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLongTime.spec"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.schedule')}</label>
                          <FormControl>
                            <Input className="mt-1 font-mono" {...field} />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLongTime.configs.batchDays"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.batchDays')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLongTime.configs.interactiveDays"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.interactiveDays')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                  </div>
                  <div className="mt-4">
                    <LoadableButton
                      variant="secondary"
                      isLoading={isLongTimeUpdating}
                      isLoadingText={t('cronPolicy.updating')}
                      onClick={() => updateLongTimeJob()}
                    >
                      {t('cronPolicy.longTimeUpdate')}
                    </LoadableButton>
                  </div>
                </div>

                <div className="rounded-md border p-4">
                  <div className="mb-4 flex items-center gap-2">
                    <h3 className="font-semibold">{t('cronPolicy.lowGpuTitle')}</h3>
                    {jobStatuses.cleanLowGpu && (
                      <CronJobStatusBadge status={jobStatuses.cleanLowGpu} />
                    )}
                  </div>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.status"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Switch
                              checked={field.value !== CronJobConfigStatus.Suspended}
                              onCheckedChange={(checked) =>
                                field.onChange(
                                  checked ? CronJobConfigStatus.Idle : CronJobConfigStatus.Suspended
                                )
                              }
                            />
                          </FormControl>
                          <span>{t('cronPolicy.enable')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.spec"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.schedule')}</label>
                          <FormControl>
                            <Input className="mt-1 font-mono" {...field} />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.configs.timeRange"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.timeRange')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.configs.util"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.util')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              step="0.1"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.configs.waitTime"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.waitTime')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                  </div>
                  <div className="mt-4">
                    <LoadableButton
                      variant="secondary"
                      isLoading={isLowGpuUpdating}
                      isLoadingText={t('cronPolicy.updating')}
                      onClick={() => updateLowGpuJob()}
                    >
                      {t('cronPolicy.lowGpuUpdate')}
                    </LoadableButton>
                  </div>
                </div>

                <div className="rounded-md border p-4">
                  <div className="mb-4 flex items-center gap-2">
                    <h3 className="font-semibold">{t('cronPolicy.jupyterTitle')}</h3>
                    {jobStatuses.cleanWaitingJupyter && (
                      <CronJobStatusBadge status={jobStatuses.cleanWaitingJupyter} />
                    )}
                  </div>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanWaitingJupyter.status"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Switch
                              checked={field.value !== CronJobConfigStatus.Suspended}
                              onCheckedChange={(checked) =>
                                field.onChange(
                                  checked ? CronJobConfigStatus.Idle : CronJobConfigStatus.Suspended
                                )
                              }
                            />
                          </FormControl>
                          <span>{t('cronPolicy.enable')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanWaitingJupyter.spec"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.schedule')}</label>
                          <FormControl>
                            <Input className="mt-1 font-mono" {...field} />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanWaitingJupyter.configs.waitMinitues"
                      render={({ field, fieldState }) => (
                        <FormItem className="flex flex-col">
                          <label className="text-sm">{t('cronPolicy.jupyterWait')}</label>
                          <FormControl>
                            <Input
                              type="number"
                              className="mt-1 w-24 font-mono"
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          {fieldState.error && (
                            <p className="text-xs text-red-500">{fieldState.error.message}</p>
                          )}
                        </FormItem>
                      )}
                    />
                  </div>
                  <div className="mt-4">
                    <LoadableButton
                      variant="secondary"
                      isLoading={isWaitingJupyterUpdating}
                      isLoadingText={t('cronPolicy.updating')}
                      onClick={() => updateWaitingJupyterJob()}
                    >
                      {t('cronPolicy.jupyterUpdate')}
                    </LoadableButton>
                  </div>
                </div>
              </div>
            </Form>
          )}
        </CardContent>
        <CardFooter className="flex flex-wrap items-center gap-4 p-4">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button onClick={runJob} variant="destructive">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="lucide lucide-broom"
                >
                  <path d="m13 11 9-9" />
                  <path d="M14.6 12.6c.8.8.9 2.1.2 3L10 22l-8-8 6.4-4.8c.9-.7 2.2-.6 3 .2Z" />
                  <path d="m6.8 10.4 6.8 6.8" />
                  <path d="m5 17 1.4-1.4" />
                </svg>
                {t('cronPolicy.runJob')}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('cronPolicy.confirmTitle')}</AlertDialogTitle>
                <AlertDialogDescription>{t('cronPolicy.confirmMessage')}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t('cronPolicy.cancel')}</AlertDialogCancel>
                <AlertDialogAction onClick={confirmJobRun}>
                  {t('cronPolicy.confirm')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </CardFooter>
      </Card>
      <CronJobRecordsTable />
    </div>
  )
}
