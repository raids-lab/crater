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
import { Link, linkOptions } from '@tanstack/react-router'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { LockIcon } from 'lucide-react'

import { IJobInfo, JobStatus, getJobStateType } from '@/services/api/vcjob'

import useIsAdmin from '@/hooks/use-admin'

import { cn } from '@/lib/utils'

import TipBadge from '../badge/tip-badge'
import SimpleTooltip from './simple-tooltip'

interface JobNameCellProps {
  jobInfo: Pick<IJobInfo, 'jobName' | 'name'> &
    Partial<Pick<IJobInfo, 'status' | 'locked' | 'permanentLocked' | 'lockedTimestamp'>>
  linkable?: boolean
  nameClassName?: string
}

// 格式化锁定日期为中文格式
const formatLockDate = (timestamp?: string) => {
  const date = new Date(timestamp ?? Date.now())
  return format(date, 'M月d日 HH:mm', { locale: zhCN })
}

const adminJobLinkOptions = linkOptions({
  to: '/admin/jobs/$name',
  params: { name: '' },
  search: { tab: '' },
})

const portalJobLinkOptions = linkOptions({
  to: '/portal/jobs/detail/$name',
  params: { name: '' },
  search: { tab: '' },
})

export const JobNameCell = ({ jobInfo, linkable = true, nameClassName }: JobNameCellProps) => {
  const isAdminView = useIsAdmin()
  const name = (
    <div className="flex min-w-0 flex-row items-center">
      <p className={cn('max-w-36 truncate', nameClassName)} title={jobInfo.name}>
        {jobInfo.name}
      </p>
      {jobInfo.locked && <LockIcon className="text-muted-foreground ml-1 h-4 w-4 shrink-0" />}
    </div>
  )

  if (!linkable) return name

  const showEvents =
    jobInfo.status !== undefined && getJobStateType(jobInfo.status) === JobStatus.NotStarted

  return (
    <SimpleTooltip
      tooltip={
        <div className="flex flex-row items-center justify-between gap-1.5">
          <p className="text-xs">查看 {jobInfo.jobName} 详情</p>
          {jobInfo.locked && (
            <TipBadge
              title={
                jobInfo.permanentLocked
                  ? '长期锁定中'
                  : `锁定至 ${formatLockDate(jobInfo.lockedTimestamp)}`
              }
              className="text-primary bg-primary-foreground z-10"
            />
          )}
        </div>
      }
    >
      <Link
        {...(isAdminView ? adminJobLinkOptions : portalJobLinkOptions)}
        preload="intent"
        params={{ name: jobInfo.jobName }}
        search={{ tab: showEvents ? 'event' : undefined }}
      >
        {name}
      </Link>
    </SimpleTooltip>
  )
}
