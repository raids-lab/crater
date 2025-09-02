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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { addHours, format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { CalendarClock, InfoIcon, Lock, UnlockIcon } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { IJobInfo, apiJobLock, apiJobUnlock } from '@/services/api/vcjob'

// Moved Zod schema to component
const getFormSchema = (t: (key: string) => string) =>
  z.object({
    isPermanent: z.boolean().default(false),
    days: z.coerce.number().min(0, t('durationDialog.form.days.min')).default(0),
    hours: z.coerce
      .number()
      .min(0, t('durationDialog.form.hours.min'))
      .max(23, t('durationDialog.form.hours.max'))
      .default(0),
  })

type FormValues = z.infer<ReturnType<typeof getFormSchema>>

interface DurationDialogProps {
  jobs: IJobInfo[]
  open: boolean
  setOpen: (open: boolean) => void
  onSuccess?: () => void
  setExtend?: boolean
  defaultDays?: number
  defaultHours?: number
}

export function DurationDialog({
  jobs,
  open,
  setOpen,
  onSuccess,
  setExtend,
  defaultDays,
  defaultHours,
}: DurationDialogProps) {
  const { t } = useTranslation()
  const [expirationDate, setExpirationDate] = useState<Date | null>(null)
  const allLocked = jobs.length > 0 && jobs.every((job) => job.locked)
  const allUnlocked = jobs.length > 0 && jobs.every((job) => !job.locked)
  const mixedState = !allLocked && !allUnlocked
  const isExtendDialogOpen = setExtend && allLocked

  // Initialize form with translated schema
  const form = useForm<FormValues>({
    resolver: zodResolver(getFormSchema(t)),
    defaultValues: {
      isPermanent: false,
      days: defaultDays ?? 0,
      hours: defaultHours ?? 0,
    },
  })

  // 先声明，避免“在赋值前使用了变量”
  const calculateExpirationDate = useCallback(
    (originExpiredTime: string | undefined, values: FormValues) => {
      const { days, hours, isPermanent } = values

      if (isPermanent) {
        setExpirationDate(null)
        return
      }

      if (days > 0 || hours > 0) {
        const now = new Date()
        let result = new Date(now)
        if (originExpiredTime !== undefined && isExtendDialogOpen) {
          result = new Date(originExpiredTime)
        }
        if (days > 0) result = addHours(result, days * 24)
        if (hours > 0) result = addHours(result, hours)
        setExpirationDate(result)
      } else {
        setExpirationDate(null)
      }
    },
    [isExtendDialogOpen]
  )

  // 打开时用默认值重置，并展示表单值
  useEffect(() => {
    if (!open) return
    const next = {
      isPermanent: false,
      days: defaultDays ?? 0,
      hours: defaultHours ?? 0,
    }
    form.reset(next)
  }, [open, defaultDays, defaultHours, form])

  // 打开对话框时用默认值重置，并计算一次预览
  useEffect(() => {
    if (open) {
      const days = defaultDays ?? 0
      const hours = defaultHours ?? 0
      form.reset({ isPermanent: false, days, hours })
      setExpirationDate(null)
      setTimeout(() => {
        calculateExpirationDate(jobs[0]?.lockedTimestamp, {
          isPermanent: false,
          days,
          hours,
        } as FormValues)
      }, 0)
    }
  }, [open, defaultDays, defaultHours, form, jobs, calculateExpirationDate])

  const handleFieldChange = useCallback(() => {
    setTimeout(() => {
      const values = form.getValues()
      calculateExpirationDate(jobs[0]?.lockedTimestamp, values)
    }, 0)
  }, [form, jobs, calculateExpirationDate])

  // Use React Query for lock mutation
  const lockMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const { isPermanent, days, hours } = values
      const promises = jobs.map((job) => {
        const payload = {
          name: job.jobName,
          isPermanent,
          days: days || 0,
          hours: hours || 0,
          minutes: 0,
        }
        return apiJobLock(payload)
      })
      return Promise.all(promises)
    },
    onSuccess: () => {
      toast.success(t('durationDialog.toast.lockSuccess', { count: jobs.length }))
      setOpen(false)
      onSuccess?.()
    },
    onError: () => {
      toast.error(t('durationDialog.toast.lockError'))
    },
  })

  // Use React Query for unlock mutation
  const unlockMutation = useMutation({
    mutationFn: async () => {
      const promises = jobs.map((job) => apiJobUnlock(job.jobName))
      return Promise.all(promises)
    },
    onSuccess: () => {
      toast.success(t('durationDialog.toast.unlockSuccess', { count: jobs.length }))
      setOpen(false)
      onSuccess?.()
    },
    onError: () => {
      toast.error(t('durationDialog.toast.unlockError'))
    },
  })

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      const days = defaultDays ?? 0
      const hours = defaultHours ?? 0
      form.reset({
        isPermanent: false,
        days,
        hours,
      })
      setExpirationDate(null)
      // 预览计算一次
      setTimeout(() => {
        calculateExpirationDate(jobs[0]?.lockedTimestamp, {
          isPermanent: false,
          days,
          hours,
        } as FormValues)
      }, 0)
    }
  }, [open, defaultDays, defaultHours, form, jobs, calculateExpirationDate])

  // Submit form
  async function onSubmit(values: FormValues) {
    const { isPermanent, days, hours } = values

    if (!isPermanent && !(days > 0 || hours > 0)) {
      toast.error(t('durationDialog.toast.noDuration'))
      return
    }

    lockMutation.mutate(values)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {allLocked && !isExtendDialogOpen
              ? t('durationDialog.title.unlock')
              : t('durationDialog.title.lock')}
          </DialogTitle>
          <DialogDescription>
            {allLocked && !isExtendDialogOpen
              ? t('durationDialog.description.unlock', { count: jobs.length })
              : t('durationDialog.description.lock', { count: jobs.length })}
          </DialogDescription>
        </DialogHeader>

        {mixedState ? (
          <div className="py-4">
            <Card className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950">
              <CardContent className="flex items-start gap-2 pt-6">
                <InfoIcon className="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-500" />
                <div>
                  <p className="font-medium">{t('durationDialog.card.title')}</p>
                  <p className="text-muted-foreground text-sm">
                    {t('durationDialog.card.description')}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : allLocked && !isExtendDialogOpen ? (
          <Button
            onClick={() => unlockMutation.mutate()}
            className="w-full"
            variant="outline"
            disabled={unlockMutation.isPending}
          >
            <UnlockIcon className="mr-2 h-4 w-4" />
            {unlockMutation.isPending
              ? t('durationDialog.button.unlocking')
              : t('durationDialog.button.unlock', { count: jobs.length })}
          </Button>
        ) : (
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="isPermanent"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start space-y-0 space-x-3 rounded-md border p-4">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        onBlur={field.onBlur}
                        onClick={() => {
                          setTimeout(handleFieldChange, 0)
                        }}
                      />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel className="text-base">
                        {t('durationDialog.form.permanentLock')}
                      </FormLabel>
                      <p className="text-muted-foreground text-sm">
                        {t('durationDialog.form.permanentLockDescription')}
                      </p>
                    </div>
                  </FormItem>
                )}
              />

              {!form.watch('isPermanent') && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="days"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('durationDialog.form.days.text')}</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min="0"
                              placeholder="0"
                              {...field}
                              onChange={(e) => {
                                field.onChange(e)
                                handleFieldChange()
                              }}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="hours"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('durationDialog.form.hours.text')}</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min="0"
                              max="23"
                              placeholder="0"
                              {...field}
                              onChange={(e) => {
                                field.onChange(e)
                                handleFieldChange()
                              }}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  {expirationDate && (
                    <Card className="border-dashed">
                      <CardContent>
                        <div className="text-muted-foreground flex items-center gap-2">
                          <CalendarClock className="h-4 w-4" />
                          <span>{t('durationDialog.form.expirationPreview')}</span>
                        </div>
                        <div className="mt-2">
                          <p className="text-lg font-medium">
                            {format(expirationDate, 'yyyy年MM月dd日 HH:mm', {
                              locale: zhCN,
                            })}
                          </p>
                          <p className="text-muted-foreground mt-1 text-sm">
                            {t('durationDialog.form.currentTime')}:{' '}
                            {format(new Date(), 'yyyy年MM月dd日 HH:mm', {
                              locale: zhCN,
                            })}
                          </p>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </>
              )}

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setOpen(false)}
                  disabled={lockMutation.isPending}
                >
                  {t('common.cancel')}
                </Button>
                <Button type="submit" disabled={lockMutation.isPending}>
                  <Lock className="mr-2 h-4 w-4" />
                  {lockMutation.isPending
                    ? t('durationDialog.button.locking')
                    : t('durationDialog.button.lock')}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        )}
      </DialogContent>
    </Dialog>
  )
}
