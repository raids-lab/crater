import { Badge } from '@/components/ui/badge'

import { formatBillingPoints } from '@/utils/billing'

import { cn } from '@/lib/utils'

interface BillingPointsBadgeProps {
  value?: number
  className?: string
  showLabel?: boolean
}

export function BillingPointsBadge({
  value = 0,
  className,
  showLabel = false,
}: BillingPointsBadgeProps) {
  return (
    <Badge variant="secondary" className={cn('font-mono font-normal', className)}>
      {showLabel ? `累计 ${formatBillingPoints(value)}` : formatBillingPoints(value)}
    </Badge>
  )
}
