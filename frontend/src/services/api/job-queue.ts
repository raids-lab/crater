import { apiV1Get } from '@/services/client'
import type { IResponse } from '@/services/types'

import type { IUserInfo, JobType } from './vcjob'

export interface QueueJob {
  name: string
  jobName: string
  jobType: JobType
  accountId: number
  account: string
  userInfo: IUserInfo
  resources?: Record<string, string>
  createdAt: string
  phase: 'Init' | 'Pending' | 'Inqueue'
  reason?: 'QueueQuotaExceeded' | 'BlockedByTimedOutJob' | 'ExceedsQueueCapability'
}

export interface QueueSnapshot {
  observedAt: string
  jobs: QueueJob[]
  reasonsAvailable: boolean
}

export const apiJobQueue = (admin: boolean, mine: boolean, signal?: AbortSignal) =>
  apiV1Get<IResponse<QueueSnapshot>>(admin ? 'admin/vcjobs/queue' : 'vcjobs/queue', {
    searchParams: mine && !admin ? { mine: 'true' } : undefined,
    signal,
  })
