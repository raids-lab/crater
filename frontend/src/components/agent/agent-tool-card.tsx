import type { TFunction } from 'i18next'
import {
  CheckCircle2Icon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  CircleEllipsisIcon,
  CircleXIcon,
  ExternalLinkIcon,
  LoaderCircleIcon,
  XIcon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import type { AgentFormPatch, AgentFormRuntime } from '@/components/agent/agent-context'
import { isAgentNavigationDestination } from '@/components/agent/form-capabilities'

import type { AgentIdentity } from '@/services/api/agent'

import { cn } from '@/lib/utils'

export interface AgentToolPart {
  type: string
  toolCallId: string
  state: string
  input?: unknown
  output?: unknown
  errorText?: string
}

interface AgentToolCardProps {
  identity: AgentIdentity
  part: AgentToolPart
  sequence?: number
  runtime?: AgentFormRuntime
  busy: boolean
  onNavigate: (path: string) => Promise<void>
  onNavigateJob: (jobName: string) => Promise<void>
  onApply: (patches: readonly AgentFormPatch[]) => Promise<string[]>
  onReject: () => Promise<void>
}

type ToolStatus = 'waiting' | 'success' | 'error' | 'cancelled'

const toolNameKeys: Record<string, string> = {
  list_jobs: 'craterAgent.toolName.listJobs',
  get_job_diagnostic_snapshot: 'craterAgent.toolName.jobDiagnostic',
  get_job_diagnostic_snapshots: 'craterAgent.toolName.jobDiagnostics',
  get_job_scheduling_diagnosis: 'craterAgent.toolName.schedulingDiagnosis',
  list_nodes: 'craterAgent.toolName.listNodes',
  get_node_capacity: 'craterAgent.toolName.nodeCapacity',
  get_cluster_capacity: 'craterAgent.toolName.clusterCapacity',
  navigate_to_page: 'craterAgent.toolName.navigate',
  navigate_to_job_detail: 'craterAgent.toolName.navigateJobDetail',
  propose_form_patch: 'craterAgent.toolName.formPatch',
}

const toolStatusStyles: Record<ToolStatus, string> = {
  waiting:
    'border-blue-200 bg-blue-50/80 text-blue-800 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-200',
  success:
    'border-green-200 bg-green-50/80 text-green-800 dark:border-green-800 dark:bg-green-950/40 dark:text-green-200',
  error:
    'border-red-200 bg-red-50/80 text-red-800 dark:border-red-800 dark:bg-red-950/40 dark:text-red-200',
  cancelled:
    'border-muted-foreground/20 bg-muted/60 text-muted-foreground dark:border-muted-foreground/30',
}

export function AgentToolCard({
  identity,
  part,
  sequence,
  runtime,
  busy,
  onNavigate,
  onNavigateJob,
  onApply,
  onReject,
}: AgentToolCardProps) {
  const { t } = useTranslation()
  const [working, setWorking] = useState(false)
  const [actionError, setActionError] = useState<string>()
  const input = isRecord(part.input) ? part.input : {}
  const resolved = part.state === 'output-available' || part.state === 'output-error'
  const output = isRecord(part.output) ? part.output : {}
  const toolName = part.type.replace(/^tool-/, '')
  const title = `${toolDisplayName(toolName, t)}${sequence ? ` #${sequence}` : ''}`
  const status = resolveToolStatus(part, output, actionError)
  const awaitingConfirmation =
    !resolved &&
    !working &&
    ['navigate_to_page', 'navigate_to_job_detail', 'propose_form_patch'].includes(toolName)
  const errorText = actionError || part.errorText || stringValue(output.error)

  const run = async (action: () => Promise<void>) => {
    setActionError(undefined)
    setWorking(true)
    try {
      await action()
    } catch (error) {
      const message = error instanceof Error ? error.message : t('craterAgent.toolFailed')
      setActionError(message)
      toast.error(message)
    } finally {
      setWorking(false)
    }
  }

  if (part.type === 'tool-navigate_to_page') {
    const path = typeof input.path === 'string' ? input.path : ''
    const allowed = isAgentNavigationDestination(identity, path)
    return (
      <ConfirmationToolCard
        title={title}
        status={status}
        pending={awaitingConfirmation}
        errorText={errorText}
      >
        <p className="text-sm font-medium wrap-anywhere">{stringValue(input.label) || path}</p>
        {resolved ? (
          <ToolResult accepted={output.status === 'accepted'} />
        ) : (
          <ToolActions
            disabled={busy || working}
            confirmDisabled={!allowed}
            working={working}
            confirmLabel={t('craterAgent.openPage')}
            onConfirm={() => void run(() => onNavigate(path))}
            onReject={() => void run(onReject)}
          />
        )}
      </ConfirmationToolCard>
    )
  }

  if (part.type === 'tool-navigate_to_job_detail') {
    const jobName = stringValue(input.jobName)
    return (
      <ConfirmationToolCard
        title={title}
        status={status}
        pending={awaitingConfirmation}
        errorText={errorText}
      >
        <p className="text-sm font-medium wrap-anywhere">{stringValue(input.label) || jobName}</p>
        {resolved ? (
          <ToolResult accepted={output.status === 'accepted'} />
        ) : (
          <ToolActions
            disabled={busy || working}
            confirmDisabled={!isAgentJobName(jobName)}
            working={working}
            confirmLabel={t('craterAgent.openJobDetail')}
            onConfirm={() => void run(() => onNavigateJob(jobName))}
            onReject={() => void run(onReject)}
          />
        )}
      </ConfirmationToolCard>
    )
  }

  if (part.type === 'tool-propose_form_patch') {
    const capabilityID = stringValue(input.capabilityId)
    const schemaVersion = numberValue(input.schemaVersion)
    const patches = parsePatches(input.patches)
    const compatible =
      runtime?.capability.id === capabilityID &&
      runtime.capability.schemaVersion === schemaVersion &&
      patches.length > 0
    return (
      <ConfirmationToolCard
        title={title}
        status={status}
        pending={awaitingConfirmation}
        errorText={errorText}
      >
        {stringValue(input.summary) && (
          <p className="text-muted-foreground text-xs wrap-anywhere">
            {stringValue(input.summary)}
          </p>
        )}
        <div className="divide-y rounded-md border">
          {patches.map((patch) => (
            <div key={patch.path} className="space-y-1 px-3 py-2 text-sm">
              <div className="flex min-w-0 items-start justify-between gap-3">
                <span className="min-w-0 font-medium wrap-anywhere">
                  {runtime?.fields.find((field) => field.path === patch.path)?.label ?? patch.path}
                </span>
                <code className="bg-muted max-w-48 shrink-0 truncate rounded px-1.5 py-0.5 text-xs">
                  {displayValue(patch.value)}
                </code>
              </div>
              <p className="text-muted-foreground text-xs wrap-anywhere">{patch.reason}</p>
            </div>
          ))}
        </div>
        {!compatible && !resolved && (
          <p className="text-destructive text-xs">{t('craterAgent.formChanged')}</p>
        )}
        {resolved ? (
          <ToolResult accepted={output.status === 'accepted'} />
        ) : (
          <ToolActions
            disabled={busy || working}
            confirmDisabled={!compatible}
            working={working}
            confirmLabel={t('craterAgent.applyForm')}
            onConfirm={() =>
              void run(async () => {
                await onApply(patches)
              })
            }
            onReject={() => void run(onReject)}
          />
        )}
      </ConfirmationToolCard>
    )
  }

  return (
    <ToolFrame
      toolName={toolName}
      title={title}
      status={status}
      awaitingConfirmation={false}
      input={part.input}
      output={part.output}
      errorText={errorText}
    />
  )
}

export function AgentToolGroup({
  parts,
  children,
}: {
  parts: readonly AgentToolPart[]
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const toolName = parts[0]?.type.replace(/^tool-/, '') ?? ''
  const status = resolveToolGroupStatus(parts)

  return (
    <ToolFrame
      toolName={toolName}
      title={`${toolDisplayName(toolName, t)} × ${parts.length}`}
      status={status}
      awaitingConfirmation={false}
    >
      <div className="space-y-1.5">{children}</div>
    </ToolFrame>
  )
}

function ConfirmationToolCard({
  title,
  status,
  pending,
  errorText,
  children,
}: {
  title: string
  status: ToolStatus
  pending: boolean
  errorText?: string
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const statusLabel = pending
    ? t('craterAgent.toolStatus.awaitingConfirmation')
    : t(`craterAgent.toolStatus.${status}`)

  return (
    <div className={cn('max-w-full min-w-0 rounded-lg border p-3', toolStatusStyles[status])}>
      <div className="flex min-w-0 items-center gap-2 text-xs font-medium">
        {pending ? (
          <CircleEllipsisIcon className="size-4 shrink-0" />
        ) : (
          <ToolStatusIcon status={status} />
        )}
        <span className="min-w-0 flex-1 truncate">{title}</span>
        <span className="shrink-0 opacity-80">{statusLabel}</span>
      </div>
      <div className="text-foreground mt-3 space-y-3">{children}</div>
      {errorText && <p className="text-destructive mt-2 text-xs wrap-anywhere">{errorText}</p>}
    </div>
  )
}

function ToolFrame({
  toolName,
  title,
  status,
  awaitingConfirmation,
  input,
  output,
  errorText,
  children,
}: {
  toolName: string
  title: string
  status: ToolStatus
  awaitingConfirmation: boolean
  input?: unknown
  output?: unknown
  errorText?: string
  children?: React.ReactNode
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const statusLabel = awaitingConfirmation
    ? t('craterAgent.toolStatus.awaitingConfirmation')
    : t(`craterAgent.toolStatus.${status}`)

  return (
    <Collapsible
      open={open}
      onOpenChange={setOpen}
      className={cn(
        'max-w-full min-w-0 overflow-hidden rounded-lg border',
        toolStatusStyles[status]
      )}
    >
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className="flex min-h-9 w-full min-w-0 items-center gap-2 px-3 py-1.5 text-left text-xs font-medium outline-none focus-visible:ring-2 focus-visible:ring-current/40"
          aria-label={open ? t('craterAgent.toolCollapse') : t('craterAgent.toolExpand')}
        >
          <ToolStatusIcon status={status} />
          <span className="min-w-0 flex-1 truncate">{title}</span>
          <span className="shrink-0 opacity-80">{statusLabel}</span>
          {open ? (
            <ChevronDownIcon className="size-3.5 shrink-0" />
          ) : (
            <ChevronRightIcon className="size-3.5 shrink-0" />
          )}
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="bg-background text-foreground space-y-3 border-t p-3">
          <ToolDetail label={t('craterAgent.toolCommand')} value={toolName} />
          {input !== undefined && (
            <ToolDetail label={t('craterAgent.toolInput')} value={displayValue(input, true)} />
          )}
          {status === 'waiting' && !awaitingConfirmation && (
            <p className="text-muted-foreground text-xs">{t('craterAgent.toolRunning')}</p>
          )}
          {output !== undefined && (
            <ToolDetail label={t('craterAgent.toolOutput')} value={displayValue(output, true)} />
          )}
          {errorText && <ToolDetail label={t('craterAgent.toolError')} value={errorText} error />}
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function ToolStatusIcon({ status }: { status: ToolStatus }) {
  if (status === 'waiting') return <LoaderCircleIcon className="size-4 shrink-0 animate-spin" />
  if (status === 'success') return <CheckCircle2Icon className="size-4 shrink-0" />
  return <CircleXIcon className="size-4 shrink-0" />
}

function ToolDetail({
  label,
  value,
  error = false,
}: {
  label: string
  value: string
  error?: boolean
}) {
  return (
    <div className="min-w-0 space-y-1">
      <p className="text-muted-foreground text-xs font-medium">{label}</p>
      <pre
        className={cn(
          'max-h-64 max-w-full overflow-auto rounded-md p-2 font-mono text-xs leading-5 break-all whitespace-pre-wrap',
          error ? 'bg-red-50 text-red-800 dark:bg-red-950/30 dark:text-red-200' : 'bg-muted/70'
        )}
      >
        {value}
      </pre>
    </div>
  )
}

function ToolActions({
  disabled,
  confirmDisabled,
  working,
  confirmLabel,
  onConfirm,
  onReject,
}: {
  disabled: boolean
  confirmDisabled: boolean
  working: boolean
  confirmLabel: string
  onConfirm: () => void
  onReject: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap justify-end gap-2">
      <Button type="button" size="sm" variant="ghost" disabled={disabled} onClick={onReject}>
        <XIcon />
        {t('craterAgent.reject')}
      </Button>
      <Button type="button" size="sm" disabled={disabled || confirmDisabled} onClick={onConfirm}>
        {working ? <LoaderCircleIcon className="animate-spin" /> : <ExternalLinkIcon />}
        {confirmLabel}
      </Button>
    </div>
  )
}

function ToolResult({ accepted }: { accepted: boolean }) {
  const { t } = useTranslation()
  return (
    <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
      {accepted ? <CheckIcon className="text-success size-4" /> : <XIcon className="size-4" />}
      {accepted ? t('craterAgent.toolAccepted') : t('craterAgent.toolRejected')}
    </p>
  )
}

function resolveToolStatus(
  part: AgentToolPart,
  output: Record<string, unknown>,
  actionError?: string
): ToolStatus {
  if (actionError || part.state === 'output-error') return 'error'
  if (part.state !== 'output-available') return 'waiting'
  if (output.status === 'error' || output.status === 'partial-error') return 'error'
  return output.status === 'rejected' ? 'cancelled' : 'success'
}

function resolveToolGroupStatus(parts: readonly AgentToolPart[]): ToolStatus {
  if (parts.some((part) => part.state === 'output-error')) return 'error'
  if (parts.some((part) => part.state !== 'output-available')) return 'waiting'
  return 'success'
}

function toolDisplayName(toolName: string, t: TFunction) {
  const key = toolNameKeys[toolName]
  return key ? t(key) : toolName
}

function parsePatches(value: unknown): AgentFormPatch[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((candidate) => {
    if (!isRecord(candidate) || typeof candidate.path !== 'string') return []
    return [
      {
        path: candidate.path,
        value: candidate.value,
        reason: stringValue(candidate.reason),
      },
    ]
  })
}

function displayValue(value: unknown, pretty = false) {
  if (typeof value === 'string') return value
  if (value === undefined) return ''
  try {
    return JSON.stringify(value, null, pretty ? 2 : undefined)
  } catch {
    return String(value)
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function numberValue(value: unknown) {
  return typeof value === 'number' ? value : 0
}

function isAgentJobName(value: string) {
  return value.length <= 253 && /^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$/.test(value)
}
