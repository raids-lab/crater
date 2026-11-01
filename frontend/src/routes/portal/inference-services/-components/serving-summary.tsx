import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { resourceSummary } from './serving-serialization'
import type { ServingFormState } from './use-serving-form'

export function ServingSummary({ state }: { state: ServingFormState }) {
  const { values, t, preview, previewCurrent } = state
  const summary = resourceSummary(values)
  return (
    <Card className="xl:sticky xl:top-4">
      <CardHeader>
        <CardTitle>{t('kthena.topology.summary')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 text-sm">
        <p className="font-medium break-all">
          {values.name || '—'} · {values.backendType}
        </p>
        <p>
          {values.layout === 'combined'
            ? t('kthena.topology.combined')
            : t('kthena.topology.disaggregated')}{' '}
          · {values.replicas} {t('kthena.topology.groups')}
        </p>
        {values.roles.map((r) => (
          <p key={r.name}>
            {r.name}: {r.instances} × {r.nodesPerInstance} Pods · TP {r.tensorParallel} / PP{' '}
            {r.pipelineParallel}
          </p>
        ))}
        <p>{summary.pods} Pods</p>
        <dl className="grid gap-2">
          {Object.entries(summary.resources)
            .filter(([, v]) => v > 0)
            .map(([name, value]) => (
              <div key={name} className="flex justify-between gap-2">
                <dt className="break-all">{name}</dt>
                <dd>{name === 'memory' ? `${value / 1024 ** 3} GiB` : value}</dd>
              </div>
            ))}
        </dl>
        {previewCurrent && preview && (
          <p>
            {preview.queue} ·{' '}
            {t(preview.queued ? 'kthena.topology.queued' : 'kthena.topology.admissible')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
