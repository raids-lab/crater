import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import {
  AgentAuditEvent,
  AgentAuditTurn,
  apiAdminGetAgentAuditSessionTurns,
  apiAdminGetAgentAuditTurnEvents,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

interface Props {
  sessionId: string
}

function stringify(v: unknown): string {
  if (v === null || v === undefined || v === '') return ''
  if (typeof v === 'string') return v
  try { return JSON.stringify(v, null, 2) } catch { return String(v) }
}

export function SessionTurnsPanel({ sessionId }: Props) {
  const { t } = useTranslation()
  const [selectedTurnId, setSelectedTurnId] = useState('')
  const turnsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'turns', sessionId],
    queryFn: async () => (await apiAdminGetAgentAuditSessionTurns(sessionId)).data as AgentAuditTurn[],
    enabled: !!sessionId,
  })
  const turns = turnsQuery.data ?? []
  const activeTurnId = selectedTurnId || turns[0]?.turnId || ''

  const eventsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'events', activeTurnId],
    queryFn: async () => (await apiAdminGetAgentAuditTurnEvents(activeTurnId)).data as AgentAuditEvent[],
    enabled: !!activeTurnId,
  })

  if (turnsQuery.isLoading) return <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
  if (turns.length === 0) {
    return (
      <Card>
        <CardContent className="py-12">
          <NothingCore title={t('agentAudit.turns.empty')} />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="grid gap-3 lg:grid-cols-[180px_minmax(0,1fr)]">
      <div className="space-y-2">
        {turns.map((turn) => (
          <button
            key={turn.turnId}
            type="button"
            onClick={() => setSelectedTurnId(turn.turnId)}
            className={cn(
              'w-full rounded-lg border p-2 text-left text-xs transition-colors',
              activeTurnId === turn.turnId
                ? 'border-primary bg-primary/5'
                : 'bg-card hover:bg-muted/50 border-border'
            )}
          >
            <div className="flex items-center justify-between gap-1">
              <Badge variant="outline" className="text-[10px]">{turn.status}</Badge>
              <span className="text-muted-foreground text-[10px]">{turn.orchestrationMode}</span>
            </div>
            <div className="text-muted-foreground mt-1 text-[10px]">{new Date(turn.startedAt).toLocaleString()}</div>
          </button>
        ))}
      </div>
      <div className="space-y-2">
        {(eventsQuery.data ?? []).map((ev) => (
          <Card key={ev.id}>
            <CardContent className="space-y-1 pt-3 text-xs">
              <div className="flex items-center gap-2">
                <Badge variant="outline">{ev.eventType}</Badge>
                {ev.eventStatus && <Badge variant="secondary">{ev.eventStatus}</Badge>}
                {ev.agentRole && <Badge variant="secondary">{ev.agentRole}</Badge>}
                <span className="text-muted-foreground ml-auto">
                  #{ev.sequence} · {new Date(ev.createdAt).toLocaleTimeString()}
                </span>
              </div>
              {ev.title && <div className="font-medium">{ev.title}</div>}
              {ev.content && (
                <pre className="bg-muted/40 max-h-40 overflow-auto whitespace-pre-wrap rounded p-2">{ev.content}</pre>
              )}
              {ev.metadata !== null && ev.metadata !== undefined && (
                <pre className="bg-muted/20 text-muted-foreground max-h-32 overflow-auto whitespace-pre-wrap rounded p-2">
                  {stringify(ev.metadata)}
                </pre>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
