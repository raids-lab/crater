import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import {
  ChevronDownIcon,
  ChevronUpIcon,
  CircleHelpIcon,
  ClipboardListIcon,
  CpuIcon,
  GaugeIcon,
  GpuIcon,
  MemoryStickIcon,
} from 'lucide-react'
import { type ComponentType, type ReactNode, type SVGProps, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import AcceleratorBadge from '@/components/badge/accelerator-badge'

import {
  type JobResourceSummaryAccelerator,
  type JobResourceSummaryUsage,
  apiContextJobResourceSummary,
} from '@/services/api/context'

import { getAcceleratorVendorStyle, parseAcceleratorString } from '@/utils/accelerator'
import { convertKResourceToResource } from '@/utils/resource'
import { atomUserContext } from '@/utils/store'

import { REFETCH_INTERVAL } from '@/lib/constants'
import { cn } from '@/lib/utils'

const MIN_BAR_WIDTH_PERCENT = 16
const SUMMARY_SECTION_MAX_HEIGHT_CLASS = 'max-h-[16rem]'
const OCCUPIED_JOBS_COLOR = '#f59e0b'
const PENDING_BAR_ALPHA = 0.55

type IconComponent = ComponentType<SVGProps<SVGSVGElement>>

interface ResourceUsageCardItem {
  key: string
  title: ReactNode
  icon: IconComponent
  meterColor: string
  value: ReactNode
  usedLabel?: string
  meterPercent?: number
  pendingMeterPercent?: number
  showTrack?: boolean
}

function formatCpuValue(value: string): string {
  const amount = convertKResourceToResource('cpu', value) ?? 0
  return `${Math.round(amount)}c`
}

function formatMemoryValue(value: string): string {
  const amount = convertKResourceToResource('memory', value) ?? 0
  if (amount >= 1024) {
    return `${Math.round(amount / 1024)}Ti`
  }
  return `${Math.round(amount)}Gi`
}

function formatPrimaryUsageParts(
  resourceKey: 'cpu' | 'memory',
  usage: JobResourceSummaryUsage
): { used: string; limit?: string } {
  const formatter = resourceKey === 'cpu' ? formatCpuValue : formatMemoryValue
  const used = formatter(usage.used)
  if (!usage.limit) {
    return { used }
  }
  return {
    used,
    limit: formatter(usage.limit),
  }
}

function hasPositiveUsage(resourceKey: 'cpu' | 'memory' | 'accelerator', value: string): boolean {
  return (convertKResourceToResource(resourceKey, value) ?? 0) > 0
}

function getAcceleratorAmount(value: string): number {
  return convertKResourceToResource('accelerator', value) ?? 0
}

function getAcceleratorDisplayAmount(accelerator: JobResourceSummaryAccelerator): number {
  return getAcceleratorAmount(accelerator.running) + getAcceleratorAmount(accelerator.pending)
}

function getMeterWidthPercent(ratio: number): number {
  if (ratio <= 0) {
    return 0
  }

  return Math.max(Math.min(ratio * 100, 100), MIN_BAR_WIDTH_PERCENT)
}

function getClampedPercent(ratio: number): number {
  return Math.max(Math.min(ratio * 100, 100), 0)
}

function getUsageRatio(used: number, limit?: number, maxUnlimitedUsed?: number): number {
  if (limit !== undefined) {
    if (limit <= 0) {
      return used > 0 ? 1 : 0
    }
    return used / limit
  }
  if (maxUnlimitedUsed && maxUnlimitedUsed > 0) {
    return used / maxUnlimitedUsed
  }
  if (used > 0) {
    return used > 0 ? 1 : 0
  }
  return 0
}

function getLimitedSegmentPercents(
  running: number,
  pending: number,
  limit: number
): { runningPercent: number; pendingPercent: number } {
  if (limit <= 0) {
    if (running > 0) {
      return { runningPercent: 100, pendingPercent: 0 }
    }
    return { runningPercent: 0, pendingPercent: pending > 0 ? 100 : 0 }
  }

  const runningPercent = getClampedPercent(running / limit)
  const remainingPercent = 100 - runningPercent
  const pendingPercent = Math.min(getClampedPercent(pending / limit), remainingPercent)
  return { runningPercent, pendingPercent }
}

function getProportionalSegmentPercents(
  running: number,
  pending: number,
  totalPercent: number
): { runningPercent: number; pendingPercent: number } {
  const fillPercent = Math.max(Math.min(totalPercent, 100), 0)
  const runningAmount = Math.max(running, 0)
  const pendingAmount = Math.max(pending, 0)
  const totalAmount = runningAmount + pendingAmount
  if (fillPercent <= 0 || totalAmount <= 0) {
    return { runningPercent: 0, pendingPercent: 0 }
  }

  const runningPercent = (fillPercent * runningAmount) / totalAmount
  return { runningPercent, pendingPercent: fillPercent - runningPercent }
}

function getColorWithAlpha(color: string, alpha: number): string {
  const normalized = color.replace('#', '')
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) {
    return color
  }

  const channels = [0, 2, 4].map((start) => Number.parseInt(normalized.slice(start, start + 2), 16))
  return `rgba(${channels[0]}, ${channels[1]}, ${channels[2]}, ${alpha})`
}

function MeterBar({
  color,
  percent,
  pendingPercent,
  showTrack,
}: {
  color: string
  percent: number
  pendingPercent?: number
  showTrack: boolean
}) {
  const width = Math.max(Math.min(percent, 100), 0)
  const pendingWidth =
    pendingPercent === undefined ? 0 : Math.min(Math.max(pendingPercent, 0), 100 - width)
  const filledWidth = Math.min(width + pendingWidth, 100)
  const pendingColor = getColorWithAlpha(color, PENDING_BAR_ALPHA)

  if (!showTrack) {
    return (
      <div className="h-2 w-full">
        {filledWidth > 0 && (
          <div
            className="flex h-2 overflow-hidden rounded-full transition-all duration-300"
            style={{ width: `${filledWidth}%` }}
          >
            {width > 0 && (
              <div
                className="h-2"
                style={{
                  backgroundColor: color,
                  width: `${(width / filledWidth) * 100}%`,
                }}
              />
            )}
            {pendingWidth > 0 && (
              <div
                className="h-2"
                style={{
                  backgroundColor: pendingColor,
                  width: `${(pendingWidth / filledWidth) * 100}%`,
                }}
              />
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="bg-muted flex h-2 w-full overflow-hidden rounded-full">
      {width > 0 && (
        <div
          className={cn('h-2 transition-all duration-300', pendingWidth <= 0 && 'rounded-full')}
          style={{
            backgroundColor: color,
            width: `${width}%`,
          }}
        />
      )}
      {pendingWidth > 0 && (
        <div
          className={cn('h-2 transition-all duration-300', width <= 0 && 'rounded-full')}
          style={{
            backgroundColor: pendingColor,
            width: `${pendingWidth}%`,
          }}
        />
      )}
    </div>
  )
}

function ResourceUsageCard({
  icon: Icon,
  title,
  usedLabel,
  value,
  color,
  percent,
  pendingPercent,
  showTrack,
}: {
  icon: IconComponent
  title: ReactNode
  usedLabel?: string
  value: ReactNode
  color: string
  percent?: number
  pendingPercent?: number
  showTrack?: boolean
}) {
  return (
    <Card className="flex h-full flex-col items-stretch justify-between gap-1 rounded-md py-0 shadow-xs">
      <CardHeader className="space-y-0 px-3.5 py-1.5">
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2">
          <CardDescription className="flex min-w-0 items-center gap-2">
            <div className="bg-primary/10 flex size-7 shrink-0 items-center justify-center rounded-full">
              <Icon className="size-3.5" style={{ color }} />
            </div>
            <div className="min-w-0">{title}</div>
          </CardDescription>
          <div className="shrink-0 text-right leading-none">
            {usedLabel && <span className="text-muted-foreground text-xs">{usedLabel}</span>}
            <span className={cn('font-mono', usedLabel ? 'ml-1' : '')}>{value}</span>
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-3.5 pt-0 pb-2">
        {percent !== undefined ? (
          <MeterBar
            color={color}
            percent={percent}
            pendingPercent={pendingPercent}
            showTrack={showTrack ?? false}
          />
        ) : (
          <div className="h-2" />
        )}
      </CardContent>
    </Card>
  )
}

function UsageValue({ used, limit, color }: { used: string; limit?: string; color: string }) {
  return (
    <>
      <span className="text-lg font-bold" style={{ color }}>
        {used}
      </span>
      {limit && <span className="text-muted-foreground text-sm font-semibold">/{limit}</span>}
    </>
  )
}

function sortLimitedAccelerators(
  a: JobResourceSummaryAccelerator,
  b: JobResourceSummaryAccelerator
): number {
  const aUsed = getAcceleratorDisplayAmount(a)
  const bUsed = getAcceleratorDisplayAmount(b)
  const aLimit = getAcceleratorAmount(a.limit!)
  const bLimit = getAcceleratorAmount(b.limit!)
  const ratioDiff = getUsageRatio(bUsed, bLimit) - getUsageRatio(aUsed, aLimit)
  if (ratioDiff !== 0) {
    return ratioDiff
  }
  if (bUsed !== aUsed) {
    return bUsed - aUsed
  }
  return a.resource.localeCompare(b.resource)
}

function sortUnlimitedAccelerators(
  a: JobResourceSummaryAccelerator,
  b: JobResourceSummaryAccelerator
): number {
  const diff = getAcceleratorDisplayAmount(b) - getAcceleratorDisplayAmount(a)
  if (diff !== 0) {
    return diff
  }
  return a.resource.localeCompare(b.resource)
}

export default function JobResourceSummary() {
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const context = useAtomValue(atomUserContext)
  const queueName = context?.queue ?? ''
  const { data } = useQuery({
    queryKey: ['context', 'job-resource-summary', queueName],
    queryFn: () => apiContextJobResourceSummary(),
    select: (res) => res.data,
    enabled: queueName !== '',
    refetchInterval: REFETCH_INTERVAL,
  })

  const resourceCards = useMemo<ResourceUsageCardItem[]>(() => {
    if (!data) {
      return []
    }

    const cards: ResourceUsageCardItem[] = []

    const cpuUsage = formatPrimaryUsageParts('cpu', {
      used: data.cpu.running,
      limit: data.cpu.limit,
      running: data.cpu.running,
      pending: data.cpu.pending,
    })
    const cpuRunning = convertKResourceToResource('cpu', data.cpu.running) ?? 0
    const cpuPending = convertKResourceToResource('cpu', data.cpu.pending) ?? 0
    const cpuLimit = data.cpu.limit
      ? (convertKResourceToResource('cpu', data.cpu.limit) ?? 0)
      : undefined
    const cpuMeter =
      cpuLimit !== undefined
        ? getLimitedSegmentPercents(cpuRunning, cpuPending, cpuLimit)
        : getProportionalSegmentPercents(
            cpuRunning,
            cpuPending,
            getMeterWidthPercent(getUsageRatio(cpuRunning + cpuPending))
          )
    cards.push({
      key: 'cpu',
      title: (
        <span className="text-muted-foreground truncate text-sm font-semibold">
          {t('jobResourceSummary.cpu')}
        </span>
      ),
      icon: CpuIcon,
      meterColor: '#2563eb',
      usedLabel: t('quotaCard.used'),
      value: <UsageValue used={cpuUsage.used} limit={cpuUsage.limit} color="#2563eb" />,
      meterPercent: cpuMeter.runningPercent,
      pendingMeterPercent: cpuMeter?.pendingPercent,
      showTrack: cpuLimit !== undefined,
    })

    const memoryUsage = formatPrimaryUsageParts('memory', {
      used: data.memory.running,
      limit: data.memory.limit,
      running: data.memory.running,
      pending: data.memory.pending,
    })
    const memoryRunning = convertKResourceToResource('memory', data.memory.running) ?? 0
    const memoryPending = convertKResourceToResource('memory', data.memory.pending) ?? 0
    const memoryLimit = data.memory.limit
      ? (convertKResourceToResource('memory', data.memory.limit) ?? 0)
      : undefined
    const memoryMeter =
      memoryLimit !== undefined
        ? getLimitedSegmentPercents(memoryRunning, memoryPending, memoryLimit)
        : getProportionalSegmentPercents(
            memoryRunning,
            memoryPending,
            getMeterWidthPercent(getUsageRatio(memoryRunning + memoryPending))
          )
    cards.push({
      key: 'memory',
      title: (
        <span className="text-muted-foreground truncate text-sm font-semibold">
          {t('jobResourceSummary.memory')}
        </span>
      ),
      icon: MemoryStickIcon,
      meterColor: '#0f766e',
      usedLabel: t('quotaCard.used'),
      value: <UsageValue used={memoryUsage.used} limit={memoryUsage.limit} color="#0f766e" />,
      meterPercent: memoryMeter.runningPercent,
      pendingMeterPercent: memoryMeter?.pendingPercent,
      showTrack: memoryLimit !== undefined,
    })

    const accelerators = data?.accelerators ?? []
    const runningJobs = data.runningJobs
    const pendingJobs = data.pendingJobs
    const occupiedJobMeter = getProportionalSegmentPercents(
      runningJobs,
      pendingJobs,
      getMeterWidthPercent(getUsageRatio(runningJobs + pendingJobs))
    )
    cards.push({
      key: 'occupied-jobs',
      title: (
        <span className="text-muted-foreground truncate text-sm font-semibold">
          {t('jobResourceSummary.occupiedJobs')}
        </span>
      ),
      icon: ClipboardListIcon,
      meterColor: OCCUPIED_JOBS_COLOR,
      usedLabel: t('quotaCard.used'),
      value: (
        <span className="text-lg font-bold" style={{ color: OCCUPIED_JOBS_COLOR }}>
          {runningJobs}
        </span>
      ),
      meterPercent: occupiedJobMeter.runningPercent,
      pendingMeterPercent: occupiedJobMeter.pendingPercent,
      showTrack: true,
    })

    const limited = accelerators
      .filter(
        (item) =>
          item.limit &&
          (hasPositiveUsage('accelerator', item.running) ||
            hasPositiveUsage('accelerator', item.pending))
      )
      .sort(sortLimitedAccelerators)

    const unlimited = accelerators
      .filter((item) => !item.limit && getAcceleratorDisplayAmount(item) > 0)
      .sort(sortUnlimitedAccelerators)

    const maxUnlimitedUsed = unlimited.length > 0 ? getAcceleratorDisplayAmount(unlimited[0]) : 0

    for (const accelerator of [...limited, ...unlimited]) {
      const { fillColor } = getAcceleratorVendorStyle(
        parseAcceleratorString(accelerator.resource).vendor
      )
      const running = getAcceleratorAmount(accelerator.running)
      const pending = getAcceleratorAmount(accelerator.pending)
      const limit = accelerator.limit ? getAcceleratorAmount(accelerator.limit) : undefined
      const acceleratorMeter =
        limit !== undefined ? getLimitedSegmentPercents(running, pending, limit) : undefined
      const unlimitedAcceleratorMeter =
        limit === undefined
          ? getProportionalSegmentPercents(
              running,
              pending,
              getMeterWidthPercent(
                getUsageRatio(getAcceleratorDisplayAmount(accelerator), undefined, maxUnlimitedUsed)
              )
            )
          : undefined
      cards.push({
        key: accelerator.resource,
        title: (
          <div className="w-full max-w-[10.5rem]">
            <AcceleratorBadge
              acceleratorString={accelerator.resource}
              className="w-full max-w-[10.5rem]"
            />
          </div>
        ),
        icon: GpuIcon,
        meterColor: fillColor,
        usedLabel: t('quotaCard.used'),
        value: (
          <UsageValue
            used={`${running}`}
            limit={limit !== undefined ? `${limit}` : undefined}
            color={fillColor}
          />
        ),
        meterPercent:
          acceleratorMeter?.runningPercent ?? unlimitedAcceleratorMeter?.runningPercent ?? 0,
        pendingMeterPercent:
          acceleratorMeter?.pendingPercent ?? unlimitedAcceleratorMeter?.pendingPercent,
        showTrack: limit !== undefined,
      })
    }

    return cards
  }, [data, t])

  if (!data) {
    return null
  }

  const resourceSummaryUsage = t('jobResourceSummary.tooltip.usage.runningPending')

  return (
    <Card className="overflow-hidden rounded-md py-0 shadow-xs">
      <CardContent
        className={cn('px-4', {
          'space-y-2.5 py-4': !collapsed,
          'py-2.5': collapsed,
        })}
      >
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <GaugeIcon className="size-4" />
              {t('jobResourceSummary.title')}
              <TooltipProvider delayDuration={100}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      aria-label={resourceSummaryUsage}
                      className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 inline-flex size-4 items-center justify-center rounded-sm border-0 bg-transparent p-0 outline-none hover:cursor-help focus-visible:ring-[3px]"
                    >
                      <CircleHelpIcon className="size-4" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="max-w-none whitespace-nowrap">
                    {resourceSummaryUsage}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </CardTitle>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={() => setCollapsed((prev) => !prev)}
          >
            {collapsed ? (
              <ChevronDownIcon className="size-3.5" />
            ) : (
              <ChevronUpIcon className="size-3.5" />
            )}
            {collapsed ? t('jobResourceSummary.expand') : t('jobResourceSummary.collapse')}
          </Button>
        </div>

        {!collapsed && (
          <div className={cn('overflow-y-auto pr-1', SUMMARY_SECTION_MAX_HEIGHT_CLASS)}>
            <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
              {resourceCards.map((item) => (
                <ResourceUsageCard
                  key={item.key}
                  icon={item.icon}
                  title={item.title}
                  usedLabel={item.usedLabel}
                  value={item.value}
                  color={item.meterColor}
                  percent={item.meterPercent}
                  pendingPercent={item.pendingMeterPercent}
                  showTrack={item.showTrack}
                />
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
