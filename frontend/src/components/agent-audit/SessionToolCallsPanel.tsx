import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

import { ToolCallCard } from '@/components/aiops/ToolCallCard'
import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'

import {
  AgentAuditToolCall,
  apiAdminGetAgentAuditSessionToolCalls,
} from '@/services/api/admin/agentAudit'

interface Props {
  sessionId: string
}

function toolStatus(
  status: string
): 'executing' | 'awaiting_confirmation' | 'done' | 'error' | 'cancelled' {
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

export function SessionToolCallsPanel({ sessionId }: Props) {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: ['admin', 'agent-audit', 'tool-calls', sessionId],
    queryFn: async () =>
      (await apiAdminGetAgentAuditSessionToolCalls(sessionId)).data as AgentAuditToolCall[],
    enabled: !!sessionId,
  })

  if (q.isLoading) return <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
  const items = q.data ?? []
  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="py-12">
          <NothingCore title={t('agentAudit.toolCalls.empty')} />
        </CardContent>
      </Card>
    )
  }
  return (
    <div className="space-y-3">
      {items.map((tc) => (
        <Card key={tc.id}>
          <CardContent className="space-y-2 pt-4">
            <div className="flex flex-wrap items-center gap-2 text-xs">
              <Badge variant="outline">
                {t(`agentAudit.toolCall.source.${tc.source ?? 'backend'}`)}
              </Badge>
              {tc.agentRole && <Badge variant="secondary">{tc.agentRole}</Badge>}
              {tc.executionBackend && <Badge variant="secondary">{tc.executionBackend}</Badge>}
              <span className="text-muted-foreground ml-auto">
                {new Date(tc.createdAt).toLocaleString()}
              </span>
            </div>
            <ToolCallCard
              toolName={tc.toolName}
              args={argsRecord(tc.toolArgs)}
              status={toolStatus(tc.resultStatus)}
              resultSummary={stringifyResult(tc.toolResult)}
            />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
