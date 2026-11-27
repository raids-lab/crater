import { LucideIcon } from 'lucide-react'
import { ReactNode } from 'react'

import { Card, CardAction, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

interface SectionCardsProps {
  items: {
    title: string
    description?: string
    value: ReactNode
    className?: string
    icon?: LucideIcon
  }[]
  className?: string
}

export function SectionCards({ items, className }: SectionCardsProps) {
  return (
    <div
      className={cn(
        'dark:*:data-[slot=card]:bg-card grid grid-cols-1 gap-4 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:shadow-xs @xl/main:grid-cols-2 @5xl/main:grid-cols-4',
        className
      )}
    >
      {items.map((item) => {
        const label = (
          <CardDescription
            className={cn('flex items-center gap-1', item.description && 'cursor-help')}
            tabIndex={item.description ? 0 : undefined}
          >
            {item.icon && <item.icon className="size-4 shrink-0" />}
            {item.title}
          </CardDescription>
        )
        return (
          <Card key={item.title} className="@container/card">
            <CardHeader>
              {item.description ? (
                <Tooltip>
                  <TooltipTrigger asChild>{label}</TooltipTrigger>
                  <TooltipContent className="max-w-72">{item.description}</TooltipContent>
                </Tooltip>
              ) : (
                label
              )}
              <CardAction>
                <CardTitle
                  className={cn(
                    'text-2xl font-semibold tabular-nums @[250px]/card:text-3xl',
                    item.className
                  )}
                >
                  {item.value}
                </CardTitle>
              </CardAction>
            </CardHeader>
          </Card>
        )
      })}
    </div>
  )
}
