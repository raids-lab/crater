import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

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

import { KthenaService, apiScaleServing, apiServingLogs } from '@/services/api/inference'

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
      await cache.invalidateQueries({ queryKey: keys.detail(service.name) })
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
  const logs = useQuery({
    queryKey: [...keys.detail(service.name), 'logs', pod, container],
    queryFn: () => apiServingLogs(service.name, pod, container).then((r) => r.data),
    enabled: !!selected,
    refetchInterval: selected ? 15000 : false,
  })
  const groups = [...new Set(pods.map((p) => p.group))].sort()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.topology.step.roles')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        <ServingLifecycle service={service} />
        {service.roles.map((role) => (
          <div key={role.name} className="rounded-lg border p-3 text-sm">
            <strong>{role.name}</strong> · {role.execution} · {role.instances} ×{' '}
            {role.nodesPerInstance} Pods / {t('kthena.topology.groups')} · TP {role.tensorParallel}{' '}
            / PP {role.pipelineParallel}
            <p className="text-muted-foreground mt-1 break-all">{role.entry.image}</p>
          </div>
        ))}
        {groups.map((group) => (
          <details key={group} open>
            <summary>{group}</summary>
            <div className="ml-3 grid gap-2 py-2">
              {service.roles.map((role) => (
                <div key={role.name}>
                  <p className="text-sm font-medium">
                    {role.name} ·{' '}
                    {t('kthena.topology.readiness', {
                      ready: pods.filter(
                        (p) => p.group === group && p.role === role.name && p.ready
                      ).length,
                      desired: role.instances * role.nodesPerInstance,
                    })}
                  </p>
                  {pods
                    .filter((p) => p.group === group && p.role === role.name)
                    .sort(
                      (a, b) =>
                        a.instance.localeCompare(b.instance) || Number(b.entry) - Number(a.entry)
                    )
                    .map((p) => (
                      <Button
                        key={p.name}
                        variant={p.name === pod ? 'secondary' : 'ghost'}
                        className="h-auto w-full justify-start text-left text-xs whitespace-normal"
                        onClick={() => setPod(p.name)}
                      >
                        {p.instance} / {p.entry ? 'Entry' : 'Worker'} · {p.phase} ·{' '}
                        {p.ready ? 'Ready' : 'Waiting'} · {p.nodeName || '—'}
                      </Button>
                    ))}
                </div>
              ))}
            </div>
          </details>
        ))}
        {selected && (
          <>
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 text-xs break-all">{pod}</span>
              <Select value={container} onValueChange={setContainer}>
                <SelectTrigger className="w-44">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="engine">Engine</SelectItem>
                  {service.modelSource === 'external' && (
                    <SelectItem value="model-downloader">Downloader</SelectItem>
                  )}
                </SelectContent>
              </Select>
              <Button variant="outline" onClick={() => void logs.refetch()}>
                {t('kthena.topology.retry')}
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
