import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

import { apiGetKthenaMetrics } from '@/services/api/inference'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

export function InferenceMetrics({ name }: { name: string }) {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const { data, error, isPending, refetch } = useQuery({
    queryKey: [...keys.detail(name), 'metrics'],
    queryFn: () => apiGetKthenaMetrics(name).then((res) => res.data),
    refetchInterval: 30000,
  })
  const metrics = data?.metrics
  const format = (value: number | null | undefined, unit = '') =>
    value == null ? '—' : `${value.toLocaleString(undefined, { maximumFractionDigits: 3 })}${unit}`
  const entries = [
    ['requests', format(metrics?.requestsPerSecond, ' req/s')],
    ['tokens', format(metrics?.tokensPerSecond, ' token/s')],
    ['ttft', format(metrics?.ttftP95Seconds, ' s')],
    ['latency', format(metrics?.latencyP95Seconds, ' s')],
    ['waiting', format(metrics?.waitingRequests)],
    ['running', format(metrics?.runningRequests)],
    ['errors', format(metrics?.httpErrorRatio == null ? null : metrics.httpErrorRatio * 100, '%')],
    ['billing', format(data?.billing.billedPoints)],
  ]
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.metrics.title')}</CardTitle>
        <CardDescription>{t('kthena.topology.metricBoundary')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div role="alert">
            {error.message}
            <Button variant="outline" onClick={() => void refetch()}>
              {t('kthena.actions.retry')}
            </Button>
          </div>
        )}
        {data?.monitoringError && (
          <p role="alert" className="text-destructive">
            {t('kthena.metrics.unavailable')}
          </p>
        )}
        <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {entries.map(([key, value]) => (
            <div key={key} className="rounded-md border p-3">
              <dt className="text-muted-foreground text-sm">{t(`kthena.metrics.${key}`)}</dt>
              <dd className="mt-2 text-xl tabular-nums">{isPending ? '…' : value}</dd>
            </div>
          ))}
        </dl>
        {Object.entries(data?.roles ?? {}).map(([role, metrics]) => (
          <details key={role}>
            <summary>
              {role} · {t('kthena.topology.roleMetrics')}
            </summary>
            <dl className="mt-3 grid grid-cols-2 gap-3 text-sm">
              {[
                ['requests', format(metrics.requestsPerSecond, ' req/s')],
                ['tokens', format(metrics.tokensPerSecond, ' token/s')],
                ['ttft', format(metrics.ttftP95Seconds, ' s')],
                ['latency', format(metrics.latencyP95Seconds, ' s')],
                ['waiting', format(metrics.waitingRequests)],
                ['running', format(metrics.runningRequests)],
              ].map(([key, value]) => (
                <div key={key}>
                  <dt>{t(`kthena.metrics.${key}`)}</dt>
                  <dd>{value}</dd>
                </div>
              ))}
            </dl>
          </details>
        ))}
        <p className="text-muted-foreground text-xs">{t('kthena.metrics.missing')}</p>
        {data?.billing.lastSettledAt && (
          <p className="text-muted-foreground text-xs">
            {t('kthena.metrics.settledAt')}: {new Date(data.billing.lastSettledAt).toLocaleString()}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
