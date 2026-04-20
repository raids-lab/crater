import { useQuery } from '@tanstack/react-query'
import { ReactNode, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'

import { BillingInlineMeter, formatBillingPair } from '@/components/custom/billing-balance-meter'
import { TimeDistance } from '@/components/custom/time-distance'

import { apiAdminGetUserBillingAccounts } from '@/services/api/billing'

import { formatBillingPoints } from '@/utils/billing'

import { cn } from '@/lib/utils'

interface UserPointsTooltipProps {
  userName?: string
  totalPoints: number
  extraPoints?: number
  periodFreePoints?: number
  effectiveIssueAmount?: number
  nextIssueLabel?: ReactNode
  fetchDetail?: boolean
  showInlineBreakdown?: boolean
  inlineVariant?: 'minimal' | 'summary'
  mode?: 'default' | 'account'
}

export function UserPointsTooltip({
  userName,
  totalPoints,
  extraPoints = 0,
  periodFreePoints = 0,
  effectiveIssueAmount = 0,
  nextIssueLabel,
  fetchDetail = false,
  showInlineBreakdown = false,
  inlineVariant = 'minimal',
  mode = 'default',
}: UserPointsTooltipProps) {
  const [open, setOpen] = useState(false)
  const { data } = useQuery({
    queryKey: ['admin', 'users', userName, 'billing-accounts', 'tooltip'],
    queryFn: () => apiAdminGetUserBillingAccounts(userName ?? '').then((res) => res.data),
    enabled: Boolean(fetchDetail && userName && open),
    staleTime: 60 * 1000,
  })

  const aggregatedPeriodFreePoints = data?.reduce((sum, item) => sum + item.periodFreeBalance, 0)
  const aggregatedIssueAmount = data?.reduce(
    (sum, item) => sum + Math.max(item.effectiveIssueAmount, 0),
    0
  )
  const useAggregatedTotals = mode !== 'account' && Boolean(data?.length)
  const resolvedPeriodFreePoints = useAggregatedTotals
    ? (aggregatedPeriodFreePoints ?? periodFreePoints)
    : periodFreePoints
  const resolvedEffectiveIssueAmount = useAggregatedTotals
    ? (aggregatedIssueAmount ?? effectiveIssueAmount)
    : effectiveIssueAmount

  const inlineContent = !showInlineBreakdown ? (
    <Badge variant="outline" className="font-mono text-xs font-normal">
      {formatBillingPoints(totalPoints)}
    </Badge>
  ) : mode === 'account' ? (
    <BillingInlineMeter
      balance={resolvedPeriodFreePoints}
      total={resolvedEffectiveIssueAmount}
      className={inlineVariant === 'minimal' ? 'w-[96px]' : 'w-[116px]'}
    />
  ) : (
    <BillingInlineMeter
      balance={resolvedPeriodFreePoints}
      total={resolvedEffectiveIssueAmount}
      className="w-[96px]"
    />
  )

  return (
    <HoverCard open={open} onOpenChange={setOpen} openDelay={150} closeDelay={100}>
      <HoverCardTrigger asChild>
        <div
          className={cn(
            'inline-flex cursor-help flex-wrap items-center gap-1.5',
            showInlineBreakdown && 'w-full'
          )}
        >
          {inlineContent}
        </div>
      </HoverCardTrigger>
      <HoverCardContent side="top" align="start" className="w-[260px] space-y-2 p-3">
        <div className="space-y-2 text-xs">
          {mode === 'account' ? (
            <>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">当前账户免费额度</span>
                <span className="font-mono">
                  {formatBillingPair(resolvedPeriodFreePoints, resolvedEffectiveIssueAmount)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">额外点数</span>
                <span className="font-mono">{formatBillingPoints(extraPoints)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">总可用</span>
                <span className="font-mono">{formatBillingPoints(totalPoints)}</span>
              </div>
              {nextIssueLabel ? (
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">重置时间</span>
                  <span className="font-medium">{nextIssueLabel}</span>
                </div>
              ) : null}
            </>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">总可用</span>
                <span className="font-mono">{formatBillingPoints(totalPoints)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">免费额度合计</span>
                <span className="font-mono">
                  {formatBillingPair(resolvedPeriodFreePoints, resolvedEffectiveIssueAmount)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">额外点数</span>
                <span className="font-mono">{formatBillingPoints(extraPoints)}</span>
              </div>
              {nextIssueLabel ? (
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">重置时间</span>
                  <span className="font-medium">{nextIssueLabel}</span>
                </div>
              ) : null}
            </>
          )}
        </div>
        {fetchDetail && (data?.length ?? 0) > 0 ? (
          <div className="space-y-2 border-t pt-2">
            {(data ?? []).map((item) => {
              return (
                <div key={item.accountId} className="space-y-1">
                  <div className="flex items-center justify-between gap-3 text-[11px]">
                    <span className="text-muted-foreground max-w-[148px] truncate">
                      {item.accountNickname || item.accountName}
                    </span>
                    <span className="font-mono">
                      {formatBillingPair(item.periodFreeBalance, item.effectiveIssueAmount)}
                    </span>
                  </div>
                  {item.nextIssueAt ? (
                    <div className="flex items-center justify-between text-[11px]">
                      <span className="text-muted-foreground">重置时间</span>
                      <span className="font-medium">
                        <TimeDistance date={item.nextIssueAt} />
                      </span>
                    </div>
                  ) : null}
                </div>
              )
            })}
          </div>
        ) : null}
      </HoverCardContent>
    </HoverCard>
  )
}
