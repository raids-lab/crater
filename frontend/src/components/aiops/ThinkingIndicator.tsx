'use client'

import { Brain, ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import { cn } from '@/lib/utils'

export interface ThinkingIndicatorProps {
  /** Optional thinking text streamed from the agent (may be partial). */
  content?: string
}

export function ThinkingIndicator({ content }: ThinkingIndicatorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <div className="flex min-w-0 items-start gap-2">
      {/* Pulsing brain icon */}
      <span className="mt-0.5 shrink-0">
        <Brain className="text-primary h-4 w-4 animate-pulse" />
      </span>

      <div className="min-w-0 flex-1">
        {content ? (
          <Collapsible open={open} onOpenChange={setOpen}>
            <CollapsibleTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground h-auto min-w-0 justify-start gap-1 px-0 py-0 text-xs hover:bg-transparent"
              >
                {/* Three bouncing dots */}
                <ThinkingDots />
                <span className="ml-1 min-w-0 truncate">
                  {t('aiops.agent.thinking.label', { defaultValue: 'Agent 思考中...' })}
                </span>
                <ChevronDown className={cn('h-3 w-3 transition-transform', open && 'rotate-180')} />
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="text-muted-foreground bg-muted/40 mt-1 max-h-36 max-w-full min-w-0 overflow-x-auto overflow-y-auto rounded-md p-2 text-[10px] leading-relaxed [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
                {content}
              </div>
            </CollapsibleContent>
          </Collapsible>
        ) : (
          <div className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
            <ThinkingDots />
            <span className="min-w-0 truncate">
              {t('aiops.agent.thinking.label', { defaultValue: 'Agent 思考中...' })}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}

/** Three animated bouncing dots */
function ThinkingDots() {
  return (
    <span className="flex items-center gap-0.5" aria-hidden>
      <span className="bg-primary/60 h-1.5 w-1.5 animate-bounce rounded-full [animation-delay:0ms]" />
      <span className="bg-primary/60 h-1.5 w-1.5 animate-bounce rounded-full [animation-delay:150ms]" />
      <span className="bg-primary/60 h-1.5 w-1.5 animate-bounce rounded-full [animation-delay:300ms]" />
    </span>
  )
}
