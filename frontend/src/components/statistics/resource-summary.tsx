// src/components/user/statistics/resource-summary.tsx
import { Cpu, Database, GpuIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { IStatisticsResp } from '@/services/api/statistics'

interface ResourceSummaryProps {
  data?: IStatisticsResp
  isLoading: boolean
}

export function ResourceSummary({ data, isLoading }: ResourceSummaryProps) {
  const { t } = useTranslation()

  if (isLoading || !data) {
    return (
      <div className="grid gap-4 md:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-muted/50 h-24 animate-pulse rounded-xl" />
        ))}
      </div>
    )
  }

  // 辅助函数：根据资源类型选择图标
  const getIcon = (type: string) => {
    if (type.includes('gpu')) return GpuIcon
    if (type.includes('common')) return Cpu
    return Database
  }

  // 辅助函数：格式化数值 (保留2位小数)
  const formatValue = (val: number) => val.toLocaleString(undefined, { maximumFractionDigits: 2 })

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {Object.entries(data.totalUsage).map(([key, detail]) => {
        const Icon = getIcon(detail.type)
        return (
          <Card key={key}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{detail.label}</CardTitle>
              <Icon className="text-muted-foreground h-4 w-4" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatValue(detail.usage)}</div>
              <p className="text-muted-foreground text-xs">
                {/* 这里可以根据单位显示，例如 "Core Hours" 或 "GPU Hours" */}
                Total Usage ({t('unit.hours', { defaultValue: 'Hours' })})
              </p>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
