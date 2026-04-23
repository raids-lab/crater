import { useQuery } from '@tanstack/react-query'
import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { ArrowLeftIcon, ExternalLinkIcon } from 'lucide-react'
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
  AgentAuditSessionListItem,
  AgentAuditSessionSource,
  AgentAuditToolCall,
  AgentAuditTurn,
  apiAdminGetAgentAuditSessionMessages,
  apiAdminGetAgentAuditSessionToolCalls,
  apiAdminGetAgentAuditSessionTurns,
  apiAdminListAgentAuditSessions,
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
  const navigate = useNavigate()
  const { sessionId } = Route.useParams()

  // Fetch the session metadata by reusing the list endpoint with a keyword hit.
  // If the session cannot be found this way (very old sessions past pagination),
  // we still render the tabs by relying on the detail endpoints below.
  const sessionMetaQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'session-meta', sessionId],
    queryFn: async () => {
      const res = await apiAdminListAgentAuditSessions({ keyword: sessionId, limit: 5 })
      return res.data.items.find((s) => s.sessionId === sessionId) ?? null
    },
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

  const session: AgentAuditSessionListItem | null = sessionMetaQuery.data ?? null

  const ownerText = useMemo(() => {
    if (!session) return '-'
    return session.nickname || session.username || t('agentAudit.session.systemActor')
  }, [session, t])

  const accountText = useMemo(() => {
    if (!session) return '-'
    return session.accountNickname || session.accountName || t('agentAudit.session.systemAccount')
  }, [session, t])

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate({ to: '/admin/more/agent-audit' })}
        >
          <ArrowLeftIcon className="mr-1 h-4 w-4" />
          {t('agentAudit.detail.backToList', { defaultValue: '返回列表' })}
        </Button>
      </div>

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
            value={session?.lastOrchestrationMode || 'single_agent'}
          />
          <DetailField
            label={t('agentAudit.detail.updatedAt')}
            value={
              session?.updatedAt ? new Date(session.updatedAt).toLocaleString() : '-'
            }
          />
          <DetailField
            label={t('agentAudit.detail.counts', { defaultValue: '统计' })}
            value={
              <div className="flex flex-wrap gap-1">
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.messages', { count: session?.messageCount ?? 0 })}
                </Badge>
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.toolCalls', { count: session?.toolCallCount ?? 0 })}
                </Badge>
                <Badge variant="secondary" className="text-[10px]">
                  {t('agentAudit.metrics.turns', { count: session?.turnCount ?? 0 })}
                </Badge>
              </div>
            }
          />
        </CardContent>
      </Card>

      {sessionMetaQuery.isLoading &&
      messagesQuery.isLoading &&
      toolCallsQuery.isLoading &&
      turnsQuery.isLoading ? (
        <Card>
          <CardContent className="text-muted-foreground py-12 text-center">
            <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
            <p className="mt-3">{t('common.loading')}</p>
          </CardContent>
        </Card>
      ) : !session &&
        !messagesQuery.isLoading &&
        (messagesQuery.data ?? []).length === 0 ? (
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
