import { useQuery } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { ExternalLinkIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { SessionConversationTimeline } from '@/components/agent-audit/SessionConversationTimeline'
import { SessionFeedbackPanel } from '@/components/agent-audit/SessionFeedbackPanel'
import { SessionQualityEvalPanel } from '@/components/agent-audit/SessionQualityEvalPanel'
import { SessionToolCallsPanel } from '@/components/agent-audit/SessionToolCallsPanel'
import { SessionTurnsPanel } from '@/components/agent-audit/SessionTurnsPanel'
import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import PageTitle from '@/components/layout/page-title'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  AgentAuditMessage,
  AgentAuditSessionSource,
  AgentAuditToolCall,
  AgentAuditTurn,
  apiAdminGetAgentAuditSessionDetail,
  apiAdminGetAgentAuditSessionMessages,
  apiAdminGetAgentAuditSessionToolCalls,
  apiAdminGetAgentAuditSessionTurns,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/admin/more/agent-audit/$sessionId')({
  component: AgentAuditDetailPage,
})

function sourceBadgeClass(source?: AgentAuditSessionSource) {
  switch (source) {
    case 'chat':
      return 'border-blue-200 bg-blue-50 text-blue-700'
    case 'ops_audit':
      return 'border-amber-200 bg-amber-50 text-amber-700'
    case 'system':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700'
    case 'benchmark':
      return 'border-fuchsia-200 bg-fuchsia-50 text-fuchsia-700'
    default:
      return ''
  }
}

function orchestrationModeBadgeClass(mode: string) {
  if (mode === 'multi_agent') return 'border-violet-200 bg-violet-50 text-violet-700'
  return 'border-sky-200 bg-sky-50 text-sky-700'
}

function DetailField({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="text-sm font-medium">{value}</div>
    </div>
  )
}

function AgentAuditDetailPage() {
  const { t } = useTranslation()
  const { sessionId } = Route.useParams()

  const sessionDetailQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'session-detail', sessionId],
    queryFn: async () => (await apiAdminGetAgentAuditSessionDetail(sessionId)).data,
    enabled: !!sessionId,
    retry: false,
  })

  const messagesQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'messages', sessionId],
    queryFn: async () =>
      (await apiAdminGetAgentAuditSessionMessages(sessionId)).data as AgentAuditMessage[],
    enabled: !!sessionId,
  })
  const toolCallsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'tool-calls', sessionId],
    queryFn: async () =>
      (await apiAdminGetAgentAuditSessionToolCalls(sessionId)).data as AgentAuditToolCall[],
    enabled: !!sessionId,
  })
  const turnsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'turns', sessionId],
    queryFn: async () =>
      (await apiAdminGetAgentAuditSessionTurns(sessionId)).data as AgentAuditTurn[],
    enabled: !!sessionId,
  })

  const session = sessionDetailQuery.data ?? null

  const ownerText = useMemo(() => {
    if (!session) return '-'
    return session.nickname || session.username || t('agentAudit.session.systemActor')
  }, [session, t])

  const accountText = useMemo(() => {
    if (!session) return '-'
    return session.accountNickname || session.accountName || t('agentAudit.session.systemAccount')
  }, [session, t])

  const modes = useMemo(() => {
    if (!session) return []
    if (session.orchestrationModes && session.orchestrationModes.length > 0) {
      return session.orchestrationModes
    }
    if (session.lastOrchestrationMode) return [session.lastOrchestrationMode]
    return []
  }, [session])

  // Counts: prefer server-enriched item but fall back to the live fetched arrays
  // so the detail page remains useful even if enrichment JOIN is stale.
  const messageCount = session?.messageCount ?? messagesQuery.data?.length ?? 0
  const toolCallCount = session?.toolCallCount ?? toolCallsQuery.data?.length ?? 0
  const turnCount = session?.turnCount ?? turnsQuery.data?.length ?? 0

  const allLoading =
    sessionDetailQuery.isLoading &&
    messagesQuery.isLoading &&
    toolCallsQuery.isLoading &&
    turnsQuery.isLoading
  const notFound = sessionDetailQuery.isError || (!sessionDetailQuery.isLoading && !session)

  return (
    <div className="space-y-4">
      <PageTitle
        title={session?.title || t('agentAudit.detail.title', { defaultValue: '会话审计详情' })}
        description={sessionId}
        descriptionCopiable
      >
        {session && (
          <Badge variant="outline" className={cn('text-xs', sourceBadgeClass(session.source))}>
            {t(`agentAudit.source.${session.source === 'ops_audit' ? 'opsAudit' : session.source}`)}
          </Badge>
        )}
      </PageTitle>

      {/* Session metadata strip */}
      <Card>
        <CardContent className="grid gap-4 pt-6 md:grid-cols-2 xl:grid-cols-5">
          <DetailField label={t('agentAudit.detail.owner')} value={ownerText} />
          <DetailField label={t('agentAudit.detail.account')} value={accountText} />
          <DetailField
            label={t('agentAudit.detail.orchestration')}
            value={
              modes.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {modes.map((m) => (
                    <Badge
                      key={m}
                      variant="outline"
                      className={cn('text-[10px]', orchestrationModeBadgeClass(m))}
                    >
                      {m}
                    </Badge>
                  ))}
                </div>
              ) : (
                <span className="text-muted-foreground">-</span>
              )
            }
          />
          <DetailField
            label={t('agentAudit.detail.updatedAt')}
            value={session?.updatedAt ? new Date(session.updatedAt).toLocaleString() : '-'}
          />
          <DetailField
            label={t('agentAudit.detail.counts', { defaultValue: '统计' })}
            value={
              <div className="flex flex-wrap gap-1">
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.messages', { count: messageCount })}
                </Badge>
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.toolCalls', { count: toolCallCount })}
                </Badge>
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.turns', { count: turnCount })}
                </Badge>
              </div>
            }
          />
        </CardContent>
      </Card>

      {allLoading ? (
        <Card>
          <CardContent className="text-muted-foreground py-12 text-center">
            <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
            <p className="mt-3">{t('common.loading')}</p>
          </CardContent>
        </Card>
      ) : notFound ? (
        <Card>
          <CardContent className="py-12">
            <NothingCore
              title={t('agentAudit.detail.notFound', { defaultValue: '未找到该会话' })}
            />
            <div className="mt-4 text-center">
              <Link to="/admin/more/agent-audit">
                <Button variant="outline" size="sm">
                  <ExternalLinkIcon className="mr-1 h-3.5 w-3.5" />
                  {t('agentAudit.detail.backToList', { defaultValue: '返回列表' })}
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Tabs defaultValue="timeline" className="space-y-4">
          <TabsList className="grid w-full grid-cols-5 md:w-[640px]">
            <TabsTrigger value="timeline">{t('agentAudit.tabs.timeline')}</TabsTrigger>
            <TabsTrigger value="qualityEval">{t('agentAudit.tabs.qualityEval')}</TabsTrigger>
            <TabsTrigger value="feedback">{t('agentAudit.tabs.feedback')}</TabsTrigger>
            <TabsTrigger value="toolCalls">{t('agentAudit.tabs.toolCalls')}</TabsTrigger>
            <TabsTrigger value="turns">{t('agentAudit.tabs.turns')}</TabsTrigger>
          </TabsList>

          <TabsContent value="timeline">
            <Card>
              <CardContent className="p-0">
                <SessionConversationTimeline
                  messages={messagesQuery.data ?? []}
                  toolCalls={toolCallsQuery.data ?? []}
                  turns={turnsQuery.data ?? []}
                  isLoading={
                    messagesQuery.isLoading || toolCallsQuery.isLoading || turnsQuery.isLoading
                  }
                />
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="qualityEval">
            <SessionQualityEvalPanel sessionId={sessionId} />
          </TabsContent>

          <TabsContent value="feedback">
            <SessionFeedbackPanel sessionId={sessionId} />
          </TabsContent>

          <TabsContent value="toolCalls">
            <SessionToolCallsPanel sessionId={sessionId} />
          </TabsContent>

          <TabsContent value="turns">
            <SessionTurnsPanel sessionId={sessionId} />
          </TabsContent>
        </Tabs>
      )}
    </div>
  )
}
