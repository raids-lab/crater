import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { formatDistanceToNow } from 'date-fns'
import {
  CheckCircle2Icon,
  ChevronRightIcon,
  CircleDashedIcon,
  ClipboardCheckIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  RefreshCw,
  SettingsIcon,
  ThumbsDownIcon,
  ThumbsUpIcon,
  XCircleIcon,
} from 'lucide-react'
import { ElementType, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import PageTitle from '@/components/layout/page-title'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AgentAuditSessionListItem,
  AgentAuditSessionSource,
  apiAdminListAgentAuditSessions,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

const PAGE_SIZE = 40

export const Route = createFileRoute('/admin/more/agent-audit/')({
  component: AgentAuditListPage,
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

function EvalStatusBadge({ status }: { status?: string | null }) {
  const { t } = useTranslation()
  if (!status) {
    return (
      <Badge variant="outline" className="text-muted-foreground gap-1 text-[10px]">
        {t('agentAudit.eval.status.none')}
      </Badge>
    )
  }
  switch (status) {
    case 'completed':
      return (
        <Badge variant="outline" className="gap-1 border-emerald-200 bg-emerald-50 text-emerald-700">
          <CheckCircle2Icon className="h-3 w-3" />
          {t('agentAudit.eval.status.completed')}
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
          <XCircleIcon className="h-3 w-3" />
          {t('agentAudit.eval.status.failed')}
        </Badge>
      )
    default:
      return null
  }
}

function FeedbackCell({ rating, has }: { rating?: number | null; has: boolean }) {
  if (!has) return <span className="text-muted-foreground text-xs">-</span>
  if (rating === 1) {
    return (
      <div className="inline-flex items-center gap-1 text-emerald-600">
        <ThumbsUpIcon className="h-3.5 w-3.5" />
      </div>
    )
  }
  if (rating === -1) {
    return (
      <div className="inline-flex items-center gap-1 text-rose-600">
        <ThumbsDownIcon className="h-3.5 w-3.5" />
      </div>
    )
  }
  return <span className="text-muted-foreground text-xs">-</span>
}

function toDateRangeISO(date: string, kind: 'from' | 'to'): string | undefined {
  if (!date) return undefined
  const d = new Date(date)
  if (kind === 'to') d.setHours(23, 59, 59, 999)
  else d.setHours(0, 0, 0, 0)
  return d.toISOString()
}

function AgentAuditListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [sourceFilter, setSourceFilter] = useState<AgentAuditSessionSource | 'all'>('all')
  const [keyword, setKeyword] = useState('')
  const [hasEvalFilter, setHasEvalFilter] = useState<'' | 'yes' | 'no'>('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [offset, setOffset] = useState(0)

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

  useEffect(() => {
    setOffset(0)
  }, [sourceFilter, keyword, hasEvalFilter, dateFrom, dateTo])

  const sessions = sessionsQuery.data?.items ?? []
  const summary = sessionsQuery.data?.summary
  const total = sessionsQuery.data?.total ?? 0

  const goDetail = (s: AgentAuditSessionListItem) => {
    navigate({ to: '/admin/more/agent-audit/$sessionId', params: { sessionId: s.sessionId } })
  }

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
          onClick={() => setSourceFilter((v) => (v === 'chat' ? 'all' : 'chat'))}
          icon={MessageSquareMoreIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.opsAudit')}
          count={summary?.opsAudit ?? 0}
          selected={sourceFilter === 'ops_audit'}
          onClick={() => setSourceFilter((v) => (v === 'ops_audit' ? 'all' : 'ops_audit'))}
          icon={ClipboardCheckIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.system')}
          count={summary?.system ?? 0}
          selected={sourceFilter === 'system'}
          onClick={() => setSourceFilter((v) => (v === 'system' ? 'all' : 'system'))}
          icon={SettingsIcon}
        />
        <SummaryCard
          label={t('agentAudit.source.benchmark')}
          count={summary?.benchmark ?? 0}
          selected={sourceFilter === 'benchmark'}
          onClick={() => setSourceFilter((v) => (v === 'benchmark' ? 'all' : 'benchmark'))}
          icon={FlaskConicalIcon}
        />
      </div>

      <Card>
        <CardContent className="grid gap-3 pt-6 md:grid-cols-[1fr_180px_180px_160px_160px]">
          <Input
            placeholder={t('agentAudit.filters.keywordPlaceholder')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <Select
            value={sourceFilter}
            onValueChange={(v) => setSourceFilter(v as AgentAuditSessionSource | 'all')}
          >
            <SelectTrigger>
              <SelectValue placeholder={t('agentAudit.filters.sourcePlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('agentAudit.filters.allSources')}</SelectItem>
              <SelectItem value="chat">{t('agentAudit.source.chat')}</SelectItem>
              <SelectItem value="ops_audit">{t('agentAudit.source.opsAudit')}</SelectItem>
              <SelectItem value="system">{t('agentAudit.source.system')}</SelectItem>
              <SelectItem value="benchmark">{t('agentAudit.source.benchmark')}</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={hasEvalFilter || 'all'}
            onValueChange={(v) => setHasEvalFilter(v === 'all' ? '' : (v as 'yes' | 'no'))}
          >
            <SelectTrigger>
              <SelectValue placeholder={t('agentAudit.filters.evalPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('agentAudit.filters.evalAll')}</SelectItem>
              <SelectItem value="yes">{t('agentAudit.filters.evalYes')}</SelectItem>
              <SelectItem value="no">{t('agentAudit.filters.evalNo')}</SelectItem>
            </SelectContent>
          </Select>
          <Input
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            placeholder={t('agentAudit.filters.dateFrom')}
          />
          <Input
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            placeholder={t('agentAudit.filters.dateTo')}
          />
        </CardContent>
      </Card>

      {sessionsQuery.isLoading ? (
        <Card>
          <CardContent className="text-muted-foreground py-12 text-center">
            <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
            <p className="mt-3">{t('common.loading')}</p>
          </CardContent>
        </Card>
      ) : sessions.length === 0 ? (
        <Card>
          <CardContent className="py-12">
            <NothingCore title={t('agentAudit.list.empty')} />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[28%]">{t('agentAudit.table.title')}</TableHead>
                  <TableHead>{t('agentAudit.table.source')}</TableHead>
                  <TableHead>{t('agentAudit.table.owner')}</TableHead>
                  <TableHead>{t('agentAudit.table.account')}</TableHead>
                  <TableHead className="text-right">{t('agentAudit.table.msgCount')}</TableHead>
                  <TableHead className="text-right">{t('agentAudit.table.toolCount')}</TableHead>
                  <TableHead className="text-right">{t('agentAudit.table.turnCount')}</TableHead>
                  <TableHead>{t('agentAudit.table.evalStatus')}</TableHead>
                  <TableHead className="text-center">{t('agentAudit.table.feedback')}</TableHead>
                  <TableHead>{t('agentAudit.table.updatedAt')}</TableHead>
                  <TableHead className="w-10"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((s) => (
                  <TableRow
                    key={s.sessionId}
                    className="hover:bg-muted/50 cursor-pointer"
                    onClick={() => goDetail(s)}
                  >
                    <TableCell>
                      <div className="min-w-0">
                        <div className="line-clamp-1 text-sm font-medium">
                          {s.title || t('agentAudit.session.untitled', { defaultValue: '未命名会话' })}
                        </div>
                        <div className="text-muted-foreground mt-0.5 font-mono text-[11px]">
                          {s.sessionId.slice(0, 8)}…
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className={cn('text-[11px]', sourceBadgeClass(s.source))}>
                        {t(`agentAudit.source.${s.source === 'ops_audit' ? 'opsAudit' : s.source}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {s.nickname || s.username || (
                        <span className="text-muted-foreground">{t('agentAudit.session.systemActor')}</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm">
                      {s.accountNickname || s.accountName || (
                        <span className="text-muted-foreground">{t('agentAudit.session.systemAccount')}</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs">{s.messageCount}</TableCell>
                    <TableCell className="text-right font-mono text-xs">{s.toolCallCount}</TableCell>
                    <TableCell className="text-right font-mono text-xs">{s.turnCount}</TableCell>
                    <TableCell>
                      <EvalStatusBadge status={s.latestEvalStatus || null} />
                    </TableCell>
                    <TableCell className="text-center">
                      <FeedbackCell rating={s.feedbackRating} has={s.hasFeedback} />
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {formatDistanceToNow(new Date(s.updatedAt), { addSuffix: true })}
                    </TableCell>
                    <TableCell>
                      <ChevronRightIcon className="text-muted-foreground h-4 w-4" />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <div className="flex items-center justify-between">
        <span className="text-muted-foreground text-xs">
          {t('agentAudit.list.range', {
            start: sessions.length === 0 ? 0 : offset + 1,
            end: offset + sessions.length,
            total,
          })}
        </span>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={offset <= 0}
            onClick={() => setOffset((c) => Math.max(0, c - PAGE_SIZE))}
          >
            {t('agentAudit.pagination.previous')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={offset + PAGE_SIZE >= total}
            onClick={() => setOffset((c) => c + PAGE_SIZE)}
          >
            {t('agentAudit.pagination.next')}
          </Button>
        </div>
      </div>
    </div>
  )
}
