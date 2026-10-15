/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { type TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'

import { type TensorboardStatus, type TensorboardStatusReason } from '@/services/api/tensorboard'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

const statusReasonTranslationKeys: Record<TensorboardStatusReason, string> = {
  deployment_failed: 'tensorboard.statusReason.deploymentFailed',
  deployment_starting: 'tensorboard.statusReason.deploymentStarting',
  waiting_for_schedule: 'tensorboard.statusReason.waitingForSchedule',
  pod_starting: 'tensorboard.statusReason.podStarting',
  job_failed: 'tensorboard.statusReason.jobFailed',
  runtime_expired: 'tensorboard.statusReason.runtimeExpired',
  ready: 'tensorboard.statusReason.ready',
}

export const getTensorboardStatusDescription = (
  status: TensorboardStatus,
  statusReason: TensorboardStatusReason | undefined,
  t: TFunction
) => {
  const translationKey = statusReason ? statusReasonTranslationKeys[statusReason] : undefined
  if (translationKey) {
    return t(translationKey)
  }

  switch (status) {
    case 'pending':
      return t('jobs.statuses.pending.description')
    case 'starting':
      return t('tensorboard.status.startingDescription')
    case 'ready':
      return t('tensorboard.status.readyDescription')
    case 'failed':
      return t('tensorboard.status.failedDescription')
    case 'expired':
      return t('tensorboard.status.expiredDescription')
    default:
      return t('tensorboard.status.unknownDescription')
  }
}

export const getTensorboardStatusLabel = (
  status: TensorboardStatus,
  statusReason: TensorboardStatusReason | undefined,
  t: TFunction
): PhaseBadgeData => {
  switch (status) {
    case 'pending':
      return {
        label: t('jobs.statuses.pending.label'),
        color: 'text-highlight-purple bg-highlight-purple/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
    case 'starting':
      return {
        label: t('tensorboard.status.starting'),
        color: 'text-highlight-purple bg-highlight-purple/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
    case 'ready':
      return {
        label: t('tensorboard.status.ready'),
        color: 'text-highlight-blue bg-highlight-blue/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
    case 'failed':
      return {
        label: t('tensorboard.status.failed'),
        color: 'text-highlight-red bg-highlight-red/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
    case 'expired':
      return {
        label: t('tensorboard.status.expired'),
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
    default:
      return {
        label: t('tensorboard.status.unknown'),
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: getTensorboardStatusDescription(status, statusReason, t),
      }
  }
}

interface TensorboardStatusBadgeProps {
  status: TensorboardStatus
  statusReason?: TensorboardStatusReason
}

const TensorboardStatusBadge = ({ status, statusReason }: TensorboardStatusBadgeProps) => {
  const { t } = useTranslation()

  return (
    <PhaseBadge
      phase={status}
      getPhaseLabel={(phase) => getTensorboardStatusLabel(phase, statusReason, t)}
    />
  )
}

export default TensorboardStatusBadge
