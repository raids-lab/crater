import { type UseQueryResult, useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { ArrowDown, ArrowUpNarrowWide, CircleAlert, Clock3, Columns2 } from 'lucide-react'
import { type ReactNode, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import type { QueueJob, QueueSnapshot } from '@/services/api/job-queue'
import { queryJobQueue } from '@/services/query/job-queue'

import { atomUserContext, atomUserInfo } from '@/utils/store'

import { JobTable, QueueObservedAt, StageLane, useQueueClock } from './rows'

const quotaReason = 'QueueQuotaExceeded'

function CheckFilter({
  checked,
  onChange,
  children,
}: {
  checked: boolean
  onChange: (value: boolean) => void
  children: ReactNode
}) {
  return (
    <label className="inline-flex cursor-pointer items-center gap-2 text-sm">
      <Checkbox checked={checked} onCheckedChange={(v) => onChange(v === true)} />
      {children}
    </label>
  )
}

function accountTotals(jobs: QueueJob[]) {
  const grouped = new Map<
    number,
    { id: number; name: string; pending: number; admitted: number; quota: number; oldest: string }
  >()
  for (const job of jobs) {
    const row = grouped.get(job.accountId) ?? {
      id: job.accountId,
      name: job.account,
      pending: 0,
      admitted: 0,
      quota: 0,
      oldest: job.createdAt,
    }
    if (job.phase === 'Inqueue') row.admitted++
    else row.pending++
    if (job.reason === quotaReason) row.quota++
    // The API returns jobs oldest first, including across accounts.
    grouped.set(row.id, row)
  }
  return Array.from(grouped.values())
}

function AdminQueue({
  jobs,
  reasonsAvailable,
  query,
}: {
  jobs: QueueJob[]
  reasonsAvailable: boolean
  query: UseQueryResult<QueueSnapshot>
}) {
  const { t } = useTranslation()
  const time = useQueueClock()
  const [account, setAccount] = useState<number>()
  const [quotaOnly, setQuotaOnly] = useState(false)
  const groups = useMemo(() => accountTotals(jobs), [jobs])
  const group = groups.find((g) => g.id === account)
  const rows = jobs.filter(
    (j) => (!group || j.accountId === group.id) && (!quotaOnly || j.reason === quotaReason)
  )
  const tableKey = `admin_job_queue_${group?.id ?? 'all'}_${quotaOnly ? 'quota' : 'all'}`
  return (
    <div className="space-y-6">
      <section>
        <h2 className="mb-2 text-base font-semibold">{t('queueBoard.accountSummary')}</h2>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('queueBoard.account')}</TableHead>
              {['pending', 'admitted', 'quotaShort'].map((k) => (
                <TableHead key={k} className="text-right">
                  {t('queueBoard.' + k)}
                </TableHead>
              ))}
              <TableHead className="text-right" aria-sort="descending">
                {t('queueBoard.longest')} <ArrowDown className="inline size-3" />
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {groups.map((g) => (
              <TableRow key={g.id} data-state={group?.id === g.id ? 'selected' : undefined}>
                <TableCell>
                  <button
                    type="button"
                    className="text-primary font-medium hover:underline"
                    aria-pressed={group?.id === g.id}
                    onClick={() => setAccount(g.id)}
                  >
                    {g.name || g.id}
                  </button>
                </TableCell>
                {[
                  g.pending,
                  g.admitted,
                  reasonsAvailable ? g.quota : '—',
                  time.elapsed(g.oldest),
                ].map((v, i) => (
                  <TableCell key={i} className="text-right tabular-nums">
                    {v}
                  </TableCell>
                ))}
              </TableRow>
            ))}
            {!groups.length && (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground h-20 text-center">
                  {t('queueBoard.emptyAccounts')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </section>
      <section>
        <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
          <h2 className="flex items-center gap-2 text-base font-semibold">
            {t('queueBoard.details')}
            <Badge variant="secondary">{rows.length}</Badge>
          </h2>
          <div className="flex flex-wrap items-center gap-4">
            <Select
              value={group ? String(group.id) : 'all'}
              onValueChange={(value) => setAccount(value === 'all' ? undefined : Number(value))}
            >
              <SelectTrigger size="sm" aria-label={t('queueBoard.account')}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('queueBoard.allAccounts')}</SelectItem>
                {groups.map((g) => (
                  <SelectItem key={g.id} value={String(g.id)}>
                    {g.name || g.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <CheckFilter checked={quotaOnly} onChange={setQuotaOnly}>
              {t('queueBoard.quotaOnly')}
            </CheckFilter>
          </div>
        </div>
        <JobTable key={tableKey} storageKey={tableKey} rows={rows} query={query} admin />
      </section>
    </div>
  )
}

export function QueueBoard({
  admin = false,
  mine = false,
  onMineChange,
}: {
  admin?: boolean
  mine?: boolean
  onMineChange?: (value: boolean) => void
}) {
  const { t } = useTranslation()
  const user = useAtomValue(atomUserInfo)
  const account = useAtomValue(atomUserContext)
  const [mode, setMode] = useState('stage')
  const query = useQuery(queryJobQueue(user?.id, account?.queue, admin, mine))
  const data = query.data
  if (!data && query.isError)
    return (
      <div role="alert" className="flex items-center gap-3 py-8 text-sm">
        <CircleAlert className="size-4" />
        {t('queueBoard.loadError')}
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          {t('queueBoard.retry')}
        </Button>
      </div>
    )
  if (!data)
    return (
      <div
        aria-busy="true"
        aria-label={t('queueBoard.loading')}
        className="grid grid-cols-1 gap-4 md:grid-cols-2"
      >
        <Skeleton className="h-36" />
        <Skeleton className="h-36" />
      </div>
    )
  const rows = data.jobs
  const tableKey = `portal_job_queue_${account?.queue}_${mine ? 'mine' : 'all'}`
  return (
    <QueueObservedAt.Provider value={Date.parse(data.observedAt) || query.dataUpdatedAt}>
      <div className="space-y-4">
        {(!data.reasonsAvailable || query.isError) && (
          <p role="status" className="text-muted-foreground text-sm">
            {t(query.isError ? 'queueBoard.refreshError' : 'queueBoard.reasonError')}
          </p>
        )}
        {admin ? (
          <AdminQueue jobs={rows} reasonsAvailable={data.reasonsAvailable} query={query} />
        ) : (
          <>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <CheckFilter checked={mine} onChange={(v) => onMineChange?.(v)}>
                {t('queueBoard.onlyMine')}
              </CheckFilter>
              <div className="flex flex-wrap items-center gap-4">
                <span className="text-muted-foreground flex items-center gap-1.5 text-xs">
                  <ArrowUpNarrowWide className="size-4" />
                  {t('queueBoard.oldest')}
                </span>
                <ToggleGroup
                  type="single"
                  variant="outline"
                  size="sm"
                  aria-label={t('queueBoard.display')}
                  value={mode}
                  onValueChange={(v) => {
                    if (v) setMode(v)
                  }}
                >
                  <ToggleGroupItem value="stage" className="gap-1.5 px-2.5">
                    <Columns2 />
                    {t('queueBoard.byStage')}
                  </ToggleGroupItem>
                  <ToggleGroupItem value="time" className="gap-1.5 px-2.5">
                    <Clock3 />
                    {t('queueBoard.byTime')}
                  </ToggleGroupItem>
                </ToggleGroup>
              </div>
            </div>
            {mode === 'time' ? (
              <JobTable
                key={tableKey}
                storageKey={tableKey}
                rows={rows}
                query={query}
                username={user?.name}
              />
            ) : (
              <div key={tableKey} className="grid items-start gap-4 @3xl/main:grid-cols-2">
                <StageLane
                  rows={rows.filter((j) => j.phase !== 'Inqueue')}
                  storageKey={tableKey + '_pending'}
                  query={query}
                  admitted={false}
                  username={user?.name}
                />
                <StageLane
                  rows={rows.filter((j) => j.phase === 'Inqueue')}
                  storageKey={tableKey + '_admitted'}
                  query={query}
                  admitted
                  username={user?.name}
                />
              </div>
            )}
          </>
        )}
      </div>
    </QueueObservedAt.Provider>
  )
}
