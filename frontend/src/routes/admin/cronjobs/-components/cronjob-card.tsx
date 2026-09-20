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
import { CalendarClockIcon, PlayIcon, SlidersHorizontalIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import CronJobStatusBadge from '@/components/badge/cronjob-status-badge'
import LoadableButton from '@/components/button/loadable-button'

import {
  CronJobConfigStatus,
  apiAdminLongTimeRunningJobsCleanup,
  apiAdminLowGPUUsageJobsCleanup,
  apiAdminWaitingCustomJobCancel,
  apiAdminWaitingJupyterJobCancel,
  apiJobScheduleChangeAdmin,
} from '@/services/api/vcjob'

import { cn } from '@/lib/utils'

export interface CronJobCardProps {
  jobId: string
  jobName: string
  jobType: string
  status: CronJobConfigStatus
  spec: string
  params: Record<string, number | string | string[]>
  onUpdate: () => void
}

const HIDDEN_PARAMS = ['jobTypes']

const DAY_PARAMS = ['batchDays', 'interactiveDays']
const MINUTE_PARAMS = ['timeRange', 'waitTime', 'waitMinitues']

interface CleanupResult {
  data: {
    deleted: unknown[]
    reminded: unknown[]
  }
}

const executeJobMap: Record<
  string,
  (params: Record<string, number | string | string[]>) => Promise<CleanupResult>
> = {
  'clean-long-time-job': async (params) => {
    const res = await apiAdminLongTimeRunningJobsCleanup({
      batchDays: params.batchDays as number,
      interactiveDays: params.interactiveDays as number,
    })
    return res
  },
  'clean-low-gpu-util-job': async (params) => {
    const res = await apiAdminLowGPUUsageJobsCleanup({
      timeRange: params.timeRange as number,
      util: params.util as number,
      waitTime: params.waitTime as number,
    })
    return res
  },
  'clean-waiting-jupyter': async (params) => {
    const res = await apiAdminWaitingJupyterJobCancel({
      waitMinutes: params.waitMinitues as number,
    })
    return res
  },
  'clean-waiting-custom': async (params) => {
    const res = await apiAdminWaitingCustomJobCancel({
      waitMinutes: params.waitMinitues as number,
    })
    return res
  },
  'trigger-gpu-analysis-job': async () => {
    throw new Error('GPU analysis job execution not implemented yet')
  },
}

export default function CronJobCard({
  jobId,
  jobName,
  jobType,
  status,
  spec,
  params,
  onUpdate,
}: CronJobCardProps) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(status !== CronJobConfigStatus.Suspended)
  const [cronSpec, setCronSpec] = useState(spec)
  const [jobParams, setJobParams] = useState(params)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const isSuspended = status === CronJobConfigStatus.Suspended
  const visibleParams = Object.entries(jobParams).filter(
    ([key, value]) => !HIDDEN_PARAMS.includes(key) && !Array.isArray(value)
  )
  const hasChanges =
    enabled !== (status !== CronJobConfigStatus.Suspended) ||
    cronSpec !== spec ||
    JSON.stringify(jobParams) !== JSON.stringify(params)

  const updateMutation = useMutation({
    mutationFn: async () => {
      const res = await apiJobScheduleChangeAdmin({
        name: jobId,
        status: enabled ? CronJobConfigStatus.Idle : CronJobConfigStatus.Suspended,
        spec: cronSpec,
        config: jobParams,
      })
      return res
    },
    onSuccess: () => {
      toast.success(t('cronPolicy.updateSuccess'))
      setSettingsOpen(false)
      onUpdate()
    },
    onError: (error: Error) => {
      toast.error(t('cronPolicy.updateError') + error.message)
    },
  })

  const executeMutation = useMutation({
    mutationFn: async () => {
      const executeFunc = executeJobMap[jobId]
      if (!executeFunc) {
        throw new Error('Job execution not implemented')
      }
      return await executeFunc(jobParams)
    },
    onSuccess: (data) => {
      const deleted = data.data.deleted || []
      const reminded = data.data.reminded || []
      const total = deleted.length + reminded.length

      toast.success(
        t('cronPolicy.cleanupSummary', {
          total,
          deleted: deleted.length,
          reminded: reminded.length,
        })
      )
      onUpdate()
    },
    onError: (error: Error) => {
      toast.error(t('cronPolicy.executeError') + error.message)
    },
  })

  const handleParamChange = (key: string, value: number | string) => {
    setJobParams((prev) => ({
      ...prev,
      [key]: value,
    }))
  }

  const handleSettingsOpenChange = (open: boolean) => {
    if (!open && !updateMutation.isPending) {
      setEnabled(status !== CronJobConfigStatus.Suspended)
      setCronSpec(spec)
      setJobParams(params)
    }
    setSettingsOpen(open)
  }

  const getParamUnit = (key: string) => {
    if (DAY_PARAMS.includes(key)) return t('cronPolicy.unitDays')
    if (MINUTE_PARAMS.includes(key)) return t('cronPolicy.unitMinutes')
    if (key === 'util') return '%'
    return ''
  }

  const getScheduleSummary = (value: string) => {
    const everyMinutes = value.match(/^\*\/(\d+) \* \* \* \*$/)
    if (everyMinutes) {
      return t('cronPolicy.scheduleEveryMinutes', { count: Number(everyMinutes[1]) })
    }

    const everyHours = value.match(/^0 \*\/(\d+) \* \* \*$/)
    if (everyHours) {
      return t('cronPolicy.scheduleEveryHours', { count: Number(everyHours[1]) })
    }

    const everyDay = value.match(/^(\d{1,2}) (\d{1,2}) \* \* \*$/)
    if (everyDay) {
      const time = `${everyDay[2].padStart(2, '0')}:${everyDay[1].padStart(2, '0')}`
      return t('cronPolicy.scheduleEveryDay', { time })
    }

    return t('cronPolicy.scheduleCustom')
  }

  const renderParamInput = (key: string, value: number | string | string[]) => {
    if (HIDDEN_PARAMS.includes(key) || Array.isArray(value)) {
      return null
    }

    const isNumber = typeof value === 'number'
    return (
      <div key={key} className="flex min-w-0 flex-col gap-2">
        <Label htmlFor={`${jobId}-${key}`} className="text-sm">
          {t(`cronPolicy.${key}`)}
        </Label>
        <div className="relative">
          <Input
            id={`${jobId}-${key}`}
            type={isNumber ? 'number' : 'text'}
            value={value as string | number}
            onChange={(event) =>
              handleParamChange(key, isNumber ? Number(event.target.value) : event.target.value)
            }
            className={cn('w-full font-mono', getParamUnit(key) && 'pr-14')}
            step={isNumber && value < 1 ? '0.1' : '1'}
          />
          {getParamUnit(key) && (
            <span className="text-muted-foreground pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs">
              {getParamUnit(key)}
            </span>
          )}
        </div>
      </div>
    )
  }

  return (
    <Card
      className={cn(
        'hover:border-primary/30 gap-0 overflow-hidden py-0 shadow-none transition-colors',
        isSuspended && 'bg-muted/20'
      )}
    >
      <CardHeader className="gap-4 p-5">
        <div className="flex min-w-0 items-start justify-between gap-4">
          <div className="min-w-0 space-y-1">
            <CardTitle className="truncate text-base leading-6">{t(jobName)}</CardTitle>
            <p className="text-muted-foreground truncate font-mono text-xs">{jobId}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {hasChanges && <Badge variant="secondary">{t('cronPolicy.unsaved')}</Badge>}
            <CronJobStatusBadge status={status} />
          </div>
        </div>

        <div className="bg-muted/45 flex min-w-0 items-center gap-3 rounded-lg px-3 py-2.5">
          <CalendarClockIcon className="text-primary size-4 shrink-0" />
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{getScheduleSummary(cronSpec)}</p>
            <p className="text-muted-foreground truncate font-mono text-xs">{cronSpec}</p>
          </div>
        </div>

        {visibleParams.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {visibleParams.map(([key, value]) => (
              <Badge
                key={key}
                variant="outline"
                className="bg-background h-7 rounded-md font-normal"
              >
                <span className="text-muted-foreground">{t(`cronPolicy.${key}Short`)}</span>
                <span className="ml-1 font-mono font-medium">
                  {String(value)}
                  {getParamUnit(key)}
                </span>
              </Badge>
            ))}
          </div>
        )}
      </CardHeader>

      <CardFooter className="justify-between gap-3 border-t px-5 py-3">
        <Dialog open={settingsOpen} onOpenChange={handleSettingsOpenChange}>
          <DialogTrigger asChild>
            <Button variant="ghost" size="sm" className="text-muted-foreground -ml-2">
              <SlidersHorizontalIcon />
              {t('cronPolicy.editSettings')}
            </Button>
          </DialogTrigger>
          <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>{t(jobName)}</DialogTitle>
              <DialogDescription>{t('cronPolicy.settingsDescription')}</DialogDescription>
            </DialogHeader>

            <div className="space-y-4 py-2">
              <div className="bg-background flex items-center justify-between gap-4 rounded-lg border px-3 py-2.5">
                <div>
                  <Label htmlFor={`${jobId}-enabled`} className="text-sm font-medium">
                    {t('cronPolicy.enable')}
                  </Label>
                  <p className="text-muted-foreground text-xs">
                    {t('cronPolicy.enableDescription')}
                  </p>
                </div>
                <Switch id={`${jobId}-enabled`} checked={enabled} onCheckedChange={setEnabled} />
              </div>

              <div className="flex flex-col gap-2">
                <Label htmlFor={`${jobId}-schedule`} className="text-sm">
                  {t('cronPolicy.schedule')}
                </Label>
                <Input
                  id={`${jobId}-schedule`}
                  value={cronSpec}
                  onChange={(event) => setCronSpec(event.target.value)}
                  className="font-mono"
                  placeholder="*/5 * * * *"
                />
                <p className="text-muted-foreground text-xs">{getScheduleSummary(cronSpec)}</p>
              </div>

              {visibleParams.length > 0 && (
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {visibleParams.map(([key, value]) => renderParamInput(key, value))}
                </div>
              )}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => handleSettingsOpenChange(false)}>
                {t('cronPolicy.cancel')}
              </Button>
              <LoadableButton
                variant="default"
                isLoading={updateMutation.isPending}
                isLoadingText={t('cronPolicy.updating')}
                onClick={() => updateMutation.mutate()}
                disabled={!hasChanges}
              >
                {t('cronPolicy.saveChanges')}
              </LoadableButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <AlertDialog>
          <AlertDialogTrigger asChild>
            <LoadableButton
              variant="default"
              size="sm"
              isLoading={executeMutation.isPending}
              isLoadingText={t('cronPolicy.executing')}
              disabled={jobType === 'patrol_function'}
            >
              <PlayIcon className="mr-1 size-3.5" />
              {t('cronPolicy.executeNow')}
            </LoadableButton>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('cronPolicy.confirmTitle')}</AlertDialogTitle>
              <AlertDialogDescription>{t('cronPolicy.confirmMessage')}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('cronPolicy.cancel')}</AlertDialogCancel>
              <AlertDialogAction onClick={() => executeMutation.mutate()}>
                {t('cronPolicy.confirm')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardFooter>
    </Card>
  )
}
