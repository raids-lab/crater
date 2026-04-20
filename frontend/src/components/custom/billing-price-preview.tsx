import { useQuery } from '@tanstack/react-query'
import { CoinsIcon } from 'lucide-react'

import { Badge } from '@/components/ui/badge'

import { apiBillingPriceList } from '@/services/api/billing'
import { apiGetBillingStatus } from '@/services/api/system-config'

import {
  BillingPriceEntryInput,
  formatBillingPoints,
  summarizeBillingPriceEntries,
} from '@/utils/billing'
import { isBillingVisibleForUser } from '@/utils/billing-visibility'

import { cn } from '@/lib/utils'

interface BillingPricePreviewProps {
  entries: BillingPriceEntryInput[]
  className?: string
}

export function BillingPricePreview({ entries, className }: BillingPricePreviewProps) {
  const { data: billingStatus } = useQuery({
    queryKey: ['system-config', 'billing-status', 'price-preview'],
    queryFn: () => apiGetBillingStatus().then((res) => res.data),
    staleTime: 60 * 1000,
  })
  const billingVisible = isBillingVisibleForUser(billingStatus)
  const { data: priceSources } = useQuery({
    queryKey: ['resources', 'billing-price-preview'],
    queryFn: () => apiBillingPriceList(),
    select: (res) =>
      res.data.reduce<Record<string, { label: string; unitPrice: number }>>((acc, resource) => {
        acc[resource.name] = {
          label: resource.label || resource.name,
          unitPrice: resource.unitPrice ?? 0,
        }
        return acc
      }, {}),
    staleTime: 60 * 1000,
    enabled: billingVisible,
  })

  const summaries = summarizeBillingPriceEntries(entries, priceSources ?? {})
  const visibleSummaries = summaries.filter((entry) => entry.items.length > 0)
  const totalPerHour = visibleSummaries.reduce((sum, entry) => sum + entry.subtotalPerHour, 0)
  const uniqueItems = new Map<string, { label: string; unitPrice: number }>()

  visibleSummaries.forEach((summary) => {
    summary.items.forEach((item) => {
      if (!uniqueItems.has(item.name)) {
        uniqueItems.set(item.name, {
          label: item.label,
          unitPrice: item.unitPrice,
        })
      }
    })
  })

  if (!billingVisible || entries.length === 0) {
    return null
  }

  return (
    <div className={cn('bg-muted/20 rounded-xl border p-4', className)}>
      <div className="flex items-center gap-2 text-sm font-medium">
        <CoinsIcon className="text-primary size-4" />
        资源价格预览（点/单位/小时）
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        {uniqueItems.size > 0 ? (
          Array.from(uniqueItems.entries()).map(([name, item]) => (
            <Badge key={name} variant="outline" className="bg-background font-normal">
              <span className="text-muted-foreground mr-1">{item.label}</span>
              <span className="font-mono">{formatBillingPoints(item.unitPrice)}</span>
            </Badge>
          ))
        ) : (
          <div className="text-muted-foreground bg-background rounded-lg border border-dashed px-3 py-2 text-xs">
            当前配置中的资源单价均为 0.00，表示免费或暂未配置计费。
          </div>
        )}
      </div>

      {visibleSummaries.length > 1 && (
        <div className="mt-3 grid gap-2">
          {visibleSummaries.map((summary, index) => (
            <div
              key={`${summary.label ?? 'default'}-${index}`}
              className="bg-background/80 flex items-center justify-between rounded-lg border px-3 py-2 text-sm"
            >
              <span className="text-muted-foreground">
                {summary.label}
                {summary.multiplier > 1 ? ` x${summary.multiplier}` : ''}
              </span>
              <span className="font-mono">{formatBillingPoints(summary.subtotalPerHour)}</span>
            </div>
          ))}
        </div>
      )}

      <div className="bg-background mt-3 flex items-center justify-between rounded-lg border px-3 py-2">
        <div>
          <div className="text-sm font-medium">当前配置理论总价（点/小时）</div>
          <div className="text-muted-foreground text-xs">
            按小时展示并保留两位小数；0.00 表示免费或暂未配置，不参与提交前额度校验
          </div>
        </div>
        <div className="font-mono text-sm font-semibold">{formatBillingPoints(totalPerHour)}</div>
      </div>

      <div className="bg-background mt-3 flex items-center justify-between rounded-lg border px-3 py-2">
        <div>
          <div className="text-sm font-medium">当前每个作业免费时长</div>
          <div className="text-muted-foreground text-xs">超出后按资源单价进入计费。</div>
        </div>
        <div className="font-mono text-sm font-semibold">
          {billingStatus?.jobFreeMinutes ?? 0} 分钟
        </div>
      </div>
    </div>
  )
}
