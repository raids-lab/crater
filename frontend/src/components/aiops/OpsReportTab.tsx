/**
 * Ops Report Tab — displays the latest intelligent patrol report + history.
 * Shown as a tab in the admin 智能运维 page.
 */
import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  CheckCircle,
  Clock,
  Cpu,
  ExternalLink,
  Info,
  Lightbulb,
  RefreshCw,
} from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  apiGetLatestOpsReport,
  apiGetOpsReportDetail,
  apiListOpsReports,
} from '@/services/api/ops-report'
import type {
  OpsReportDetail,
  OpsReportJSON,
  OpsReportListItem,
} from '@/services/api/ops-report'

// ─── Main Component ─────────────────────────────────────────────────────────

export function OpsReportTab() {
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null)
  const [historyPage, setHistoryPage] = useState(1)

  const {
    data: latestData,
    isLoading: latestLoading,
  } = useQuery({
    queryKey: ['ops-report', 'latest'],
    queryFn: async () => {
      const res = await apiGetLatestOpsReport()
      return res.data as OpsReportDetail
    },
    refetchInterval: 60000,
  })

  const { data: selectedReport } = useQuery({
    queryKey: ['ops-report', 'detail', selectedReportId],
    queryFn: async () => {
      if (!selectedReportId) return null
      const res = await apiGetOpsReportDetail(selectedReportId)
      return res.data as OpsReportDetail
    },
    enabled: !!selectedReportId,
  })

  const { data: historyData } = useQuery({
    queryKey: ['ops-report', 'history', historyPage],
    queryFn: async () => {
      const res = await apiListOpsReports(historyPage, 10)
      return res.data
    },
    refetchInterval: 60000,
  })

  const activeReport = selectedReportId && selectedReport ? selectedReport : latestData
  const reportJSON = activeReport?.report_json as OpsReportJSON | null

  if (latestLoading) {
    return (
      <Card>
        <CardContent className="text-muted-foreground py-12 text-center">
          <RefreshCw className="mx-auto mb-2 h-5 w-5 animate-spin" />
          加载巡检报告中...
        </CardContent>
      </Card>
    )
  }

  if (!activeReport || !activeReport.id) {
    return (
      <Card>
        <CardContent className="text-muted-foreground py-12 text-center">
          <Info className="mx-auto mb-2 h-6 w-6" />
          <p>巡检报告将在定时任务执行后生成</p>
          <p className="mt-1 text-sm">
            前往{' '}
            <a href="/admin/cronjobs" className="text-primary underline">
              定时任务配置
            </a>{' '}
            查看巡检任务状态
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      <ReportSummaryCard report={activeReport} reportJSON={reportJSON} />

      {reportJSON?.failure_analysis && (
        <FailureAnalysisCard analysis={reportJSON.failure_analysis} />
      )}

      {reportJSON?.success_analysis && (
        <SuccessAnalysisCard analysis={reportJSON.success_analysis} />
      )}

      {reportJSON?.resource_utilization && (
        <ResourceUtilizationCard utilization={reportJSON.resource_utilization} />
      )}

      {reportJSON?.recommendations && reportJSON.recommendations.length > 0 && (
        <RecommendationsCard recommendations={reportJSON.recommendations} />
      )}

      <ReportHistoryTable
        items={historyData?.items || []}
        total={historyData?.total || 0}
        page={historyPage}
        onPageChange={setHistoryPage}
        selectedId={selectedReportId}
        onSelect={(id) => setSelectedReportId(id === selectedReportId ? null : id)}
      />
    </div>
  )
}

// ─── Sub-components ─────────────────────────────────────────────────────────

function ReportSummaryCard({
  report,
  reportJSON,
}: {
  report: OpsReportDetail
  reportJSON: OpsReportJSON | null
}) {
  const overview = reportJSON?.job_overview
  const delta = overview?.delta
  const periodTime = report.period_end
    ? new Date(report.period_end).toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
      })
    : new Date(report.created_at).toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
      })

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{periodTime} 巡检报告</CardTitle>
          <Badge variant="outline" className="text-xs">
            {report.status === 'completed' ? '已完成' : report.status}
          </Badge>
        </div>
        {reportJSON?.executive_summary && (
          <p className="text-muted-foreground mt-1 text-sm">{reportJSON.executive_summary}</p>
        )}
      </CardHeader>
      <CardContent>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard label="总作业" value={overview?.total ?? report.job_total} icon={Activity} delta={delta?.total} />
          <StatCard label="成功" value={overview?.success ?? report.job_success} icon={CheckCircle} variant="success"
            suffix={overview ? `${overview.success_rate.toFixed(1)}%` : undefined} />
          <StatCard label="失败" value={overview?.failed ?? report.job_failed} icon={AlertCircle} variant="destructive" delta={delta?.failed} />
          <StatCard label="等待中" value={overview?.pending ?? report.job_pending} icon={Clock} variant="warning" delta={delta?.pending} />
        </div>
      </CardContent>
    </Card>
  )
}

function StatCard({ label, value, icon: Icon, variant = 'default', delta, suffix }: {
  label: string; value: number; icon: React.ElementType;
  variant?: 'default' | 'success' | 'destructive' | 'warning'; delta?: number; suffix?: string
}) {
  const variantBg = {
    default: 'bg-muted/50', success: 'bg-green-50 dark:bg-green-950/20',
    destructive: 'bg-red-50 dark:bg-red-950/20', warning: 'bg-yellow-50 dark:bg-yellow-950/20',
  }
  return (
    <div className={`rounded-lg p-3 ${variantBg[variant]}`}>
      <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
        <Icon className="h-3.5 w-3.5" />{label}
      </div>
      <div className="mt-1 flex items-baseline gap-2">
        <span className="text-2xl font-bold">{value}</span>
        {suffix && <span className="text-muted-foreground text-sm">{suffix}</span>}
        {delta !== undefined && delta !== 0 && <DeltaBadge delta={delta} />}
      </div>
    </div>
  )
}

function DeltaBadge({ delta }: { delta: number }) {
  if (delta > 0) return <span className="inline-flex items-center gap-0.5 text-xs text-red-600"><ArrowUp className="h-3 w-3" />+{delta}</span>
  if (delta < 0) return <span className="inline-flex items-center gap-0.5 text-xs text-green-600"><ArrowDown className="h-3 w-3" />{delta}</span>
  return null
}

function FailureAnalysisCard({ analysis }: { analysis: OpsReportJSON['failure_analysis'] }) {
  const navigate = useNavigate()
  if (!analysis.categories || analysis.categories.length === 0) return null

  const navigateToFailedJobs = () => {
    localStorage.setItem(
      'admin_job_overview-column-filters',
      JSON.stringify([{ id: 'status', value: ['Failed'] }])
    )
    navigate({ to: '/admin/jobs' })
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <AlertCircle className="h-4 w-4 text-red-500" />失败分析
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>失败类型</TableHead>
              <TableHead className="w-20 text-center">数量</TableHead>
              <TableHead>代表作业</TableHead>
              <TableHead className="w-20 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {analysis.categories.map((cat) => (
              <TableRow key={cat.reason}>
                <TableCell><Badge variant="outline">{cat.reason}</Badge></TableCell>
                <TableCell className="text-center font-medium">{cat.count}</TableCell>
                <TableCell className="text-muted-foreground text-sm">{cat.top_job?.name || '-'}</TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={navigateToFailedJobs}>
                    查看<ExternalLink className="ml-1 h-3 w-3" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {analysis.top_affected_users && analysis.top_affected_users.length > 0 && (
          <p className="text-muted-foreground mt-3 text-sm">
            受影响用户: {analysis.top_affected_users.join('、')}
          </p>
        )}
        {analysis.patterns && <p className="text-muted-foreground mt-3 text-sm">{analysis.patterns}</p>}
      </CardContent>
    </Card>
  )
}

function SuccessAnalysisCard({ analysis }: { analysis: OpsReportJSON['success_analysis'] }) {
  const durations = Object.entries(analysis.avg_duration_by_type || {})
  const efficiency = analysis.resource_efficiency
  const hasEfficiency =
    efficiency.avg_cpu_ratio > 0 || efficiency.avg_gpu_ratio > 0 || efficiency.avg_memory_ratio > 0

  if (!durations.length && !hasEfficiency) return null

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <CheckCircle className="h-4 w-4 text-green-500" />成功作业画像
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasEfficiency && (
          <div className="grid gap-3 sm:grid-cols-3">
            <StatCard
              label="平均 CPU 利用率"
              value={Number((efficiency.avg_cpu_ratio * 100).toFixed(1))}
              icon={Cpu}
              suffix="%"
            />
            <StatCard
              label="平均 GPU 利用率"
              value={Number((efficiency.avg_gpu_ratio * 100).toFixed(1))}
              icon={Cpu}
              suffix="%"
            />
            <StatCard
              label="平均内存利用率"
              value={Number((efficiency.avg_memory_ratio * 100).toFixed(1))}
              icon={Cpu}
              suffix="%"
            />
          </div>
        )}
        {durations.length > 0 && (
          <div className="space-y-2">
            <div className="text-muted-foreground text-xs">按作业类型统计的平均运行时长</div>
            <div className="grid gap-2 sm:grid-cols-2">
              {durations.slice(0, 6).map(([jobType, seconds]) => (
                <div key={jobType} className="bg-muted/40 rounded-lg px-3 py-2 text-sm">
                  <div className="font-medium">{jobType}</div>
                  <div className="text-muted-foreground mt-1">
                    平均 {Math.round(Number(seconds) / 60)} 分钟
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ResourceUtilizationCard({ utilization }: { utilization: OpsReportJSON['resource_utilization'] }) {
  const navigate = useNavigate()

  const navigateToJobs = () => {
    localStorage.removeItem('admin_job_overview-column-filters')
    navigate({ to: '/admin/jobs' })
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Cpu className="h-4 w-4 text-blue-500" />资源利用率
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 sm:grid-cols-3">
          <UtilBar label="GPU 平均" value={utilization.cluster_gpu_avg} />
          <UtilBar label="CPU 平均" value={utilization.cluster_cpu_avg} />
          <UtilBar label="内存 平均" value={utilization.cluster_memory_avg} />
        </div>
        <div className="text-muted-foreground mt-3 space-y-1 text-sm">
          {utilization.over_provisioned_count > 0 && (
            <p>
              <AlertTriangle className="mr-1 inline h-3.5 w-3.5 text-yellow-500" />
              <button
                type="button"
                className="cursor-pointer underline-offset-2 hover:underline"
                onClick={navigateToJobs}
              >
                {utilization.over_provisioned_count} 个作业资源过度申请
              </button>
            </p>
          )}
          {utilization.idle_gpu_jobs > 0 && (
            <p>
              <Clock className="mr-1 inline h-3.5 w-3.5 text-yellow-500" />
              <button
                type="button"
                className="cursor-pointer underline-offset-2 hover:underline"
                onClick={navigateToJobs}
              >
                {utilization.idle_gpu_jobs} 个 GPU 空闲作业
              </button>
            </p>
          )}
          {utilization.node_hotspots.length > 0 && (
            <p><AlertCircle className="mr-1 inline h-3.5 w-3.5 text-red-500" />热点节点: {utilization.node_hotspots.join(', ')}</p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function UtilBar({ label, value }: { label: string; value: number }) {
  const pct = Math.min(100, Math.max(0, value))
  const color = pct > 90 ? 'bg-red-500' : pct > 70 ? 'bg-yellow-500' : 'bg-green-500'
  return (
    <div>
      <div className="mb-1 flex justify-between text-sm"><span>{label}</span><span className="font-medium">{pct.toFixed(1)}%</span></div>
      <div className="bg-muted h-2 rounded-full"><div className={`h-2 rounded-full ${color}`} style={{ width: `${pct}%` }} /></div>
    </div>
  )
}

function RecommendationsCard({ recommendations }: { recommendations: OpsReportJSON['recommendations'] }) {
  const cfg = {
    high: { color: 'text-red-600 bg-red-50 dark:bg-red-950/30', icon: AlertCircle },
    medium: { color: 'text-yellow-600 bg-yellow-50 dark:bg-yellow-950/30', icon: AlertTriangle },
    low: { color: 'text-blue-600 bg-blue-50 dark:bg-blue-950/30', icon: Info },
  }
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Lightbulb className="h-4 w-4 text-yellow-500" />运维建议
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {recommendations.map((rec, idx) => {
          const c = cfg[rec.severity] || cfg.low
          const Icon = c.icon
          return (
            <div key={idx} className={`flex items-start gap-2 rounded-lg p-3 ${c.color}`}>
              <Icon className="mt-0.5 h-4 w-4 flex-shrink-0" /><span className="text-sm">{rec.text}</span>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}

function ReportHistoryTable({ items, total, page, onPageChange, selectedId, onSelect }: {
  items: OpsReportListItem[]; total: number; page: number;
  onPageChange: (p: number) => void; selectedId: string | null; onSelect: (id: string) => void
}) {
  const totalPages = Math.ceil(total / 10)
  return (
    <Card>
      <CardHeader className="pb-2"><CardTitle className="text-base">历史报告</CardTitle></CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-sm">暂无历史报告</p>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead className="text-center">总作业</TableHead>
                  <TableHead className="text-center">失败</TableHead>
                  <TableHead className="text-center">失败率</TableHead>
                  <TableHead className="w-20 text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => {
                  const rate = item.job_total > 0 ? ((item.job_failed / item.job_total) * 100).toFixed(1) : '0.0'
                  const time = new Date(item.created_at).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
                  const isSelected = item.id === selectedId
                  return (
                    <TableRow key={item.id} className={isSelected ? 'bg-primary/5' : undefined}>
                      <TableCell className="text-sm">{time}</TableCell>
                      <TableCell className="text-center">{item.job_total}</TableCell>
                      <TableCell className="text-center">
                        {item.job_failed > 0 ? <span className="font-medium text-red-600">{item.job_failed}</span> : item.job_failed}
                      </TableCell>
                      <TableCell className="text-center">{rate}%</TableCell>
                      <TableCell className="text-right">
                        <Button variant={isSelected ? 'default' : 'ghost'} size="sm" className="h-7 text-xs"
                          onClick={() => onSelect(item.id)}>
                          {isSelected ? '当前' : '查看'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            {totalPages > 1 && (
              <div className="mt-3 flex justify-center gap-2">
                <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>上一页</Button>
                <span className="text-muted-foreground flex items-center text-sm">{page} / {totalPages}</span>
                <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>下一页</Button>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
