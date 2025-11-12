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
import { CronJobConfigStatus, CronJobConfigStatusType } from '@/services/api/vcjob'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export type CronJobStatus = CronJobConfigStatusType

const getCronJobStatusLabel = (status: CronJobStatus): PhaseBadgeData => {
  switch (status) {
    case CronJobConfigStatus.Unknown:
      return {
        label: '未知',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '定时任务状态未知',
      }
    case CronJobConfigStatus.Suspended:
      return {
        label: '已暂停',
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: '定时任务已被暂停，不会自动执行',
      }
    case CronJobConfigStatus.Idle:
      return {
        label: '空闲',
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: '定时任务处于空闲状态，等待下次执行',
      }
    case CronJobConfigStatus.Running:
      return {
        label: '运行中',
        color: 'text-highlight-blue bg-highlight-blue/20',
        description: '定时任务正在执行',
      }
    default:
      return {
        label: '未知',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '定时任务状态未知',
      }
  }
}

const CronJobStatusBadge = ({ status }: { status: string }) => {
  return <PhaseBadge phase={status as CronJobStatus} getPhaseLabel={getCronJobStatusLabel} />
}

export default CronJobStatusBadge
