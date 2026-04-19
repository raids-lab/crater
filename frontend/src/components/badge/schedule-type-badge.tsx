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
import { t } from 'i18next'

import { ScheduleType } from '@/services/api/vcjob'

import { PhaseBadge } from './phase-badge'

const getScheduleTypeLabel = (scheduleType: ScheduleType) => {
  switch (scheduleType) {
    case ScheduleType.Backfill:
      return {
        label: t('jobs.scheduleTypes.backfill'),
        color: 'text-highlight-amber bg-highlight-amber/10',
        description: t('jobs.scheduleTypes.backfillDescription'),
      }
    default:
      return {
        label: t('jobs.scheduleTypes.normal'),
        color: 'text-highlight-emerald bg-highlight-emerald/10',
        description: t('jobs.scheduleTypes.normalDescription'),
      }
  }
}

const ScheduleTypeLabel = ({ scheduleType }: { scheduleType?: ScheduleType }) => {
  return (
    <PhaseBadge phase={scheduleType ?? ScheduleType.Normal} getPhaseLabel={getScheduleTypeLabel} />
  )
}

export default ScheduleTypeLabel
