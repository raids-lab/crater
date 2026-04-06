'use client'

import { AlertTriangle, Check, X } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'

import { cn } from '@/lib/utils'

export interface BatchConfirmItem {
  job_name: string
  user: string
  gpu_util: string
  selected: boolean
}

export interface BatchConfirmCardProps {
  batchId: string
  action: string
  description: string
  items: BatchConfirmItem[]
  onConfirmSelected: (batchId: string, selectedJobNames: string[]) => void
  onRejectAll: (batchId: string) => void
  settled?: 'confirmed' | 'rejected' | null
}

export function BatchConfirmCard({
  batchId,
  action,
  description,
  items,
  onConfirmSelected,
  onRejectAll,
  settled,
}: BatchConfirmCardProps) {
  const [selections, setSelections] = useState<Record<string, boolean>>(() => {
    const init: Record<string, boolean> = {}
    for (const item of items) {
      init[item.job_name] = item.selected
    }
    return init
  })

  const selectedCount = Object.values(selections).filter(Boolean).length
  const allSelected = selectedCount === items.length
  const noneSelected = selectedCount === 0

  const toggleAll = () => {
    const nextVal = !allSelected
    const next: Record<string, boolean> = {}
    for (const item of items) {
      next[item.job_name] = nextVal
    }
    setSelections(next)
  }

  const toggleItem = (jobName: string) => {
    setSelections((prev) => ({ ...prev, [jobName]: !prev[jobName] }))
  }

  const handleConfirm = () => {
    const selected = items.filter((i) => selections[i.job_name]).map((i) => i.job_name)
    onConfirmSelected(batchId, selected)
  }

  if (settled === 'confirmed') {
    return (
      <Card className="border-green-300/60 bg-green-50/30 min-w-0 p-4 dark:border-green-800/40 dark:bg-green-950/20">
        <p className="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400">
          <Check className="h-3.5 w-3.5" />
          已确认执行选中的 {selectedCount} 项操作
        </p>
      </Card>
    )
  }

  if (settled === 'rejected') {
    return (
      <Card className="border-red-300/60 bg-red-50/30 min-w-0 p-4 dark:border-red-800/40 dark:bg-red-950/20">
        <p className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400">
          <X className="h-3.5 w-3.5" />
          已拒绝全部操作
        </p>
      </Card>
    )
  }

  return (
    <Card className="border-warning/40 bg-warning/5 min-w-0 space-y-3 overflow-hidden p-4">
      {/* Header */}
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
        <div className="min-w-0 flex-1 space-y-1">
          <p className="text-sm font-semibold leading-snug">{action}</p>
          <p className="text-muted-foreground text-xs [overflow-wrap:anywhere] break-words">
            {description}
          </p>
        </div>
      </div>

      {/* Select all */}
      <div className="flex items-center gap-2 border-b pb-2">
        <Checkbox
          checked={allSelected}
          onCheckedChange={toggleAll}
          id={`batch-${batchId}-all`}
        />
        <label
          htmlFor={`batch-${batchId}-all`}
          className="cursor-pointer text-xs font-medium"
        >
          {allSelected ? '取消全选' : '全选'} ({items.length})
        </label>
      </div>

      {/* Item list */}
      <div className="max-h-48 space-y-1 overflow-y-auto">
        {items.map((item) => (
          <div
            key={item.job_name}
            className={cn(
              'flex items-center gap-2 rounded px-2 py-1.5 text-xs transition-colors',
              selections[item.job_name] && 'bg-amber-50/50 dark:bg-amber-950/20'
            )}
          >
            <Checkbox
              checked={!!selections[item.job_name]}
              onCheckedChange={() => toggleItem(item.job_name)}
              id={`batch-${batchId}-${item.job_name}`}
            />
            <code className="min-w-0 flex-1 truncate font-mono text-xs">{item.job_name}</code>
            <span className="text-muted-foreground shrink-0">{item.user}</span>
            <Badge variant="outline" className="h-4 shrink-0 px-1 text-[9px]">
              GPU {item.gpu_util}
            </Badge>
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          size="sm"
          className="h-7 flex-1 bg-green-600 text-xs text-white hover:bg-green-700"
          onClick={handleConfirm}
          disabled={noneSelected}
        >
          <Check className="mr-1 h-3.5 w-3.5" />
          确认选中 ({selectedCount})
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="text-destructive hover:text-destructive h-7 flex-1 border-red-400 text-xs hover:bg-red-50 dark:hover:bg-red-950"
          onClick={() => onRejectAll(batchId)}
        >
          <X className="mr-1 h-3.5 w-3.5" />
          全部拒绝
        </Button>
      </div>
    </Card>
  )
}
