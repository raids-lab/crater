import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t as tGlobal } from 'i18next'
import {
  ClipboardCheckIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  RefreshCw,
  SettingsIcon,
} from 'lucide-react'
import { ElementType, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SessionConversationTimeline } from '@/components/agent-audit/SessionConversationTimeline'
import { SessionFeedbackPanel } from '@/components/agent-audit/SessionFeedbackPanel'
import { SessionListCard } from '@/components/agent-audit/SessionListCard'
import { SessionQualityEvalPanel } from '@/components/agent-audit/SessionQualityEvalPanel'
import { SessionToolCallsPanel } from '@/components/agent-audit/SessionToolCallsPanel'
import { SessionTurnsPanel } from '@/components/agent-audit/SessionTurnsPanel'
import PageTitle from '@/components/layout/page-title'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  AgentAuditMessage,
  AgentAuditSessionSource,
  AgentAuditToolCall,
  AgentAuditTurn,
  apiAdminGetAgentAuditSessionMessages,
  apiAdminGetAgentAuditSessionToolCalls,
  apiAdminGetAgentAuditSessionTurns,
  apiAdminListAgentAuditSessions,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

const PAGE_SIZE = 40

export const Route = createFileRoute('/admin/more/agent-audit')({
  component: RouteComponent,
  loader: () => ({ crumb: tGlobal('navigation.agentAudit') }),
})

function SummaryCard({
  label,
  count,
  selected,
  onClick,
  icon: Icon,
}: {
  label: string
  count: number
  selected: boolean
  onClick: () => void
  icon: ElementType
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full rounded-xl border p-4 text-left transition-colors',
        selected ? 'border-primary bg-primary/5' : 'bg-card hover:bg-muted/50 border-border'
      )}
    >
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Icon className="h-4 w-4" />
        {label}
      </div>
      <div className="mt-2 text-2xl font-semibold">{count}</div>
    </button>
  )
}

function toDateRangeISO(date: string, kind: 'from' | 'to'): string | undefined {
  if (!date) return undefined
  const d = new Date(date)
  if (kind === 'to') d.setHours(23, 59, 59, 999)
  else d.setHours(0, 0, 0, 0)
  return d.toISOString()
}

function RouteComponent() {
  const { t } = useTranslation()
  const [sourceFilter, setSourceFilter] = useState<AgentAuditSessionSource | 'all'>('all')
  const [keyword, setKeyword] = useState('')
  const [hasEvalFilter, setHasEvalFilter] = useState<'' | 'yes' | 'no'>('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [offset, setOffset] = useState(0)
  const [selectedSessionId, setSelectedSessionId] = useState('')

  const listParams = useMemo(
    () => ({
      source: sourceFilter,
      keyword,
      hasEval: hasEvalFilter,
      from: toDateRangeISO(dateFrom, 'from'),
      to: toDateRangeISO(dateTo, 'to'),
      limit: PAGE_SIZE,
      offset,
    }),
    [sourceFilter, keyword, hasEvalFilter, dateFrom, dateTo, offset]
  )

  const sessionsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'sessions', listParams],
    queryFn: async () => (await apiAdminListAgentAuditSessions(listParams)).data,
  })

  const sessions = sessionsQuery.data?.items ?? []
  const summary = sessionsQuery.data?.summary
  const total = sessionsQuery.data?.total ?? 0
  const selectedSession =
    sessions.find((s) => s.sessionId === selectedSessionId) ?? sessions[0] ?? null

  useEffect(() => {
    if (!sessions.length) {
      setSelectedSessionId('')
      return
    }
    if (!selectedSessionId || !sessions.some((s) => s.sessionId === selectedSessionId)) {
      setSelectedSessionId(sessions[0].sessionId)
    }
  }, [selectedSessionId, sessions])

  const sessionId = selectedSession?.sessionId ?? ''

  const messagesQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'messages', sessionId],
    queryFn: async () => (await apiAdminGetAgentAuditSessionMessages(sessionId)).data as AgentAuditMessage[],
    enabled: !!sessionId,
  })
  const toolCallsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'tool-calls', sessionId],
    queryFn: async () => (await apiAdminGetAgentAuditSessionToolCalls(sessionId)).data as AgentAuditToolCall[],
    enabled: !!sessionId,
  })
  const turnsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'turns', sessionId],
    queryFn: async () => (await apiAdminGetAgentAuditSessionTurns(sessionId)).data as AgentAuditTurn[],
    enabled: !!sessionId,
  })

  return (
    <div className="space-y-4">
      <PageTitle title={t('agentAudit.page.title')} description={t('agentAudit.page.description')}>
        <Button variant="outline" size="sm" onClick={() => sessionsQuery.refetch()}>
          <RefreshCw className={cn('mr-2 h-4 w-4', sessionsQuery.isFetching && 'animate-spin')} />
          {t('common.refresh')}
        </Button>
      </PageTitle>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          label={t('agentAudit.source.chat')}
          count={summary?.chat ?? 0}
          selected={sourceFilter === 'chat'}
          onClick={() => { setSourceFilter('chat'); setOffset(0) }}
          icon={MessageSquareMoreIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.opsAudit')}
          count={summary?.opsAudit ?? 0}
          selected={sourceFilter === 'ops_audit'}
          onClick={() => { setSourceFilter('ops_audit'); setOffset(0) }}
          icon={ClipboardCheckIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.system')}
          count={summary?.system ?? 0}
          selected={sourceFilter === 'system'}
          onClick={() => { setSourceFilter('system'); setOffset(0) }}
          icon={SettingsIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.benchmark')}
          count={summary?.benchmark ?? 0}
          selected={sourceFilter === 'benchmark'}
          onClick={() => { setSourceFilter('benchmark'); setOffset(0) }}
          icon={FlaskConicalIcon}
        />
      </div>

      {sessionsQuery.isLoading ? null : sessions.length === 0 ? (
        <Card>
          <CardContent className="py-12">
            <NothingCore title={t('agentAudit.list.empty')} />
          </CardContent>
        </Card>
      ) : (
        <ResizablePanelGroup direction="horizontal" className="min-h-[78vh] rounded-xl border">
          {/* Left: session list + filters */}
          <ResizablePanel defaultSize={28} minSize={22}>
            <SessionListCard
              sessions={sessions}
              total={total}
              isLoading={sessionsQuery.isLoading}
              selectedSessionId={selectedSession?.sessionId ?? ''}
              onSelectSession={setSelectedSessionId}
              sourceFilter={sourceFilter}
              onSourceChange={(v) => { setSourceFilter(v); setOffset(0) }}
              keyword={keyword}
              onKeywordChange={(v) => { setKeyword(v); setOffset(0) }}
              hasEvalFilter={hasEvalFilter}
              onHasEvalChange={(v) => { setHasEvalFilter(v); setOffset(0) }}
              dateFrom={dateFrom}
              dateTo={dateTo}
              onDateFromChange={(v) => { setDateFrom(v); setOffset(0) }}
              onDateToChange={(v) => { setDateTo(v); setOffset(0) }}
              offset={offset}
              pageSize={PAGE_SIZE}
              onPrev={() => setOffset((c) => Math.max(0, c - PAGE_SIZE))}
              onNext={() => setOffset((c) => c + PAGE_SIZE)}
            />
          </ResizablePanel>

          <ResizableHandle withHandle />

          {/* Middle: conversation timeline */}
          <ResizablePanel defaultSize={38} minSize={30}>
            <div className="bg-card flex h-full flex-col">
              {selectedSession ? (
                <>
                  <div className="flex items-center gap-2 border-b px-4 py-3">
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{selectedSession.title || selectedSession.sessionId}</div>
                      <div className="text-muted-foreground mt-0.5 font-mono text-[11px]">{selectedSession.sessionId}</div>
                    </div>
                    <Badge variant="outline" className="text-[10px]">
                      {selectedSession.lastOrchestrationMode || 'single_agent'}
                    </Badge>
                  </div>
                  <ScrollArea className="flex-1">
                    <SessionConversationTimeline
                      messages={messagesQuery.data ?? []}
                      toolCalls={toolCallsQuery.data ?? []}
                      turns={turnsQuery.data ?? []}
                      isLoading={messagesQuery.isLoading || toolCallsQuery.isLoading || turnsQuery.isLoading}
                    />
                  </ScrollArea>
                </>
              ) : (
                <div className="flex h-full items-center justify-center">
                  <NothingCore title={t('agentAudit.detail.emptySelection')} />
                </div>
              )}
            </div>
          </ResizablePanel>

          <ResizableHandle withHandle />

          {/* Right: data panels */}
          <ResizablePanel defaultSize={34} minSize={24}>
            <div className="bg-muted/10 h-full overflow-hidden">
              {selectedSession ? (
                <Tabs defaultValue="qualityEval" className="flex h-full flex-col">
                  <TabsList className="m-2 grid grid-cols-4">
                    <TabsTrigger value="qualityEval">{t('agentAudit.tabs.qualityEval')}</TabsTrigger>
                    <TabsTrigger value="feedback">{t('agentAudit.tabs.feedback')}</TabsTrigger>
                    <TabsTrigger value="toolCalls">{t('agentAudit.tabs.toolCalls')}</TabsTrigger>
                    <TabsTrigger value="turns">{t('agentAudit.tabs.turns')}</TabsTrigger>
                  </TabsList>
                  <ScrollArea className="flex-1">
                    <div className="p-3">
                      <TabsContent value="qualityEval" className="mt-0">
                        <SessionQualityEvalPanel sessionId={selectedSession.sessionId} />
                      </TabsContent>
                      <TabsContent value="feedback" className="mt-0">
                        <SessionFeedbackPanel sessionId={selectedSession.sessionId} />
                      </TabsContent>
                      <TabsContent value="toolCalls" className="mt-0">
                        <SessionToolCallsPanel sessionId={selectedSession.sessionId} />
                      </TabsContent>
                      <TabsContent value="turns" className="mt-0">
                        <SessionTurnsPanel sessionId={selectedSession.sessionId} />
                      </TabsContent>
                    </div>
                  </ScrollArea>
                </Tabs>
              ) : (
                <div className="flex h-full items-center justify-center">
                  <NothingCore title={t('agentAudit.detail.emptySelection')} />
                </div>
              )}
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      )}
    </div>
  )
}
