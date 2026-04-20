import { ReactNode } from 'react'

import { BillingBalanceMeter, formatBillingPair } from '@/components/custom/billing-balance-meter'
import { TimeDistance } from '@/components/custom/time-distance'

import { BillingSummaryResp } from '@/services/api/context'

import { formatBillingPoints } from '@/utils/billing'

import { cn } from '@/lib/utils'

interface BillingAccountLike {
  accountId: number
  accountName: string
  accountNickname?: string
  periodFreeBalance: number
  nextIssueAt?: string
  effectiveIssueAmount: number
}

interface BillingSummaryCardsProps {
  summary?: Partial<BillingSummaryResp>
  accounts?: BillingAccountLike[]
  className?: string
  emphasis?: 'inline' | 'user'
  compact?: boolean
}

function getActiveBucket(periodFreeBalance: number, extraBalance: number) {
  return periodFreeBalance <= 0 && extraBalance > 0 ? '额外额度' : '免费额度'
}

function DetailRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono">{value}</span>
    </div>
  )
}

export function BillingSummaryCards({
  summary,
  accounts,
  className,
  emphasis = 'inline',
  compact = false,
}: BillingSummaryCardsProps) {
  const periodFreeBalance = summary?.periodFreeBalance ?? 0
  const extraBalance = summary?.extraBalance ?? 0
  const effectiveIssueAmount = summary?.effectiveIssueAmount ?? 0
  const nextIssueAt = summary?.nextIssueAt
  const currentBucket = getActiveBucket(periodFreeBalance, extraBalance)

  if (emphasis === 'user') {
    const accountRows = accounts?.length
      ? accounts
      : [
          {
            accountId: 0,
            accountName: '当前账户',
            accountNickname: '当前账户',
            periodFreeBalance,
            nextIssueAt,
            effectiveIssueAmount,
          },
        ]
    const totalBalance =
      accountRows.reduce((sum, item) => sum + item.periodFreeBalance, 0) + extraBalance
    const issuedTotal =
      accountRows.reduce((sum, item) => sum + Math.max(item.effectiveIssueAmount, 0), 0) +
      Math.max(extraBalance, 0)
    const totalAmount = issuedTotal > 0 ? issuedTotal : Math.max(totalBalance, 0)

    return (
      <div className={cn('inline-flex', className)}>
        <BillingBalanceMeter
          balance={totalBalance}
          total={totalAmount}
          meterClassName="w-[126px]"
          hoverContent={
            <div className="space-y-2">
              {accountRows.map((item) => (
                <div key={item.accountId} className="space-y-1">
                  <DetailRow
                    label={item.accountNickname || item.accountName}
                    value={formatBillingPair(item.periodFreeBalance, item.effectiveIssueAmount)}
                  />
                  <DetailRow
                    label="重置时间"
                    value={item.nextIssueAt ? <TimeDistance date={item.nextIssueAt} /> : '-'}
                  />
                </div>
              ))}
              <DetailRow label="额外额度" value={formatBillingPoints(extraBalance)} />
            </div>
          }
        />
      </div>
    )
  }

  const layerBalance = currentBucket === '额外额度' ? extraBalance : periodFreeBalance
  const layerTotal = currentBucket === '额外额度' ? extraBalance : effectiveIssueAmount
  return (
    <div className={cn('inline-flex', className)}>
      <BillingBalanceMeter
        balance={layerBalance}
        total={layerTotal}
        label={currentBucket === '额外额度' ? '额外' : '免费'}
        meterClassName="w-[122px]"
        compact={compact}
        hideValueText={compact}
        hoverContent={
          <div className="space-y-2">
            <DetailRow label="当前使用层级" value={currentBucket} />
            <DetailRow
              label="剩余/账户总额"
              value={formatBillingPair(layerBalance, effectiveIssueAmount)}
            />
            <DetailRow label="额外额度" value={formatBillingPoints(extraBalance)} />
            <DetailRow
              label="下次重置时间"
              value={nextIssueAt ? <TimeDistance date={nextIssueAt} /> : '-'}
            />
          </div>
        }
      />
    </div>
  )
}
