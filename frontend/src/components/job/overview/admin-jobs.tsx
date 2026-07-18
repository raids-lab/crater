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
// i18n-processed-v1.1.0
// Modified code
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { CalendarIcon, LockIcon, Trash2Icon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import ScheduleTypeLabel from '@/components/badge/schedule-type-badge'
import { BillingPointsBadge } from '@/components/custom/billing-points-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import { JobActionsMenu } from '@/components/job/overview/job-actions-menu'
import { getRemoteJobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import UserLabel from '@/components/label/user-label'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { RemoteDataTable } from '@/components/query-table/remote'
import { buildFacetQueryKey, buildRemoteQueryKey } from '@/components/query-table/remote-state'

import { apiAdminGetBillingStatus } from '@/services/api/system-config'
import {
  IJobInfo,
  JobPhase,
  JobType,
  ScheduleType,
  apiAdminGetJobFacets,
  apiAdminGetJobList,
  apiJobDeleteForAdmin,
  getDisplayJobPhase,
} from '@/services/api/vcjob'

import useRemoteTableState from '@/hooks/use-remote-table-state'

import { isBillingVisibleForAdmin } from '@/utils/billing-visibility'
import { logger } from '@/utils/loglevel'

import { DurationDialog } from '../../../routes/admin/jobs/-components/duration-dialog'

export type StatusValue =
  | 'Prequeue'
  | 'Queueing'
  | 'Created'
  | 'Pending'
  | 'Running'
  | 'Failed'
  | 'Succeeded'
  | 'Preempted'
  | 'Deleted'

type JobTableRow = IJobInfo

export const getHeader = (key: string): string => {
  switch (key) {
    case 'jobName':
      return t('jobs.headers.jobName')
    case 'jobType':
      return t('jobs.headers.jobType')
    case 'scheduleType':
      return t('jobs.headers.scheduleType')
    case 'queue':
      return t('jobs.headers.queue')
    case 'owner':
      return t('jobs.headers.owner')
    case 'status':
      return t('jobs.headers.status')
    case 'createdAt':
      return t('jobs.headers.createdAt')
    case 'startedAt':
      return t('jobs.headers.startedAt')
    case 'completedAt':
      return t('jobs.headers.completedAt')
    case 'nodes':
      return t('jobs.headers.nodes')
    case 'resources':
      return t('jobs.headers.resources')
    default:
      return key
  }
}

const AdminJobOverview = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedJobs, setSelectedJobs] = useState<IJobInfo[]>([])
  const [isLockDialogOpen, setIsLockDialogOpen] = useState(false)
  const [isExtendDialogOpen, setIsExtendDialogOpen] = useState(false)
  const { data: billingStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })
  const billingVisible = isBillingVisibleForAdmin(billingStatus)
  const tableState = useRemoteTableState('admin_job_overview', {
    sorting: [{ id: 'createdAt', desc: true }],
    columnFilters: [{ id: 'days', value: ['7'] }],
  })
  const days = Number(
    (
      tableState.columnFilters.find(({ id }) => id === 'days')?.value as string[] | undefined
    )?.[0] ?? 7
  )

  const vcjobQuery = useQuery({
    queryKey: buildRemoteQueryKey('admin-jobs', tableState.params),
    queryFn: async ({ signal }) => (await apiAdminGetJobList(tableState.params, signal)).data,
    placeholderData: keepPreviousData,
  })
  const facetsQuery = useQuery({
    queryKey: buildFacetQueryKey('admin-jobs', tableState.params),
    queryFn: async ({ signal }) => (await apiAdminGetJobFacets(tableState.params, signal)).data,
  })
  const toolbarConfig = useMemo(
    () => ({ ...getRemoteJobToolbarConfig(facetsQuery.data), getHeader }),
    [facetsQuery.data]
  )

  const refetchTaskList = useCallback(async () => {
    try {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ['remote-list', 'admin-jobs'] })
        ),
        queryClient.invalidateQueries({ queryKey: ['remote-list-facets', 'admin-jobs'] }),
      ])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }, [queryClient])

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDeleteForAdmin,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('jobs.successMessage'))
    },
  })

  const vcjobColumns = useMemo<ColumnDef<JobTableRow>[]>(() => {
    const getHeader = (key: string): string => {
      switch (key) {
        case 'jobName':
          return t('jobs.headers.jobName')
        case 'jobType':
          return t('jobs.headers.jobType')
        case 'scheduleType':
          return t('jobs.headers.scheduleType')
        case 'queue':
          return t('jobs.headers.queue')
        case 'owner':
          return t('jobs.headers.owner')
        case 'status':
          return t('jobs.headers.status')
        case 'createdAt':
          return t('jobs.headers.createdAt')
        case 'startedAt':
          return t('jobs.headers.startedAt')
        case 'completedAt':
          return t('jobs.headers.completedAt')
        case 'nodes':
          return t('jobs.headers.nodes')
        case 'resources':
          return t('jobs.headers.resources')
        default:
          return key
      }
    }
    return [
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
        accessorKey: 'jobName',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('jobName')} />
        ),
        cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
      },
      {
        accessorKey: 'owner',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('owner')} />
        ),
        cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
      },
      {
        accessorKey: 'queue',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('queue')} />
        ),
        cell: ({ row }) => <div>{row.getValue('queue')}</div>,
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
          return <TimeDistance date={row.getValue('createdAt')} />
        },
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'startedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('startedAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('startedAt')} />
        },
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'completedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('completedAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('completedAt')} />
        },
        sortingFn: 'datetime',
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const jobInfo = row.original
          return (
            <JobActionsMenu
              jobInfo={jobInfo}
              onDelete={deleteTask}
              isAdminView={true}
              onLockSuccess={refetchTaskList}
            />
          )
        },
      },
    ]
  }, [billingVisible, deleteTask, refetchTaskList, t])

  return (
    <>
      <RemoteDataTable
        info={{
          title: t('adminJobOverview.title'),
          description: t('adminJobOverview.description'),
        }}
        query={vcjobQuery}
        state={tableState}
        columns={vcjobColumns}
        getRowId={(row) => row.jobName}
        toolbarConfig={toolbarConfig}
        multipleHandlers={[
          {
            title: (rows) =>
              t('jobs.handlers.stopOrDeleteTitle', {
                count: rows.length,
              }),
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
          {
            title: (rows) =>
              t('adminJobOverview.handlers.lockOrUnlockTitle', {
                count: rows.length,
              }),
            description: (rows) => (
              <>
                {t('adminJobOverview.handlers.lockOrUnlockDescription', {
                  jobs: rows.map((row) => row.original.name).join(', '),
                })}
              </>
            ),
            icon: <LockIcon className="text-highlight-purple" />,
            handleSubmit: (rows) => {
              const jobInfos = rows.map((row) => row.original)
              setSelectedJobs(jobInfos)
              setIsLockDialogOpen(true)
            },
            isDanger: false,
          },
        ]}
      >
        <Select
          value={days.toString()}
          onValueChange={(value) => {
            tableState.setColumnFilters((current) => [
              ...current.filter(({ id }) => id !== 'days'),
              { id: 'days', value: [value] },
            ])
          }}
        >
          <SelectTrigger className="bg-background h-9 pr-2 pl-3">
            <CalendarIcon />
            <SelectValue placeholder={days.toString()} />
          </SelectTrigger>
          <SelectContent side="top">
            {[7, 14, 30, 90, -1].map((pageSize) => (
              <SelectItem key={pageSize} value={`${pageSize}`}>
                {pageSize === -1
                  ? t('jobs.select.all')
                  : t('jobs.select.recentDays', { days: pageSize })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </RemoteDataTable>

      {/* Duration Dialog for locking/unlocking jobs (batch operation) */}
      <DurationDialog
        jobs={selectedJobs}
        open={isLockDialogOpen}
        setOpen={setIsLockDialogOpen}
        onSuccess={refetchTaskList}
      />

      {/* Duration Dialog for extending locked jobs (batch operation) */}
      <DurationDialog
        jobs={selectedJobs}
        open={isExtendDialogOpen}
        setOpen={setIsExtendDialogOpen}
        onSuccess={refetchTaskList}
        setExtend={true}
      />
    </>
  )
}

export default AdminJobOverview
