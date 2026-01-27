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
import { PlayIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
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

  const renderParamInput = (key: string, value: number | string | string[]) => {
    if (HIDDEN_PARAMS.includes(key)) {
      return null
    }

    if (Array.isArray(value)) {
      return null
    }

    const isNumber = typeof value === 'number'
    return (
      <div key={key} className="flex flex-col gap-2">
        <Label className="text-sm">{t(`cronPolicy.${key}`)}</Label>
        <Input
          type={isNumber ? 'number' : 'text'}
          value={value as string | number}
          onChange={(e) =>
            handleParamChange(key, isNumber ? Number(e.target.value) : e.target.value)
          }
          className="w-32 font-mono"
          step={isNumber && value < 1 ? '0.1' : '1'}
        />
      </div>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="text-lg">{t(jobName)}</span>
          <CronJobStatusBadge status={status} />
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-2">
          <Switch checked={enabled} onCheckedChange={setEnabled} />
          <Label className="text-sm">{t('cronPolicy.enable')}</Label>
        </div>

        <div className="flex flex-col gap-2">
          <Label className="text-sm">{t('cronPolicy.schedule')}</Label>
          <Input
            value={cronSpec}
            onChange={(e) => setCronSpec(e.target.value)}
            className="font-mono"
            placeholder="*/5 * * * *"
          />
        </div>

        {Object.keys(jobParams).filter((key) => !HIDDEN_PARAMS.includes(key)).length > 0 && (
          <div className="flex flex-wrap gap-4">
            {Object.entries(jobParams)
              .filter(([key]) => !HIDDEN_PARAMS.includes(key))
              .map(([key, value]) => renderParamInput(key, value))}
          </div>
        )}
      </CardContent>
      <CardFooter className="flex gap-2">
        <LoadableButton
          variant="secondary"
          isLoading={updateMutation.isPending}
          isLoadingText={t('cronPolicy.updating')}
          onClick={() => updateMutation.mutate()}
        >
          {t('cronPolicy.updateParams')}
        </LoadableButton>
        <LoadableButton
          variant="default"
          isLoading={executeMutation.isPending}
          isLoadingText={t('cronPolicy.executing')}
          onClick={() => executeMutation.mutate()}
          disabled={jobType === 'patrol_function'}
        >
          <PlayIcon className="mr-2 h-4 w-4" />
          {t('cronPolicy.executeNow')}
        </LoadableButton>
      </CardFooter>
    </Card>
  )
}
