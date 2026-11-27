import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { ArrowRight, Clock, FlaskConical, Gpu, type LucideIcon, UsersRound } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import AcceleratorBadge from '@/components/badge/accelerator-badge'
import { getJobPhaseLabel, jobPhases } from '@/components/badge/job-phase-badge'
import { PhaseBadge } from '@/components/badge/phase-badge'
import NivoPie from '@/components/chart/nivo-pie'
import { JobNameCell } from '@/components/label/job-name-label'

import type { INodeBriefInfo } from '@/services/api/cluster'
import type { QueueJob } from '@/services/api/job-queue'
import type { Resource } from '@/services/api/resource'
import type { JobDisplayPhaseValue } from '@/services/api/vcjob'
import { queryJobQueue } from '@/services/query/job-queue'

import { convertKResourceToResource } from '@/utils/resource'
import { atomUserContext, atomUserInfo } from '@/utils/store'

import { cn } from '@/lib/utils'

import { QueueObservedAt, useQueueClock } from '../queue/rows'
import { JobDateRangeSelect } from './job-date-range-select'

interface ActivityCount {
  id: string
  label: string
  value: number
}

interface AcceleratorTotals {
  total: number
  schedulable: number
  used: number
}

interface AcceleratorAllocation extends AcceleratorTotals {
  name: string
}

interface PanelData {
  loading: boolean
  error: boolean
}

const extraPhaseColors: Record<string, string> = {
  Inqueue: 'var(--highlight-indigo)',
  Starting: 'var(--highlight-sky)',
}

function OverviewPanel({
  title,
  icon: Icon,
  description,
  action,
  children,
}: {
  title: string
  icon: LucideIcon
  description?: string
  action?: ReactNode
  children: ReactNode
}) {
  const heading = (
    <h2
      className={cn(
        'flex items-center gap-2 text-base font-semibold',
        description && 'cursor-help'
      )}
      tabIndex={description ? 0 : undefined}
    >
      <Icon className="text-muted-foreground size-4 shrink-0" />
      {title}
    </h2>
  )
  return (
    <Card className="@container/overview-panel min-w-0 gap-4 py-5">
      <CardHeader className="flex items-center justify-between gap-3 px-5">
        {description ? (
          <Tooltip>
            <TooltipTrigger asChild>{heading}</TooltipTrigger>
            <TooltipContent className="max-w-80">{description}</TooltipContent>
          </Tooltip>
        ) : (
          heading
        )}
        {action && <CardAction className="self-center">{action}</CardAction>}
      </CardHeader>
      <CardContent className="px-5">{children}</CardContent>
    </Card>
  )
}

function PanelPlaceholder({ loading, error, empty }: PanelData & { empty: string }) {
  const { t } = useTranslation()
  if (loading)
    return (
      <div className="space-y-3 py-1" aria-busy="true" aria-label={t('overviewDashboard.loading')}>
        <Skeleton className="h-5 w-full" />
        <Skeleton className="h-5 w-full" />
        <Skeleton className="h-5 w-full" />
      </div>
    )

  return (
    <p role={error ? 'status' : undefined} className="text-muted-foreground py-6 text-sm">
      {error ? t('overviewDashboard.loadError') : empty}
    </p>
  )
}

export function JobActivityPanel({
  rows,
  days,
  onDaysChange,
  onOpen,
  loading,
  error,
}: PanelData & {
  rows: { id: string; phase: JobDisplayPhaseValue; value: number }[]
  days: number
  onDaysChange: (days: number) => void
  onOpen: (phase: string) => void
}) {
  const { t, i18n } = useTranslation()
  const chartRows = rows.map(({ id, phase, value }) => ({
    id,
    label: getJobPhaseLabel(phase).label,
    value,
  }))
  return (
    <OverviewPanel
      title={t('overviewDashboard.jobStatus')}
      icon={FlaskConical}
      description={t('overviewDashboard.jobStatusHint', { days })}
      action={<JobDateRangeSelect days={days} onChange={onDaysChange} size="sm" />}
    >
      {rows.length ? (
        <div className="grid min-w-0 items-center gap-4 @lg/overview-panel:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
          <div className="h-52 min-w-0">
            <NivoPie
              data={chartRows}
              margin={{ top: 12, right: 12, bottom: 12, left: 12 }}
              colors={({ id }) =>
                jobPhases.find((item) => item.value === id)?.color ??
                extraPhaseColors[String(id)] ??
                'var(--highlight-slate)'
              }
              enableArcLinkLabels={false}
              arcLabelsSkipAngle={28}
              arcLabelsTextColor="#ffffff"
              onClick={({ id }) => onOpen(String(id))}
            />
          </div>
          <ul className="divide-y">
            {rows.map(({ id, phase, value }) => (
              <li key={id} className="flex items-center justify-between gap-3 py-1.5 text-sm">
                <PhaseBadge
                  phase={phase}
                  getPhaseLabel={getJobPhaseLabel}
                  onClick={() => onOpen(id)}
                />
                <span className="font-medium tabular-nums">
                  {value.toLocaleString(i18n.language)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <PanelPlaceholder
          loading={loading}
          error={error}
          empty={t('overviewDashboard.noJobs', { days })}
        />
      )}
    </OverviewPanel>
  )
}

export function UserSubmissionsPanel({
  rows,
  days,
  loading,
  error,
}: PanelData & { rows: ActivityCount[]; days: number }) {
  const { t, i18n } = useTranslation()
  const top = rows.slice(0, 5)
  const other = rows.slice(5)
  const count = (value: number) => value.toLocaleString(i18n.language)
  return (
    <OverviewPanel
      title={t('overviewDashboard.userSubmissions')}
      icon={UsersRound}
      description={t('overviewDashboard.userSubmissionsHint', { days })}
    >
      {rows.length ? (
        <div className="grid min-w-0 items-center gap-4 @lg/overview-panel:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
          <div className="h-52 min-w-0">
            <NivoPie
              data={rows}
              margin={{ top: 12, right: 12, bottom: 12, left: 12 }}
              enableArcLinkLabels={false}
              enableArcLabels={false}
              padAngle={Math.min(0.35, 60 / rows.length)}
              cornerRadius={1}
            />
          </div>
          <div className="min-w-0">
            <div className="text-muted-foreground flex justify-between text-xs">
              <span>{t('overviewDashboard.user')}</span>
              <span>{t('overviewDashboard.submittedJobs')}</span>
            </div>
            <dl className="mt-1 divide-y">
              {top.map(({ id, label, value }) => (
                <div key={id} className="flex items-center justify-between gap-2 py-2 text-sm">
                  <dt className="truncate" title={label}>
                    {label}
                  </dt>
                  <dd className="font-medium tabular-nums">{count(value)}</dd>
                </div>
              ))}
              {other.length > 0 && (
                <div className="text-muted-foreground flex items-center justify-between gap-2 py-2 text-sm">
                  <dt>{t('overviewDashboard.otherUsers', { count: other.length })}</dt>
                  <dd className="tabular-nums">
                    {count(other.reduce((sum, row) => sum + row.value, 0))}
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>
      ) : (
        <PanelPlaceholder
          loading={loading}
          error={error}
          empty={t('overviewDashboard.noJobs', { days })}
        />
      )}
    </OverviewPanel>
  )
}

export function getAcceleratorAllocations(nodes: INodeBriefInfo[], resources: Resource[]) {
  return resources.flatMap(({ name }): AcceleratorAllocation[] => {
    let total = 0
    let schedulable = 0
    let used = 0
    for (const node of nodes) {
      const allocatable =
        convertKResourceToResource('accelerator', node.allocatable?.[name] ?? '0') ?? 0
      const capacity = convertKResourceToResource('accelerator', node.capacity?.[name] ?? '0') ?? 0
      total += Math.max(capacity, allocatable)
      if (!node.schedulableForAccount) continue
      const requested = convertKResourceToResource('accelerator', node.used?.[name] ?? '0') ?? 0
      schedulable += allocatable
      used += Math.min(allocatable, requested)
    }
    return total > 0 ? [{ name, total, schedulable, used }] : []
  })
}

export function AcceleratorPanel({
  rows,
  summary,
  onOpen,
  loading,
  error,
}: PanelData & { rows: AcceleratorAllocation[]; summary: AcceleratorTotals; onOpen: () => void }) {
  const { t, i18n } = useTranslation()
  const count = (value: number) => value.toLocaleString(i18n.language)
  return (
    <OverviewPanel
      title={t('overviewDashboard.accelerators')}
      icon={Gpu}
      description={t('overviewDashboard.acceleratorHint')}
      action={
        <Button variant="ghost" size="sm" onClick={onOpen}>
          {t('queueBoard.nodes')}
          <ArrowRight />
        </Button>
      }
    >
      {rows.length ? (
        <div className="space-y-4">
          <dl className="grid grid-cols-2 gap-3 border-y py-3 @sm/overview-panel:grid-cols-3 @xl/overview-panel:grid-cols-5">
            {[
              { key: 'totalCards', value: summary.total },
              { key: 'schedulableCards', value: summary.schedulable },
              { key: 'allocatedCards', value: summary.used },
              { key: 'freeCards', value: summary.schedulable - summary.used },
              { key: 'unavailableCards', value: summary.total - summary.schedulable },
            ].map(({ key, value }) => (
              <div key={key} className="min-w-0">
                <dt className="text-muted-foreground text-xs">{t('overviewDashboard.' + key)}</dt>
                <dd className="mt-1 text-lg font-semibold tabular-nums">{count(value)}</dd>
              </div>
            ))}
          </dl>
          <ul className="space-y-3">
            {rows.map(({ name, total, schedulable, used }) => (
              <li key={name} className="space-y-1.5">
                <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                  <AcceleratorBadge acceleratorString={name} />
                  <div className="flex items-center gap-2 tabular-nums">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-help font-medium" tabIndex={0}>
                          {count(used)} / {count(schedulable)}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-72">
                        {t('overviewDashboard.modelAllocationHint', { total, schedulable, used })}
                      </TooltipContent>
                    </Tooltip>
                    <span className="text-muted-foreground text-xs">
                      {t('overviewDashboard.unallocated', { count: schedulable - used })}
                    </span>
                    {total > schedulable && (
                      <span className="text-muted-foreground text-xs">
                        {t('overviewDashboard.unavailable', { count: total - schedulable })}
                      </span>
                    )}
                  </div>
                </div>
                <Progress
                  aria-hidden="true"
                  value={schedulable ? (used / schedulable) * 100 : 0}
                  className="bg-muted h-1.5"
                />
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <PanelPlaceholder
          loading={loading}
          error={error}
          empty={t('overviewDashboard.noAccelerators')}
        />
      )}
    </OverviewPanel>
  )
}

function OldestJobs({ rows, username }: { rows: QueueJob[]; username?: string }) {
  const { t } = useTranslation()
  const clock = useQueueClock()
  return (
    <div>
      <div className="text-muted-foreground flex justify-between gap-2 pb-1 text-xs">
        <span>{t('overviewDashboard.oldestJobs')}</span>
        <span>{t('queueBoard.wait')}</span>
      </div>
      <ul className="divide-y">
        {rows.map((job) => (
          <li key={job.jobName} className="flex items-center justify-between gap-2 py-2 text-sm">
            <div className="min-w-0">
              <JobNameCell
                jobInfo={{ name: job.name || job.jobName, jobName: job.jobName }}
                linkable={job.userInfo.username === username}
                nameClassName="max-w-full"
              />
            </div>
            <span className="text-muted-foreground shrink-0 tabular-nums">
              {clock.elapsed(job.createdAt)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function AccountQueuePanel({ onOpen }: { onOpen: () => void }) {
  const { t } = useTranslation()
  const user = useAtomValue(atomUserInfo)
  const account = useAtomValue(atomUserContext)
  const query = useQuery(queryJobQueue(user?.id, account?.queue))
  const rows = query.data?.jobs ?? []
  const admitted = rows.filter((job) => job.phase === 'Inqueue').length
  const own = rows.filter((job) => job.userInfo.username === user?.name).length
  return (
    <OverviewPanel
      title={t('queueBoard.accountQueue')}
      icon={Clock}
      description={t('overviewDashboard.accountQueueHint')}
      action={
        <Button variant="ghost" size="sm" onClick={onOpen}>
          {t('queueBoard.viewQueue')}
          <ArrowRight />
        </Button>
      }
    >
      {query.data ? (
        <div className="space-y-4">
          <dl className="grid grid-cols-3 divide-x border-y py-3">
            {[
              { label: t('queueBoard.pending'), value: rows.length - admitted },
              { label: t('queueBoard.admitted'), value: admitted },
              { label: t('overviewDashboard.myJobs'), value: own },
            ].map(({ label, value }) => (
              <div key={label} className="min-w-0 px-3 first:pl-0 sm:px-4">
                <dt className="text-muted-foreground text-xs">{label}</dt>
                <dd className="mt-1 text-xl font-semibold tabular-nums">{value}</dd>
              </div>
            ))}
          </dl>
          {rows.length > 0 ? (
            <QueueObservedAt.Provider
              value={Date.parse(query.data.observedAt) || query.dataUpdatedAt}
            >
              <OldestJobs rows={rows.slice(0, 3)} username={user?.name} />
            </QueueObservedAt.Provider>
          ) : (
            <p className="text-muted-foreground text-sm">{t('overviewDashboard.noQueue')}</p>
          )}
        </div>
      ) : (
        <PanelPlaceholder
          loading={query.isPending}
          error={query.isError}
          empty={t('overviewDashboard.noQueue')}
        />
      )}
    </OverviewPanel>
  )
}
