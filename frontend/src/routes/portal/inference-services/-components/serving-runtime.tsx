import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  KthenaRuntimeGroup,
  KthenaRuntimePod,
  KthenaService,
  apiScaleServing,
  apiServingLogs,
} from '@/services/api/inference'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { showErrorToast } from '@/utils/toast'

export function ServingLifecycle({ service }: { service: KthenaService }) {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const cache = useQueryClient()
  const [groups, setGroups] = useState(service.desiredReplicas || 1)
  const scale = useMutation({
    mutationFn: (replicas: number) => apiScaleServing(service.name, replicas),
    onSuccess: async () => {
      await cache.invalidateQueries({ queryKey: keys.scope })
    },
    onError: showErrorToast,
  })
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Input
        type="number"
        aria-label={t('kthena.topology.groups')}
        min={1}
        max={10000}
        value={groups}
        className="w-24"
        onChange={(e) => setGroups(Number(e.target.value))}
      />
      <Button
        variant="outline"
        disabled={scale.isPending || groups < 1}
        onClick={() => scale.mutate(groups)}
      >
        {t(service.phase === 'Suspended' ? 'kthena.topology.resume' : 'kthena.topology.scale')}
      </Button>
      {service.phase !== 'Suspended' && (
        <Button variant="outline" disabled={scale.isPending} onClick={() => scale.mutate(0)}>
          {t('kthena.topology.suspend')}
        </Button>
      )}
      <Button variant="outline" asChild>
        <Link to="/portal/inference-services/new" search={{ clone: undefined, edit: service.name }}>
          {t('kthena.topology.edit')}
        </Link>
      </Button>
    </div>
  )
}
export function ServingRuntime({ service }: { service: KthenaService }) {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const [pod, setPod] = useState('')
  const [container, setContainer] = useState('engine')
  const pods = service.runtimePods ?? []
  const selected = pods.find((p) => p.name === pod)
  const podsByName = new Map(pods.map((p) => [p.name, p]))
  const groups = service.runtimeGroups ?? []
  const logs = useQuery({
    queryKey: [...keys.detail(service.name), 'logs', pod, container],
    queryFn: () => apiServingLogs(service.name, pod, container).then((r) => r.data),
    enabled: !!selected,
    refetchInterval: selected ? 15000 : false,
  })
  function openLogs(pod: KthenaRuntimePod) {
    setPod(pod.name)
    setContainer(pod.stage === 'downloading' ? 'model-downloader' : 'engine')
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.runtime.title')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        <ServingLifecycle service={service} />
        <p className="text-muted-foreground text-sm">
          {t('kthena.runtime.groupReadiness', {
            ready: service.readyReplicas ?? '—',
            desired: service.desiredReplicas,
          })}
        </p>
        {service.diagnostics?.some((item) => item.reason === 'RoutingResourcesMissing') && (
          <p className="text-muted-foreground text-sm">{t('kthena.runtime.routingPending')}</p>
        )}
        {groups.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t('kthena.runtime.noPods')}</p>
        ) : (
          <Accordion type="single" collapsible defaultValue={`group:${groups[0].name}`}>
            {groups.map((group) => (
              <RuntimeGroup key={group.name} group={group} pods={podsByName} openLogs={openLogs} />
            ))}
          </Accordion>
        )}
        {selected && (
          <>
            <div className="flex flex-wrap items-center gap-2">
              <span className="min-w-0 flex-1 text-xs break-all">{pod}</span>
              <Select value={container} onValueChange={setContainer}>
                <SelectTrigger className="w-44">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="engine">{t('kthena.runtime.engine')}</SelectItem>
                  {service.modelSource === 'external' && (
                    <SelectItem value="model-downloader">
                      {t('kthena.runtime.downloader')}
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
              <Button variant="outline" onClick={() => void logs.refetch()}>
                {t('kthena.topology.retry')}
              </Button>
              <Button variant="ghost" onClick={() => setPod('')}>
                {t('kthena.runtime.closeLogs')}
              </Button>
            </div>
            {logs.isError ? (
              <p role="alert">{logs.error.message}</p>
            ) : (
              <pre className="bg-muted max-h-96 overflow-auto rounded-lg p-3 text-xs">
                {logs.data ?? t('kthena.topology.loading')}
              </pre>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}

function RuntimeGroup({
  group,
  pods,
  openLogs,
}: {
  group: KthenaRuntimeGroup
  pods: Map<string, KthenaRuntimePod>
  openLogs: (pod: KthenaRuntimePod) => void
}) {
  const { t } = useTranslation()
  return (
    <AccordionItem value={`group:${group.name}`}>
      <AccordionTrigger>
        <span className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="break-all">{group.name || '—'}</span>
          <Badge variant={group.ready ? 'default' : 'secondary'}>
            {t(group.ready ? 'kthena.runtime.stage.ready' : 'kthena.runtime.waiting')}
          </Badge>
        </span>
      </AccordionTrigger>
      <AccordionContent className="space-y-3">
        {group.roles.map((role) => (
          <section key={role.name} className="space-y-2 rounded-lg border p-3">
            <h4 className="flex flex-wrap items-center gap-2 font-medium">
              {role.name || '—'}
              <span className="text-muted-foreground text-xs">
                {t('kthena.runtime.instanceReadiness', {
                  ready: role.readyInstances,
                  desired: role.desiredInstances,
                })}
              </span>
            </h4>
            {role.instances.map((instance) => (
              <div key={instance.name} className="space-y-1">
                <p className="text-muted-foreground text-xs">
                  {t('kthena.runtime.instance', { name: instance.name || '—' })}
                  {' · '}
                  {t(instance.ready ? 'kthena.runtime.stage.ready' : 'kthena.runtime.waiting')}
                </p>
                <ul className="divide-y">
                  {instance.podNames.map((name) => {
                    const pod = pods.get(name)
                    return pod ? <RuntimePod key={name} pod={pod} openLogs={openLogs} /> : null
                  })}
                </ul>
              </div>
            ))}
          </section>
        ))}
      </AccordionContent>
    </AccordionItem>
  )
}

function RuntimePod({
  pod,
  openLogs,
}: {
  pod: KthenaRuntimePod
  openLogs: (pod: KthenaRuntimePod) => void
}) {
  const { t } = useTranslation()
  const failed = pod.stage === 'failed' || pod.stage === 'blocked'
  return (
    <li className="flex flex-wrap items-start justify-between gap-2 py-2 text-xs">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">
            {t(pod.entry ? 'kthena.runtime.entry' : 'kthena.runtime.worker')}
          </span>
          <Badge variant={failed ? 'destructive' : pod.ready ? 'default' : 'secondary'}>
            {t(`kthena.runtime.stage.${pod.stage}`)}
          </Badge>
        </div>
        <p className="text-muted-foreground font-mono break-all">
          {pod.name} · {pod.nodeName || '—'}
        </p>
        {pod.reason && (
          <p className="text-muted-foreground break-words">
            {[pod.reason, pod.message].filter(Boolean).join(' · ')}
          </p>
        )}
      </div>
      <Button size="sm" variant="outline" onClick={() => openLogs(pod)}>
        {t('kthena.runtime.logs')}
      </Button>
    </li>
  )
}
