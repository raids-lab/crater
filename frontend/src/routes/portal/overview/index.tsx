/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { type Locale, enUS, ja, ko, zhCN } from 'date-fns/locale'
import { useAtomValue } from 'jotai'
import { ClockIcon, FlaskConicalIcon, GpuIcon, UsersRoundIcon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent } from '@/components/ui/tabs'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import DocsButton from '@/components/button/docs-button'
import { BillingPointsBadge } from '@/components/custom/billing-points-badge'
import { BillingSummaryCards } from '@/components/custom/billing-summary-cards'
import { TimeDistance } from '@/components/custom/time-distance'
import ListedNewJobButton from '@/components/job/new-job-button'
import { getHeader } from '@/components/job/overview/admin-jobs'
import { JobDateRangeSelect } from '@/components/job/overview/job-date-range-select'
import {
  AcceleratorPanel,
  AccountQueuePanel,
  JobActivityPanel,
  UserSubmissionsPanel,
  getAcceleratorAllocations,
} from '@/components/job/overview/overview-panels'
import { QueueBoard } from '@/components/job/queue/board'
import { JobPageTabs } from '@/components/job/queue/navigation'
import { getJobOverviewPhases, getRemoteJobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import { SectionCards } from '@/components/metrics/section-cards'
import { useAccountNameLookup } from '@/components/node/getaccountnickname'
import { getNodeColumns, nodesToolbarConfig } from '@/components/node/node-list'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { RemoteDataTable } from '@/components/query-table/remote'
import {
  type RemoteTableParams,
  buildFacetQueryKey,
  buildRemoteQueryKey,
} from '@/components/query-table/remote-state'
import { UserBanAlert } from '@/components/user/user-ban-alert'

import { apiContextBillingSummary } from '@/services/api/context'
import { apiGetBillingStatus } from '@/services/api/system-config'
import {
  IJobInfo,
  JobDisplayPhase,
  JobPhase,
  JobType,
  apiJobActiveCounts,
  apiJobAllFacets,
  apiJobAllList,
  getDisplayJobPhase,
} from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import useJobTableState from '@/hooks/use-job-table-state'

import { isBillingVisibleForUser } from '@/utils/billing-visibility'
import { getUserPseudonym } from '@/utils/pseudonym'
import { atomUserContext, atomUserInfo, globalHideUsername, globalJobUrl } from '@/utils/store'

import { REFETCH_INTERVAL } from '@/lib/constants'

type OverviewSearch = { tab?: 'jobs' | 'queue' | 'nodes'; mine?: boolean; submitted?: boolean }

export const Route = createFileRoute('/portal/overview/')({
  validateSearch: (raw: Record<string, unknown>): OverviewSearch => ({
    tab: raw.tab === 'jobs' || raw.tab === 'nodes' || raw.tab === 'queue' ? raw.tab : undefined,
    mine: raw.tab === 'queue' && (raw.mine === true || raw.mine === 'true') ? true : undefined,
    submitted: raw.tab === 'jobs' && raw.submitted === true ? true : undefined,
  }),
  component: Overview,
})

type JobTableRow = IJobInfo
const waitingPhases = [JobPhase.Pending, JobDisplayPhase.Inqueue, JobDisplayPhase.Starting]
const activePhases = [JobPhase.Running, ...waitingPhases]

function Overview() {
  const { i18n, t } = useTranslation()
  const [overviewDays, setOverviewDays] = useState(7)
  const account = useAtomValue(atomUserContext)
  const username = useAtomValue(atomUserInfo)?.name
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const volcano = useAtomValue(globalJobUrl) === 'vcjobs'
  const tab = search.tab === 'queue' && !volcano ? 'overview' : (search.tab ?? 'overview')
  const setTab = (next: string) =>
    void navigate({
      search: { tab: next === 'overview' ? undefined : (next as OverviewSearch['tab']) },
    })
  const nodeQuery = useQuery({
    ...queryNodes(true, account?.queue),
    enabled: !!account?.queue && (tab === 'overview' || tab === 'nodes'),
    staleTime: REFETCH_INTERVAL,
    refetchInterval: tab === 'overview' ? 3 * REFETCH_INTERVAL : false,
  })
  const { getNicknameByName } = useAccountNameLookup()
  const { data: billingStatus } = useQuery({
    queryKey: ['system-config', 'billing-status'],
    queryFn: () => apiGetBillingStatus().then((res) => res.data),
  })
  const billingVisible = isBillingVisibleForUser(billingStatus)
  const tableState = useJobTableState('overview_joblist', {
    sorting: [{ id: 'createdAt', desc: true }],
    columnFilters: [
      { id: 'days', value: ['7'] },
      { id: 'status', value: activePhases },
    ],
  })
  const tableDays = Number(tableState.params.filters.days?.[0] ?? 7)
  const tableParams: RemoteTableParams = search.submitted
    ? tableState.params
    : {
        ...tableState.params,
        filters: { ...tableState.params.filters, include_running: ['true'] },
      }
  const overviewSummaryParams: RemoteTableParams = {
    page: 1,
    page_size: 1,
    filters: { days: [String(overviewDays)], summary: ['submissions'] },
  }

  // 获取当前语言对应的 date-fns locale
  const getDateLocale = useCallback((): Locale => {
    switch (i18n.language) {
      case 'en':
        return enUS
      case 'ja':
        return ja
      case 'ko':
        return ko
      default:
        return zhCN
    }
  }, [i18n.language])

  const jobColumns = useMemo<ColumnDef<JobTableRow>[]>(
    () => [
      {
        accessorKey: 'jobType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('jobType')} />
        ),
        cell: ({ row }) => <JobTypeLabel jobType={row.getValue<JobType>('jobType')} />,
      },
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('jobName')} />
        ),
        cell: ({ row }) => (
          <JobNameCell
            jobInfo={row.original}
            linkable={!!username && row.original.userInfo?.username === username}
          />
        ),
      },
      {
        accessorKey: 'queue',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('queue')} />
        ),
        cell: ({ row }) => <div>{row.getValue('queue')}</div>,
      },
      {
        accessorKey: 'owner',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('owner')} />
        ),
        cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
      },
      {
        accessorKey: 'nodes',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('nodes')} />
        ),
        cell: ({ row }) => {
          const nodes = row.getValue<string[]>('nodes')
          return <NodeBadges nodes={nodes} />
        },
      },
      {
        accessorKey: 'resources',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.getValue<Record<string, string> | undefined>('resources')
          return <ResourceBadges resources={resources} />
        },
        sortingFn: (rowA, rowB) => {
          const resourcesA = rowA.original.resources
          const resourcesB = rowB.original.resources
          if (resourcesA && resourcesB) {
            // compare the number of GPUs, key with nvidia.com/ prefix
            const gpuA = Object.keys(resourcesA).filter((key) =>
              key.startsWith('nvidia.com')
            ).length
            const gpuB = Object.keys(resourcesB).filter((key) =>
              key.startsWith('nvidia.com')
            ).length
            return gpuA - gpuB
          }
          return 0
        },
      },
      ...(billingVisible
        ? [
            {
              accessorKey: 'billedPointsTotal',
              header: ({ column }) => <DataTableColumnHeader column={column} title="累计点数" />,
              cell: ({ row }) => <BillingPointsBadge value={row.original.billedPointsTotal ?? 0} />,
            } as ColumnDef<JobTableRow>,
          ]
        : []),
      {
        accessorFn: (row) => getDisplayJobPhase(row.status),
        id: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('status')} />
        ),
        cell: ({ row }) => {
          return (
            <JobPhaseLabel
              jobPhase={row.getValue<JobPhase>('status')}
              podGroupPhase={row.original.podGroupPhase}
            />
          )
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('createdAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('createdAt')}></TimeDistance>
        },
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'startedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('startedAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('startedAt')}></TimeDistance>
        },
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'completedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('completedAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('completedAt')}></TimeDistance>
        },
        sortingFn: 'datetime',
      },
    ],
    [billingVisible, username]
  )

  const jobQuery = useQuery({
    queryKey: buildRemoteQueryKey('overview-jobs', tableParams),
    queryFn: async ({ signal }) => (await apiJobAllList(tableParams, signal)).data,
    enabled: tab === 'jobs',
    placeholderData: keepPreviousData,
    refetchInterval: REFETCH_INTERVAL,
  })
  const jobFacetsQuery = useQuery({
    queryKey: buildFacetQueryKey('overview-jobs', tableParams),
    queryFn: async ({ signal }) => (await apiJobAllFacets(tableParams, signal)).data,
    enabled: tab === 'jobs',
    refetchInterval: REFETCH_INTERVAL,
  })
  const jobSummaryFacetsQuery = useQuery({
    queryKey: buildFacetQueryKey('overview-job-summary', overviewSummaryParams),
    queryFn: async ({ signal }) => (await apiJobAllFacets(overviewSummaryParams, signal)).data,
    enabled: tab === 'overview',
    refetchInterval: overviewDays > 14 ? 3 * REFETCH_INTERVAL : REFETCH_INTERVAL,
  })
  const activeJobCountsQuery = useQuery({
    queryKey: ['overview', 'active-job-counts'],
    queryFn: async ({ signal }) => (await apiJobActiveCounts(signal)).data,
    enabled: tab === 'overview',
    refetchInterval: REFETCH_INTERVAL,
  })
  const toolbarConfig = (() => {
    const config = getRemoteJobToolbarConfig(jobFacetsQuery.data)
    return {
      ...config,
      getHeader,
      filterOptions: config.filterOptions.map((filter) =>
        filter.key === 'status' ? { ...filter, defaultValues: activePhases } : filter
      ),
    }
  })()

  const resourcesQuery = useQuery({
    ...queryResources(true, (resource) => resource.type === 'gpu'),
    enabled: tab === 'overview' || tab === 'nodes',
  })
  const billingSummaryQuery = useQuery({
    queryKey: ['context', 'billing-summary', 'overview'],
    queryFn: () => apiContextBillingSummary().then((res) => res.data),
    enabled: billingVisible,
  })

  const jobStatus = getJobOverviewPhases(jobSummaryFacetsQuery.data)
  const activeJobCounts = activeJobCountsQuery.data

  const hideUsername = useAtomValue(globalHideUsername)
  const userSubmissions = (jobSummaryFacetsQuery.data?.facets.submitter ?? []).map(
    ({ value, label, count }) => ({
      id: value,
      label: hideUsername || !label ? getUserPseudonym(value) : label,
      value: count,
    })
  )

  const hasAccountScheduling = nodeQuery.data?.every(
    (node) => node.schedulableForAccount !== undefined
  )
  const accelerators = useMemo(
    () =>
      hasAccountScheduling
        ? getAcceleratorAllocations(nodeQuery.data ?? [], resourcesQuery.data ?? [])
        : [],
    [hasAccountScheduling, nodeQuery.data, resourcesQuery.data]
  )
  const acceleratorSummary = accelerators.reduce(
    (sum, row) => ({
      total: sum.total + row.total,
      schedulable: sum.schedulable + row.schedulable,
      used: sum.used + row.used,
    }),
    { total: 0, schedulable: 0, used: 0 }
  )
  const allocation = acceleratorSummary.schedulable
    ? Math.round((acceleratorSummary.used / acceleratorSummary.schedulable) * 100)
    : undefined
  const jobPanelState = {
    loading: jobSummaryFacetsQuery.isPending,
    error: jobSummaryFacetsQuery.isError,
  }

  const openClusterJobs = (phases: string[], days = overviewDays) => {
    tableState.setSearch('')
    tableState.setColumnFilters([
      { id: 'days', value: [String(days)] },
      { id: 'status', value: phases },
    ])
    void navigate({ search: { tab: 'jobs', submitted: days > 0 || undefined } })
  }

  return (
    <Tabs value={tab} onValueChange={setTab} className="min-w-0 gap-4">
      <UserBanAlert />
      <PageTitle title={t('queueBoard.platform')}>
        <div className="flex flex-wrap items-center justify-end gap-2">
          {billingVisible ? (
            <BillingSummaryCards
              summary={billingSummaryQuery.data}
              emphasis="inline"
              compact
              className="shrink-0"
            />
          ) : null}
          <DocsButton title="平台文档" url="" />
          <ListedNewJobButton mode="all" />
        </div>
      </PageTitle>
      <JobPageTabs
        label={t('queueBoard.platform')}
        items={[
          { value: 'overview', label: t('queueBoard.overview') },
          { value: 'jobs', label: t('queueBoard.clusterJobs') },
          ...(volcano ? [{ value: 'queue', label: t('queueBoard.accountQueue') }] : []),
          { value: 'nodes', label: t('queueBoard.nodes') },
        ]}
      />
      <TabsContent value="overview" className="grid min-w-0 gap-4">
        <SectionCards
          items={[
            {
              title: t('overviewDashboard.running'),
              value: activeJobCounts?.running ?? '—',
              className: 'text-highlight-blue',
              description: t('overviewDashboard.runningHint'),
              icon: FlaskConicalIcon,
            },
            {
              title: t('jobs.statuses.waiting.label'),
              value: activeJobCounts ? (
                <button
                  type="button"
                  className="hover:underline"
                  aria-label={t('queueBoard.currentWaiting')}
                  onClick={() => openClusterJobs(waitingPhases, -1)}
                >
                  {activeJobCounts.waiting}
                </button>
              ) : (
                '—'
              ),
              className: 'text-highlight-purple',
              description: t('overviewDashboard.waitingHint'),
              icon: ClockIcon,
            },
            {
              title: t('overviewDashboard.activeUsers'),
              value: activeJobCounts?.activeUsers ?? '—',
              className: 'text-highlight-emerald',
              description: t('overviewDashboard.currentActiveUsersHint'),
              icon: UsersRoundIcon,
            },
            {
              title: t('overviewDashboard.allocation'),
              value: allocation === undefined ? '—' : `${allocation}%`,
              className: 'text-highlight-orange',
              description: t('overviewDashboard.allocationHint'),
              icon: GpuIcon,
            },
          ]}
        />
        <div className="grid min-w-0 gap-4 @4xl/main:grid-cols-2">
          <JobActivityPanel
            rows={jobStatus}
            days={overviewDays}
            onDaysChange={setOverviewDays}
            onOpen={(phase) => openClusterJobs([phase])}
            {...jobPanelState}
          />
          <UserSubmissionsPanel rows={userSubmissions} days={overviewDays} {...jobPanelState} />
          <AcceleratorPanel
            rows={accelerators}
            summary={acceleratorSummary}
            loading={nodeQuery.isPending || resourcesQuery.isPending}
            error={nodeQuery.isError || resourcesQuery.isError || hasAccountScheduling === false}
            onOpen={() => setTab('nodes')}
          />
          {volcano && <AccountQueuePanel onOpen={() => setTab('queue')} />}
        </div>
      </TabsContent>
      <TabsContent value="jobs">
        <RemoteDataTable
          info={{
            title: '作业信息',
            description:
              tableDays === -1
                ? t('overviewDashboard.allClusterJobs')
                : t(
                    search.submitted
                      ? 'overviewDashboard.clusterJobsRange'
                      : 'overviewDashboard.clusterJobsWithRunning',
                    { days: tableDays }
                  ),
          }}
          query={jobQuery}
          state={tableState}
          columns={jobColumns}
          getRowId={(row) => row.jobName}
          toolbarConfig={toolbarConfig}
        >
          <JobDateRangeSelect
            days={tableDays}
            allowAll
            onChange={(value) =>
              tableState.setColumnFilters((current) => [
                ...current.filter(({ id }) => id !== 'days'),
                { id: 'days', value: [String(value)] },
              ])
            }
          />
        </RemoteDataTable>
      </TabsContent>
      {volcano && (
        <TabsContent value="queue">
          <QueueBoard
            mine={search.mine}
            onMineChange={(mine) =>
              void navigate({ search: { tab: 'queue', mine: mine || undefined } })
            }
          />
        </TabsContent>
      )}
      <TabsContent value="nodes">
        <DataTable
          info={{
            title: '节点信息',
            description: '集群节点维度的资源分配情况',
          }}
          storageKey="overview_nodelist"
          query={nodeQuery}
          columns={getNodeColumns(
            getNicknameByName,
            resourcesQuery.data?.map((r) => r.name),
            false,
            getDateLocale()
          )}
          toolbarConfig={nodesToolbarConfig}
        />
      </TabsContent>
    </Tabs>
  )
}
