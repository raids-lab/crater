import { Cpu, Database, GpuIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'

import SimpleTooltip from '@/components/label/simple-tooltip'
import {
  formatStatisticsNumber,
  statisticsResourceLabel,
  statisticsResourceUnit,
  utilizationTone,
} from '@/components/statistics/statistics-display'

import { IResourceDetail, IStatisticsResp } from '@/services/api/statistics'

import { getAcceleratorVendorStyle, parseAcceleratorString } from '@/utils/accelerator'

import { cn } from '@/lib/utils'

interface ResourceSummaryProps {
  data?: IStatisticsResp
  isLoading: boolean
}

const percent = (v: number) => `${v.toLocaleString(undefined, { maximumFractionDigits: 1 })}%`

type ResourceEntry = [string, IResourceDetail]

const COMPUTE_ORDER = ['cpu', 'memory']
const isAccelerator = (type?: string) => type === 'gpu' || type === 'vgpu'

// groupResources 将资源拆分为通用计算资源（cpu、memory 优先）与加速卡资源。
function groupResources(usage: IStatisticsResp['totalUsage']) {
  const entries = Object.entries(usage)
  const rank = (key: string) => {
    const i = COMPUTE_ORDER.indexOf(key)
    return i === -1 ? COMPUTE_ORDER.length : i
  }
  const compute = entries
    .filter(([, detail]) => !isAccelerator(detail.type))
    .sort(([a], [b]) => rank(a) - rank(b))
  const accelerators = entries.filter(([, detail]) => isAccelerator(detail.type))
  return { compute, accelerators }
}

// resourceIcon 按资源类型与厂商选择图标与配色，与节点页加速器配色保持一致。
function resourceIcon(key: string, type?: string) {
  if (isAccelerator(type)) {
    const { vendor } = parseAcceleratorString(key)
    if (vendor) {
      const { fillColor } = getAcceleratorVendorStyle(vendor)
      return {
        Icon: GpuIcon,
        tone: '',
        style: { color: fillColor, backgroundColor: `${fillColor}20` },
      }
    }
    return { Icon: GpuIcon, tone: 'bg-highlight-purple/15 text-highlight-purple' }
  }
  if (key === 'cpu') return { Icon: Cpu, tone: 'bg-highlight-blue/15 text-highlight-blue' }
  return { Icon: Database, tone: 'bg-highlight-emerald/15 text-highlight-emerald' }
}

// ResourceSummary 展示各资源的用量汇总。
export function ResourceSummary({ data, isLoading }: ResourceSummaryProps) {
  const { t } = useTranslation()
  if (isLoading)
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="bg-muted/50 h-36 animate-pulse rounded-xl" />
        ))}
      </div>
    )
  if (!data || Object.keys(data.totalUsage).length === 0)
    return <p className="text-muted-foreground">{t('statistics.empty')}</p>
  const { compute, accelerators } = groupResources(data.totalUsage)
  const grouped = compute.length > 0 && accelerators.length > 0
  if (!grouped) return <ResourceGrid entries={compute.length > 0 ? compute : accelerators} />
  return (
    <div className="space-y-6">
      <ResourceGroup title={t('statistics.computeGroup')} entries={compute} />
      <ResourceGroup title={t('statistics.gpuGroup')} entries={accelerators} />
    </div>
  )
}

function ResourceGroup({ title, entries }: { title: string; entries: ResourceEntry[] }) {
  return (
    <section className="space-y-3">
      <h3 className="text-muted-foreground text-xs font-medium">{title}</h3>
      <ResourceGrid entries={entries} />
    </section>
  )
}

function ResourceGrid({ entries }: { entries: ResourceEntry[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {entries.map(([key, detail]) => (
        <ResourceCard key={key} resourceKey={key} detail={detail} />
      ))}
    </div>
  )
}

// ResourceCard 展示单个资源的占用量，个人视图额外展示利用率。
function ResourceCard({ resourceKey, detail }: { resourceKey: string; detail: IResourceDetail }) {
  const { t } = useTranslation()
  const { Icon, tone, style } = resourceIcon(resourceKey, detail.type)
  const unit = statisticsResourceUnit(t, resourceKey, detail.type)
  return (
    <Card className="min-w-0 gap-0 py-0">
      <CardContent className="flex h-full flex-col gap-4 p-5">
        <div className="space-y-1">
          <div className="flex items-center justify-between gap-2">
            <p className="text-muted-foreground min-w-0 truncate text-sm font-medium">
              {statisticsResourceLabel(t, resourceKey, detail.label)}
            </p>
            <span className={cn('shrink-0 rounded-lg p-2', tone)} style={style}>
              <Icon className="size-4" />
            </span>
          </div>
          <p className="flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5">
            <UsageNumber
              value={detail.usage}
              unit={unit}
              className="text-2xl font-semibold tracking-tight"
            />
            <span className="text-muted-foreground text-sm">{unit}</span>
          </p>
          <p className="text-muted-foreground text-xs">{t('statistics.allocated')}</p>
        </div>
        {detail.effectiveUsage !== undefined && <ProfileSection detail={detail} unit={unit} />}
      </CardContent>
    </Card>
  )
}

// ProfileSection 展示有监控作业的平均利用率与有效用量。
function ProfileSection({ detail, unit }: { detail: IResourceDetail; unit: string }) {
  const { t } = useTranslation()
  const utilization = detail.utilization ?? null
  const monitored = utilization !== null && (detail.profiledUsage ?? 0) > 0
  const value = monitored ? utilization * 100 : 0
  const tone = monitored ? utilizationTone(utilization) : null
  const utilizationText = (
    <span
      className={cn('text-sm font-medium tabular-nums', tone ? tone.text : 'text-muted-foreground')}
    >
      {monitored ? percent(value) : t('statistics.notEvaluated')}
    </span>
  )
  return (
    <div className="mt-auto space-y-2 border-t pt-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground text-xs">{t('statistics.utilization')}</span>
        {monitored && detail.coverage !== null && detail.coverage !== undefined ? (
          <SimpleTooltip
            ariaLabel={t('statistics.coverage')}
            tooltip={`${t('statistics.coverage')} ${percent(detail.coverage * 100)}`}
          >
            <span className="decoration-muted-foreground/50 cursor-help underline decoration-dotted underline-offset-4">
              {utilizationText}
            </span>
          </SimpleTooltip>
        ) : (
          utilizationText
        )}
      </div>
      <div className="bg-muted h-1.5 overflow-hidden rounded-full">
        <div
          className={cn('h-full rounded-full transition-[width]', tone?.bar)}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
      <div className="flex items-center justify-between gap-2 text-xs">
        <span className="text-muted-foreground">{t('statistics.effective')}</span>
        <span className="tabular-nums">
          {monitored ? (
            <>
              <UsageNumber
                value={detail.effectiveUsage ?? 0}
                unit={unit}
                className="text-foreground font-medium"
              />
              <span className="text-muted-foreground ml-1">{unit}</span>
            </>
          ) : (
            <span className="text-muted-foreground">—</span>
          )}
        </span>
      </div>
    </div>
  )
}

// UsageNumber 展示用量数值，紧凑写法时通过提示显示完整数值与单位。
function UsageNumber({
  value,
  unit,
  className,
}: {
  value: number
  unit: string
  className: string
}) {
  const { text, exact, compact } = formatStatisticsNumber(value)
  const content = (
    <span className={cn('tabular-nums', compact && 'cursor-help', className)}>{text}</span>
  )
  if (!compact) return content
  return (
    <SimpleTooltip ariaLabel={`${exact} ${unit}`} tooltip={`${exact} ${unit}`}>
      {content}
    </SimpleTooltip>
  )
}
