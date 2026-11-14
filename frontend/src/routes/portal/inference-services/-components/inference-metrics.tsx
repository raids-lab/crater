import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

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
  const [scope, setScope] = useState('overall')
  const roles = Object.keys(data?.roles ?? {})
  const selectedRole = roles.find((role) => role === scope)
  const metrics = selectedRole ? data?.roles[selectedRole] : data?.metrics
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
      <CardHeader className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1.5">
          <CardTitle>{t('kthena.metrics.title')}</CardTitle>
          <CardDescription>
            {t(selectedRole ? 'kthena.metrics.roleBoundary' : 'kthena.topology.metricBoundary')}
          </CardDescription>
        </div>
        {roles.length > 0 && (
          <Select value={selectedRole ?? 'overall'} onValueChange={setScope}>
            <SelectTrigger className="w-40" aria-label={t('kthena.metrics.scope')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="overall">{t('kthena.metrics.overall')}</SelectItem>
              {roles.map((role) => (
                <SelectItem key={role} value={role}>
                  {role}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
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
          <div
            role="alert"
            className="bg-muted/50 flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"
          >
            <span>{t('kthena.metrics.unavailable')}</span>
            <Button variant="ghost" size="sm" onClick={() => void refetch()}>
              {t('common.refresh')}
            </Button>
          </div>
        )}
        <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {entries.map(([key, value]) => (
            <div key={key} className="rounded-md border p-3">
              <dt className="text-muted-foreground text-sm">{t(`kthena.metrics.${key}`)}</dt>
              <dd className="mt-2 text-xl tabular-nums">{isPending ? '…' : value}</dd>
            </div>
          ))}
        </dl>
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
