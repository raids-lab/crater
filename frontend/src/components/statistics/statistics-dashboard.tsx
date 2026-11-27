import { useQuery } from '@tanstack/react-query'
import { subDays } from 'date-fns'
import { InfoIcon, TriangleAlertIcon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { DateRange } from 'react-day-picker'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { DateRangePicker } from '@/components/custom/date-range-picker'
import SimpleTooltip from '@/components/label/simple-tooltip'
import { ActivityHeatmap } from '@/components/statistics/activity-heatmap'
import { ResourceRankings } from '@/components/statistics/resource-rankings'
import { ResourceSummary } from '@/components/statistics/resource-summary'
import { ResourceTrend } from '@/components/statistics/resource-trend'
import {
  hasStatisticsProfile,
  statisticsDateRange,
} from '@/components/statistics/statistics-display'

import { StatisticsScope, apiAdminGetStatistics, apiGetStatistics } from '@/services/api/statistics'

import useIsAdmin from '@/hooks/use-admin'

interface StatisticsDashboardProps {
  scope: StatisticsScope
  targetID?: number
  enabled?: boolean
}

// StatisticsDashboard 组织资源统计查询和展示。
export function StatisticsDashboard({ scope, targetID, enabled = true }: StatisticsDashboardProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const [timeRangeType, setTimeRangeType] = useState('30')
  const [customDate, setCustomDate] = useState<DateRange | undefined>({
    from: subDays(new Date(), 29),
    to: new Date(),
  })
  const [utcDay, setUTCDay] = useState(() => new Date().toISOString().slice(0, 10))
  useEffect(() => {
    const updateUTCDay = () => setUTCDay(new Date().toISOString().slice(0, 10))
    const timer = window.setInterval(updateUTCDay, 60000)
    window.addEventListener('focus', updateUTCDay)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', updateUTCDay)
    }
  }, [])
  const { startTime, endTime } = useMemo(
    () =>
      statisticsDateRange(
        utcDay,
        timeRangeType === 'custom' ? 30 : Number(timeRangeType),
        timeRangeType === 'custom' ? customDate : undefined
      ),
    [utcDay, timeRangeType, customDate]
  )
  const request = { startTime, endTime, step: 'day' as const, scope, targetID }
  const { data, isLoading, isError } = useQuery({
    queryKey: ['statistics', isAdmin, request],
    queryFn: () => (isAdmin ? apiAdminGetStatistics(request) : apiGetStatistics(request)),
    select: (res) => res.data,
    enabled,
    refetchInterval: 60000,
  })
  const progress = data?.progress
  const [tab, setTab] = useState('overview')
  const showRankings = scope !== 'user'
  const incomplete =
    progress && (!progress.ready || !progress.backfillComplete || progress.pendingDays > 0)
  const overview = (
    <>
      <ResourceSummary data={data} isLoading={isLoading} />
      <ResourceTrend data={data} isLoading={isLoading} />
      <ActivityHeatmap
        data={data}
        isLoading={isLoading}
        from={startTime.slice(0, 10)}
        to={new Date(Date.parse(endTime) - 86400000).toISOString().slice(0, 10)}
      />
    </>
  )
  return (
    <Tabs value={tab} onValueChange={setTab} className="min-w-0 gap-6 p-1">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2">
          {showRankings ? (
            <TabsList>
              <TabsTrigger value="overview" className="px-3">
                {t('statistics.overview')}
              </TabsTrigger>
              <TabsTrigger value="rankings" className="px-3">
                {t('statistics.rankings')}
              </TabsTrigger>
            </TabsList>
          ) : (
            <h3 className="text-lg font-medium">{t('statistics.overview')}</h3>
          )}
          {/* 统计口径说明收进悬浮提示 */}
          <SimpleTooltip
            ariaLabel={t('statistics.utcDay')}
            tooltip={
              <div className="max-w-xs space-y-1">
                <p>{t('statistics.utcDay')}</p>
                {hasStatisticsProfile(data) && <p>{t('statistics.estimateNote')}</p>}
              </div>
            }
          >
            <InfoIcon className="text-muted-foreground size-4" />
          </SimpleTooltip>
          {progress?.coveredUntil && (
            <span className="text-muted-foreground text-xs" role="status">
              {t('statistics.cutoff', { time: new Date(progress.coveredUntil).toLocaleString() })}
            </span>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Select value={timeRangeType} onValueChange={setTimeRangeType}>
            <SelectTrigger className="w-[140px]" aria-label={t('common.timeRange')}>
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
                if (range?.from && range?.to) setCustomDate(range)
              }}
            />
          )}
        </div>
      </div>
      {incomplete && (
        <p
          className="flex items-center gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400"
          role="status"
        >
          <TriangleAlertIcon className="size-4 shrink-0" />
          {t('statistics.incomplete', { count: progress.pendingDays })}
        </p>
      )}
      {isError ? (
        <p role="alert">{t('statistics.loadError')}</p>
      ) : showRankings ? (
        <>
          <TabsContent value="overview" className="min-w-0 space-y-6">
            {overview}
          </TabsContent>
          <TabsContent value="rankings" className="min-w-0">
            <ResourceRankings
              key={`${scope}-${targetID}-${startTime}-${endTime}`}
              request={request}
              data={data}
              enabled={enabled && !!data}
            />
          </TabsContent>
        </>
      ) : (
        <div className="min-w-0 space-y-6">{overview}</div>
      )}
    </Tabs>
  )
}
