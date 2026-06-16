import { useMutation, useQuery } from '@tanstack/react-query'
import { Eye, History, RefreshCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { Textarea } from '@/components/ui/textarea'

import {
  ReplaySummary,
  StorageDecisionRecordDetail,
  apiAdminGetStorageDecision,
  apiAdminGetStorageDecisions,
  apiAdminReplayStorageDecisions,
  apiAdminRunAutoShrink,
} from '@/services/api/storage'

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** exponent).toFixed(1)} ${units[exponent]}`
}

function formatRatio(value?: number): string {
  if (value === undefined || Number.isNaN(value)) return 'N/A'
  return `${(value * 100).toFixed(1)}%`
}

function safeStringify(value: unknown): string {
  return JSON.stringify(value, null, 2) ?? ''
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null) {
    const candidate = error as {
      data?: { msg?: string }
      response?: { data?: { msg?: string } }
    }
    return candidate.data?.msg ?? candidate.response?.data?.msg ?? fallback
  }
  return fallback
}

function decisionStatusTone(status: string): string {
  switch (status) {
    case 'done':
      return 'bg-green-100 text-green-700 border-green-300'
    case 'error':
      return 'bg-red-100 text-red-700 border-red-300'
    case 'running':
      return 'bg-blue-100 text-blue-700 border-blue-300'
    default:
      return 'bg-yellow-100 text-yellow-700 border-yellow-300'
  }
}

function decisionStatusLabel(status: string): string {
  switch (status) {
    case 'pending':
      return '待处理'
    case 'running':
      return '执行中'
    case 'done':
      return '已完成'
    case 'error':
      return '失败'
    default:
      return status
  }
}

function decisionSourceTone(source: string): string {
  switch (source) {
    case 'patrol':
      return 'bg-sky-100 text-sky-700 border-sky-300'
    case 'manual':
      return 'bg-violet-100 text-violet-700 border-violet-300'
    default:
      return 'bg-slate-100 text-slate-700 border-slate-300'
  }
}

function decisionSourceLabel(source: string): string {
  switch (source) {
    case 'manual':
      return '手动触发'
    case 'patrol':
      return '定时巡检'
    case 'replay':
      return '回放任务'
    default:
      return source
  }
}

function appliedActionLabel(action: string): string {
  switch (action) {
    case 'expand':
      return '执行扩容'
    case 'expand_failed':
      return '扩容失败'
    case 'freeze':
      return '冻结新作业'
    case 'freeze_failed':
      return '冻结失败'
    case 'observe':
      return '仅观察'
    case 'expand_and_freeze':
      return '扩容并冻结'
    case 'manual_expand':
      return '手动执行扩容'
    case 'manual_expand_and_freeze':
      return '手动扩容并冻结'
    case 'manual_freeze':
      return '手动执行冻结'
    case 'manual_freeze_failed':
      return '手动冻结失败'
    default:
      return action || '待处理'
  }
}

function shrinkStageLabel(stage?: string): string {
  switch (stage) {
    case 'expanded':
      return '扩容阶段'
    case 'buffer_reduction':
      return '缩容缓冲期'
    default:
      return stage || '无'
  }
}

function shrinkStageReason(stage?: string): string {
  switch (stage) {
    case 'expanded':
      return '用户当前处于临时扩容后的观察阶段，系统暂不回收扩容额度。'
    case 'buffer_reduction':
      return '用户占用已回落到安全区间，系统先回收到中间缓冲配额，等待观察窗口结束后再完全恢复到原始配额。'
    default:
      return '当前没有处于自动缩容阶段。'
  }
}

export default function StorageGovernancePanel() {
  const [filters, setFilters] = useState({
    user: '',
    status: '',
    source: '',
  })
  const [isDetailOpen, setIsDetailOpen] = useState(false)
  const [selectedDecisionJobId, setSelectedDecisionJobId] = useState<string | null>(null)
  const [isReplayOpen, setIsReplayOpen] = useState(false)
  const [replaySummary, setReplaySummary] = useState<ReplaySummary | null>(null)
  const [replayConfig, setReplayConfig] = useState({
    limit: 50,
    maxExpandRatio: 0.3,
    maxExpandBytesGiB: 500,
    minPlatformReservedRatio: 0.1,
    minPlatformReservedBytesGiB: 200,
    expansionCooldownHours: 6,
    forceFreezeWhenOverQuota: true,
  })

  const decisionsQuery = useQuery({
    queryKey: ['admin', 'storage-decisions', filters],
    queryFn: () =>
      apiAdminGetStorageDecisions(1, 100, {
        user: filters.user || undefined,
        status: filters.status || undefined,
        source: filters.source || undefined,
      }).then((res) => res.data.items),
    staleTime: 30 * 1000,
  })

  const detailQuery = useQuery({
    queryKey: ['admin', 'storage-decision', selectedDecisionJobId],
    queryFn: () => apiAdminGetStorageDecision(selectedDecisionJobId!).then((res) => res.data),
    enabled: isDetailOpen && !!selectedDecisionJobId,
  })

  const replayMutation = useMutation({
    mutationFn: () =>
      apiAdminReplayStorageDecisions({
        limit: replayConfig.limit,
        max_expand_ratio: replayConfig.maxExpandRatio,
        max_expand_bytes: replayConfig.maxExpandBytesGiB * 1024 ** 3,
        min_platform_reserved_ratio: replayConfig.minPlatformReservedRatio,
        min_platform_reserved_bytes: replayConfig.minPlatformReservedBytesGiB * 1024 ** 3,
        expansion_cooldown_hours: replayConfig.expansionCooldownHours,
        force_freeze_when_over_quota: replayConfig.forceFreezeWhenOverQuota,
      }),
    onSuccess: (res) => {
      setReplaySummary(res.data)
      toast.success('回放评估已完成')
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '回放评估失败'))
    },
  })

  const autoShrinkMutation = useMutation({
    mutationFn: () => apiAdminRunAutoShrink(),
    onSuccess: (res) => {
      toast.success(res.data.message || '自动缩容扫描已完成')
      void decisionsQuery.refetch()
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '自动缩容扫描失败'))
    },
  })

  const latestRecords = useMemo(
    () => (decisionsQuery.data ?? []).slice(0, 20),
    [decisionsQuery.data]
  )
  const detail = detailQuery.data

  return (
    <>
      <Card>
        <CardHeader className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              决策记录与回放评估
            </CardTitle>
            <CardDescription>
              查看已持久化的 LLM 决策记录、约束调整结果，并对历史决策执行新的策略回放。
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => void decisionsQuery.refetch()}
              disabled={decisionsQuery.isFetching}
            >
              <RefreshCcw className="mr-2 h-4 w-4" />
              刷新
            </Button>
            <Button
              variant="outline"
              onClick={() => autoShrinkMutation.mutate()}
              disabled={autoShrinkMutation.isPending}
            >
              {autoShrinkMutation.isPending ? '缩容中...' : '执行自动缩容'}
            </Button>
            <Button onClick={() => setIsReplayOpen(true)}>回放评估</Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-3">
            <div className="space-y-2">
              <Label>用户</Label>
              <Input
                value={filters.user}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, user: event.target.value }))
                }
                placeholder="按用户名筛选"
              />
            </div>
            <div className="space-y-2">
              <Label>状态</Label>
              <Select
                value={filters.status || 'all'}
                onValueChange={(value) =>
                  setFilters((current) => ({ ...current, status: value === 'all' ? '' : value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="全部状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="pending">待处理</SelectItem>
                  <SelectItem value="running">执行中</SelectItem>
                  <SelectItem value="done">已完成</SelectItem>
                  <SelectItem value="error">失败</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>来源</Label>
              <Select
                value={filters.source || 'all'}
                onValueChange={(value) =>
                  setFilters((current) => ({ ...current, source: value === 'all' ? '' : value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="全部来源" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部来源</SelectItem>
                  <SelectItem value="manual">手动触发</SelectItem>
                  <SelectItem value="patrol">定时巡检</SelectItem>
                  <SelectItem value="replay">回放任务</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>创建时间</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>来源</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>最终动作</TableHead>
                  <TableHead>最终扩容量</TableHead>
                  <TableHead>约束结果</TableHead>
                  <TableHead>耗时</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {latestRecords.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={9} className="text-muted-foreground h-24 text-center">
                      {decisionsQuery.isLoading ? '正在加载决策记录...' : '暂无决策记录'}
                    </TableCell>
                  </TableRow>
                ) : (
                  latestRecords.map((record) => (
                    <TableRow key={record.job_id}>
                      <TableCell className="font-mono text-xs">{record.created_at}</TableCell>
                      <TableCell>{record.username}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className={decisionSourceTone(record.source)}>
                          {decisionSourceLabel(record.source)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={decisionStatusTone(record.status)}>
                          {decisionStatusLabel(record.status)}
                        </Badge>
                      </TableCell>
                      <TableCell>{appliedActionLabel(record.applied_action)}</TableCell>
                      <TableCell>
                        {record.final_allow_expand
                          ? formatBytes(record.final_expand_bytes)
                          : '不扩容'}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-2">
                          {record.constraint_adjusted && <Badge variant="outline">已调整</Badge>}
                          {record.constraint_blocked && <Badge variant="outline">已阻断</Badge>}
                          {!record.constraint_adjusted && !record.constraint_blocked && (
                            <Badge variant="outline">直接通过</Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{record.latency_ms} ms</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedDecisionJobId(record.job_id)
                            setIsDetailOpen(true)
                          }}
                        >
                          <Eye className="mr-1 h-4 w-4" />
                          详情
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Dialog open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>决策详情</DialogTitle>
          </DialogHeader>
          {detailQuery.isLoading ? (
            <div className="text-muted-foreground py-8 text-sm">正在加载决策详情...</div>
          ) : !detail ? (
            <div className="text-muted-foreground py-8 text-sm">暂无可用的决策详情。</div>
          ) : (
            <DecisionDetailContent detail={detail} />
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={isReplayOpen} onOpenChange={setIsReplayOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>回放评估</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>回放数量上限</Label>
              <Input
                type="number"
                min={1}
                value={replayConfig.limit}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    limit: Number(event.target.value) || 1,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>最大扩容比例</Label>
              <Input
                type="number"
                step="0.05"
                min={0}
                value={replayConfig.maxExpandRatio}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    maxExpandRatio: Number(event.target.value) || 0,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>最大扩容量（GiB）</Label>
              <Input
                type="number"
                min={1}
                value={replayConfig.maxExpandBytesGiB}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    maxExpandBytesGiB: Number(event.target.value) || 1,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>平台预留比例</Label>
              <Input
                type="number"
                step="0.01"
                min={0}
                value={replayConfig.minPlatformReservedRatio}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    minPlatformReservedRatio: Number(event.target.value) || 0,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>平台预留容量（GiB）</Label>
              <Input
                type="number"
                min={0}
                value={replayConfig.minPlatformReservedBytesGiB}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    minPlatformReservedBytesGiB: Number(event.target.value) || 0,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>冷却时间（小时）</Label>
              <Input
                type="number"
                min={0}
                value={replayConfig.expansionCooldownHours}
                onChange={(event) =>
                  setReplayConfig((current) => ({
                    ...current,
                    expansionCooldownHours: Number(event.target.value) || 0,
                  }))
                }
              />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-md border p-3">
            <div>
              <div className="font-medium">超额后强制冻结</div>
              <div className="text-muted-foreground text-sm">
                如果回放后的策略拒绝扩容，且用户已经超过理论配额，则强制给出冻结建议。
              </div>
            </div>
            <Button
              variant={replayConfig.forceFreezeWhenOverQuota ? 'default' : 'outline'}
              onClick={() =>
                setReplayConfig((current) => ({
                  ...current,
                  forceFreezeWhenOverQuota: !current.forceFreezeWhenOverQuota,
                }))
              }
            >
              {replayConfig.forceFreezeWhenOverQuota ? '已启用' : '已禁用'}
            </Button>
          </div>

          {replaySummary && <ReplaySummaryView summary={replaySummary} />}

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsReplayOpen(false)}>
              关闭
            </Button>
            <Button onClick={() => replayMutation.mutate()} disabled={replayMutation.isPending}>
              {replayMutation.isPending ? '回放中...' : '开始回放'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function DecisionDetailContent({ detail }: { detail: StorageDecisionRecordDetail }) {
  const snapshot = detail.snapshot
  const currentShrinkStage = detail.current_shrink_stage || snapshot?.shrink_stage

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="用户" value={detail.username} />
        <MetricCard label="来源" value={decisionSourceLabel(detail.source)} />
        <MetricCard label="状态" value={decisionStatusLabel(detail.status)} />
        <MetricCard label="最终动作" value={appliedActionLabel(detail.applied_action)} />
        <MetricCard label="缩容阶段" value={shrinkStageLabel(currentShrinkStage)} />
        <MetricCard label="耗时" value={`${detail.latency_ms} ms`} />
        <MetricCard
          label="原始扩容建议"
          value={detail.raw_allow_expand ? formatBytes(detail.raw_expand_bytes) : '否'}
        />
        <MetricCard
          label="最终扩容建议"
          value={detail.final_allow_expand ? formatBytes(detail.final_expand_bytes) : '否'}
        />
        <MetricCard label="约束是否调整" value={detail.constraint_adjusted ? '是' : '否'} />
        <MetricCard label="约束是否阻断" value={detail.constraint_blocked ? '是' : '否'} />
      </div>

      {snapshot && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">决策快照</CardTitle>
            <CardDescription>记录决策时的用户存储状态与平台上下文。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard label="当前占用" value={formatBytes(snapshot.current_usage_bytes)} />
            <MetricCard label="当前配额" value={formatBytes(snapshot.current_quota_bytes)} />
            <MetricCard label="理论配额" value={formatBytes(snapshot.theoretical_quota_bytes)} />
            <MetricCard label="使用率" value={formatRatio(snapshot.usage_ratio)} />
            <MetricCard
              label="增长速率"
              value={
                snapshot.growth_rate_bytes_per_hour
                  ? `${formatBytes(snapshot.growth_rate_bytes_per_hour)}/h`
                  : '未知'
              }
            />
            <MetricCard label="平台总量" value={formatBytes(snapshot.platform_total_bytes)} />
            <MetricCard label="平台已用" value={formatBytes(snapshot.platform_used_bytes)} />
            <MetricCard label="平台可用" value={formatBytes(snapshot.platform_available_bytes)} />
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">缩容阶段说明</CardTitle>
          <CardDescription>
            当前记录对应用户在自动缩容状态机中的阶段，以及系统为什么处于这个阶段。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <MetricCard label="当前缩容阶段" value={shrinkStageLabel(currentShrinkStage)} />
          <p className="text-muted-foreground text-sm leading-relaxed">
            {shrinkStageReason(currentShrinkStage)}
          </p>
        </CardContent>
      </Card>

      <div className="grid gap-4 xl:grid-cols-3">
        <JsonPanel title="原始决策" value={detail.raw_decision} />
        <JsonPanel title="最终决策" value={detail.final_decision} />
        <JsonPanel title="约束评估结果" value={detail.constraint_result} />
      </div>
    </div>
  )
}

function ReplaySummaryView({ summary }: { summary: ReplaySummary }) {
  const records = summary.records ?? []

  return (
    <div className="space-y-4">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <MetricCard label="策略版本" value={summary.policy_version} />
        <MetricCard label="样本数" value={String(summary.total_cases)} />
        <MetricCard label="发生变化" value={String(summary.changed_cases)} />
        <MetricCard label="被阻断" value={String(summary.blocked_cases)} />
        <MetricCard label="冻结升级" value={String(summary.freeze_escalations)} />
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户</TableHead>
              <TableHead>原始扩容</TableHead>
              <TableHead>回放扩容</TableHead>
              <TableHead>原始冻结</TableHead>
              <TableHead>回放冻结</TableHead>
              <TableHead>回放说明</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                  暂无回放结果。
                </TableCell>
              </TableRow>
            ) : (
              records.slice(0, 10).map((record) => (
                <TableRow key={record.job_id}>
                  <TableCell>{record.username}</TableCell>
                  <TableCell>
                    {record.stored_allow_expand ? formatBytes(record.stored_expand_bytes) : '否'}
                  </TableCell>
                  <TableCell>
                    {record.replay_allow_expand ? formatBytes(record.replay_expand_bytes) : '否'}
                  </TableCell>
                  <TableCell>{record.stored_freeze ? '是' : '否'}</TableCell>
                  <TableCell>{record.replay_freeze ? '是' : '否'}</TableCell>
                  <TableCell className="text-muted-foreground max-w-[320px] text-xs whitespace-pre-wrap">
                    {record.evaluation.adjustments.join('；') ||
                      record.evaluation.violations.join('；') ||
                      '无变化'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-muted/30 rounded-md border p-3">
      <div className="text-muted-foreground text-xs tracking-wide uppercase">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  )
}

function JsonPanel({ title, value }: { title: string; value: unknown }) {
  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">{title}</div>
      <Textarea className="min-h-[220px] font-mono text-xs" readOnly value={safeStringify(value)} />
    </div>
  )
}
