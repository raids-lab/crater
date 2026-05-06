/**
 * AIOps Health Overview Page - User Version (个人数据)
 */
import { useQuery } from '@tanstack/react-query'
import { Activity, AlertCircle, Calendar, Clock, Info, TrendingUp } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import PageTitle from '@/components/layout/page-title'

import { apiGetHealthOverview } from '@/services/api/aiops'
import type { IHealthOverview } from '@/services/api/aiops'

const TIME_RANGES = [
  { value: '1', labelKey: 'aiops.time.today' },
  { value: '7', labelKey: 'aiops.time.week' },
  { value: '30', labelKey: 'aiops.time.month' },
  { value: '0', labelKey: 'aiops.time.all' },
]

export function HealthOverview() {
  const { t } = useTranslation()
  const [timeRange, setTimeRange] = useState('7')

  const { data, isLoading, error } = useQuery({
    queryKey: ['aiops', 'health-overview', timeRange],
    queryFn: async () => {
      const days = timeRange === '0' ? 0 : parseInt(timeRange)
      const response = await apiGetHealthOverview(days)
      return response.data
    },
    refetchInterval: 30000, // Refetch every 30s
  })

  if (isLoading) {
    return (
      <div className="space-y-4">
        <PageTitle title={t('aiops.page.title')} description={t('aiops.page.loadingUser')} />
        <Card>
          <CardContent className="text-muted-foreground py-8 text-center">
            {t('aiops.common.loading')}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="space-y-4">
        <PageTitle title={t('aiops.page.title')} description={t('aiops.page.loadFailedUser')} />
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>{t('aiops.common.loadFailed')}</AlertTitle>
          <AlertDescription>
            {error ? t('aiops.common.backendUnavailable') : t('aiops.common.dataLoadFailed')}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const healthData = data as IHealthOverview
  const timeRangeLabel = t(
    TIME_RANGES.find((r) => r.value === timeRange)?.labelKey || 'aiops.time.week'
  )

  return (
    <div className="space-y-4">
      <PageTitle title={t('aiops.page.userTitle')} description={t('aiops.page.userDesc')}>
        <div className="flex items-center gap-2">
          <Calendar className="text-muted-foreground h-4 w-4" />
          <Select value={timeRange} onValueChange={setTimeRange}>
            <SelectTrigger className="w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TIME_RANGES.map((range) => (
                <SelectItem key={range.value} value={range.value}>
                  {t(range.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </PageTitle>

      {/* Info Alert */}
      <Alert>
        <Info className="h-4 w-4" />
        <AlertTitle>{t('aiops.common.aboutPage')}</AlertTitle>
        <AlertDescription>
          {t('aiops.page.userAbout', { timeRange: timeRangeLabel })}
        </AlertDescription>
      </Alert>

      {/* Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title={t('aiops.metric.totalJobs')}
          value={healthData?.totalJobs || 0}
          description={t('aiops.metric.totalJobsDesc', { timeRange: timeRangeLabel })}
          icon={Activity}
          variant="default"
        />
        <MetricCard
          title={t('aiops.metric.failedJobs')}
          value={healthData?.failedJobs || 0}
          description={t('aiops.metric.failedJobsDesc')}
          icon={AlertCircle}
          variant={healthData && healthData.failedJobs > 0 ? 'destructive' : 'default'}
        />
        <MetricCard
          title={t('aiops.metric.pendingJobs')}
          value={healthData?.pendingJobs || 0}
          description={t('aiops.metric.pendingJobsDesc')}
          icon={Clock}
          variant="warning"
        />
        <MetricCard
          title={t('aiops.metric.failureRate')}
          value={`${healthData?.failureRate?.toFixed(1) || 0}%`}
          description={t('aiops.metric.failureRateDesc', { timeRange: timeRangeLabel })}
          icon={TrendingUp}
          variant={healthData && healthData.failureRate > 10 ? 'destructive' : 'default'}
        />
      </div>

      {/* Failure Trend Chart */}
      <Card>
        <CardHeader>
          <CardTitle>{t('aiops.chart.failureTrend')}</CardTitle>
        </CardHeader>
        <CardContent>
          {healthData && healthData.failureTrend && healthData.failureTrend.length > 0 ? (
            <div className="overflow-x-auto">
              <div className="flex h-[200px] min-w-max items-end gap-2 px-2">
                {healthData.failureTrend.map((item) => {
                  const maxCount = Math.max(...healthData.failureTrend.map((t) => t.count), 1)
                  return (
                    <div key={item.date} className="flex min-w-[40px] flex-col items-center gap-1">
                      <div
                        className="bg-destructive w-full rounded-t"
                        style={{
                          height: `${(item.count / maxCount) * 160}px`,
                          minHeight: '4px',
                        }}
                      />
                      <div className="text-muted-foreground text-xs whitespace-nowrap">
                        {item.date.slice(5)}
                      </div>
                      <div className="text-xs font-medium">{item.count}</div>
                    </div>
                  )
                })}
              </div>
            </div>
          ) : (
            <div className="text-muted-foreground flex h-[200px] items-center justify-center">
              {t('aiops.common.noData')}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Top Failure Reasons */}
      <Card>
        <CardHeader>
          <CardTitle>{t('aiops.chart.topReasons')}</CardTitle>
        </CardHeader>
        <CardContent>
          {healthData && healthData.topFailureReasons && healthData.topFailureReasons.length > 0 ? (
            <div className="space-y-3">
              {healthData.topFailureReasons.map((reason, index) => (
                <div key={reason.reason} className="flex items-center gap-3">
                  <div className="bg-primary/10 text-primary flex h-8 w-8 flex-none items-center justify-center rounded-full text-sm font-semibold">
                    {index + 1}
                  </div>
                  <div className="flex-1">
                    <div className="font-medium">{translateReason(reason.reason, t)}</div>
                    <div className="text-muted-foreground text-sm">{reason.reason}</div>
                  </div>
                  <div className="flex-none text-2xl font-bold">{reason.count}</div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-muted-foreground py-8 text-center">
              {t('aiops.common.noFailedJobs')}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

// Metric Card Component
interface MetricCardProps {
  title: string
  value: number | string
  description: string
  icon: React.ElementType
  variant?: 'default' | 'destructive' | 'warning'
}

function MetricCard({
  title,
  value,
  description,
  icon: Icon,
  variant = 'default',
}: MetricCardProps) {
  const variantStyles = {
    default: 'border-border',
    destructive: 'border-destructive',
    warning: 'border-yellow-500',
  }

  return (
    <Card className={`${variantStyles[variant]}`}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="text-muted-foreground h-4 w-4" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </CardContent>
    </Card>
  )
}

// Translate internal reason names to Chinese
function translateReason(
  reason: string,
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  return t(`aiops.reason.${reason}`, { defaultValue: reason })
}
