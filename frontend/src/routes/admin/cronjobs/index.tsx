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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { AlarmClockIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import TipBadge from '@/components/badge/tip-badge'

import { apiAdminGetGpuAnalysisStatus } from '@/services/api/system-config'
import {
  CronJobConfig,
  CronJobConfigStatus,
  apiAdminCronJobConfigStatus,
  apiJobScheduleAdmin,
} from '@/services/api/vcjob'

import { cn } from '@/lib/utils'

import CronJobCard from './-components/cronjob-card'
import CronJobRecordsTable from './-components/cronjob-records-table'

export const Route = createFileRoute('/admin/cronjobs/')({
  component: CronPolicy,
  loader: () => ({ crumb: t('navigation.cronPolicy') }),
})

const JOB_CONFIGS = [
  {
    jobId: 'clean-long-time-job',
    jobName: 'cronPolicy.longTimeTitle',
    jobType: 'cleaner_function',
  },
  {
    jobId: 'clean-low-gpu-util-job',
    jobName: 'cronPolicy.lowGpuTitle',
    jobType: 'cleaner_function',
  },
  {
    jobId: 'clean-waiting-jupyter',
    jobName: 'cronPolicy.jupyterTitle',
    jobType: 'cleaner_function',
  },
  {
    jobId: 'clean-waiting-custom',
    jobName: 'cronPolicy.customTitle',
    jobType: 'cleaner_function',
  },
  {
    jobId: 'trigger-gpu-analysis-job',
    jobName: 'cronPolicy.gpuAnalysisTitle',
    jobType: 'patrol_function',
  },
]

function CronPolicy({ className }: { className?: string }) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('cleaner_function')

  const { data: gpuStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'gpu-status'],
    queryFn: () => apiAdminGetGpuAnalysisStatus().then((res) => res.data),
    staleTime: 1000 * 60 * 5,
  })

  const showGpuAnalysis = gpuStatus?.enabled ?? false

  const jobsQuery = useQuery({
    queryKey: ['admin', 'cronjobs', 'configs'],
    queryFn: async () => {
      const res = await apiJobScheduleAdmin()
      const jobs = res.data?.configs || []

      const jobMap: Record<string, CronJobConfig> = {}
      jobs.forEach((job: CronJobConfig) => {
        jobMap[job.name] = job
      })

      return jobMap
    },
    refetchInterval: 5000,
  })

  const statusQuery = useQuery({
    queryKey: ['admin', 'cronjobs', 'status'],
    queryFn: async () => {
      const res = await apiAdminCronJobConfigStatus({
        name: JOB_CONFIGS.map((job) => job.jobId),
      })
      return res.data || {}
    },
    refetchInterval: 5000,
  })

  const cleanerJobs = JOB_CONFIGS.filter((job) => job.jobType === 'cleaner_function')
  const patrolJobs = JOB_CONFIGS.filter((job) => job.jobType === 'patrol_function')

  const tabToJobNames: Record<string, string[]> = {
    cleaner_function: cleanerJobs.map((j) => j.jobId),
    patrol_function: patrolJobs.map((j) => j.jobId),
  }

  const renderJobCards = (jobs: typeof JOB_CONFIGS) => {
    if (jobsQuery.isLoading) {
      return (
        <div className="space-y-4">
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      )
    }

    const jobData = jobsQuery.data || {}

    return (
      <div className="grid gap-4 md:grid-cols-2">
        {jobs.map((job) => {
          const config = jobData[job.jobId]
          if (!config) return null

          const currentStatus =
            (statusQuery.data?.[job.jobId]?.status as CronJobConfigStatus) || config.status

          return (
            <CronJobCard
              key={job.jobId}
              jobId={job.jobId}
              jobName={job.jobName}
              jobType={job.jobType}
              status={currentStatus}
              spec={config.spec}
              params={config.config as Record<string, number | string | string[]>}
              onUpdate={() => {
                jobsQuery.refetch()
                statusQuery.refetch()
              }}
            />
          )
        })}
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <Card className="flex flex-col">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5">
              <AlarmClockIcon className="text-primary" />
              {t('cronPolicy.title')}
              <TipBadge />
            </CardTitle>
            <TabsList
              className={cn('grid w-full', showGpuAnalysis ? 'grid-cols-2' : 'grid-cols-1')}
            >
              <TabsTrigger value="cleaner_function">{t('cronPolicy.cleanerJobsTitle')}</TabsTrigger>
              {showGpuAnalysis && (
                <TabsTrigger value="patrol_function">{t('cronPolicy.patrolJobsTitle')}</TabsTrigger>
              )}
            </TabsList>
          </CardHeader>
          <CardContent className="p-6">
            <TabsContent value="cleaner_function" className="mt-0">
              {renderJobCards(cleanerJobs)}
            </TabsContent>
            {showGpuAnalysis && (
              <TabsContent value="patrol_function" className="mt-0">
                {renderJobCards(patrolJobs)}
              </TabsContent>
            )}
          </CardContent>
        </Card>
      </Tabs>
      <CronJobRecordsTable filteredJobNames={tabToJobNames[activeTab]} />
    </div>
  )
}
