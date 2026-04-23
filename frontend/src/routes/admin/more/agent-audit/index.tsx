import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import {
  CheckCircle2Icon,
  ChevronRightIcon,
  CircleDashedIcon,
  ClipboardCheckIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  SettingsIcon,
  ThumbsDownIcon,
  ThumbsUpIcon,
  XCircleIcon,
} from 'lucide-react'
import { ElementType, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { TimeDistance } from '@/components/custom/time-distance'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  AgentAuditSessionListItem,
  AgentAuditSessionSource,
  apiAdminListAgentAuditSessions,
} from '@/services/api/admin/agentAudit'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/admin/more/agent-audit/')({
  component: AgentAuditListPage,
})

// ── Static option sets (match approval-order pattern) ─────────────────────────

const EVAL_STATUS_OPTIONS = [
  { value: 'completed', label: '已完成' },
  { value: 'running', label: '评测中' },
  { value: 'pending', label: '排队中' },
  { value: 'failed', label: '失败' },
  { value: 'none', label: '未评测' },
] as const

const FEEDBACK_OPTIONS = [
  { value: 'up', label: '正向 👍' },
  { value: 'down', label: '负向 👎' },
  { value: 'none', label: '无反馈' },
] as const

// ── Derived column getters ────────────────────────────────────────────────────

/** Normalise eval status to one of the faceted-filter values. */
function evalStatusValue(item: AgentAuditSessionListItem): string {
  const s = item.latestEvalStatus
  return s ? s : 'none'
}

function feedbackValue(item: AgentAuditSessionListItem): string {
  if (!item.hasFeedback) return 'none'
  if (item.feedbackRating === 1) return 'up'
  if (item.feedbackRating === -1) return 'down'
  return 'none'
}

// ── Small cell components ─────────────────────────────────────────────────────

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

function SourceBadge({ source }: { source: AgentAuditSessionSource }) {
  const { t } = useTranslation()
  return (
    <Badge variant="outline" className={cn('text-[11px]', sourceBadgeClass(source))}>
      {t(`agentAudit.source.${source === 'ops_audit' ? 'opsAudit' : source}`)}
    </Badge>
  )
}

function EvalStatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  switch (status) {
    case 'completed':
      return (
        <Badge variant="outline" className="gap-1 border-emerald-200 bg-emerald-50 text-[11px] text-emerald-700">
          <CheckCircle2Icon className="h-3 w-3" />
          {t('agentAudit.eval.status.completed')}
        </Badge>
      )
    case 'running':
    case 'pending':
      return (
        <Badge variant="outline" className="gap-1 border-sky-200 bg-sky-50 text-[11px] text-sky-700">
          <CircleDashedIcon className="h-3 w-3 animate-spin" />
          {t(`agentAudit.eval.status.${status}`)}
        </Badge>
      )
    case 'failed':
      return (
        <Badge variant="outline" className="gap-1 border-rose-200 bg-rose-50 text-[11px] text-rose-700">
          <XCircleIcon className="h-3 w-3" />
          {t('agentAudit.eval.status.failed')}
        </Badge>
      )
    default:
      return (
        <Badge variant="outline" className="text-muted-foreground text-[11px]">
          {t('agentAudit.eval.status.none')}
        </Badge>
      )
  }
}

function FeedbackIcon({ value }: { value: string }) {
  if (value === 'up') return <ThumbsUpIcon className="h-3.5 w-3.5 text-emerald-600" />
  if (value === 'down') return <ThumbsDownIcon className="h-3.5 w-3.5 text-rose-600" />
  return <span className="text-muted-foreground text-xs">-</span>
}

function TitleCell({
  session,
  onClick,
}: {
  session: AgentAuditSessionListItem
  onClick: () => void
}) {
  const { t } = useTranslation()
  const fallbackTitle = t('agentAudit.session.untitled', { defaultValue: '未命名会话' })
  const title = session.title?.trim() || fallbackTitle
  const shortId = session.sessionId.slice(0, 8)

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={onClick}
            className="group flex min-w-0 max-w-[260px] flex-col gap-0.5 text-left"
          >
            <span className="truncate text-sm font-medium group-hover:underline group-hover:underline-offset-4">
              {title}
            </span>
            <span className="text-muted-foreground font-mono text-[11px]">{shortId}…</span>
          </button>
        </TooltipTrigger>
        <TooltipContent className="max-w-sm">
          <div className="space-y-1">
            <div className="font-medium break-all">{title}</div>
            <div className="text-muted-foreground font-mono text-xs break-all">
              {session.sessionId}
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

function UserCell({ session }: { session: AgentAuditSessionListItem }) {
  const { t } = useTranslation()
  const name = session.nickname || session.username
  if (!name) {
    return <span className="text-muted-foreground text-sm">{t('agentAudit.session.systemActor')}</span>
  }
  return <span className="truncate text-sm">{name}</span>
}

function AccountCell({ session }: { session: AgentAuditSessionListItem }) {
  const { t } = useTranslation()
  const name = session.accountNickname || session.accountName
  if (!name) {
    return <span className="text-muted-foreground text-sm">{t('agentAudit.session.systemAccount')}</span>
  }
  return <span className="truncate text-sm">{name}</span>
}

function SummaryCard({
  label,
  count,
  selected,
  onClick,
  icon: Icon,
  accent,
}: {
  label: string
  count: number
  selected: boolean
  onClick: () => void
  icon: ElementType
  accent: 'blue' | 'amber' | 'emerald' | 'fuchsia'
}) {
  const accentClasses: Record<string, { border: string; bg: string; text: string; iconBg: string }> = {
    blue: {
      border: 'border-blue-300',
      bg: 'bg-blue-50/60',
      text: 'text-blue-700',
      iconBg: 'bg-blue-100 text-blue-600',
    },
    amber: {
      border: 'border-amber-300',
      bg: 'bg-amber-50/60',
      text: 'text-amber-700',
      iconBg: 'bg-amber-100 text-amber-600',
    },
    emerald: {
      border: 'border-emerald-300',
      bg: 'bg-emerald-50/60',
      text: 'text-emerald-700',
      iconBg: 'bg-emerald-100 text-emerald-600',
    },
    fuchsia: {
      border: 'border-fuchsia-300',
      bg: 'bg-fuchsia-50/60',
      text: 'text-fuchsia-700',
      iconBg: 'bg-fuchsia-100 text-fuchsia-600',
    },
  }
  const c = accentClasses[accent]
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full rounded-xl border p-4 text-left transition-all',
        selected
          ? `${c.border} ${c.bg} shadow-sm`
          : 'bg-card hover:bg-muted/40 border-border'
      )}
    >
      <div className="flex items-center gap-2 text-sm">
        <span className={cn('flex h-7 w-7 items-center justify-center rounded-md', c.iconBg)}>
          <Icon className="h-4 w-4" />
        </span>
        <span className="text-muted-foreground">{label}</span>
      </div>
      <div className={cn('mt-2 text-2xl font-semibold tabular-nums', selected ? c.text : '')}>
        {count}
      </div>
    </button>
  )
}

// ── Column header labels ──────────────────────────────────────────────────────

const COLUMN_HEADERS: Record<string, string> = {
  title: '会话',
  source: '来源',
  owner: '执行者',
  account: '账户',
  messageCount: '消息',
  toolCallCount: '工具',
  turnCount: '轮次',
  evalStatus: '评测',
  feedback: '反馈',
  updatedAt: '最近更新',
}

const getHeaderLabel = (key: string) => COLUMN_HEADERS[key] ?? key

// ── Main page ─────────────────────────────────────────────────────────────────

function AgentAuditListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [sourceFilter, setSourceFilter] = useState<AgentAuditSessionSource | 'all'>('all')

  // Fetch up to 100 most recent sessions (backend cap). Client-side toolbar then
  // handles search + faceted filters + pagination (matches approval-order pattern).
  // Two selectors over the same cached fetch: one surfaces the summary block for
  // the overview cards, one produces the pre-filtered list that feeds DataTable.
  const SESSIONS_KEY = ['admin', 'agent-audit', 'sessions']
  const fetchSessions = async () =>
    (await apiAdminListAgentAuditSessions({ limit: 100 })).data

  const summaryQuery = useQuery({
    queryKey: SESSIONS_KEY,
    queryFn: fetchSessions,
    select: (data) => data.summary,
  })
  const summary = summaryQuery.data

  const sessionsQuery = useQuery({
    queryKey: SESSIONS_KEY,
    queryFn: fetchSessions,
    select: (data) =>
      sourceFilter === 'all'
        ? data.items
        : data.items.filter((s) => s.source === sourceFilter),
  })

  const toggleSource = (v: AgentAuditSessionSource) =>
    setSourceFilter((curr) => (curr === v ? 'all' : v))

  const goDetail = (session: AgentAuditSessionListItem) => {
    navigate({
      to: '/admin/more/agent-audit/$sessionId',
      params: { sessionId: session.sessionId },
    })
  }

  const toolbarConfig: DataTableToolbarConfig = {
    globalSearch: {
      enabled: true,
      placeholder: t('agentAudit.filters.keywordPlaceholder'),
    },
    filterOptions: [
      { key: 'evalStatus', title: COLUMN_HEADERS.evalStatus, option: [...EVAL_STATUS_OPTIONS] },
      { key: 'feedback', title: COLUMN_HEADERS.feedback, option: [...FEEDBACK_OPTIONS] },
    ],
    getHeader: getHeaderLabel,
  }

  const columns: ColumnDef<AgentAuditSessionListItem>[] = [
    {
      accessorKey: 'title',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeaderLabel('title')} />,
      cell: ({ row }) => <TitleCell session={row.original} onClick={() => goDetail(row.original)} />,
      enableColumnFilter: false,
    },
    {
      accessorKey: 'source',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeaderLabel('source')} />,
      cell: ({ row }) => <SourceBadge source={row.original.source} />,
      filterFn: (row, id, value) =>
        Array.isArray(value) ? value.includes(row.getValue(id)) : true,
    },
    {
      accessorKey: 'owner',
      accessorFn: (row) => row.nickname || row.username || '',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeaderLabel('owner')} />,
      cell: ({ row }) => <UserCell session={row.original} />,
    },
    {
      accessorKey: 'account',
      accessorFn: (row) => row.accountNickname || row.accountName || '',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeaderLabel('account')} />,
      cell: ({ row }) => <AccountCell session={row.original} />,
    },
    {
      accessorKey: 'messageCount',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('messageCount')} />
      ),
      cell: ({ row }) => (
        <span className="font-mono text-xs tabular-nums text-sky-700 dark:text-sky-400">
          {row.original.messageCount}
        </span>
      ),
      enableColumnFilter: false,
    },
    {
      accessorKey: 'toolCallCount',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('toolCallCount')} />
      ),
      cell: ({ row }) => (
        <span className="font-mono text-xs tabular-nums text-orange-700 dark:text-orange-400">
          {row.original.toolCallCount}
        </span>
      ),
      enableColumnFilter: false,
    },
    {
      accessorKey: 'turnCount',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('turnCount')} />
      ),
      cell: ({ row }) => (
        <span className="font-mono text-xs tabular-nums text-violet-700 dark:text-violet-400">
          {row.original.turnCount}
        </span>
      ),
      enableColumnFilter: false,
    },
    {
      id: 'evalStatus',
      accessorFn: (row) => evalStatusValue(row),
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('evalStatus')} />
      ),
      cell: ({ row }) => <EvalStatusBadge status={evalStatusValue(row.original)} />,
      filterFn: (row, id, value) =>
        Array.isArray(value) ? value.includes(row.getValue(id)) : true,
    },
    {
      id: 'feedback',
      accessorFn: (row) => feedbackValue(row),
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('feedback')} />
      ),
      cell: ({ row }) => <FeedbackIcon value={feedbackValue(row.original)} />,
      filterFn: (row, id, value) =>
        Array.isArray(value) ? value.includes(row.getValue(id)) : true,
    },
    {
      accessorKey: 'updatedAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeaderLabel('updatedAt')} />
      ),
      cell: ({ row }) => <TimeDistance date={row.original.updatedAt} />,
      sortingFn: 'datetime',
      enableColumnFilter: false,
    },
    {
      id: 'goto',
      header: () => null,
      cell: ({ row }) => (
        <button
          type="button"
          onClick={() => goDetail(row.original)}
          className="text-muted-foreground hover:text-foreground transition-colors"
          aria-label="查看详情"
        >
          <ChevronRightIcon className="h-4 w-4" />
        </button>
      ),
      enableSorting: false,
      enableColumnFilter: false,
    },
  ]

  return (
    <>
      <div className="mb-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          label={t('agentAudit.source.chat')}
          count={summary?.chat ?? 0}
          selected={sourceFilter === 'chat'}
          onClick={() => toggleSource('chat')}
          icon={MessageSquareMoreIcon}
          accent="blue"
        />
        <SummaryCard
          label={t('agentAudit.source.opsAudit')}
          count={summary?.opsAudit ?? 0}
          selected={sourceFilter === 'ops_audit'}
          onClick={() => toggleSource('ops_audit')}
          icon={ClipboardCheckIcon}
          accent="amber"
        />
        <SummaryCard
          label={t('agentAudit.source.system')}
          count={summary?.system ?? 0}
          selected={sourceFilter === 'system'}
          onClick={() => toggleSource('system')}
          icon={SettingsIcon}
          accent="emerald"
        />
        <SummaryCard
          label={t('agentAudit.source.benchmark')}
          count={summary?.benchmark ?? 0}
          selected={sourceFilter === 'benchmark'}
          onClick={() => toggleSource('benchmark')}
          icon={FlaskConicalIcon}
          accent="fuchsia"
        />
      </div>

      <DataTable
        info={{
          title: t('agentAudit.page.title'),
          description: t('agentAudit.page.description'),
        }}
        storageKey="admin-agent-audit"
        query={sessionsQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
    </>
  )
}
