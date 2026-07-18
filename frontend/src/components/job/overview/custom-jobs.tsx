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
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { Trash2Icon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import ScheduleTypeLabel from '@/components/badge/schedule-type-badge'
import DocsButton from '@/components/button/docs-button'
import { BillingPointsBadge } from '@/components/custom/billing-points-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import JobResourceSummary from '@/components/job/job-resource-summary'
import { JobActionsMenu } from '@/components/job/overview/job-actions-menu'
import { getHeader, getRemoteJobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { RemoteDataTable } from '@/components/query-table/remote'
import { buildFacetQueryKey, buildRemoteQueryKey } from '@/components/query-table/remote-state'

import { apiGetBillingStatus } from '@/services/api/system-config'
import {
  IJobInfo,
  JobPhase,
  JobType,
  ScheduleType,
  apiJobBatchFacets,
  apiJobBatchList,
  apiJobDelete,
  batchJobTypes,
  getDisplayJobPhase,
} from '@/services/api/vcjob'

import useRemoteTableState from '@/hooks/use-remote-table-state'

import { isBillingVisibleForUser } from '@/utils/billing-visibility'
import { logger } from '@/utils/loglevel'

import { REFETCH_INTERVAL } from '@/lib/constants'

import ListedNewJobButton from '../new-job-button'

type JobTableRow = IJobInfo

const VolcanoOverview = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: billingStatus } = useQuery({
    queryKey: ['system-config', 'billing-status'],
    queryFn: () => apiGetBillingStatus().then((res) => res.data),
  })
  const billingVisible = isBillingVisibleForUser(billingStatus)
  const tableState = useRemoteTableState('portal_batch_job_overview', {
    sorting: [{ id: 'createdAt', desc: true }],
  })

  const batchQuery = useQuery({
    queryKey: buildRemoteQueryKey('jobs-batch', tableState.params),
    queryFn: async ({ signal }) => (await apiJobBatchList(tableState.params, signal)).data,
    placeholderData: keepPreviousData,
    refetchInterval: REFETCH_INTERVAL,
  })
  const facetsQuery = useQuery({
    queryKey: buildFacetQueryKey('jobs-batch', tableState.params),
    queryFn: async ({ signal }) => (await apiJobBatchFacets(tableState.params, signal)).data,
  })
  const toolbarConfig = useMemo(
    () => getRemoteJobToolbarConfig(facetsQuery.data, batchJobTypes),
    [facetsQuery.data]
  )

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['remote-list', 'jobs-batch'] }),
        queryClient.invalidateQueries({ queryKey: ['remote-list-facets', 'jobs-batch'] }),
        queryClient.invalidateQueries({ queryKey: ['job'] }),
        queryClient.invalidateQueries({ queryKey: ['job', 'billing'] }),
        queryClient.invalidateQueries({ queryKey: ['aitask', 'quota'] }),
        queryClient.invalidateQueries({ queryKey: ['aitask', 'stats'] }),
        queryClient.invalidateQueries({ queryKey: ['context', 'job-resource-summary'] }),
      ])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('jobs.successMessage'))
    },
  })

  const batchColumns = useMemo<ColumnDef<JobTableRow>[]>(
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
        accessorKey: 'name',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
        cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
      },
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
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const jobInfo = row.original
          return <JobActionsMenu jobInfo={jobInfo} onDelete={deleteTask} />
        },
      },
    ],
    [billingVisible, deleteTask]
  )

  return (
    <RemoteDataTable
      info={{
        title: '自定义作业',
        description: '使用自定义作业进行训练、推理等任务',
      }}
      query={batchQuery}
      state={tableState}
      columns={batchColumns}
      getRowId={(row) => row.jobName}
      toolbarConfig={toolbarConfig}
      briefChildren={<JobResourceSummary />}
      multipleHandlers={[
        {
          title: (rows) => t('jobs.handlers.stopOrDeleteTitle', { count: rows.length }),
          description: (rows) => (
            <>
              {t('jobs.handlers.stopOrDeleteDescription', {
                jobs: rows.map((row) => row.original.name).join(', '),
              })}
            </>
          ),
          icon: <Trash2Icon className="text-destructive" />,
          handleSubmit: (rows) => {
            rows.forEach((row) => {
              deleteTask(row.original.jobName)
            })
          },
          isDanger: true,
        },
      ]}
    >
      <div className="flex flex-row gap-3">
        <DocsButton title="查看文档" url="quick-start/batchprocess" />
        <ListedNewJobButton mode="custom" />
      </div>
    </RemoteDataTable>
  )
}

export default VolcanoOverview
