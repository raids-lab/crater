import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { LogsIcon, ServerIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import NodeBadges from '@/components/badge/node-badges'
import { PhaseBadge } from '@/components/badge/phase-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import TooltipButton from '@/components/button/tooltip-button'
import { PodMonitorSheet, type PodMonitorTarget } from '@/components/job/detail/pod-monitor'
import CardTitle from '@/components/label/card-title'
import TooltipCopy from '@/components/label/tooltop-copy'
import { NothingCore } from '@/components/placeholder/nothing'

import {
  KthenaRuntimeGroup,
  KthenaRuntimePod,
  KthenaService,
  apiScaleServing,
} from '@/services/api/inference'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { shortestImageName } from '@/utils/formatter'
import { showErrorToast } from '@/utils/toast'

import { ServingLogDialog, ServingPodLog } from './serving-pod-logs'

export function ServingLifecycle({ service }: { service: KthenaService }) {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const cache = useQueryClient()
  const [replicas, setReplicas] = useState(service.desiredReplicas || 1)
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
        value={replicas}
        className="w-24"
        onChange={(e) => setReplicas(Number(e.target.value))}
      />
      <Button
        variant="outline"
        disabled={scale.isPending || replicas < 1}
        onClick={() => scale.mutate(replicas)}
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
  const [podName, setPodName] = useState<string>()
  const [monitorPod, setMonitorPod] = useState<PodMonitorTarget>()
  const pods = new Map((service.runtimePods ?? []).map((pod) => [pod.name, pod]))
  const groups = service.runtimeGroups ?? []
  const selected = podName ? pods.get(podName) : undefined
  const [role] = service.roles
  const singlePod =
    pods.size === 1 &&
    service.desiredReplicas === 1 &&
    service.roles.length === 1 &&
    role.instances === 1 &&
    role.nodesPerInstance === 1
      ? pods.values().next().value
      : undefined

  return (
    <>
      <Card>
        <CardHeader className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1.5">
            <CardTitle icon={ServerIcon}>{t('kthena.runtime.title')}</CardTitle>
            <CardDescription>
              {t('kthena.runtime.groupReadiness', {
                ready: service.readyReplicas ?? '—',
                desired: service.desiredReplicas,
              })}
            </CardDescription>
          </div>
          <ServingLifecycle service={service} />
        </CardHeader>
        <CardContent className="space-y-3">
          {service.diagnostics?.some((item) => item.reason === 'RoutingResourcesMissing') && (
            <p className="text-muted-foreground text-sm">{t('kthena.runtime.routingPending')}</p>
          )}
          {singlePod ? (
            <div className="grid gap-6 md:grid-cols-3 xl:grid-cols-4">
              <div className="min-w-0 md:col-span-2 xl:col-span-3">
                <ServingPodLog key={singlePod.name} service={service} pod={singlePod} />
              </div>
              <ServingPodDetails service={service} pod={singlePod} openMonitor={setMonitorPod} />
            </div>
          ) : groups.length === 0 ? (
            <NothingCore className="py-8" title={t('kthena.runtime.noPods')} />
          ) : (
            <Accordion type="single" collapsible defaultValue={groups[0].name}>
              {groups.map((group) => (
                <RuntimeGroup
                  key={group.name}
                  group={group}
                  pods={pods}
                  openLogs={setPodName}
                  openMonitor={setMonitorPod}
                />
              ))}
            </Accordion>
          )}
        </CardContent>
      </Card>
      <PodMonitorSheet pod={monitorPod} onClose={() => setMonitorPod(undefined)} />
      {selected && (
        <ServingLogDialog
          key={selected.name}
          service={service}
          pod={selected}
          onClose={() => setPodName(undefined)}
        />
      )}
    </>
  )
}

function ServingPodDetails({
  service,
  pod,
  openMonitor,
}: {
  service: KthenaService
  pod: KthenaRuntimePod
  openMonitor: (target: PodMonitorTarget) => void
}) {
  const { t } = useTranslation()
  const role = service.roles.find((item) => item.name === pod.role)
  const profile = role && (pod.entry ? role.entry : (role.worker ?? role.entry))
  const resources = profile
    ? {
        cpu: profile.cpu,
        memory: profile.memory,
        [profile.gpuModel]: profile.gpu,
        ...profile.extendedResources,
      }
    : undefined
  return (
    <fieldset className="border-input min-w-0 rounded-lg border p-4 shadow-xs">
      <legend className="-ml-1 px-2 text-sm font-medium">{t('kthena.runtime.podDetails')}</legend>
      <dl className="grid grid-cols-[auto_minmax(0,1fr)] items-start gap-x-4 gap-y-3 text-sm">
        <dt className="text-muted-foreground">{t('kthena.table.status')}</dt>
        <dd>
          <PodStageBadge stage={pod.stage} />
        </dd>
        <dt className="text-muted-foreground">{t('kthena.runtime.podName')}</dt>
        <dd>
          <ServingPodLink pod={pod} onClick={() => openMonitor(pod)} />
        </dd>
        <dt className="text-muted-foreground">{t('kthena.runtime.roleInstance')}</dt>
        <dd>{[pod.role, pod.instance].filter(Boolean).join(' · ') || '—'}</dd>
        {profile && (
          <>
            <dt className="text-muted-foreground">{t('kthena.topology.imageLabel')}</dt>
            <dd className="font-mono break-all" title={profile.image}>
              {shortestImageName(profile.image)}
            </dd>
          </>
        )}
        <dt className="text-muted-foreground">{t('kthena.runtime.podIP')}</dt>
        <dd>{pod.podIP ? <TooltipCopy name={pod.podIP} className="font-mono" /> : '—'}</dd>
        <dt className="text-muted-foreground">{t('kthena.runtime.node')}</dt>
        <dd>{pod.nodeName ? <NodeBadges nodes={[pod.nodeName]} /> : '—'}</dd>
        {resources && (
          <>
            <dt className="text-muted-foreground">{t('kthena.runtime.requests')}</dt>
            <dd className="flex flex-wrap gap-1">
              <ResourceBadges resources={resources} />
            </dd>
          </>
        )}
        <dt className="text-muted-foreground">{t('kthena.runtime.restarts')}</dt>
        <dd className="font-mono">{pod.restarts}</dd>
      </dl>
      <PodStageDetails pod={pod} className="mt-3" />
    </fieldset>
  )
}

const stageColors: Record<KthenaRuntimePod['stage'], string> = {
  scheduling: 'text-highlight-purple bg-highlight-purple/20',
  downloading: 'text-highlight-blue bg-highlight-blue/20',
  starting: 'text-highlight-blue bg-highlight-blue/20',
  blocked: 'text-highlight-red bg-highlight-red/20',
  failed: 'text-highlight-red bg-highlight-red/20',
  ready: 'text-highlight-emerald bg-highlight-emerald/20',
  terminating: 'text-highlight-slate bg-highlight-slate/20',
}

function PodStageBadge({ stage }: { stage: KthenaRuntimePod['stage'] }) {
  const { t } = useTranslation()
  const label = t(`kthena.runtime.stage.${stage}`)
  return (
    <PhaseBadge
      phase={stage}
      getPhaseLabel={(value) => ({ label, description: label, color: stageColors[value] })}
      disableDefaultTooltip
    />
  )
}

function PodStageDetails({ pod, className }: { pod: KthenaRuntimePod; className?: string }) {
  const { t } = useTranslation()
  return pod.message ? (
    <details
      className={`text-muted-foreground max-w-80 text-xs whitespace-normal ${className ?? ''}`}
    >
      <summary className="cursor-pointer">
        {pod.reason || t('kthena.detail.tabs.diagnostics')}
      </summary>
      <p className="mt-1 break-words">{pod.message}</p>
    </details>
  ) : pod.reason ? (
    <p className={`text-muted-foreground text-xs ${className ?? ''}`}>{pod.reason}</p>
  ) : null
}

function ServingPodLink({ pod, onClick }: { pod: KthenaRuntimePod; onClick: () => void }) {
  const { t } = useTranslation()
  if (!pod.nodeName) return <span className="font-mono break-all">{pod.name}</span>
  return (
    <TooltipButton
      variant="link"
      className="h-auto justify-start p-0 text-left font-mono break-all whitespace-normal underline-offset-4"
      tooltipContent={t('common.podMonitor')}
      onClick={onClick}
    >
      {pod.name}
    </TooltipButton>
  )
}

function RuntimeGroup({
  group,
  pods,
  openLogs,
  openMonitor,
}: {
  group: KthenaRuntimeGroup
  pods: Map<string, KthenaRuntimePod>
  openLogs: (podName: string) => void
  openMonitor: (target: PodMonitorTarget) => void
}) {
  const { t } = useTranslation()
  const groupPods = group.roles
    .flatMap((role) => role.instances)
    .flatMap((instance) => instance.podNames)
    .map((name) => pods.get(name))
    .filter((pod) => pod !== undefined)
  const readyLabel = t(group.ready ? 'kthena.runtime.stage.ready' : 'kthena.runtime.waiting')
  return (
    <AccordionItem value={group.name}>
      <AccordionTrigger className="hover:no-underline">
        <span className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="break-all">{group.name || '—'}</span>
          <PhaseBadge
            phase={group.ready}
            getPhaseLabel={(ready) => ({
              label: readyLabel,
              description: readyLabel,
              color: stageColors[ready ? 'ready' : 'scheduling'],
            })}
            disableDefaultTooltip
          />
        </span>
      </AccordionTrigger>
      <AccordionContent className="space-y-3">
        <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-sm">
          {group.roles.map((role) => (
            <p key={role.name}>
              <span className="text-foreground font-medium">{role.name}</span>
              {' · '}
              {t('kthena.runtime.instanceReadiness', {
                ready: role.readyInstances,
                desired: role.desiredInstances,
              })}
            </p>
          ))}
        </div>
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="bg-muted/40">
              <TableRow>
                <TableHead>{t('kthena.runtime.podName')}</TableHead>
                <TableHead>{t('kthena.runtime.roleInstance')}</TableHead>
                <TableHead>{t('kthena.table.status')}</TableHead>
                <TableHead>{t('kthena.runtime.node')}</TableHead>
                <TableHead>{t('kthena.runtime.podIP')}</TableHead>
                <TableHead className="w-16 text-right">{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groupPods.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-muted-foreground py-6 text-center">
                    {t('kthena.runtime.noPods')}
                  </TableCell>
                </TableRow>
              )}
              {groupPods.map((pod) => (
                <TableRow key={pod.name}>
                  <TableCell>
                    <ServingPodLink pod={pod} onClick={() => openMonitor(pod)} />
                  </TableCell>
                  <TableCell>
                    <p>{[pod.role, pod.instance].filter(Boolean).join(' · ') || '—'}</p>
                    <p className="text-muted-foreground text-xs">
                      {t(pod.entry ? 'kthena.runtime.entry' : 'kthena.runtime.worker')}
                    </p>
                  </TableCell>
                  <TableCell>
                    <PodStageBadge stage={pod.stage} />
                    <PodStageDetails pod={pod} className="mt-1" />
                  </TableCell>
                  <TableCell>
                    {pod.nodeName ? <NodeBadges nodes={[pod.nodeName]} /> : '—'}
                  </TableCell>
                  <TableCell>
                    {pod.podIP ? <TooltipCopy name={pod.podIP} className="font-mono" /> : '—'}
                  </TableCell>
                  <TableCell className="text-right">
                    <TooltipButton
                      variant="outline"
                      size="icon"
                      className="size-8"
                      aria-label={t('kthena.runtime.logs')}
                      tooltipContent={t('kthena.runtime.logs')}
                      onClick={() => openLogs(pod.name)}
                    >
                      <LogsIcon className="text-highlight-orange size-4" />
                    </TooltipButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}
