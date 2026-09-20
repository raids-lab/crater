import { t } from 'i18next'

import { KthenaService } from '@/services/api/inference'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export type KthenaDisplayState =
  | 'submitted'
  | 'scheduling'
  | 'deploying'
  | 'running'
  | 'degraded'
  | 'failed'

const hasProblemDiagnostics = (service: KthenaService) =>
  service.diagnostics?.some((diagnostic) => {
    const reason = diagnostic.reason?.toLowerCase() ?? ''
    const message = diagnostic.message?.toLowerCase() ?? ''
    return (
      diagnostic.level === 'error' ||
      reason.includes('backoff') ||
      reason.includes('failed') ||
      reason.includes('error') ||
      reason.includes('unhealthy') ||
      message.includes('back-off') ||
      message.includes('readiness probe failed') ||
      message.includes('exit code')
    )
  }) ?? false

const hasErrorDiagnostics = (service: KthenaService) =>
  service.diagnostics?.some((diagnostic) => diagnostic.level === 'error') ?? false

export const getKthenaDisplayState = (service: KthenaService): KthenaDisplayState => {
  const phase = service.phase || 'Pending'
  if (phase === 'Failed') return 'failed'
  if (phase === 'Degraded') return 'degraded'
  // Kubernetes Warning events are historical and can remain attached to a pod
  // after it becomes ready. Keep them in diagnostics, but don't let a
  // transient readiness failure override Kthena's current Ready/Active phase.
  // A live error diagnostic still takes precedence so real runtime failures
  // remain visible in the header.
  if (phase === 'Ready' || phase === 'Active') {
    return hasErrorDiagnostics(service) ? 'degraded' : 'running'
  }
  if (hasProblemDiagnostics(service)) return 'degraded'
  if (phase === 'Pending') return service.resources?.length ? 'scheduling' : 'submitted'
  return 'deploying'
}

export const getKthenaStatusLabel = (state: KthenaDisplayState): PhaseBadgeData => {
  switch (state) {
    case 'submitted':
      return {
        label: t('kthena.state.submitted'),
        color: 'bg-highlight-slate/20 text-highlight-slate',
        description: t('kthena.stateDescription.submitted'),
      }
    case 'scheduling':
      return {
        label: t('kthena.state.scheduling'),
        color: 'bg-highlight-purple/20 text-highlight-purple',
        description: t('kthena.stateDescription.scheduling'),
      }
    case 'deploying':
      return {
        label: t('kthena.state.deploying'),
        color: 'bg-highlight-blue/20 text-highlight-blue',
        description: t('kthena.stateDescription.deploying'),
      }
    case 'running':
      return {
        label: t('kthena.state.running'),
        color: 'bg-highlight-blue/20 text-highlight-blue',
        description: t('kthena.stateDescription.running'),
      }
    case 'degraded':
      return {
        label: t('kthena.state.degraded'),
        color: 'bg-highlight-orange/20 text-highlight-orange',
        description: t('kthena.stateDescription.degraded'),
      }
    case 'failed':
      return {
        label: t('kthena.state.failed'),
        color: 'bg-highlight-red/20 text-highlight-red',
        description: t('kthena.stateDescription.failed'),
      }
  }
}

export default function KthenaStatusBadge({ service }: { service: KthenaService }) {
  return <PhaseBadge phase={getKthenaDisplayState(service)} getPhaseLabel={getKthenaStatusLabel} />
}
