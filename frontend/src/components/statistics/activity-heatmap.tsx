// src/components/user/statistics/activity-heatmap.tsx
import { ResponsiveTimeRange } from '@nivo/calendar'
import { format } from 'date-fns'
import { Filter } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
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

interface ActivityHeatmapProps {
  data?: IStatisticsResp
  isLoading: boolean
  from: string
  to: string
}

// 定义 tooltip 接收的数据结构
interface HeatmapDatum {
  day: string
  value: number
  details: Record<string, number>
  // Nivo 可能会注入额外的属性（如 color, x, y），如果需要使用可以补全，不需要则忽略
}

export function ActivityHeatmap({ data, isLoading, from, to }: ActivityHeatmapProps) {
  const { t } = useTranslation()
  const { nivoTheme, theme } = useNivoTheme()

  // 存储当前选中的资源 Key
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])

  // 当数据加载完成后，默认选中所有资源
  useEffect(() => {
    if (data && Object.keys(data.totalUsage).length > 0) {
      setSelectedKeys((prev) => {
        const allKeys = Object.keys(data.totalUsage)
        return prev.length === 0 ? allKeys : prev
      })
    }
  }, [data])

  // 处理多选逻辑
  const toggleResource = (key: string) => {
    setSelectedKeys((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]))
  }

  // 计算聚合数据
  const calendarData = useMemo(() => {
    if (!data) return []

    return data.series
      .map((item) => {
        const totalValue = selectedKeys.reduce((sum, key) => sum + (item.usage[key] || 0), 0)
        const details: Record<string, number> = {}
        selectedKeys.forEach((key) => {
          const val = item.usage[key] || 0
          if (val > 0) details[key] = val
        })

        return {
          day: format(new Date(item.timestamp), 'yyyy-MM-dd'),
          value: totalValue, // 去掉 Math.floor，保留原始值用于判断
          details,
        }
      })
      .filter((d) => d.value > 0) // 只要有值就显示
  }, [data, selectedKeys])

  if (isLoading || !data)
    return <div className="bg-muted/50 h-[250px] w-full animate-pulse rounded-xl" />

  const allResourceKeys = Object.keys(data.totalUsage)

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{t('loginHeatmap.title', { defaultValue: 'User Activity Heatmap' })}</CardTitle>

        {/* 多选下拉菜单 */}
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

      <CardContent>
        <div style={{ height: '200px' }}>
          <ResponsiveTimeRange
            data={calendarData}
            from={from}
            to={to}
            emptyColor={theme === 'dark' ? '#1f283b' : '#eeeeee'}
            colors={['#61cdbb', '#97e3d5', '#e8c1a0', '#f47560']}
            margin={{ top: 20, right: 20, bottom: 20, left: 20 }}
            dayBorderWidth={2}
            dayBorderColor={theme === 'dark' ? '#10172a' : '#ffffff'}
            firstWeekday="monday"
            theme={nivoTheme}
            // 修复 TS 报错：使用 (node: any) 绕过严格类型检查，因为 Nivo 类型定义里没有 details
            tooltip={(nodeInput) => {
              const node = nodeInput as unknown as HeatmapDatum

              const { day, value, details } = node

              // 确保 details 存在
              const safeDetails = details || {}
              const hasDetails = Object.keys(safeDetails).length > 0
              // console.log('Nivo Node:', node);

              // const { day, value, data: rawData } = node;

              // // 检查 rawData 里面是否有 details
              // console.log('Raw Data from node:', rawData);
              // const details = rawData?.details || {};
              // const hasDetails = Object.keys(details).length > 0;

              return (
                <div className="bg-popover animate-in fade-in-0 zoom-in-95 z-50 min-w-[180px] rounded-lg border p-3 shadow-xl outline-none">
                  <div className="mb-2 flex items-center justify-between border-b pb-2">
                    <span className="text-foreground text-sm font-bold">{day}</span>
                    <span className="bg-primary/10 text-primary rounded px-1.5 py-0.5 font-mono text-xs font-medium">
                      {Number(value).toFixed(2)}
                    </span>
                  </div>
                  <div className="space-y-1.5">
                    {hasDetails ? (
                      Object.entries(details).map(([key, val]) => (
                        <div key={key} className="flex items-center justify-between gap-4 text-xs">
                          <span className="text-muted-foreground">
                            {data.totalUsage[key]?.label || key}
                          </span>
                          <span className="text-foreground font-mono font-medium tabular-nums">
                            {Number(val).toFixed(2)}
                          </span>
                        </div>
                      ))
                    ) : (
                      <div className="text-muted-foreground py-1 text-center text-xs italic">
                        {t('common.noData', { defaultValue: 'No detailed usage' })}
                      </div>
                    )}
                  </div>
                </div>
              )
            }}
          />
        </div>
      </CardContent>
    </Card>
  )
}
