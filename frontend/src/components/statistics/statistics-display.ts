import type { TFunction } from 'i18next'
import type { DateRange } from 'react-day-picker'

import type { IStatisticsResp, ITimePointData } from '@/services/api/statistics'

// allGPUResource 表示所有整卡 GPU 的聚合选项。
export const allGPUResource = '__all_gpu__'

// statisticsResourceLabel 获取资源的展示名称。
export function statisticsResourceLabel(t: TFunction, id: string, label = id) {
  switch (id) {
    case 'cpu':
      return t('statistics.cpu')
    case 'memory':
      return t('statistics.memory')
    case allGPUResource:
      return t('statistics.allGPU')
    default:
      return label
  }
}

// statisticsResourceUnit 获取资源用量的单位。
export function statisticsResourceUnit(t: TFunction, id: string, type?: string) {
  if (id === 'cpu') return t('statistics.coreHours')
  if (id === 'memory') return t('statistics.gibHours')
  if (id === allGPUResource || type === 'gpu') return t('statistics.cardHours')
  return t('statistics.resourceHours')
}

// statisticsResourceIDs 获取统计结果中的资源名称。
export function statisticsResourceIDs(data: IStatisticsResp) {
  const ids = Object.keys(data.totalUsage)
  return ids.length > 0 ? ids : ['cpu']
}

// hasStatisticsProfile 判断统计结果是否包含监控指标。
export function hasStatisticsProfile(data?: IStatisticsResp) {
  return Object.values(data?.totalUsage ?? {}).some((detail) => detail.effectiveUsage !== undefined)
}

// utilizationTone 按利用率（0~1）分档返回文字与进度条配色：<30% 偏低、30%~60% 中等、≥60% 健康。
export function utilizationTone(utilization: number) {
  if (utilization < 0.3) return { text: 'text-highlight-rose', bar: 'bg-highlight-rose' }
  if (utilization < 0.6) return { text: 'text-highlight-amber', bar: 'bg-highlight-amber' }
  return { text: 'text-highlight-emerald', bar: 'bg-highlight-emerald' }
}

// allocatedChartValue 获取图表中指定资源的分配用量。
export function allocatedChartValue(point: ITimePointData, resource: string) {
  return point.status === 'complete' ? (point.usage[resource] ?? 0) : null
}

// statisticsDateRange 根据预设或自定义日期生成查询范围。
export function statisticsDateRange(utcDay: string, presetDays: number, custom?: DateRange) {
  const end = new Date(utcDay)
  end.setUTCDate(end.getUTCDate() + 1)
  const start = new Date(end)
  start.setUTCDate(start.getUTCDate() - presetDays)

  const { from, to } = custom ?? {}
  if (from && to) {
    return {
      startTime: new Date(
        Date.UTC(from.getFullYear(), from.getMonth(), from.getDate())
      ).toISOString(),
      endTime: new Date(Date.UTC(to.getFullYear(), to.getMonth(), to.getDate() + 1)).toISOString(),
    }
  }
  return { startTime: start.toISOString(), endTime: end.toISOString() }
}

const compactNumber = new Intl.NumberFormat('en-US', {
  notation: 'compact',
  maximumFractionDigits: 1,
})

// formatStatisticsNumber 格式化用量：≥10,000 时使用 K/M/B 紧凑写法，exact 保留完整数值供提示使用。
export function formatStatisticsNumber(value: number) {
  const exact = value.toLocaleString(undefined, { maximumFractionDigits: 2 })
  if (Math.abs(value) < 10_000) return { text: exact, exact, compact: false }
  return { text: compactNumber.format(value), exact, compact: true }
}
