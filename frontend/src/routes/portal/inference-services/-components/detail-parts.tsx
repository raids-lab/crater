import { ActivityIcon, BotIcon, UserRoundIcon } from 'lucide-react'
import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import KthenaStatusBadge from '@/components/badge/kthena-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { CopyButton } from '@/components/button/copy-button'
import TooltipCopy from '@/components/label/tooltop-copy'

import {
  ChatCompletionReq,
  KthenaDiagnostic,
  KthenaResource,
  KthenaRuntimePod,
  KthenaService,
} from '@/services/api/inference'

import { shellQuote } from '@/hooks/inference/format'

export const getResourcePhaseVariant = (
  phase: string
): 'default' | 'secondary' | 'outline' | 'destructive' => {
  if (phase === 'Ready' || phase === 'Active') return 'default'
  if (phase === 'Failed') return 'destructive'
  if (phase === 'Pending' || phase === 'Progressing') return 'secondary'
  return 'outline'
}

export type ChatMessage = ChatCompletionReq['messages'][number]

export const emptyChatMessages: ChatMessage[] = []

export const draftChatSessionKey = '__new__'

export function KthenaDetailMeta({
  icon: Icon,
  label,
  children,
}: {
  icon: typeof ActivityIcon
  label: string
  children: ReactNode
}) {
  return (
    <div className="flex min-w-0 items-center">
      <Icon className="text-muted-foreground mr-1.5 size-4 shrink-0" />
      <span className="text-muted-foreground mr-1.5 truncate text-sm">{label}:</span>
      <span className="min-w-0 truncate">{children}</span>
    </div>
  )
}

export function OverviewSection({
  icon: Icon,
  title,
  children,
  className,
}: {
  icon: typeof ActivityIcon
  title: string
  children: ReactNode
  className?: string
}) {
  return (
    <section className={className}>
      <div className="flex items-center gap-2 px-4 py-3">
        <Icon className="text-primary size-4" />
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      <div className="px-4 pb-4">{children}</div>
    </section>
  )
}

export function OverviewField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="bg-muted/20 min-w-0 rounded-lg border px-3 py-2.5">
      <div className="text-muted-foreground text-xs font-medium">{label}</div>
      <div className="mt-1 min-w-0 text-sm">{children}</div>
    </div>
  )
}

export function RuntimePanel({
  service,
  primaryPod,
  resources,
}: {
  service: KthenaService
  primaryPod?: KthenaRuntimePod
  resources: Record<string, string>
}) {
  const { t } = useTranslation()
  const pods = service.runtimePods ?? []
  return (
    <section className="bg-card overflow-hidden rounded-xl border">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <div className="flex min-w-0 items-center gap-2">
          <ActivityIcon className="text-primary size-4 shrink-0" />
          <h3 className="truncate text-sm font-semibold">{t('kthena.table.status')}</h3>
        </div>
        <KthenaStatusBadge service={service} />
      </div>
      <div className="grid gap-3 p-3">
        <OverviewField label={t('kthena.detail.runtimePod')}>
          <CopyValue value={primaryPod?.name || '-'} />
        </OverviewField>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
          <OverviewField label={t('kthena.detail.runtimeNode')}>
            <CopyValue value={primaryPod?.nodeName || '-'} />
          </OverviewField>
          <OverviewField label={t('kthena.detail.runtimeResources')}>
            {Object.keys(resources).length ? (
              <ResourceBadges resources={resources} />
            ) : (
              <span className="text-muted-foreground">-</span>
            )}
          </OverviewField>
        </div>
      </div>
      {pods.length ? (
        <div className="grid gap-1.5 border-t p-3">
          <div className="text-muted-foreground flex items-center justify-between gap-2 text-xs font-medium">
            <span>{t('kthena.detail.runtimePods')}</span>
            <Badge variant="secondary">{pods.length}</Badge>
          </div>
          {pods.map((pod) => (
            <div
              key={`${pod.namespace}/${pod.name}`}
              className="bg-muted/20 flex min-w-0 items-start justify-between gap-3 rounded-lg border px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                <TooltipCopy
                  name={pod.name}
                  copyMessage={t('kthena.copy.generic', { label: 'Pod' })}
                  className="max-w-full min-w-0 truncate font-mono text-xs"
                />
                <div className="text-muted-foreground mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs">
                  <span>{pod.nodeName || '-'}</span>
                  {pod.podIP && <span>{pod.podIP}</span>}
                  <span>
                    {pod.readyContainers}/{pod.totalContainers}
                  </span>
                  {pod.restarts > 0 && (
                    <span>{t('kthena.detail.restarts', { count: pod.restarts })}</span>
                  )}
                </div>
              </div>
              <Badge className="shrink-0" variant={pod.ready ? 'default' : 'secondary'}>
                {pod.phase || '-'}
              </Badge>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  )
}

export function CopyValue({ value }: { value?: string }) {
  const { t } = useTranslation()
  const display = value || '-'
  return (
    <TooltipCopy
      name={display}
      copyMessage={t('kthena.copy.generic', { label: display })}
      showIcon={display !== '-'}
      className="max-w-full text-left font-mono break-all"
    />
  )
}

export function InvocationValue({ label, value }: { label: string; value?: string }) {
  const { t } = useTranslation()
  const display = value || '-'
  return (
    <div className="grid min-w-0 gap-1.5">
      <div className="text-muted-foreground text-xs font-medium">{label}</div>
      <div className="bg-muted/35 flex min-w-0 items-center gap-1 rounded-lg border px-2.5 py-1.5">
        <span className="min-w-0 flex-1 truncate font-mono text-xs" title={display}>
          {display}
        </span>
        {display !== '-' && (
          <CopyButton
            content={display}
            copyMessage={t('kthena.copy.generic', { label })}
            className="shrink-0"
          />
        )}
      </div>
    </div>
  )
}

export function ChatBubble({ message, muted = false }: { message: ChatMessage; muted?: boolean }) {
  const isUser = message.role === 'user'
  const content = isUser ? message.content : stripThinkBlocks(message.content)
  return (
    <div
      className={[
        'flex items-end gap-2',
        isUser ? 'flex-row-reverse justify-start' : 'justify-start',
      ].join(' ')}
    >
      <div
        className={[
          'flex size-7 shrink-0 items-center justify-center rounded-full border',
          isUser ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground',
        ].join(' ')}
      >
        {isUser ? <UserRoundIcon className="size-3.5" /> : <BotIcon className="size-3.5" />}
      </div>
      <div
        className={[
          'max-w-[min(82%,48rem)] rounded-2xl px-4 py-3 text-sm leading-6 whitespace-pre-wrap shadow-sm',
          isUser
            ? 'bg-primary text-primary-foreground rounded-br-md'
            : 'bg-background rounded-bl-md border',
          muted ? 'text-muted-foreground' : '',
        ].join(' ')}
      >
        {content}
      </div>
    </div>
  )
}

export function ResourceCard({ resource }: { resource: KthenaResource }) {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="font-medium">{resource.kind}</div>
        <Badge variant={getResourcePhaseVariant(resource.phase)}>
          {resource.phase || 'Pending'}
        </Badge>
      </div>
      <TooltipCopy
        name={`${resource.namespace}/${resource.name}`}
        copyMessage={t('kthena.copy.resource')}
        className="max-w-full truncate font-mono text-xs"
      />
      {resource.conditions?.length > 0 && (
        <div className="text-muted-foreground mt-2 line-clamp-2 text-xs">
          {resource.conditions
            .map((condition) => `${String(condition.type ?? '')}:${String(condition.status ?? '')}`)
            .join(' · ')}
        </div>
      )}
    </div>
  )
}

export function DiagnosticCard({ diagnostic }: { diagnostic: KthenaDiagnostic }) {
  const { t } = useTranslation()
  const isError = diagnostic.level === 'error'
  return (
    <div
      className={[
        'relative rounded-md border p-3',
        isError ? 'border-destructive/40 bg-destructive/5' : '',
      ].join(' ')}
    >
      <div
        className={[
          'border-background absolute top-4 -left-[17px] size-2.5 rounded-full border-2',
          isError ? 'bg-destructive' : 'bg-amber-500',
        ].join(' ')}
      />
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge variant={isError ? 'destructive' : 'secondary'}>
          {diagnostic.reason || diagnostic.level}
        </Badge>
        {diagnostic.pod && <span className="text-muted-foreground text-xs">{diagnostic.pod}</span>}
        {diagnostic.container && (
          <span className="text-muted-foreground text-xs">/{diagnostic.container}</span>
        )}
      </div>
      <div className="text-sm leading-6 break-words">{diagnostic.message}</div>
      {diagnostic.details && (
        <details className="mt-3">
          <summary className="text-muted-foreground cursor-pointer text-xs">
            {t('kthena.detail.logDetails')}
          </summary>
          <pre className="bg-muted mt-2 max-h-72 overflow-auto rounded-md p-3 text-xs leading-5 break-words whitespace-pre-wrap">
            {diagnostic.details}
          </pre>
        </details>
      )}
      {(diagnostic.resource || diagnostic.timestamp) && (
        <div className="text-muted-foreground mt-2 flex flex-wrap gap-3 text-xs">
          {diagnostic.resource && <span>{diagnostic.resource}</span>}
          {diagnostic.timestamp && <span>{new Date(diagnostic.timestamp).toLocaleString()}</span>}
        </div>
      )}
    </div>
  )
}

export function DiagnosticSummary({ diagnostics }: { diagnostics: KthenaDiagnostic[] }) {
  const { t } = useTranslation()
  const firstError =
    diagnostics.find((diagnostic) => diagnostic.level === 'error') ?? diagnostics[0]
  return (
    <div className="rounded-md border p-3">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge variant={firstError.level === 'error' ? 'destructive' : 'secondary'}>
          {firstError.reason || firstError.level}
        </Badge>
        <span className="text-muted-foreground text-xs">{t('kthena.detail.primaryIssue')}</span>
      </div>
      <div className="text-sm leading-6 break-words">{firstError.message}</div>
    </div>
  )
}

export function buildCurl(service: KthenaService, messages: ChatMessage[], placeholder: string) {
  const model = service.access?.modelName || service.name
  const baseURL = displayAPIBaseURL(service)
  const requestBody = JSON.stringify({
    model,
    messages: messages.length ? messages : [{ role: 'user', content: placeholder }],
    temperature: 0.2,
  })
  return `curl -X POST '${baseURL}/chat/completions' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <your-crater-token>' \\
  --data-raw ${shellQuote(requestBody)}`
}

export function displayAPIBaseURL(service: KthenaService) {
  return `${window.location.origin}/api${service.access?.proxyBaseURL || `/v1/kthena/inference-services/${service.name}/openai/v1`}`
}

export function stripThinkBlocks(content: string) {
  return content
    .replace(/<think>[\s\S]*?<\/think>/gi, '')
    .replace(/^\s+/, '')
    .trimEnd()
}

export function createChatID() {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
}
