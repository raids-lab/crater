// src/components/statistics/statistics-dashboard.tsx
import { useQuery } from '@tanstack/react-query'
import { format, subDays } from 'date-fns'
import { useMemo, useState } from 'react'
import { DateRange } from 'react-day-picker'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { DateRangePicker } from '@/components/custom/date-range-picker'
// 建议移动到 components/statistics
import { ActivityHeatmap } from '@/components/statistics/activity-heatmap'
import { ResourceSummary } from '@/components/statistics/resource-summary'
// 建议移动到 components/statistics
import { ResourceTrend } from '@/components/statistics/resource-trend'

// 建议移动到 components/statistics
import { StatisticsScope, apiGetStatistics } from '@/services/api/statistics'

interface StatisticsDashboardProps {
  scope: StatisticsScope
  targetID?: number // Cluster 级别可能不需要 targetID
  enabled?: boolean // 控制是否开始请求
}

export function StatisticsDashboard({ scope, targetID, enabled = true }: StatisticsDashboardProps) {
  const { t } = useTranslation()

  // 1. 状态管理
  const [timeRangeType, setTimeRangeType] = useState<string>('30')
  const [customDate, setCustomDate] = useState<DateRange | undefined>({
    from: subDays(new Date(), 30),
    to: new Date(),
  })

  // 2. 计算时间参数
  const { startTimeStr, endTimeStr } = useMemo(() => {
    if (timeRangeType !== 'custom') {
      const end = new Date()
      const start = subDays(end, parseInt(timeRangeType))
      return {
        startTimeStr: start.toISOString(),
        endTimeStr: end.toISOString(),
      }
    } else {
      return {
        startTimeStr: customDate?.from?.toISOString() || subDays(new Date(), 30).toISOString(),
        endTimeStr: customDate?.to?.toISOString() || new Date().toISOString(),
      }
    }
  }, [timeRangeType, customDate])

  // 3. API 请求
  const { data: statsData, isLoading: isStatsLoading } = useQuery({
    queryKey: ['statistics', scope, targetID, startTimeStr, endTimeStr],
    queryFn: () =>
      apiGetStatistics({
        startTime: startTimeStr,
        endTime: endTimeStr,
        step: 'day',
        scope: scope,
        targetID: targetID,
      }),
    select: (res) => res.data,
    enabled: enabled,
  })

  return (
    <div className="space-y-6 p-1">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-lg font-medium">
          {t('statistics.overview', { defaultValue: 'Resource Overview' })}
        </h3>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-muted-foreground text-sm">
            {t('common.timeRange', { defaultValue: 'Time Range' })}:
          </span>

          <Select value={timeRangeType} onValueChange={(val) => setTimeRangeType(val)}>
            <SelectTrigger className="w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="30">{t('time.last30Days')}</SelectItem>
              <SelectItem value="90">{t('time.last3Months')}</SelectItem>
              <SelectItem value="365">{t('time.lastYear')}</SelectItem>
              <SelectItem value="custom">{t('common.customRange')}</SelectItem>
            </SelectContent>
          </Select>

          {timeRangeType === 'custom' && (
            <DateRangePicker
              value={customDate}
              onUpdate={(range) => {
                if (range?.from && range?.to) {
                  setCustomDate(range)
                }
              }}
            />
          )}
        </div>
      </div>

      <ResourceSummary data={statsData} isLoading={isStatsLoading} />
      <ResourceTrend data={statsData} isLoading={isStatsLoading} />
      {/* 只有 Cluster 级别可能不需要热力图，或者热力图含义不同，可根据需要渲染 */}
      <ActivityHeatmap
        data={statsData}
        isLoading={isStatsLoading}
        from={format(new Date(startTimeStr), 'yyyy-MM-dd')}
        to={format(new Date(endTimeStr), 'yyyy-MM-dd')}
      />
    </div>
  )
}
