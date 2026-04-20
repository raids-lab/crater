import { ReactNode } from 'react'

import { ProgressBar, ProgressMode, progressTextColor } from '@/components/ui-custom/colorful-progress'

import { cn } from '@/lib/utils'

interface MetricMeterProps {
  percent: number
  primary: ReactNode
  secondary?: ReactNode
  leading?: ReactNode
  trailing?: ReactNode
  layout?: 'stacked' | 'inline'
  mode?: ProgressMode
  className?: string
  barClassName?: string
}

export function MetricMeter({
  percent,
  primary,
  secondary,
  leading,
  trailing,
  layout = 'stacked',
  mode = 'usage',
  className,
  barClassName,
}: MetricMeterProps) {
  const normalizedPercent = Math.min(100, Math.max(0, percent))

  if (layout === 'inline') {
    return (
      <div className={cn('inline-flex min-w-0 items-center gap-2', className)}>
        {leading}
        <span className="font-mono text-xs font-semibold">{primary}</span>
        <div className="min-w-0 flex-1">
          <ProgressBar
            percent={normalizedPercent}
            mode={mode}
            className={cn('h-1 w-full min-w-12', barClassName)}
          />
        </div>
        {trailing}
      </div>
    )
  }

  return (
    <div className={cn('w-20', className)}>
      <div className="flex items-end justify-between gap-1">
        <p className={progressTextColor(normalizedPercent, mode)}>{primary}</p>
        {trailing ? <div className="text-muted-foreground text-[10px]">{trailing}</div> : null}
      </div>
      <ProgressBar percent={normalizedPercent} mode={mode} className={cn('h-1 w-full', barClassName)} />
      {secondary ? <p className="text-muted-foreground pt-1 font-mono text-xs">{secondary}</p> : null}
    </div>
  )
}
