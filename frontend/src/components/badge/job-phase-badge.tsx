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
import { useTranslation } from 'react-i18next'

import {
  JobDisplayPhase,
  JobDisplayPhaseValue,
  JobPhase,
  PodGroupPhase,
  encodeJobFilterPhase,
  getJobDisplayPhase,
} from '@/services/api/vcjob'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export const jobPhases = [
  {
    value: 'Prequeue',
    color: '#8b5cf6',
  },
  {
    value: 'Pending',
    color: '#a855f7',
  },
  {
    value: 'Aborting',
    color: '#ec4899',
  },
  {
    value: 'Aborted',
    color: '#ec4899',
  },
  {
    value: 'Running',
    color: '#3b82f6',
  },
  {
    value: 'Restarting',
    color: '#f43f5e',
  },
  {
    value: 'Completing',
    color: '#10b981',
  },
  {
    value: 'Completed',
    color: '#10b981',
  },
  {
    value: 'Terminating',
    color: '#f97316',
  },
  {
    value: 'Terminated',
    color: '#f97316',
  },
  {
    value: 'Deleted',
    color: '#f97316',
  },
  {
    value: 'Freed',
    color: '#f97316',
  },
  {
    value: 'Failed',
    color: '#ef4444',
  },
].map((phase) => ({
  ...phase,
  get label() {
    return phase.value === JobPhase.Pending
      ? t('jobs.statuses.waiting.label')
      : getJobPhaseLabel(phase.value as JobPhase).label
  },
}))

// Candidate sets differ by job implementation; all display text comes from the existing resolver.
export function getJobFilterOptions() {
  const phases: JobDisplayPhaseValue[] = [JobPhase.Init]
  for (const { value } of jobPhases) {
    if (value === JobPhase.Prequeue) continue
    phases.push(value as JobPhase)
    if (value === JobPhase.Pending) phases.push(JobDisplayPhase.Inqueue, JobDisplayPhase.Starting)
  }
  return phases.map((phase) => ({
    value: encodeJobFilterPhase(phase),
    label: getJobPhaseLabel(phase).label,
  }))
}

export function getJobPhaseLabel(phase: JobDisplayPhaseValue): PhaseBadgeData {
  switch (phase) {
    case JobPhase.Prequeue:
      return {
        label: getJobPhaseLabel(JobPhase.Pending).label,
        color: 'text-highlight-violet bg-highlight-violet/20',
        description: t('jobs.statuses.prequeue.description'),
      }
    case JobPhase.Pending:
      return {
        label: t('jobs.statuses.pending.label'),
        color: 'text-highlight-purple bg-highlight-purple/20',
        description: t('jobs.statuses.pending.description'),
      }
    case JobDisplayPhase.Inqueue:
      return {
        label: t('jobs.statuses.inqueue.label'),
        color: 'text-highlight-indigo bg-highlight-indigo/20',
        description: t('jobs.statuses.inqueue.description'),
      }
    case JobDisplayPhase.Starting:
      return {
        label: t('jobs.statuses.starting.label'),
        color: 'text-highlight-sky bg-highlight-sky/20',
        description: t('jobs.statuses.starting.description'),
      }
    case JobPhase.Init:
      return {
        label: t('jobs.statuses.created.label'),
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: t('jobs.statuses.created.description'),
      }
    case JobPhase.Aborting:
      return {
        label: t('jobs.statuses.aborting.label'),
        color: 'text-highlight-pink bg-highlight-pink/20',
        description: t('jobs.statuses.aborting.description'),
      }
    case JobPhase.Aborted:
      return {
        label: t('jobs.statuses.aborted.label'),
        color: 'text-highlight-pink bg-highlight-pink/20',
        description: t('jobs.statuses.aborted.description'),
      }
    case JobPhase.Running:
      return {
        label: t('jobs.statuses.running.label'),
        color: 'text-highlight-blue bg-highlight-blue/20',
        description: t('jobs.statuses.running.description'),
      }
    case JobPhase.Restarting:
      return {
        label: t('jobs.statuses.restarting.label'),
        color: 'text-highlight-rose bg-highlight-rose/20',
        description: t('jobs.statuses.restarting.description'),
      }
    case JobPhase.Completing:
      return {
        label: t('jobs.statuses.completing.label'),
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: t('jobs.statuses.completing.description'),
      }
    case JobPhase.Completed:
      return {
        label: t('jobs.statuses.completed.label'),
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: t('jobs.statuses.completed.description'),
      }
    case JobPhase.Terminating:
      return {
        label: t('jobs.statuses.terminating.label'),
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: t('jobs.statuses.terminating.description'),
      }
    case JobPhase.Terminated:
      return {
        label: t('jobs.statuses.terminated.label'),
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: t('jobs.statuses.terminated.description'),
      }
    case JobPhase.Failed:
      return {
        label: t('jobs.statuses.failed.label'),
        color: 'text-highlight-red bg-highlight-red/20',
        description: t('jobs.statuses.failed.description'),
      }
    case JobPhase.Deleted:
      return {
        label: t('jobs.statuses.deleted.label'),
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: t('jobs.statuses.deleted.description'),
      }
    case JobPhase.Freed:
      return {
        label: t('jobs.statuses.freed.label'),
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: t('jobs.statuses.freed.description'),
      }
    default:
      return {
        label: t('jobs.statuses.unknown.label'),
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: t('jobs.statuses.unknown.description'),
      }
  }
}

const JobPhaseLabel = ({
  jobPhase,
  podGroupPhase,
}: {
  jobPhase: JobPhase
  podGroupPhase?: PodGroupPhase
}) => {
  useTranslation()
  return (
    <PhaseBadge
      phase={getJobDisplayPhase(jobPhase, podGroupPhase)}
      getPhaseLabel={getJobPhaseLabel}
    />
  )
}

export default JobPhaseLabel
