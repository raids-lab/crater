import { LucideIcon } from 'lucide-react'
import { ReactNode } from 'react'

import { Card, CardAction, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

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
      {items.map((item) => (
        <Card key={item.title} className="@container/card">
          <CardHeader>
            <CardDescription className="flex flex-row items-center gap-1">
              {item.icon && <item.icon className="size-4" />}
              {item.title}
            </CardDescription>
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
      ))}
    </div>
  )
}
