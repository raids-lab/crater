import { useAtomValue } from 'jotai'
import { GaugeIcon, ServerIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import GrafanaIframe from '@/components/layout/embed/grafana-iframe'

import type { KthenaRuntimePod } from '@/services/api/inference'

import { configGrafanaJobAtom } from '@/utils/store/config'

import { cn } from '@/lib/utils'

type ResourceUsageProps = {
  runtimePods?: KthenaRuntimePod[]
  className?: string
}

const podKey = (pod: Pick<KthenaRuntimePod, 'namespace' | 'name'>) => `${pod.namespace}/${pod.name}`

export const getKthenaPodMonitorURL = (
  baseURL: string,
  pod: Pick<KthenaRuntimePod, 'name' | 'nodeName'>
) => {
  const params = new URLSearchParams({
    orgId: '1',
    refresh: '5s',
    var_node_name: pod.nodeName,
    var_pod_name: pod.name,
    var_gpu: 'All',
    from: 'now-15m',
    to: 'now',
  })
  return `${baseURL}${baseURL.includes('?') ? '&' : '?'}${params.toString()}`
}

export default function ResourceUsage({ runtimePods, className }: ResourceUsageProps) {
  const { t } = useTranslation()
  const grafanaJob = useAtomValue(configGrafanaJobAtom)
  const [selectedPodKey, setSelectedPodKey] = useState<string>()
  const pods = useMemo(
    () =>
      (runtimePods ?? [])
        .filter((pod) => pod.name && pod.namespace)
        .sort((left, right) => {
          if (left.ready !== right.ready) return left.ready ? -1 : 1
          return podKey(left).localeCompare(podKey(right))
        }),
    [runtimePods]
  )
  const preferredPod = pods.find((pod) => pod.ready) ?? pods[0]
  const selectedPod = pods.find((pod) => podKey(pod) === selectedPodKey) ?? preferredPod
  const hasGrafana = Boolean(grafanaJob.pod?.trim())

  if (!selectedPod) {
    return <ResourceUsageEmpty className={className} description={t('kthena.detail.usageNoPod')} />
  }

  if (!hasGrafana) {
    return (
      <ResourceUsageEmpty
        className={className}
        description={t('kthena.detail.usageNoGrafana')}
        pod={selectedPod}
      />
    )
  }

  if (!selectedPod.nodeName) {
    return (
      <ResourceUsageEmpty
        className={className}
        description={t('kthena.detail.usageNotScheduled')}
        pod={selectedPod}
      />
    )
  }

  return (
    <Card className={cn('gap-0 overflow-hidden', className)}>
      <CardHeader className="border-b px-5 py-4 sm:flex sm:items-center sm:justify-between sm:gap-4">
        <div className="min-w-0">
          <CardTitle className="flex items-center gap-2 text-base">
            <GaugeIcon className="text-primary size-4" />
            {t('kthena.detail.usageTitle')}
          </CardTitle>
          <CardDescription className="mt-1">{t('kthena.detail.usageDescription')}</CardDescription>
        </div>
        <div className="flex min-w-0 items-center gap-2">
          <Badge
            variant="outline"
            className="hidden max-w-64 truncate font-mono text-xs md:inline-flex"
          >
            {selectedPod.namespace}
          </Badge>
          {pods.length > 1 && (
            <Select value={podKey(selectedPod)} onValueChange={setSelectedPodKey}>
              <SelectTrigger className="w-[min(100%,22rem)] min-w-48 font-mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {pods.map((pod) => (
                  <SelectItem key={podKey(pod)} value={podKey(pod)} className="font-mono">
                    {pod.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="flex items-center gap-2 border-b px-5 py-2.5 text-xs">
          <ServerIcon className="text-muted-foreground size-3.5" />
          <span className="text-muted-foreground">{t('kthena.detail.runtimePod')}</span>
          <span className="truncate font-mono font-medium">{selectedPod.name}</span>
          <span className="text-muted-foreground">·</span>
          <span className="text-muted-foreground">{t('kthena.detail.runtimeNode')}</span>
          <span className="truncate font-mono font-medium">{selectedPod.nodeName}</span>
        </div>
        <div className="h-[min(70vh,48rem)] min-h-[30rem] p-3 sm:p-4">
          <GrafanaIframe baseSrc={getKthenaPodMonitorURL(grafanaJob.pod, selectedPod)} />
        </div>
      </CardContent>
    </Card>
  )
}

function ResourceUsageEmpty({
  className,
  description,
  pod,
}: {
  className?: string
  description: string
  pod?: Pick<KthenaRuntimePod, 'name' | 'namespace'>
}) {
  const { t } = useTranslation()
  return (
    <Card className={cn('gap-0', className)}>
      <CardContent className="text-muted-foreground flex min-h-52 flex-col items-center justify-center gap-3 px-6 py-10 text-center">
        <div className="bg-muted flex size-10 items-center justify-center rounded-full">
          <GaugeIcon className="size-5" />
        </div>
        <div className="space-y-1">
          <div className="text-foreground font-medium">{t('kthena.detail.usageUnavailable')}</div>
          <p className="max-w-lg text-sm leading-6">{description}</p>
          {pod && <p className="font-mono text-xs">{podKey(pod)}</p>}
        </div>
      </CardContent>
    </Card>
  )
}
