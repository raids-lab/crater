import { CpuIcon } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

import CardTitle from '@/components/label/card-title'

import { quantityValue } from '@/hooks/inference/format'

import { resourceSummary } from './serving-serialization'
import type { ServingFormState } from './use-serving-form'

export function ServingSummary({ state }: { state: ServingFormState }) {
  const { values, t, preview, previewCurrent } = state
  const checked = previewCurrent ? preview : undefined
  const summary = checked
    ? {
        pods: checked.pods,
        resources: Object.fromEntries(
          Object.entries(checked.totalResources).map(([name, value]) => [
            name,
            quantityValue(value),
          ])
        ),
      }
    : resourceSummary(values)
  return (
    <Card className="gap-3 py-4">
      <CardHeader>
        <CardTitle icon={CpuIcon}>
          {t(checked ? 'kthena.topology.summary' : 'kthena.form.summary.estimate')}
        </CardTitle>
      </CardHeader>
      <CardContent className="grid gap-3 text-sm">
        {values.name && <p className="font-medium break-all">{values.name}</p>}
        <p className="text-muted-foreground">
          {values.backendType} ·{' '}
          {values.layout === 'combined'
            ? t('kthena.topology.combined')
            : t('kthena.topology.disaggregated')}{' '}
          · {t('kthena.form.summary.groupCount', { count: values.replicas })}
        </p>
        {summary ? (
          <div className="flex flex-wrap gap-1.5">
            {Object.entries(summary.resources)
              .filter(([, v]) => v > 0)
              .map(([name, value]) => (
                <Badge key={name} variant="secondary" className="font-mono" title={name}>
                  {name === 'cpu'
                    ? `${value} vCPU`
                    : name === 'memory'
                      ? `${value / 1024 ** 3} GiB`
                      : `${name.split('/').at(-1)}: ${value}`}
                </Badge>
              ))}
            <Badge variant="secondary" className="font-mono">
              {summary.pods} {summary.pods === 1 ? 'Pod' : 'Pods'}
            </Badge>
          </div>
        ) : (
          <p className="text-muted-foreground">{t('kthena.form.summary.incomplete')}</p>
        )}
      </CardContent>
    </Card>
  )
}
