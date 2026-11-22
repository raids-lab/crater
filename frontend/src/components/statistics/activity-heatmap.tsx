import { ResponsiveTimeRange, TimeRangeTooltipProps } from '@nivo/calendar'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  allocatedChartValue,
  statisticsResourceIDs,
  statisticsResourceLabel,
} from '@/components/statistics/statistics-display'

import { IStatisticsResp } from '@/services/api/statistics'

import useNivoTheme from '@/hooks/use-nivo-theme'

interface ActivityHeatmapProps {
  data?: IStatisticsResp
  isLoading: boolean
  from: string
  to: string
}

const CHART_HEIGHT = 200
const CHART_MARGIN = { top: 20, right: 20, bottom: 20, left: 20 }
// 提示框距离日期格边缘超过该值时保持居中，否则向内侧对齐。
const TOOLTIP_EDGE = 140

// tooltipPlacement 让提示框始终朝图表内侧展开，避免被横向滚动容器裁剪。
function tooltipPlacement(cell: TimeRangeTooltipProps, chartWidth: number) {
  const x = CHART_MARGIN.left + cell.coordinates.x + cell.width / 2
  const y = CHART_MARGIN.top + cell.coordinates.y + cell.height / 2
  // Nivo 默认把提示框放在指针上方 14px 处，上半区改为放到指针下方。
  const translateY = y < CHART_HEIGHT / 2 ? 'calc(100% + 28px)' : '0px'
  const translateX = x < TOOLTIP_EDGE ? '50%' : x > chartWidth - TOOLTIP_EDGE ? '-50%' : '0px'
  return { transform: `translate(${translateX}, ${translateY})` }
}

// ActivityHeatmap 展示资源用量的日历热力图。
export function ActivityHeatmap({ data, isLoading, from, to }: ActivityHeatmapProps) {
  const { t } = useTranslation()
  const { nivoTheme, theme } = useNivoTheme()
  const [resource, setResource] = useState('cpu')
  const [chart, setChart] = useState<HTMLDivElement | null>(null)
  if (isLoading) return <div className="bg-muted/50 h-[250px] animate-pulse rounded-xl" />
  if (!data) return null
  const resources = statisticsResourceIDs(data)
  const key = resources.includes(resource) ? resource : resources[0]
  const calendarData = data.series.flatMap((item) => {
    const value = allocatedChartValue(item, key)
    return value === null ? [] : [{ day: item.timestamp.slice(0, 10), value }]
  })
  return (
    <Card className="min-w-0">
      <CardHeader className="flex flex-wrap items-center justify-between gap-3 sm:flex-row">
        <CardTitle>{t('statistics.heatmap')}</CardTitle>
        <Select value={key} onValueChange={setResource}>
          <SelectTrigger aria-label={t('statistics.resource')} className="w-full sm:w-[200px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {resources.map((name) => (
              <SelectItem value={name} key={name}>
                {statisticsResourceLabel(t, name, data.totalUsage[name]?.label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        {calendarData.length < data.series.length && (
          <p className="text-muted-foreground text-sm">{t('statistics.incompleteChartDays')}</p>
        )}
        <div ref={setChart} className="min-w-[600px]" style={{ height: CHART_HEIGHT }}>
          <ResponsiveTimeRange
            data={calendarData}
            from={from}
            to={to}
            emptyColor={theme === 'dark' ? '#1f283b' : '#eeeeee'}
            colors={['#e0f2fe', '#7dd3fc', '#0284c7', '#075985']}
            margin={CHART_MARGIN}
            dayBorderWidth={2}
            dayBorderColor={theme === 'dark' ? '#10172a' : '#ffffff'}
            firstWeekday="monday"
            theme={nivoTheme}
            tooltip={(cell) => (
              <div
                className="bg-popover rounded border p-3 text-sm whitespace-nowrap shadow"
                // Nivo 类型把 tooltip 参数标为 CalendarTooltipProps，运行时实际传入完整的 TimeRange 日期格数据。
                style={tooltipPlacement(cell as TimeRangeTooltipProps, chart?.clientWidth ?? 0)}
              >
                {cell.day}: {Number(cell.value).toFixed(2)} (
                {statisticsResourceLabel(t, key, data.totalUsage[key]?.label)})
              </div>
            )}
          />
        </div>
      </CardContent>
    </Card>
  )
}
