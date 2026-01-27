// src/components/user/statistics/resource-trend.tsx
import { ResponsiveLine } from '@nivo/line'
import { format } from 'date-fns'
import { Filter } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { IStatisticsResp } from '@/services/api/statistics'

import useNivoTheme from '@/hooks/use-nivo-theme'

interface ResourceTrendProps {
  data?: IStatisticsResp
  isLoading: boolean
}

// 定义 Nivo Line 图表的类型
interface SlicePoint {
  id: string
  seriesId: string | number
  seriesColor: string
  data: {
    x: string | number
    y: number
    xFormatted: string
    yFormatted: string
  }
}

interface SliceTooltipProps {
  slice: {
    points: readonly SlicePoint[]
  }
}

export function ResourceTrend({ data, isLoading }: ResourceTrendProps) {
  const { nivoTheme } = useNivoTheme()
  const { t } = useTranslation()

  // 1. State for filtering
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])

  // 2. Initialize selected keys when data loads
  useEffect(() => {
    if (data && Object.keys(data.totalUsage).length > 0) {
      setSelectedKeys((prev) => {
        const allKeys = Object.keys(data.totalUsage)
        return prev.length === 0 ? allKeys : prev
      })
    }
  }, [data])

  // 3. Toggle function
  const toggleResource = (key: string) => {
    setSelectedKeys((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]))
  }

  if (isLoading || !data)
    return <div className="bg-muted/50 h-[350px] w-full animate-pulse rounded-xl" />

  // 4. Prepare Chart Data (Filtered by selectedKeys)
  const allResourceKeys = Object.keys(data.totalUsage)

  // Only map the keys that are currently selected
  const chartData = selectedKeys.map((key) => {
    return {
      id: data.totalUsage[key].label,
      data: data.series.map((item) => ({
        x: format(new Date(item.timestamp), 'yyyy-MM-dd'),
        y: item.usage[key] || 0,
      })),
    }
  })

  return (
    <Card className="col-span-4">
      {/* 5. Header with Filter Dropdown */}
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>
          {t('statistics.trend.title', { defaultValue: 'Resource Usage Trend' })}
        </CardTitle>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="ml-auto flex h-8 gap-2">
              <Filter className="size-3.5" />
              <span className="text-xs">
                {selectedKeys.length === allResourceKeys.length
                  ? t('common.allResources', { defaultValue: 'All Resources' })
                  : t('common.selectedCount', {
                      count: selectedKeys.length,
                      defaultValue: `${selectedKeys.length} Selected`,
                    })}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-[200px]">
            <DropdownMenuLabel>
              {t('common.filterResources', { defaultValue: 'Filter Resources' })}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {allResourceKeys.map((key) => {
              const info = data.totalUsage[key]
              return (
                <DropdownMenuCheckboxItem
                  key={key}
                  checked={selectedKeys.includes(key)}
                  onCheckedChange={() => toggleResource(key)}
                >
                  <span className="truncate">{info.label}</span>
                </DropdownMenuCheckboxItem>
              )
            })}
            {selectedKeys.length === 0 && (
              <div className="text-muted-foreground p-2 text-center text-xs">
                {t('common.selectAtLeastOne', { defaultValue: 'Select at least one' })}
              </div>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>

      <CardContent className="pl-0">
        <div style={{ height: '350px' }}>
          <ResponsiveLine
            data={chartData}
            margin={{ top: 20, right: 110, bottom: 50, left: 60 }}
            xScale={{
              type: 'time',
              format: '%Y-%m-%d',
              precision: 'day',
              useUTC: false,
            }}
            xFormat="time:%Y-%m-%d"
            yScale={{
              type: 'linear',
              min: 'auto',
              max: 'auto',
              stacked: false,
              reverse: false,
            }}
            axisTop={null}
            axisRight={null}
            axisBottom={{
              format: '%b %Y',
              tickValues: 'every 1 month',
              tickSize: 5,
              tickPadding: 5,
              tickRotation: 0,
              legend: t('common.date', { defaultValue: 'Date' }),
              legendOffset: 36,
              legendPosition: 'middle',
            }}
            axisLeft={{
              tickSize: 5,
              tickPadding: 5,
              tickRotation: 0,
              legend: t('common.usage', { defaultValue: 'Usage' }),
              legendOffset: -40,
              legendPosition: 'middle',
            }}
            colors={{ scheme: 'category10' }}
            lineWidth={2}
            pointSize={0}
            pointBorderWidth={2}
            pointBorderColor={{ from: 'serieColor' }}
            enableArea={true}
            areaOpacity={0.1}
            useMesh={true}
            enableGridX={false}
            theme={nivoTheme}
            enableSlices="x"
            sliceTooltip={({ slice }: SliceTooltipProps) => {
              return (
                <div className="bg-popover animate-in fade-in-0 zoom-in-95 z-50 w-fit min-w-[190px] rounded-lg border p-3 shadow-xl">
                  <div className="text-foreground mb-2 border-b pb-2 text-sm font-bold">
                    {format(new Date(slice.points[0].data.x as string), 'PPP')}
                  </div>
                  <div className="space-y-2">
                    {slice.points.map((point: SlicePoint) => (
                      <div key={point.id} className="flex items-center justify-between gap-6">
                        <div className="flex items-center gap-2.5">
                          <div
                            className="size-2.5 shrink-0 rounded-full shadow-sm"
                            style={{ backgroundColor: point.seriesColor }}
                          />
                          <span className="text-muted-foreground max-w-[120px] truncate text-xs font-medium">
                            {String(point.seriesId)}
                          </span>
                        </div>
                        <span className="text-foreground font-mono text-xs font-bold tabular-nums">
                          {Number(point.data.yFormatted).toFixed(2)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )
            }}
            legends={[
              {
                anchor: 'bottom-right',
                direction: 'column',
                justify: false,
                translateX: 100,
                translateY: 0,
                itemsSpacing: 0,
                itemDirection: 'left-to-right',
                itemWidth: 80,
                itemHeight: 20,
                itemOpacity: 0.75,
                symbolSize: 12,
                symbolShape: 'circle',
                symbolBorderColor: 'rgba(0, 0, 0, .5)',
                effects: [
                  {
                    on: 'hover',
                    style: {
                      itemBackground: 'rgba(0, 0, 0, .03)',
                      itemOpacity: 1,
                    },
                  },
                ],
              },
            ]}
          />
        </div>
      </CardContent>
    </Card>
  )
}
