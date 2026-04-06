'use client'

import { Check, Lock } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

export interface ParameterReviewParameter {
  key: string
  label: string
  value: any
  source: 'recommended' | 'default' | 'user'
  editable: boolean
  type: 'text' | 'number' | 'select' | 'textarea'
  options?: Array<{ label: string; value: string }>
  constraints?: { min?: number; max?: number }
  hint?: string
}

export interface ParameterReviewCardProps {
  reviewId: string
  scenario: string
  complexity: 'simple' | 'complex'
  step: number
  totalSteps: number
  title: string
  description: string
  parameters: ParameterReviewParameter[]
  onConfirm: (reviewId: string, parameters: Record<string, any>) => void
  onModify?: (reviewId: string, parameters: Record<string, any>) => void
  settled?: 'confirmed' | null
}

const SOURCE_BADGE: Record<string, { label: string; className: string }> = {
  recommended: { label: '推荐', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300' },
  default: { label: '默认', className: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400' },
  user: { label: '用户', className: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' },
}

export function ParameterReviewCard({
  reviewId,
  scenario: _scenario,
  complexity: _complexity,
  step,
  totalSteps,
  title,
  description,
  parameters,
  onConfirm,
  onModify,
  settled,
}: ParameterReviewCardProps) {
  const [values, setValues] = useState<Record<string, any>>(() => {
    const init: Record<string, any> = {}
    for (const p of parameters) {
      init[p.key] = p.value
    }
    return init
  })

  const hasChanges = parameters.some((p) => {
    const current = values[p.key]
    return p.editable && String(current) !== String(p.value)
  })

  const collectValues = (): Record<string, any> => {
    const result: Record<string, any> = {}
    for (const p of parameters) {
      result[p.key] = values[p.key]
    }
    return result
  }

  const handleConfirm = () => {
    if (hasChanges && onModify) {
      onModify(reviewId, collectValues())
    } else {
      onConfirm(reviewId, collectValues())
    }
  }

  const updateValue = (key: string, val: any) => {
    setValues((prev) => ({ ...prev, [key]: val }))
  }

  const renderInput = (param: ParameterReviewParameter) => {
    const val = values[param.key] ?? ''

    if (!param.editable) {
      return (
        <div className="flex items-center gap-1.5 rounded-md bg-gray-100 px-3 py-1.5 text-sm text-gray-500 dark:bg-gray-800 dark:text-gray-400">
          <Lock className="h-3 w-3 shrink-0" />
          <span className="truncate">{String(val)}</span>
        </div>
      )
    }

    if (param.type === 'select' && param.options) {
      return (
        <Select
          value={String(val) || undefined}
          onValueChange={(v) => updateValue(param.key, v)}
        >
          <SelectTrigger className="h-8 w-full text-sm">
            <SelectValue placeholder={param.label} />
          </SelectTrigger>
          <SelectContent>
            {param.options.map((opt) => (
              <SelectItem key={`${param.key}-${opt.value}`} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    }

    if (param.type === 'textarea') {
      return (
        <Textarea
          value={String(val)}
          onChange={(e) => updateValue(param.key, e.target.value)}
          className="min-h-[60px] text-sm"
        />
      )
    }

    return (
      <Input
        type={param.type === 'number' ? 'number' : 'text'}
        value={String(val)}
        min={param.constraints?.min}
        max={param.constraints?.max}
        onChange={(e) =>
          updateValue(
            param.key,
            param.type === 'number' ? Number(e.target.value) : e.target.value
          )
        }
        className="h-8 text-sm"
      />
    )
  }

  if (settled === 'confirmed') {
    return (
      <Card className="border-green-300/60 bg-green-50/30 min-w-0 p-4 dark:border-green-800/40 dark:bg-green-950/20">
        <p className="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400">
          <Check className="h-3.5 w-3.5" />
          配置已确认
        </p>
      </Card>
    )
  }

  return (
    <Card className="border-border min-w-0 space-y-3 overflow-hidden p-4">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1 space-y-1">
          <p className="text-sm font-semibold leading-snug">{title}</p>
          {description && (
            <p className="text-muted-foreground text-xs">{description}</p>
          )}
        </div>
        {totalSteps > 1 && (
          <Badge variant="outline" className="shrink-0 text-[10px]">
            步骤 {step}/{totalSteps}
          </Badge>
        )}
      </div>

      {/* Parameters */}
      <div className="space-y-2.5">
        {parameters.map((param) => {
          const badge = SOURCE_BADGE[param.source]
          return (
            <div key={param.key} className="space-y-1">
              <div className="flex items-center gap-2">
                <Label className="text-xs font-medium">{param.label}</Label>
                {badge && (
                  <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium ${badge.className}`}>
                    {badge.label}
                  </span>
                )}
              </div>
              {renderInput(param)}
              {param.hint && (
                <p className="text-muted-foreground text-[11px]">{param.hint}</p>
              )}
            </div>
          )
        })}
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          size="sm"
          className="h-7 flex-1 bg-green-600 text-xs text-white hover:bg-green-700"
          onClick={handleConfirm}
        >
          <Check className="mr-1 h-3.5 w-3.5" />
          {hasChanges ? '修改后确认' : '确认'}
        </Button>
      </div>
    </Card>
  )
}
