'use client'

import { AlertTriangle, Bell, ChevronDown, Info, OctagonAlert, StopCircle } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import { cn } from '@/lib/utils'

export interface PipelineReportItem {
  job_name: string
  user: string
  gpu_util: string
  duration?: string
  gpu_requested?: number
  gpu_actual?: number
}

export interface PipelineReportCategory {
  action: string
  severity: 'critical' | 'warning' | 'info'
  count: number
  items: PipelineReportItem[]
}

export interface PipelineReportCardProps {
  reportId: string
  reportType: string
  completedAt: string
  summary: {
    total_scanned: number
    idle_detected: number
    gpu_waste_hours: number
  }
  summaryLabels?: {
    total_label?: string
    middle_label?: string
    right_label?: string
  }
  categories: PipelineReportCategory[]
  onBatchStop?: (jobNames: string[]) => void
  onNotifyOwners?: (jobNames: string[]) => void
  onDismiss?: () => void
}

const SEVERITY_CONFIG = {
  critical: {
    icon: OctagonAlert,
    color: 'text-red-600 dark:text-red-400',
    bg: 'bg-red-50 dark:bg-red-950/30',
    border: 'border-red-200 dark:border-red-800',
    badge: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-amber-600 dark:text-amber-400',
    bg: 'bg-amber-50 dark:bg-amber-950/30',
    border: 'border-amber-200 dark:border-amber-800',
    badge: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
  },
  info: {
    icon: Info,
    color: 'text-blue-600 dark:text-blue-400',
    bg: 'bg-blue-50 dark:bg-blue-950/30',
    border: 'border-blue-200 dark:border-blue-800',
    badge: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  },
}

export function PipelineReportCard({
  reportId: _reportId,
  reportType,
  completedAt,
  summary,
  summaryLabels,
  categories,
  onBatchStop,
  onNotifyOwners,
  onDismiss,
}: PipelineReportCardProps) {
  const [openCategories, setOpenCategories] = useState<Record<number, boolean>>({})

  const toggleCategory = (idx: number) => {
    setOpenCategories((prev) => ({ ...prev, [idx]: !prev[idx] }))
  }

  const allCriticalJobs = categories
    .filter((c) => c.severity === 'critical')
    .flatMap((c) => c.items.map((i) => i.job_name))

  const allJobNames = categories.flatMap((c) => c.items.map((i) => i.job_name))

  return (
    <Card className="border-border min-w-0 space-y-3 overflow-hidden p-4">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="space-y-0.5">
          <p className="text-sm font-semibold">{reportType}</p>
          <p className="text-muted-foreground text-[11px]">完成于 {completedAt}</p>
        </div>
        {onDismiss && (
          <Button variant="ghost" size="sm" className="h-6 px-2 text-[10px]" onClick={onDismiss}>
            关闭
          </Button>
        )}
      </div>

      {/* Summary metrics */}
      <div className="grid grid-cols-3 gap-2">
        <div className="rounded-md border px-3 py-2 text-center">
          <p className="text-lg font-bold tabular-nums">{summary.total_scanned}</p>
          <p className="text-muted-foreground text-[10px]">
            {summaryLabels?.total_label ?? '扫描任务'}
          </p>
        </div>
        <div className="rounded-md border border-amber-200 bg-amber-50/50 px-3 py-2 text-center dark:border-amber-800 dark:bg-amber-950/30">
          <p className="text-lg font-bold tabular-nums text-amber-600 dark:text-amber-400">
            {summary.idle_detected}
          </p>
          <p className="text-muted-foreground text-[10px]">
            {summaryLabels?.middle_label ?? '闲置检出'}
          </p>
        </div>
        <div className="rounded-md border border-red-200 bg-red-50/50 px-3 py-2 text-center dark:border-red-800 dark:bg-red-950/30">
          <p className="text-lg font-bold tabular-nums text-red-600 dark:text-red-400">
            {summary.gpu_waste_hours}h
          </p>
          <p className="text-muted-foreground text-[10px]">
            {summaryLabels?.right_label ?? 'GPU 浪费时'}
          </p>
        </div>
      </div>

      {/* Categories */}
      <div className="space-y-2">
        {categories.map((cat, idx) => {
          const cfg = SEVERITY_CONFIG[cat.severity]
          const SeverityIcon = cfg.icon
          const isOpen = !!openCategories[idx]

          return (
            <Collapsible key={idx} open={isOpen} onOpenChange={() => toggleCategory(idx)}>
              <CollapsibleTrigger asChild>
                <button
                  className={cn(
                    'flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors',
                    cfg.bg,
                    cfg.border
                  )}
                >
                  <SeverityIcon className={cn('h-3.5 w-3.5 shrink-0', cfg.color)} />
                  <span className="flex-1 font-medium">{cat.action}</span>
                  <span className={cn('rounded px-1.5 py-0.5 text-[10px] font-medium', cfg.badge)}>
                    {cat.count}
                  </span>
                  <ChevronDown
                    className={cn('h-3 w-3 transition-transform', isOpen && 'rotate-180')}
                  />
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="mt-1 space-y-1 pl-1">
                  {cat.items.map((item, iIdx) => (
                    <div
                      key={iIdx}
                      className="flex items-center gap-2 rounded px-2 py-1 text-[11px] hover:bg-muted/50"
                    >
                      <code className="min-w-0 flex-1 truncate font-mono">{item.job_name}</code>
                      <span className="text-muted-foreground shrink-0">{item.user}</span>
                      <Badge variant="outline" className="h-4 shrink-0 px-1 text-[9px]">
                        GPU {item.gpu_util}
                      </Badge>
                      {item.duration && (
                        <span className="text-muted-foreground shrink-0 text-[10px]">
                          {item.duration}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </CollapsibleContent>
            </Collapsible>
          )
        })}
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-2">
        {onBatchStop && allCriticalJobs.length > 0 && (
          <Button
            size="sm"
            variant="destructive"
            className="h-7 gap-1 text-xs"
            onClick={() => onBatchStop(allCriticalJobs)}
          >
            <StopCircle className="h-3.5 w-3.5" />
            批量停止严重项 ({allCriticalJobs.length})
          </Button>
        )}
        {onNotifyOwners && allJobNames.length > 0 && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 text-xs"
            onClick={() => onNotifyOwners(allJobNames)}
          >
            <Bell className="h-3.5 w-3.5" />
            通知负责人
          </Button>
        )}
      </div>
    </Card>
  )
}
