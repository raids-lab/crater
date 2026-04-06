'use client'

import { Check, Lightbulb } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

import { cn } from '@/lib/utils'

export interface ResourceRecommendation {
  gpu_model: string
  available: number
  queue_depth: number
  estimated_wait: string
  match_score: number
  reason: string
}

export interface ResourceSuggestionCardProps {
  suggestionId: string
  context: string
  recommendations: ResourceRecommendation[]
  tip: string
  onSelect?: (gpuModel: string) => void
}

export function ResourceSuggestionCard({
  suggestionId: _suggestionId,
  context: _context,
  recommendations,
  tip,
  onSelect,
}: ResourceSuggestionCardProps) {
  const [selected, setSelected] = useState<string | null>(null)

  const bestScore = Math.max(...recommendations.map((r) => r.match_score), 0)

  const handleSelect = (gpuModel: string) => {
    setSelected(gpuModel)
    onSelect?.(gpuModel)
  }

  return (
    <Card className="min-w-0 space-y-3 overflow-hidden border-blue-200/60 bg-blue-50/20 p-4 dark:border-blue-800/40 dark:bg-blue-950/20">
      {/* Tip */}
      {tip && (
        <div className="flex items-start gap-2">
          <Lightbulb className="mt-0.5 h-4 w-4 shrink-0 text-blue-500" />
          <p className="text-xs leading-relaxed text-blue-700 dark:text-blue-300">{tip}</p>
        </div>
      )}

      {/* Table header */}
      <div className="grid grid-cols-[1fr_auto_auto_auto_60px_auto] items-center gap-x-2 gap-y-1 text-[10px] font-medium text-muted-foreground">
        <span>GPU 型号</span>
        <span className="w-10 text-center">可用</span>
        <span className="w-10 text-center">队列</span>
        <span className="w-16 text-center">等待</span>
        <span className="text-center">匹配度</span>
        <span className="w-12" />
      </div>

      {/* Rows */}
      {recommendations.map((rec) => {
        const isBest = rec.match_score === bestScore
        return (
          <div
            key={rec.gpu_model}
            className={cn(
              'grid grid-cols-[1fr_auto_auto_auto_60px_auto] items-center gap-x-2 rounded-md px-2 py-1.5 text-xs',
              isBest && 'bg-blue-100/50 dark:bg-blue-900/30',
              selected === rec.gpu_model && 'ring-2 ring-blue-400'
            )}
          >
            <div className="flex items-center gap-1.5 min-w-0">
              <code className="truncate font-mono text-xs font-medium">{rec.gpu_model}</code>
              {isBest && (
                <Badge className="h-4 shrink-0 bg-blue-500 px-1 text-[9px] text-white">
                  最佳
                </Badge>
              )}
            </div>
            <span className="w-10 text-center tabular-nums">{rec.available}</span>
            <span className="w-10 text-center tabular-nums">{rec.queue_depth}</span>
            <span className="w-16 text-center text-muted-foreground">{rec.estimated_wait}</span>
            <div className="flex items-center gap-1">
              <Progress value={rec.match_score * 100} className="h-1.5 w-8" />
              <span className="w-7 text-right tabular-nums text-[10px]">
                {Math.round(rec.match_score * 100)}%
              </span>
            </div>
            <div className="w-12">
              {selected === rec.gpu_model ? (
                <Badge variant="outline" className="h-5 gap-0.5 px-1 text-[10px] text-green-600">
                  <Check className="h-3 w-3" />
                  已选
                </Badge>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  className="h-5 px-1.5 text-[10px]"
                  onClick={() => handleSelect(rec.gpu_model)}
                >
                  选择
                </Button>
              )}
            </div>
          </div>
        )
      })}

      {/* Reasons (collapsed under each best) */}
      {recommendations.length > 0 && recommendations.find((r) => r.match_score === bestScore)?.reason && (
        <p className="text-muted-foreground text-[11px] leading-relaxed">
          推荐理由：{recommendations.find((r) => r.match_score === bestScore)?.reason}
        </p>
      )}
    </Card>
  )
}
