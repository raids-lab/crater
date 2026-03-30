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
  args: Record<string, any>
  status: 'executing' | 'done' | 'error'
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
        'border px-3 py-2 text-xs',
        status === 'error' && 'border-destructive/40 bg-destructive/5',
        status === 'done' && 'border-border bg-muted/30',
        status === 'executing' && 'border-primary/30 bg-primary/5',
      )}
    >
      {/* Header row */}
      <div className="flex items-center gap-2">
        {/* Status icon */}
        {status === 'executing' && (
          <Loader2 className="text-primary h-3.5 w-3.5 shrink-0 animate-spin" />
        )}
        {status === 'done' && (
          <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-500" />
        )}
        {status === 'error' && (
          <XCircle className="text-destructive h-3.5 w-3.5 shrink-0" />
        )}

        {/* Tool icon + name */}
        <Terminal className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
        <code className="text-foreground flex-1 font-mono text-xs font-medium">{toolName}</code>

        {/* Status badge */}
        <Badge
          variant={status === 'error' ? 'destructive' : status === 'done' ? 'secondary' : 'outline'}
          className="h-4 shrink-0 px-1.5 text-[10px]"
        >
          {status === 'executing' && t('aiops.agent.toolCall.executing', { defaultValue: '执行中' })}
          {status === 'done' && t('aiops.agent.toolCall.done', { defaultValue: '完成' })}
          {status === 'error' && t('aiops.agent.toolCall.error', { defaultValue: '错误' })}
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
            <pre className="bg-background mt-1 max-h-32 overflow-auto rounded p-2 font-mono text-[10px] whitespace-pre-wrap">
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
            <div className="bg-background mt-1 max-h-40 overflow-auto rounded p-2 font-mono text-[10px] whitespace-pre-wrap">
              {resultSummary}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </Card>
  )
}
