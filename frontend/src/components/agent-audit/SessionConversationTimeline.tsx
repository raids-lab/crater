import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { ToolCallCard } from '@/components/aiops/ToolCallCard'
import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import {
  AgentAuditMessage,
  AgentAuditToolCall,
  AgentAuditTurn,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

type TimelineItem =
  | { kind: 'turn-divider'; key: string; index: number; mode: string; turnId: string; at: string }
  | { kind: 'message'; key: string; msg: AgentAuditMessage }
  | { kind: 'tool'; key: string; call: AgentAuditToolCall }

interface Props {
  messages: AgentAuditMessage[]
  toolCalls: AgentAuditToolCall[]
  turns: AgentAuditTurn[]
  isLoading: boolean
}

function toolStatus(status: string): 'executing' | 'awaiting_confirmation' | 'done' | 'error' | 'cancelled' {
  switch (status) {
    case 'success':
    case 'completed':
      return 'done'
    case 'confirmation_required':
    case 'await_confirm':
      return 'awaiting_confirmation'
    case 'cancelled':
    case 'canceled':
      return 'cancelled'
    case 'running':
      return 'executing'
    default:
      return 'error'
  }
}

function argsRecord(v: unknown): Record<string, unknown> {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>
  return {}
}

function stringifyResult(v: unknown): string {
  if (v === null || v === undefined || v === '') return ''
  if (typeof v === 'string') return v
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

export function SessionConversationTimeline({ messages, toolCalls, turns, isLoading }: Props) {
  const { t } = useTranslation()

  const timeline = useMemo<TimelineItem[]>(() => {
    const items: TimelineItem[] = []

    const sortedTurns = [...turns].sort(
      (a, b) => new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime()
    )

    sortedTurns.forEach((turn, idx) => {
      items.push({
        kind: 'turn-divider',
        key: `turn-${turn.turnId}`,
        index: idx + 1,
        mode: turn.orchestrationMode || 'single_agent',
        turnId: turn.turnId,
        at: turn.startedAt,
      })
    })

    messages.forEach((m) =>
      items.push({ kind: 'message', key: `m-${m.id}`, msg: m })
    )
    toolCalls.forEach((tc) =>
      items.push({ kind: 'tool', key: `t-${tc.id}`, call: tc })
    )

    items.sort((a, b) => {
      const ta = a.kind === 'turn-divider'
        ? new Date(a.at).getTime() - 1
        : new Date(a.kind === 'message' ? a.msg.createdAt : a.call.createdAt).getTime()
      const tb = b.kind === 'turn-divider'
        ? new Date(b.at).getTime() - 1
        : new Date(b.kind === 'message' ? b.msg.createdAt : b.call.createdAt).getTime()
      return ta - tb
    })

    return items
  }, [messages, toolCalls, turns])

  if (isLoading) {
    return (
      <div className="text-muted-foreground flex h-full items-center justify-center">
        <LoadingCircleIcon className="mr-2 h-5 w-5 animate-spin" />
        {t('agentAudit.timeline.loading')}
      </div>
    )
  }

  if (timeline.length === 0) {
    return (
      <div className="flex h-full items-center justify-center py-12">
        <NothingCore title={t('agentAudit.timeline.empty')} />
      </div>
    )
  }

  return (
    <div className="space-y-3 p-4">
      {timeline.map((item) => {
        if (item.kind === 'turn-divider') {
          return (
            <div key={item.key} className="flex items-center gap-3 py-2">
              <div className="bg-border h-px flex-1" />
              <Badge variant="outline" className="shrink-0 text-[11px]">
                {t('agentAudit.timeline.turnLabel', { index: item.index, mode: item.mode })}
              </Badge>
              <div className="bg-border h-px flex-1" />
            </div>
          )
        }

        if (item.kind === 'message') {
          const m = item.msg
          const isUser = m.role === 'user'
          return (
            <div key={item.key} className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
              <div
                className={cn(
                  'max-w-[85%] rounded-lg border px-3 py-2 text-sm',
                  isUser
                    ? 'border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-100'
                    : m.role === 'tool'
                      ? 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950/40'
                      : 'bg-card'
                )}
              >
                <div className="text-muted-foreground mb-1 flex items-center gap-2 text-[11px]">
                  <span>{t(`agentAudit.message.role.${m.role}`, { defaultValue: m.role })}</span>
                  {m.toolName && <Badge variant="secondary" className="text-[10px]">{m.toolName}</Badge>}
                  <span className="ml-auto">{new Date(m.createdAt).toLocaleTimeString()}</span>
                </div>
                <pre className="max-h-60 overflow-auto whitespace-pre-wrap text-sm">
                  {m.content || '-'}
                </pre>
              </div>
            </div>
          )
        }

        const call = item.call
        return (
          <div key={item.key} className="pl-2">
            <ToolCallCard
              toolName={call.toolName}
              args={argsRecord(call.toolArgs)}
              status={toolStatus(call.resultStatus)}
              resultSummary={stringifyResult(call.toolResult)}
            />
          </div>
        )
      })}
    </div>
  )
}
