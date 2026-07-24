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
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import JobPhaseLabel, { getJobPhaseLabel, jobPhases } from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import ScheduleTypeLabel from '@/components/badge/schedule-type-badge'
import DocsButton from '@/components/button/docs-button'
import NivoPie from '@/components/chart/nivo-pie'
import PieCard from '@/components/chart/pie-card'
import { BillingPointsBadge } from '@/components/custom/billing-points-badge'
import { BillingSummaryCards } from '@/components/custom/billing-summary-cards'
import { TimeDistance } from '@/components/custom/time-distance'
import ListedNewJobButton from '@/components/job/new-job-button'
import { getHeader } from '@/components/job/overview/admin-jobs'
import { getRemoteJobToolbarConfig } from '@/components/job/statuses'
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
  JobPhase,
  JobType,
  ScheduleType,
  apiJobAllFacets,
  apiJobAllList,
  getDisplayJobPhase,
} from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import useRemoteTableState from '@/hooks/use-remote-table-state'

import { isBillingVisibleForUser } from '@/utils/billing-visibility'
import { getUserPseudonym } from '@/utils/pseudonym'
import { atomUserInfo, globalHideUsername } from '@/utils/store'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const Route = createFileRoute('/portal/overview/')({
  component: Overview,
})

type JobTableRow = IJobInfo

const overviewSummaryParams: RemoteTableParams = {
  page: 1,
  page_size: 1,
  filters: { days: ['7'] },
}

function Overview() {
  const { i18n, t } = useTranslation()
  const userInfo = useAtomValue(atomUserInfo)
  const nodeQuery = useQuery(queryNodes(true))
  const { getNicknameByName } = useAccountNameLookup()
  const { data: billingStatus } = useQuery({
    queryKey: ['system-config', 'billing-status'],
    queryFn: () => apiGetBillingStatus().then((res) => res.data),
  })
  const billingVisible = isBillingVisibleForUser(billingStatus)
  const tableState = useRemoteTableState('overview_joblist', {
    sorting: [{ id: 'createdAt', desc: true }],
    columnFilters: [
      { id: 'days', value: ['7'] },
      { id: 'status', value: ['Running', 'Pending', 'Prequeue'] },
    ],
  })

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
        accessorFn: (row) => String(row.scheduleType ?? ScheduleType.Normal),
        id: 'scheduleType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('scheduleType')} />
        ),
        cell: ({ row }) => <ScheduleTypeLabel scheduleType={row.original.scheduleType} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
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
          return <JobPhaseLabel jobPhase={row.getValue<JobPhase>('status')} />
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
    [billingVisible]
  )

  const jobQuery = useQuery({
    queryKey: buildRemoteQueryKey('overview-jobs', tableState.params),
    queryFn: async ({ signal }) => (await apiJobAllList(tableState.params, signal)).data,
    placeholderData: keepPreviousData,
    refetchInterval: REFETCH_INTERVAL,
  })
  const jobFacetsQuery = useQuery({
    queryKey: buildFacetQueryKey('overview-jobs', tableState.params),
    queryFn: async ({ signal }) => (await apiJobAllFacets(tableState.params, signal)).data,
    refetchInterval: REFETCH_INTERVAL,
  })
  const jobSummaryFacetsQuery = useQuery({
    queryKey: buildFacetQueryKey('overview-job-summary', overviewSummaryParams),
    queryFn: async ({ signal }) => (await apiJobAllFacets(overviewSummaryParams, signal)).data,
    refetchInterval: REFETCH_INTERVAL,
  })
  const toolbarConfig = useMemo(() => {
    const config = getRemoteJobToolbarConfig(jobFacetsQuery.data)
    return {
      ...config,
      getHeader,
      filterOptions: config.filterOptions.map((filter) =>
        filter.key === 'status'
          ? { ...filter, defaultValues: ['Running', 'Pending', 'Prequeue'] }
          : filter
      ),
    }
  }, [jobFacetsQuery.data])

  const resourcesQuery = useQuery(
    queryResources(true, (resource) => {
      return resource.type == 'gpu'
    })
  )
  const billingSummaryQuery = useQuery({
    queryKey: ['context', 'billing-summary', 'overview'],
    queryFn: () => apiContextBillingSummary().then((res) => res.data),
    enabled: billingVisible,
  })

  const jobStatus = useMemo(() => {
    return (jobSummaryFacetsQuery.data?.facets.status ?? [])
      .filter(({ value }) => value !== JobPhase.Deleted && value !== JobPhase.Freed)
      .map(({ value, count }) => ({
        id: value,
        label: getJobPhaseLabel(value as JobPhase).label,
        value: count,
      }))
  }, [jobSummaryFacetsQuery.data])

  const hideUsername = useAtomValue(globalHideUsername)
  const userStatus = useMemo(() => {
    return (jobSummaryFacetsQuery.data?.facets.owner ?? []).map(({ value, count }) => ({
      id: value,
      label: hideUsername ? getUserPseudonym(value) : value,
      value: count,
    }))
  }, [hideUsername, jobSummaryFacetsQuery.data])

  const gpuStatus = useMemo(() => {
    return (jobSummaryFacetsQuery.data?.facets.gpu_resource ?? []).map(({ value, count }) => ({
      id: value,
      label: value,
      value: count,
    }))
  }, [jobSummaryFacetsQuery.data])

  const gpuAllocation = useMemo(() => {
    if (resourcesQuery.data === undefined) {
      return 0
    }
    const total = resourcesQuery.data.reduce((acc, resource) => {
      if (resource.type === 'gpu') {
        return acc + resource.amount
      }
      return acc
    }, 0)
    const used = gpuStatus.reduce((acc, item) => {
      return acc + item.value
    }, 0)
    return total > 0 ? (used / total) * 100 : 0
  }, [resourcesQuery.data, gpuStatus])

  return (
    <>
      <div className="grid gap-4 lg:grid-cols-2">
        <UserBanAlert className="lg:col-span-2" />
        <PageTitle
          title={`欢迎回来，${userInfo?.nickname} 👋`}
          description="使用异构集群管理平台 Crater 加速您的科研工作"
          className="lg:col-span-2"
        >
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
        <SectionCards
          items={[
            {
              title: '运行中作业',
              value: jobStatus.find(({ id }) => id === JobPhase.Running)?.value ?? 0,
              className: 'text-highlight-blue',
              description: '正在运行的作业数量',
              icon: FlaskConicalIcon,
            },
            {
              title: t('statuses.awaitingAdmission'),
              value: jobStatus.find(({ id }) => id === JobPhase.Prequeue)?.value ?? 0,
              className: 'text-highlight-violet',
              description: t('jobs.statuses.prequeue.description'),
              icon: ClockIcon,
            },
            {
              title: t('statuses.pendingScheduling'),
              value: jobStatus.find(({ id }) => id === JobPhase.Pending)?.value ?? 0,
              className: 'text-highlight-purple',
              description: t('jobs.statuses.pending.description'),
              icon: ClockIcon,
            },
            {
              title: '活跃用户',
              value: userStatus.length,
              className: 'text-highlight-emerald',
              description: '当前活跃的用户数量',
              icon: UsersRoundIcon,
            },
            {
              title: '加速卡分配率',
              value: `${gpuAllocation.toFixed()}%`,
              className: 'text-highlight-orange',
              description: '当前 GPU 资源的分配率',
              icon: GpuIcon,
            },
          ]}
          className="lg:col-span-2 @5xl/main:grid-cols-5"
        />
        <PieCard
          icon={FlaskConicalIcon}
          cardTitle="作业状态"
          cardDescription="查看集群近 7 天作业的状态统计"
          isLoading={jobSummaryFacetsQuery.isLoading}
        >
          <NivoPie
            data={jobStatus}
            margin={{ top: 20, bottom: 30 }}
            colors={({ id }) => {
              return jobPhases.find((x) => x.value === id)?.color ?? '#000'
            }}
            arcLabelsTextColor="#ffffff"
          />
        </PieCard>
        <PieCard
          icon={UsersRoundIcon}
          cardTitle="用户统计"
          cardDescription="当前正在运行作业所属的用户"
          isLoading={jobSummaryFacetsQuery.isLoading}
        >
          <NivoPie data={userStatus} margin={{ top: 20, bottom: 30 }} />
        </PieCard>
      </div>
      <RemoteDataTable
        info={{
          title: '作业信息',
          description: '查看近 7 天集群作业的运行情况',
        }}
        query={jobQuery}
        state={tableState}
        columns={jobColumns}
        getRowId={(row) => row.jobName}
        toolbarConfig={toolbarConfig}
      />
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
    </>
  )
}
