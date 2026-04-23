import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  ActivityIcon,
  CalendarIcon,
  ClockIcon,
  CreditCardIcon,
  LayersIcon,
  MessageSquareIcon,
  MessagesSquareIcon,
  SparklesIcon,
  ThumbsUpIcon,
  UserRoundIcon,
  WrenchIcon,
} from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { SessionConversationTimeline } from '@/components/agent-audit/SessionConversationTimeline'
import { SessionFeedbackPanel } from '@/components/agent-audit/SessionFeedbackPanel'
import { SessionQualityEvalPanel } from '@/components/agent-audit/SessionQualityEvalPanel'
import { SessionToolCallsPanel } from '@/components/agent-audit/SessionToolCallsPanel'
import { SessionTurnsPanel } from '@/components/agent-audit/SessionTurnsPanel'
import { TimeDistance } from '@/components/custom/time-distance'
import PageTitle from '@/components/layout/page-title'
import DetailPage, {
  detailLinkOptions,
  detailValidateSearch,
} from '@/components/layout/detail-page'
import { Badge } from '@/components/ui/badge'
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
  validateSearch: detailValidateSearch,
  component: AgentAuditDetailPage,
  loader: ({ params }) => ({
    // The breadcrumb is rendered by the shell; keep it short so it doesn't
    // push the rest of the path off screen on narrow viewports.
    crumb: params.sessionId.slice(0, 8) + '…',
  }),
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

function AgentAuditDetailPage() {
  const { t } = useTranslation()
  const { sessionId } = Route.useParams()
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()

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
    return session.lastOrchestrationMode ? [session.lastOrchestrationMode] : []
  }, [session])

  const messageCount = session?.messageCount ?? messagesQuery.data?.length ?? 0
  const toolCallCount = session?.toolCallCount ?? toolCallsQuery.data?.length ?? 0
  const turnCount = session?.turnCount ?? turnsQuery.data?.length ?? 0

  const feedbackLabel = useMemo(() => {
    if (!session || !session.hasFeedback) return '-'
    if (session.feedbackRating === 1) return t('agentAudit.feedback.ratingUp')
    if (session.feedbackRating === -1) return t('agentAudit.feedback.ratingDown')
    return '-'
  }, [session, t])

  const evalLabel = useMemo(() => {
    if (!session || !session.latestEvalStatus) return t('agentAudit.eval.status.none')
    return t(`agentAudit.eval.status.${session.latestEvalStatus}`, {
      defaultValue: session.latestEvalStatus,
    })
  }, [session, t])

  const header = (
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
  )

  const info = [
    {
      title: t('agentAudit.detail.owner'),
      icon: UserRoundIcon,
      value: <span className="truncate">{ownerText}</span>,
    },
    {
      title: t('agentAudit.detail.account'),
      icon: CreditCardIcon,
      value: <span className="truncate">{accountText}</span>,
    },
    {
      title: t('agentAudit.detail.orchestration'),
      icon: LayersIcon,
      value:
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
        ),
    },
    {
      title: t('agentAudit.table.evalStatus'),
      icon: SparklesIcon,
      value: <span>{evalLabel}</span>,
    },
    {
      title: t('agentAudit.metrics.messages', { count: messageCount }),
      icon: MessageSquareIcon,
      value: (
        <span className="font-mono tabular-nums text-sky-700 dark:text-sky-400">
          {messageCount}
        </span>
      ),
    },
    {
      title: t('agentAudit.metrics.toolCalls', { count: toolCallCount }),
      icon: WrenchIcon,
      value: (
        <span className="font-mono tabular-nums text-orange-700 dark:text-orange-400">
          {toolCallCount}
        </span>
      ),
    },
    {
      title: t('agentAudit.metrics.turns', { count: turnCount }),
      icon: ActivityIcon,
      value: (
        <span className="font-mono tabular-nums text-violet-700 dark:text-violet-400">
          {turnCount}
        </span>
      ),
    },
    {
      title: t('agentAudit.feedback.title'),
      icon: ThumbsUpIcon,
      value: <span>{feedbackLabel}</span>,
    },
    {
      title: t('agentAudit.detail.updatedAt'),
      icon: ClockIcon,
      value: session?.updatedAt ? <TimeDistance date={session.updatedAt} /> : '-',
    },
    {
      title: t('agentAudit.detail.createdAt', { defaultValue: '创建于' }),
      icon: CalendarIcon,
      value: session?.createdAt ? <TimeDistance date={session.createdAt} /> : '-',
    },
  ]

  const tabs = [
    {
      key: 'timeline',
      icon: MessagesSquareIcon,
      label: t('agentAudit.tabs.timeline'),
      scrollable: true,
      children: (
        <SessionConversationTimeline
          messages={messagesQuery.data ?? []}
          toolCalls={toolCallsQuery.data ?? []}
          turns={turnsQuery.data ?? []}
          isLoading={
            messagesQuery.isLoading || toolCallsQuery.isLoading || turnsQuery.isLoading
          }
        />
      ),
    },
    {
      key: 'qualityEval',
      icon: SparklesIcon,
      label: t('agentAudit.tabs.qualityEval'),
      scrollable: true,
      children: <SessionQualityEvalPanel sessionId={sessionId} />,
    },
    {
      key: 'feedback',
      icon: ThumbsUpIcon,
      label: t('agentAudit.tabs.feedback'),
      scrollable: true,
      children: <SessionFeedbackPanel sessionId={sessionId} />,
    },
    {
      key: 'toolCalls',
      icon: WrenchIcon,
      label: t('agentAudit.tabs.toolCalls'),
      scrollable: true,
      children: <SessionToolCallsPanel sessionId={sessionId} />,
    },
    {
      key: 'turns',
      icon: ActivityIcon,
      label: t('agentAudit.tabs.turns'),
      scrollable: true,
      children: <SessionTurnsPanel sessionId={sessionId} />,
    },
  ]

  return (
    <DetailPage
      header={header}
      info={info}
      tabs={tabs}
      currentTab={tab}
      setCurrentTab={(nextTab) => navigate(detailLinkOptions(nextTab))}
    />
  )
}
