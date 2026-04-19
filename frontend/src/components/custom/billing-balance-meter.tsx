import { CSSProperties, ReactNode } from 'react'

import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'

import { ProgressBar, progressTextColor } from '@/components/ui-custom/colorful-progress'
import { MetricMeter } from '@/components/ui-custom/metric-meter'

import { formatBillingPoints } from '@/utils/billing'

import { cn } from '@/lib/utils'

interface BillingBalanceMeterProps {
  balance: number
  total: number
  label?: ReactNode
  hoverContent?: ReactNode
  className?: string
  meterClassName?: string
  compact?: boolean
  hideValueText?: boolean
}

interface BillingInlineMeterProps {
  balance: number
  total: number
  className?: string
}

export function getBillingDisplayTotal(balance: number, total: number) {
  if (total > 0) {
    return total
  }
  return Math.max(balance, 0)
}

export function formatBillingPair(balance: number, total: number) {
  const displayTotal = getBillingDisplayTotal(balance, total)
  if (displayTotal <= 0) {
    return formatBillingPoints(balance)
  }
  return `${formatBillingPoints(balance)}/${formatBillingPoints(displayTotal)}`
}

export function getBillingBalancePercent(balance: number, total: number) {
  const displayTotal = getBillingDisplayTotal(balance, total)
  if (displayTotal <= 0) {
    return balance > 0 ? 100 : 0
  }
  return Math.min(100, Math.max(0, (balance / displayTotal) * 100))
}

export function BillingInlineMeter({ balance, total, className }: BillingInlineMeterProps) {
  const percent = getBillingBalancePercent(balance, total)

  return (
    <MetricMeter
      percent={percent}
      mode="balance"
      primary={
        <>
          {percent.toFixed(1)}
          <span className="ml-0.5">%</span>
        </>
      }
      secondary={formatBillingPair(balance, total)}
      className={className}
    />
  )
}

function getCompactBalanceColor(percent: number) {
  if (percent > 90) {
    return 'var(--highlight-emerald)'
  }
  if (percent > 70) {
    return 'var(--highlight-sky)'
  }
  if (percent > 50) {
    return 'var(--highlight-yellow)'
  }
  if (percent > 20) {
    return 'var(--highlight-orange)'
  }
  return 'var(--highlight-red)'
}

function getCompactBorderProgressStyle(percent: number): CSSProperties {
  const normalizedPercent = Math.min(100, Math.max(0, percent))
  const progressAngle = normalizedPercent * 3.6
  const progressColor = getCompactBalanceColor(normalizedPercent)
  const surfaceColor = `color-mix(in srgb, ${progressColor} 14%, var(--card))`

  return {
    background: `linear-gradient(${surfaceColor}, ${surfaceColor}) padding-box, conic-gradient(from -90deg, ${progressColor} 0deg ${progressAngle}deg, var(--border) ${progressAngle}deg 360deg) border-box`,
  }
}

export function BillingBalanceMeter({
  balance,
  total,
  label,
  hoverContent,
  className,
  meterClassName,
  compact = false,
  hideValueText = false,
}: BillingBalanceMeterProps) {
  const percent = getBillingBalancePercent(balance, total)
  const compactBorderProgressStyle = getCompactBorderProgressStyle(percent)

  const content = (
    <div className={cn('inline-flex min-w-0', className)}>
      <div
        className={cn(
          'w-[118px]',
          compact &&
            'flex h-9 w-[156px] items-center justify-between rounded-md border border-transparent px-3 shadow-xs transition-[background,transform]',
          meterClassName
        )}
        style={compact ? compactBorderProgressStyle : undefined}
      >
        {compact ? (
          <>
            <div className="text-muted-foreground min-w-0 truncate text-[12px] leading-none">
              {label}
            </div>
            <p
              className={cn(
                progressTextColor(percent, 'balance'),
                'mb-0 text-[14px] leading-none font-bold'
              )}
            >
              {percent.toFixed(1)}
              <span className="ml-0.5">%</span>
            </p>
          </>
        ) : (
          <>
            <div className="mb-0.5 flex items-center justify-between gap-2">
              <div className="text-muted-foreground min-w-0 truncate text-[10px] leading-none">
                {label}
              </div>
              <p className={progressTextColor(percent, 'balance')}>
                {percent.toFixed(1)}
                <span className="ml-0.5">%</span>
              </p>
            </div>
            <ProgressBar percent={percent} mode="balance" className="h-1 w-full" />
            {!hideValueText && (
              <p className="text-muted-foreground pt-1 font-mono text-xs">
                {formatBillingPair(balance, total)}
              </p>
            )}
          </>
        )}
      </div>
    </div>
  )

  if (!hoverContent) {
    return content
  }

  return (
    <HoverCard openDelay={150} closeDelay={100}>
      <HoverCardTrigger asChild>
        <div className="cursor-help">{content}</div>
      </HoverCardTrigger>
      <HoverCardContent side="bottom" align="start" className="w-[240px] space-y-2 p-3">
        {hoverContent}
      </HoverCardContent>
    </HoverCard>
  )
}
