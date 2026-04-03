'use client'

import { CheckCircle, ChevronDown, Loader2, Terminal, XCircle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import { cn } from '@/lib/utils'

export interface ToolCallCardProps {
  toolName: string
  args: Record<string, unknown>
  status: 'executing' | 'awaiting_confirmation' | 'done' | 'error' | 'cancelled'
  resultSummary?: string
}

export function ToolCallCard({ toolName, args, status, resultSummary }: ToolCallCardProps) {
  const { t } = useTranslation()
  const [argsOpen, setArgsOpen] = useState(false)
  const [resultOpen, setResultOpen] = useState(false)

  const hasArgs = Object.keys(args).length > 0
  const hasResult = !!resultSummary

  return (
    <Card
      className={cn(
        'min-w-0 overflow-hidden border px-3 py-2 text-xs',
        status === 'error' && 'border-destructive/40 bg-destructive/5',
        status === 'cancelled' &&
          'border-slate-300 bg-slate-50/80 dark:border-slate-700 dark:bg-slate-900/40',
        status === 'done' && 'border-border bg-muted/30',
        status === 'awaiting_confirmation' && 'border-amber-400/40 bg-amber-50/50',
        status === 'executing' && 'border-primary/30 bg-primary/5'
      )}
    >
      {/* Header row */}
      <div className="flex min-w-0 items-center gap-2">
        {/* Status icon */}
        {status === 'executing' && (
          <Loader2 className="text-primary h-3.5 w-3.5 shrink-0 animate-spin" />
        )}
        {status === 'awaiting_confirmation' && (
          <Loader2 className="h-3.5 w-3.5 shrink-0 text-amber-500" />
        )}
        {status === 'done' && <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-500" />}
        {status === 'error' && <XCircle className="text-destructive h-3.5 w-3.5 shrink-0" />}
        {status === 'cancelled' && <XCircle className="h-3.5 w-3.5 shrink-0 text-slate-500" />}

        {/* Tool icon + name */}
        <Terminal className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
        <code className="text-foreground min-w-0 flex-1 overflow-hidden font-mono text-xs font-medium text-ellipsis whitespace-nowrap">
          {toolName}
        </code>

        {/* Status badge */}
        <Badge
          variant={status === 'error' ? 'destructive' : status === 'done' ? 'secondary' : 'outline'}
          className="h-4 shrink-0 px-1.5 text-[10px]"
        >
          {status === 'executing' &&
            t('aiops.agent.toolCall.executing', { defaultValue: '执行中' })}
          {status === 'awaiting_confirmation' &&
            t('aiops.agent.toolCall.awaitingConfirmation', {
              defaultValue: '等待确认',
            })}
          {status === 'done' && t('aiops.agent.toolCall.done', { defaultValue: '完成' })}
          {status === 'error' && t('aiops.agent.toolCall.error', { defaultValue: '错误' })}
          {status === 'cancelled' &&
            t('aiops.agent.toolCall.cancelled', { defaultValue: '已取消' })}
        </Badge>
      </div>

      {/* Args collapsible */}
      {hasArgs && (
        <Collapsible open={argsOpen} onOpenChange={setArgsOpen} className="mt-1.5">
          <CollapsibleTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground h-5 w-full justify-start gap-1 px-0 text-[10px]"
            >
              <ChevronDown
                className={cn('h-3 w-3 transition-transform', argsOpen && 'rotate-180')}
              />
              {t('aiops.agent.toolCall.args', { defaultValue: '参数' })}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <pre className="bg-background mt-1 max-h-32 max-w-full min-w-0 overflow-x-auto overflow-y-auto rounded p-2 font-mono text-[10px] [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
              {JSON.stringify(args, null, 2)}
            </pre>
          </CollapsibleContent>
        </Collapsible>
      )}

      {/* Result collapsible */}
      {hasResult && (
        <Collapsible open={resultOpen} onOpenChange={setResultOpen} className="mt-1">
          <CollapsibleTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground h-5 w-full justify-start gap-1 px-0 text-[10px]"
            >
              <ChevronDown
                className={cn('h-3 w-3 transition-transform', resultOpen && 'rotate-180')}
              />
              {t('aiops.agent.toolCall.result', { defaultValue: '结果' })}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="bg-background mt-1 max-h-40 max-w-full min-w-0 overflow-x-auto overflow-y-auto rounded p-2 font-mono text-[10px] [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
              {resultSummary}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </Card>
  )
}
