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
import { createFileRoute } from '@tanstack/react-router'
import { isValidCron } from 'cron-validator'
import { t } from 'i18next'
import { AlarmClockIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
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
import { Checkbox } from '@/components/ui/checkbox'
import { Form, FormControl, FormField, FormItem } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

import TipBadge from '@/components/badge/tip-badge'

import {
  apiAdminLongTimeRunningJobsCleanup,
  apiAdminLowGPUUsageJobsCleanup,
  apiAdminWaitingJupyterJobCancel,
  apiJobScheduleAdmin,
  apiJobScheduleChangeAdmin,
} from '@/services/api/vcjob'

import { cn } from '@/lib/utils'

import CronJobRecordsTable from './cronjob-records-table'

export const Route = createFileRoute('/admin/cronjobs/')({
  component: CronPolicy,
  loader: () => ({ crumb: t('navigation.cronPolicy') }),
})

interface Job {
  name: string
  suspend?: boolean
  schedule?: string
  configs?: Record<string, number | string | boolean>
}

// Moved Zod schema to component
const getCronErrorMessage = (t: (key: string) => string) => t('cronPolicy.invalidCron')

const getCleanLongTimeSchema = (t: (key: string) => string) =>
  z.object({
    suspend: z.boolean(),
    schedule: z.string().refine((value) => isValidCron(value), {
      message: getCronErrorMessage(t),
    }),
    configs: z.object({
      batchDays: z.coerce.number().int().positive(),
      interactiveDays: z.coerce.number().int().positive(),
    }),
  })

const getCleanLowGpuSchema = (t: (key: string) => string) =>
  z.object({
    suspend: z.boolean(),
    schedule: z.string().refine((value) => isValidCron(value), {
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
    suspend: z.boolean(),
    schedule: z.string().refine((value) => isValidCron(value), {
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

export default function CronPolicy({ className }: { className?: string }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState<boolean>(false)

  const formSchema = getFormSchema(t)
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
  })

  useEffect(() => {
    async function loadJobSchedule() {
      setLoading(true)
      try {
        const res = await apiJobScheduleAdmin()
        const detail = res.data
        if (detail) {
          const jobs = Array.isArray(detail) ? detail : []
          const formData: FormValues = {
            cleanLongTime:
              jobs.find((job: Job) => job.name === 'clean-long-time-job') ||
              form.getValues('cleanLongTime'),
            cleanLowGpu:
              jobs.find((job: Job) => job.name === 'clean-low-gpu-util-job') ||
              form.getValues('cleanLowGpu'),
            cleanWaitingJupyter:
              jobs.find((job: Job) => job.name === 'clean-waiting-jupyter') ||
              form.getValues('cleanWaitingJupyter'),
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
    }
    loadJobSchedule()
  }, [form, t])

  const onSubmitLongTime = async () => {
    const data = form.getValues('cleanLongTime')
    try {
      const res = await apiJobScheduleChangeAdmin(data)
      if (res.code === 0) {
        toast.success(t('cronPolicy.longTimeSuccess'))
      } else {
        toast.error(t('cronPolicy.longTimeError') + res.msg)
      }
    } catch (error) {
      toast.error(t('cronPolicy.longTimeError') + error)
    }
  }

  const onSubmitLowGpu = async () => {
    const data = form.getValues('cleanLowGpu')
    try {
      const res = await apiJobScheduleChangeAdmin(data)
      if (res.code === 0) {
        toast.success(t('cronPolicy.lowGpuSuccess'))
      } else {
        toast.error(t('cronPolicy.lowGpuError') + res.msg)
      }
    } catch (error) {
      toast.error(t('cronPolicy.lowGpuError') + error)
    }
  }

  const onSubmitWaitingJupyter = async () => {
    const data = form.getValues('cleanWaitingJupyter')
    try {
      const res = await apiJobScheduleChangeAdmin(data)
      if (res.code === 0) {
        toast.success(t('cronPolicy.jupyterSuccess'))
      } else {
        toast.error(t('cronPolicy.jupyterError') + res.msg)
      }
    } catch (error) {
      toast.error(t('cronPolicy.jupyterError') + error)
    }
  }

  const runJob = async () => {}

  const confirmJobRun = async () => {
    try {
      const longTimeSuspend = form.getValues('cleanLongTime.suspend')
      const lowGpuSuspend = form.getValues('cleanLowGpu.suspend')
      const waitingJupyterSuspend = form.getValues('cleanWaitingJupyter.suspend')

      const longTimeData = form.getValues('cleanLongTime.configs')
      const lowGpuData = form.getValues('cleanLowGpu.configs')
      const waitingJupyterData = form.getValues('cleanWaitingJupyter.configs')

      const promises = []

      if (!longTimeSuspend) {
        promises.push(
          apiAdminLongTimeRunningJobsCleanup({
            batchDays: Number(longTimeData.batchDays),
            interactiveDays: Number(longTimeData.interactiveDays),
          })
        )
      } else {
        promises.push(Promise.resolve(null))
      }

      if (!lowGpuSuspend) {
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

      if (!waitingJupyterSuspend) {
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
                  <h3 className="mb-4 font-semibold">{t('cronPolicy.longTimeTitle')}</h3>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanLongTime.suspend"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                          </FormControl>
                          <span>{t('cronPolicy.suspend')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLongTime.schedule"
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
                    <Button variant="secondary" onClick={form.handleSubmit(onSubmitLongTime)}>
                      {t('cronPolicy.longTimeUpdate')}
                    </Button>
                  </div>
                </div>

                <div className="rounded-md border p-4">
                  <h3 className="mb-4 font-semibold">{t('cronPolicy.lowGpuTitle')}</h3>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.suspend"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                          </FormControl>
                          <span>{t('cronPolicy.suspend')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanLowGpu.schedule"
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
                    <Button variant="secondary" onClick={form.handleSubmit(onSubmitLowGpu)}>
                      {t('cronPolicy.lowGpuUpdate')}
                    </Button>
                  </div>
                </div>

                <div className="rounded-md border p-4">
                  <h3 className="mb-4 font-semibold">{t('cronPolicy.jupyterTitle')}</h3>
                  <div className="flex flex-wrap gap-4">
                    <FormField
                      control={form.control}
                      name="cleanWaitingJupyter.suspend"
                      render={({ field }) => (
                        <FormItem className="flex items-center space-x-2">
                          <FormControl>
                            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                          </FormControl>
                          <span>{t('cronPolicy.suspend')}</span>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cleanWaitingJupyter.schedule"
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
                    <Button variant="secondary" onClick={form.handleSubmit(onSubmitWaitingJupyter)}>
                      {t('cronPolicy.jupyterUpdate')}
                    </Button>
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
