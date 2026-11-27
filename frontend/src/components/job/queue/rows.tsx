import type { UseQueryResult } from '@tanstack/react-query'
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { CircleAlert, UserRound } from 'lucide-react'
import { createContext, useContext, useEffect, useId, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'

import JobPhaseLabel, { getJobPhaseLabel } from '@/components/badge/job-phase-badge'
import JobTypeLabel, { jobTypes } from '@/components/badge/job-type-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { JobNameCell } from '@/components/label/job-name-label'
import UserLabel from '@/components/label/user-label'
import { DataTable } from '@/components/query-table'
import { DataTablePagination } from '@/components/query-table/pagination'

import type { QueueJob, QueueSnapshot } from '@/services/api/job-queue'
import { JobDisplayPhase, JobPhase, PodGroupPhase } from '@/services/api/vcjob'

import usePaginationWithStorage from '@/hooks/use-pagination-with-storage'

import { cn } from '@/lib/utils'

export const QueueObservedAt = createContext<number>(0)

const stageColumns: ColumnDef<QueueJob>[] = []

function JobIdentity({
  job,
  admin = false,
  mine = false,
}: {
  job: QueueJob
  admin?: boolean
  mine?: boolean
}) {
  const knownType = jobTypes.some(({ value }) => value === job.jobType)
  return (
    <div className="min-w-0 space-y-1.5">
      <div className="flex min-w-0 items-center gap-2">
        <div className="min-w-0 font-medium">
          <JobNameCell
            jobInfo={{ name: job.name || job.jobName, jobName: job.jobName }}
            linkable={admin || mine}
            nameClassName="max-w-full"
          />
        </div>
        {job.jobType &&
          (knownType ? (
            <JobTypeLabel jobType={job.jobType} />
          ) : (
            <Badge variant="secondary">{job.jobType}</Badge>
          ))}
      </div>
      <ResourceBadges resources={job.resources} className="flex-row" />
    </div>
  )
}

export function useQueueClock() {
  const { i18n, t } = useTranslation()
  const now = useContext(QueueObservedAt)
  const stamp = useMemo(
    () =>
      new Intl.DateTimeFormat(i18n.language, {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }),
    [i18n.language]
  )
  const units = useMemo(
    () =>
      new Intl.NumberFormat(i18n.language, { style: 'unit', unit: 'minute', unitDisplay: 'short' }),
    [i18n.language]
  )
  const hours = useMemo(
    () =>
      new Intl.NumberFormat(i18n.language, { style: 'unit', unit: 'hour', unitDisplay: 'short' }),
    [i18n.language]
  )
  const days = useMemo(
    () =>
      new Intl.NumberFormat(i18n.language, { style: 'unit', unit: 'day', unitDisplay: 'short' }),
    [i18n.language]
  )
  return {
    submitted: (date: string) => stamp.format(new Date(date)),
    elapsed: (date: string) => {
      const mins = Math.max(0, Math.floor((now - Date.parse(date)) / 60000))
      if (mins < 1) return t('queueBoard.justNow')
      if (mins < 60) return units.format(mins)
      const hrs = Math.floor(mins / 60)
      if (hrs < 24)
        return [hours.format(hrs), mins % 60 ? units.format(mins % 60) : '']
          .filter(Boolean)
          .join(' ')
      return [days.format(Math.floor(hrs / 24)), hrs % 24 ? hours.format(hrs % 24) : '']
        .filter(Boolean)
        .join(' ')
    },
  }
}

export function StageLane({
  rows,
  query,
  storageKey,
  admitted,
  username,
}: {
  rows: QueueJob[]
  query: UseQueryResult<QueueSnapshot>
  storageKey: string
  admitted: boolean
  username?: string
}) {
  const { t } = useTranslation()
  const time = useQueueClock()
  const headingId = useId()
  const { icon } = getJobPhaseLabel(admitted ? JobDisplayPhase.Inqueue : JobPhase.Pending)
  const [pagination, setPagination] = usePaginationWithStorage(storageKey)
  const table = useReactTable({
    data: rows,
    columns: stageColumns,
    getRowId: (job) => job.jobName,
    state: { pagination },
    onPaginationChange: setPagination,
    autoResetPageIndex: false,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })
  const lastPageIndex = Math.max(0, table.getPageCount() - 1)
  useEffect(() => {
    if (pagination.pageIndex > lastPageIndex) {
      setPagination((state) => ({ ...state, pageIndex: lastPageIndex }))
    }
  }, [lastPageIndex, pagination.pageIndex, setPagination])

  return (
    <Card
      role="region"
      aria-labelledby={headingId}
      className="min-w-0 gap-0 overflow-hidden rounded-md py-0 shadow-xs"
    >
      <CardHeader className="flex items-center justify-between px-4 py-3">
        <h2 id={headingId} className="flex items-center gap-2 text-sm font-semibold">
          <span className="text-muted-foreground [&_svg]:size-4" aria-hidden>
            {icon}
          </span>
          {t(admitted ? 'queueBoard.admitted' : 'queueBoard.pending')}
        </h2>
        <Badge variant="secondary" className="tabular-nums">
          {rows.length}
        </Badge>
      </CardHeader>
      <CardContent className="border-t px-4">
        {rows.length > 0 ? (
          <ul className="divide-y">
            {table.getRowModel().rows.map(({ id, original: job }) => (
              <li key={id} className="space-y-1.5 py-3">
                <div className="flex items-start justify-between gap-3">
                  <JobIdentity job={job} mine={job.userInfo.username === username} />
                  <div className="text-muted-foreground max-w-28 shrink-0 truncate">
                    {job.userInfo.username === username ? (
                      <Badge variant="secondary" className="gap-1">
                        <UserRound className="size-3" />
                        {t('queueBoard.me')}
                      </Badge>
                    ) : (
                      <UserLabel info={job.userInfo} />
                    )}
                  </div>
                </div>
                {!admitted && (
                  <div
                    className={cn(
                      'flex items-center gap-1 text-xs',
                      job.reason ? 'text-highlight-orange' : 'text-muted-foreground'
                    )}
                  >
                    {job.reason && <CircleAlert className="size-3.5" />}
                    {t(
                      'queueBoard.' +
                        (job.reason || (job.phase === 'Init' ? 'created' : 'checking'))
                    )}
                  </div>
                )}
                <div className="text-muted-foreground flex flex-wrap justify-between gap-x-2 text-xs tabular-nums">
                  <time dateTime={job.createdAt}>
                    {t('queueBoard.submittedAt', { time: time.submitted(job.createdAt) })}
                  </time>
                  <span>{t('queueBoard.waited', { time: time.elapsed(job.createdAt) })}</span>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <div className="text-muted-foreground py-12 text-center text-sm">
            {t(admitted ? 'queueBoard.noAdmitted' : 'queueBoard.noPending')}
          </div>
        )}
      </CardContent>
      <CardFooter className="border-t px-4 py-3 [.border-t]:pt-3">
        <DataTablePagination
          table={table}
          refetch={() => void query.refetch()}
          updatedAt={query.dataUpdatedAt}
          className="sm:flex-wrap"
        />
      </CardFooter>
    </Card>
  )
}

export function JobTable({
  rows,
  query,
  storageKey,
  admin,
  username,
}: {
  rows: QueueJob[]
  query: UseQueryResult<QueueSnapshot>
  storageKey: string
  admin?: boolean
  username?: string
}) {
  const { t } = useTranslation()
  const time = useQueueClock()
  const columns: ColumnDef<QueueJob>[] = [
    {
      id: 'job',
      header: t('queueBoard.job'),
      cell: ({ row: { original: job } }) => (
        <div className="max-w-96 min-w-40 whitespace-normal">
          <JobIdentity job={job} admin={admin} mine={job.userInfo.username === username} />
        </div>
      ),
    },
    {
      accessorKey: 'account',
      header: t('queueBoard.account'),
    },
    {
      id: 'owner',
      header: t('queueBoard.owner'),
      cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
    },
    {
      id: 'phase',
      header: t('jobs.headers.status'),
      cell: ({ row: { original: job } }) => (
        <>
          <JobPhaseLabel
            jobPhase={job.phase === 'Init' ? JobPhase.Init : JobPhase.Pending}
            podGroupPhase={job.phase === 'Inqueue' ? PodGroupPhase.Inqueue : undefined}
          />
          {job.reason && (
            <div className="text-highlight-orange mt-1 text-xs whitespace-normal">
              {t('queueBoard.' + job.reason)}
            </div>
          )}
        </>
      ),
    },
    {
      id: 'submitted',
      header: t('queueBoard.submitted'),
      cell: ({ row }) => (
        <time className="text-muted-foreground tabular-nums" dateTime={row.original.createdAt}>
          {time.submitted(row.original.createdAt)}
        </time>
      ),
    },
    {
      id: 'wait',
      header: () => <div className="text-right">{t('queueBoard.wait')}</div>,
      cell: ({ row }) => (
        <div className="pr-2 text-right tabular-nums">{time.elapsed(row.original.createdAt)}</div>
      ),
    },
  ]

  const { error, isError, isLoading, dataUpdatedAt, refetch } = query
  return (
    <DataTable
      storageKey={storageKey}
      query={{ data: rows, error, isError, isLoading, dataUpdatedAt, refetch }}
      columns={columns}
      initialColumnVisibility={{ account: !!admin }}
      withI18n
    />
  )
}
