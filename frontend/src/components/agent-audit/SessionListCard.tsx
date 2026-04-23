import { formatDistanceToNow } from 'date-fns'
import { CheckCircle2Icon, CircleDashedIcon, ThumbsDownIcon, ThumbsUpIcon, XCircleIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { AgentAuditSessionListItem, AgentAuditSessionSource } from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

interface Props {
  sessions: AgentAuditSessionListItem[]
  total: number
  isLoading: boolean
  selectedSessionId: string
  onSelectSession: (id: string) => void

  sourceFilter: AgentAuditSessionSource | 'all'
  onSourceChange: (v: AgentAuditSessionSource | 'all') => void
  keyword: string
  onKeywordChange: (v: string) => void
  hasEvalFilter: '' | 'yes' | 'no'
  onHasEvalChange: (v: '' | 'yes' | 'no') => void
  dateFrom: string
  dateTo: string
  onDateFromChange: (v: string) => void
  onDateToChange: (v: string) => void

  offset: number
  pageSize: number
  onPrev: () => void
  onNext: () => void
}

function sourceBadgeClass(source: AgentAuditSessionSource) {
  switch (source) {
    case 'chat':
      return 'border-blue-200 bg-blue-50 text-blue-700'
    case 'ops_audit':
      return 'border-amber-200 bg-amber-50 text-amber-700'
    case 'system':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700'
    case 'benchmark':
      return 'border-fuchsia-200 bg-fuchsia-50 text-fuchsia-700'
  }
}

function EvalBadge({ status }: { status?: string | null }) {
  const { t } = useTranslation()
  if (!status) return null
  switch (status) {
    case 'completed':
      return (
        <Badge variant="outline" className="gap-1 border-emerald-200 bg-emerald-50 text-emerald-700">
          <CheckCircle2Icon className="h-3 w-3" /> {t('agentAudit.eval.status.completed')}
        </Badge>
      )
    case 'running':
    case 'pending':
      return (
        <Badge variant="outline" className="gap-1 border-sky-200 bg-sky-50 text-sky-700">
          <CircleDashedIcon className="h-3 w-3 animate-spin" />
          {t(`agentAudit.eval.status.${status}`)}
        </Badge>
      )
    case 'failed':
      return (
        <Badge variant="outline" className="gap-1 border-rose-200 bg-rose-50 text-rose-700">
          <XCircleIcon className="h-3 w-3" /> {t('agentAudit.eval.status.failed')}
        </Badge>
      )
    default:
      return null
  }
}

function FeedbackBadge({ rating }: { rating?: number | null }) {
  if (rating === 1) return <ThumbsUpIcon className="h-3.5 w-3.5 text-emerald-600" />
  if (rating === -1) return <ThumbsDownIcon className="h-3.5 w-3.5 text-rose-600" />
  return null
}

export function SessionListCard({
  sessions,
  total,
  selectedSessionId,
  onSelectSession,
  sourceFilter,
  onSourceChange,
  keyword,
  onKeywordChange,
  hasEvalFilter,
  onHasEvalChange,
  dateFrom,
  dateTo,
  onDateFromChange,
  onDateToChange,
  offset,
  pageSize,
  onPrev,
  onNext,
}: Props) {
  const { t } = useTranslation()
  return (
    <div className="bg-card flex h-full flex-col">
      {/* Filter block */}
      <div className="space-y-2 border-b px-3 py-3">
        <Input
          placeholder={t('agentAudit.filters.keywordPlaceholder')}
          value={keyword}
          onChange={(e) => onKeywordChange(e.target.value)}
        />
        <div className="grid grid-cols-2 gap-2">
          <Select value={sourceFilter} onValueChange={(v) => onSourceChange(v as AgentAuditSessionSource | 'all')}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('agentAudit.filters.allSources')}</SelectItem>
              <SelectItem value="chat">{t('agentAudit.source.chat')}</SelectItem>
              <SelectItem value="ops_audit">{t('agentAudit.source.opsAudit')}</SelectItem>
              <SelectItem value="system">{t('agentAudit.source.system')}</SelectItem>
              <SelectItem value="benchmark">{t('agentAudit.source.benchmark')}</SelectItem>
            </SelectContent>
          </Select>
          <Select value={hasEvalFilter || 'all'} onValueChange={(v) => onHasEvalChange(v === 'all' ? '' : (v as 'yes' | 'no'))}>
            <SelectTrigger><SelectValue placeholder={t('agentAudit.filters.evalPlaceholder')} /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('agentAudit.filters.evalAll')}</SelectItem>
              <SelectItem value="yes">{t('agentAudit.filters.evalYes')}</SelectItem>
              <SelectItem value="no">{t('agentAudit.filters.evalNo')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <Input type="date" value={dateFrom} onChange={(e) => onDateFromChange(e.target.value)} />
          <Input type="date" value={dateTo} onChange={(e) => onDateToChange(e.target.value)} />
        </div>
      </div>

      {/* Session list */}
      <ScrollArea className="flex-1">
        <div className="space-y-2 p-3">
          {sessions.map((s) => (
            <button
              key={s.sessionId}
              type="button"
              onClick={() => onSelectSession(s.sessionId)}
              className={cn(
                'w-full rounded-lg border p-3 text-left transition-colors',
                selectedSessionId === s.sessionId
                  ? 'border-primary bg-primary/5'
                  : 'bg-card hover:bg-muted/50 border-border'
              )}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="line-clamp-2 text-sm font-medium">{s.title || s.sessionId}</div>
                <Badge variant="outline" className={cn('shrink-0 text-xs', sourceBadgeClass(s.source))}>
                  {t(`agentAudit.source.${s.source === 'ops_audit' ? 'opsAudit' : s.source}`)}
                </Badge>
              </div>
              <div className="text-muted-foreground mt-1 text-xs">
                {(s.nickname || s.username || t('agentAudit.session.systemActor'))}
                {' · '}
                {(s.accountNickname || s.accountName || t('agentAudit.session.systemAccount'))}
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-1.5 text-[11px]">
                <Badge variant="secondary">{t('agentAudit.metrics.messages', { count: s.messageCount })}</Badge>
                <Badge variant="secondary">{t('agentAudit.metrics.toolCalls', { count: s.toolCallCount })}</Badge>
                <Badge variant="secondary">{t('agentAudit.metrics.turns', { count: s.turnCount })}</Badge>
                <EvalBadge status={s.latestEvalStatus || null} />
                <FeedbackBadge rating={s.feedbackRating} />
              </div>
              <div className="text-muted-foreground mt-2 text-[11px]">
                {t('agentAudit.list.updatedAt', { time: formatDistanceToNow(new Date(s.updatedAt), { addSuffix: true }) })}
              </div>
            </button>
          ))}
        </div>
      </ScrollArea>

      {/* Pagination */}
      <div className="flex items-center justify-between border-t px-3 py-2 text-xs">
        <span className="text-muted-foreground">
          {t('agentAudit.list.range', {
            start: sessions.length === 0 ? 0 : offset + 1,
            end: offset + sessions.length,
            total,
          })}
        </span>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={offset <= 0} onClick={onPrev}>
            {t('agentAudit.pagination.previous')}
          </Button>
          <Button variant="outline" size="sm" disabled={offset + pageSize >= total} onClick={onNext}>
            {t('agentAudit.pagination.next')}
          </Button>
        </div>
      </div>
    </div>
  )
}
