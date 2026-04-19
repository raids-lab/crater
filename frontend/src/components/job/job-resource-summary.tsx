import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import {
  ChevronDownIcon,
  ChevronUpIcon,
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
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import AcceleratorBadge from '@/components/badge/accelerator-badge'

import {
  type JobResourceSummaryAccelerator,
  type JobResourceSummaryScope,
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

type IconComponent = ComponentType<SVGProps<SVGSVGElement>>

interface ResourceUsageCardItem {
  key: string
  title: ReactNode
  icon: IconComponent
  meterColor: string
  value: ReactNode
  usedLabel?: string
  meterPercent?: number
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

function getMeterWidthPercent(ratio: number): number {
  if (ratio <= 0) {
    return 0
  }

  return Math.max(Math.min(ratio * 100, 100), MIN_BAR_WIDTH_PERCENT)
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

function getOccupiedJobsMeterPercent(occupiedJobs: number): number {
  return occupiedJobs > 0 ? 100 : 0
}

function MeterBar({
  color,
  percent,
  showTrack,
}: {
  color: string
  percent: number
  showTrack: boolean
}) {
  const width = Math.max(Math.min(percent, 100), 0)

  if (!showTrack) {
    return (
      <div className="h-2 w-full">
        <div
          className="h-2 rounded-full transition-all duration-300"
          style={{
            backgroundColor: color,
            width: `${width}%`,
          }}
        />
      </div>
    )
  }

  return (
    <div className="bg-muted h-2 w-full rounded-full">
      <div
        className="h-2 rounded-full transition-all duration-300"
        style={{
          backgroundColor: color,
          width: `${width}%`,
        }}
      />
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
  showTrack,
}: {
  icon: IconComponent
  title: ReactNode
  usedLabel?: string
  value: ReactNode
  color: string
  percent?: number
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
          <MeterBar color={color} percent={percent} showTrack={showTrack ?? false} />
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
  const aUsed = getAcceleratorAmount(a.used)
  const bUsed = getAcceleratorAmount(b.used)
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
  const diff = getAcceleratorAmount(b.used) - getAcceleratorAmount(a.used)
  if (diff !== 0) {
    return diff
  }
  return a.resource.localeCompare(b.resource)
}

export default function JobResourceSummary() {
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [scope, setScope] = useState<JobResourceSummaryScope>('personal')
  const context = useAtomValue(atomUserContext)
  const queueName = context?.queue ?? ''
  const canSelectAccountScope = queueName !== '' && queueName !== 'default'
  const activeScope: JobResourceSummaryScope = canSelectAccountScope ? scope : 'personal'
  const { data } = useQuery({
    queryKey: ['context', 'job-resource-summary', queueName, activeScope],
    queryFn: () => apiContextJobResourceSummary(activeScope),
    select: (res) => res.data,
    enabled: queueName !== '',
    refetchInterval: REFETCH_INTERVAL,
  })

  const resourceCards = useMemo<ResourceUsageCardItem[]>(() => {
    if (!data) {
      return []
    }

    const cards: ResourceUsageCardItem[] = []

    const cpuUsage = formatPrimaryUsageParts('cpu', data.cpu)
    const cpuUsed = convertKResourceToResource('cpu', data.cpu.used) ?? 0
    const cpuLimit = data.cpu.limit
      ? (convertKResourceToResource('cpu', data.cpu.limit) ?? 0)
      : undefined
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
      meterPercent: getMeterWidthPercent(getUsageRatio(cpuUsed, cpuLimit)),
      showTrack: cpuLimit !== undefined,
    })

    const memoryUsage = formatPrimaryUsageParts('memory', data.memory)
    const memoryUsed = convertKResourceToResource('memory', data.memory.used) ?? 0
    const memoryLimit = data.memory.limit
      ? (convertKResourceToResource('memory', data.memory.limit) ?? 0)
      : undefined
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
      meterPercent: getMeterWidthPercent(getUsageRatio(memoryUsed, memoryLimit)),
      showTrack: memoryLimit !== undefined,
    })

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
          {data.occupiedJobs ?? 0}
        </span>
      ),
      meterPercent: getOccupiedJobsMeterPercent(data.occupiedJobs ?? 0),
      showTrack: true,
    })

    const accelerators = data?.accelerators ?? []
    const limited = accelerators
      .filter((item) => item.limit && hasPositiveUsage('accelerator', item.used))
      .sort(sortLimitedAccelerators)

    const unlimited = accelerators
      .filter((item) => !item.limit && hasPositiveUsage('accelerator', item.used))
      .sort(sortUnlimitedAccelerators)

    const maxUnlimitedUsed = unlimited.length > 0 ? getAcceleratorAmount(unlimited[0].used) : 0

    for (const accelerator of [...limited, ...unlimited]) {
      const { fillColor } = getAcceleratorVendorStyle(
        parseAcceleratorString(accelerator.resource).vendor
      )
      const used = getAcceleratorAmount(accelerator.used)
      const limit = accelerator.limit ? getAcceleratorAmount(accelerator.limit) : undefined
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
            used={`${used}`}
            limit={limit !== undefined ? `${limit}` : undefined}
            color={fillColor}
          />
        ),
        meterPercent: getMeterWidthPercent(getUsageRatio(used, limit, maxUnlimitedUsed)),
        showTrack: limit !== undefined,
      })
    }

    return cards
  }, [data, t])

  if (!data) {
    return null
  }

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
            </CardTitle>
            {canSelectAccountScope && (
              <ToggleGroup
                type="single"
                variant="outline"
                size="sm"
                value={activeScope}
                onValueChange={(value) => {
                  if (value === 'personal' || value === 'account') {
                    setScope(value)
                  }
                }}
              >
                <ToggleGroupItem className="px-2 text-xs" value="personal">
                  {t('jobResourceSummary.scope.personal')}
                </ToggleGroupItem>
                <ToggleGroupItem className="px-2 text-xs" value="account">
                  {t('jobResourceSummary.scope.account')}
                </ToggleGroupItem>
              </ToggleGroup>
            )}
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
