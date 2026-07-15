import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  BotIcon,
  ClipboardCheckIcon,
  MessageSquareMoreIcon,
  SearchIcon,
  SettingsIcon,
} from 'lucide-react'
import { useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { TimeDistance } from '@/components/custom/time-distance'

import {
  AgentAuditSessionListItem,
  AgentAuditSessionSource,
  apiAdminListAgentAuditSessions,
} from '@/services/api/admin/agentAudit'

import { cn } from '@/lib/utils'

const sourceMeta: Record<
  AgentAuditSessionSource,
  { label: string; icon: typeof MessageSquareMoreIcon; className: string }
> = {
  chat: {
    label: '用户对话',
    icon: MessageSquareMoreIcon,
    className: 'border-blue-200 bg-blue-50 text-blue-700',
  },
  ops_audit: {
    label: '审批 Agent',
    icon: ClipboardCheckIcon,
    className: 'border-amber-200 bg-amber-50 text-amber-700',
  },
  system: {
    label: '系统任务',
    icon: SettingsIcon,
    className: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  },
}

function actorName(session: AgentAuditSessionListItem) {
  return session.nickname || session.username || '系统'
}

function accountName(session: AgentAuditSessionListItem) {
  return session.accountNickname || session.accountName || '系统'
}

function sourceBadge(source: AgentAuditSessionSource) {
  const meta = sourceMeta[source]
  return (
    <Badge variant="outline" className={cn('text-[11px]', meta.className)}>
      {meta.label}
    </Badge>
  )
}

function evalStatusText(session: AgentAuditSessionListItem) {
  switch (session.latestEvalStatus) {
    case 'completed':
      return '已评测'
    case 'running':
      return '评测中'
    case 'pending':
      return '排队中'
    case 'failed':
      return '评测失败'
    default:
      return '未评测'
  }
}

function feedbackText(session: AgentAuditSessionListItem) {
  if (!session.hasFeedback) return '无反馈'
  if (session.feedbackRating === 1) return '正向'
  if (session.feedbackRating === -1) return '负向'
  return '有反馈'
}

function SummaryButton({
  source,
  count,
  selected,
  onClick,
}: {
  source: AgentAuditSessionSource
  count: number
  selected: boolean
  onClick: () => void
}) {
  const meta = sourceMeta[source]
  const Icon = meta.icon
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex min-w-0 items-center justify-between rounded-xl border px-4 py-3 text-left transition-colors',
        selected ? meta.className : 'bg-card hover:bg-muted/50'
      )}
    >
      <span className="flex min-w-0 items-center gap-2 text-sm">
        <Icon className="h-4 w-4 shrink-0" />
        <span className="truncate">{meta.label}</span>
      </span>
      <span className="ml-3 font-mono text-lg font-semibold tabular-nums">{count}</span>
    </button>
  )
}

export function AgentAuditSessionsDialog() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [sourceFilter, setSourceFilter] = useState<AgentAuditSessionSource | 'all'>('all')

  const sessionsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'sessions', 'dialog', sourceFilter, keyword],
    queryFn: async () =>
      (
        await apiAdminListAgentAuditSessions({
          source: sourceFilter,
          keyword,
          limit: 100,
        })
      ).data,
    enabled: open,
  })

  const summary = sessionsQuery.data?.summary
  const filteredSessions = useMemo(() => {
    return sessionsQuery.data?.items ?? []
  }, [sessionsQuery.data?.items])

  const openDetail = (session: AgentAuditSessionListItem) => {
    setOpen(false)
    navigate({
      to: '/admin/more/agent-audit/$sessionId',
      params: { sessionId: session.sessionId },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <BotIcon className="h-4 w-4" />
          智能体审计
        </Button>
      </DialogTrigger>
      <DialogContent className="flex h-[86vh] max-h-[86vh] flex-col overflow-hidden sm:max-w-[1120px]">
        <DialogHeader>
          <DialogTitle>智能体审计</DialogTitle>
        </DialogHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-4">
          <div className="grid gap-3 md:grid-cols-3">
            <SummaryButton
              source="chat"
              count={summary?.chat ?? 0}
              selected={sourceFilter === 'chat'}
              onClick={() => setSourceFilter((curr) => (curr === 'chat' ? 'all' : 'chat'))}
            />
            <SummaryButton
              source="ops_audit"
              count={summary?.opsAudit ?? 0}
              selected={sourceFilter === 'ops_audit'}
              onClick={() =>
                setSourceFilter((curr) => (curr === 'ops_audit' ? 'all' : 'ops_audit'))
              }
            />
            <SummaryButton
              source="system"
              count={summary?.system ?? 0}
              selected={sourceFilter === 'system'}
              onClick={() => setSourceFilter((curr) => (curr === 'system' ? 'all' : 'system'))}
            />
          </div>

          <div className="relative">
            <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="搜索会话、用户、账户或 session id"
              className="pl-9"
            />
          </div>

          <div className="min-h-0 flex-1 overflow-auto rounded-md border">
            <Table>
              <TableHeader className="bg-background sticky top-0 z-10">
                <TableRow>
                  <TableHead>会话</TableHead>
                  <TableHead>来源</TableHead>
                  <TableHead>执行者</TableHead>
                  <TableHead>账户</TableHead>
                  <TableHead className="text-right">消息</TableHead>
                  <TableHead className="text-right">工具</TableHead>
                  <TableHead>评测</TableHead>
                  <TableHead>反馈</TableHead>
                  <TableHead>更新</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessionsQuery.isLoading ? (
                  <TableRow>
                    <TableCell colSpan={10} className="text-muted-foreground h-48 text-center">
                      加载中...
                    </TableCell>
                  </TableRow>
                ) : filteredSessions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={10} className="text-muted-foreground h-48 text-center">
                      {sourceFilter === 'all'
                        ? '暂无审计会话'
                        : `暂无${sourceMeta[sourceFilter].label}会话`}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredSessions.map((session) => {
                    const title = session.title?.trim() || '未命名会话'
                    return (
                      <TableRow key={session.sessionId}>
                        <TableCell className="max-w-[260px]">
                          <button
                            type="button"
                            onClick={() => openDetail(session)}
                            className="flex min-w-0 flex-col text-left hover:underline hover:underline-offset-4"
                          >
                            <span className="truncate font-medium">{title}</span>
                            <span className="text-muted-foreground truncate font-mono text-[11px]">
                              {session.sessionId}
                            </span>
                          </button>
                        </TableCell>
                        <TableCell>{sourceBadge(session.source)}</TableCell>
                        <TableCell className="max-w-[140px] truncate">
                          {actorName(session)}
                        </TableCell>
                        <TableCell className="max-w-[140px] truncate">
                          {accountName(session)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs tabular-nums">
                          {session.messageCount}
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs tabular-nums">
                          {session.toolCallCount}
                        </TableCell>
                        <TableCell>
                          <span className="text-muted-foreground text-xs">
                            {evalStatusText(session)}
                          </span>
                        </TableCell>
                        <TableCell>
                          <span className="text-muted-foreground text-xs">
                            {feedbackText(session)}
                          </span>
                        </TableCell>
                        <TableCell>
                          <TimeDistance date={session.updatedAt} className="text-xs" />
                        </TableCell>
                        <TableCell>
                          <Button size="sm" variant="ghost" onClick={() => openDetail(session)}>
                            详情
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
