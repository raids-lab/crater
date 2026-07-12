/**
 * User Jobs Overview Component
 *
 * This component displays a list of jobs for a specific user.
 * It is designed to be used in the user detail view where we don't need
 * to show the user column (since we're already viewing a specific user's jobs)
 * and in the user view, we don't show the name column.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { Trash2Icon } from 'lucide-react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import ScheduleTypeLabel from '@/components/badge/schedule-type-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import { JobActionsMenu } from '@/components/job/overview/job-actions-menu'
import { getHeader, getRemoteJobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { RemoteDataTable } from '@/components/query-table/remote'
import { buildFacetQueryKey, buildRemoteQueryKey } from '@/components/query-table/remote-state'

import {
  IJobInfo,
  ScheduleType,
  apiAdminGetUserJobFacets,
  apiAdminGetUserJobList,
  apiGetUserJobFacets,
  apiGetUserJobs,
  apiJobDeleteForAdmin,
  getDisplayJobPhase,
} from '@/services/api/vcjob'

import useIsAdmin from '@/hooks/use-admin'
import useRemoteTableState from '@/hooks/use-remote-table-state'

// Define the props for the component
interface UserJobsOverviewProps {
  /**
   * The username for which to display jobs
   */
  username: string
}

export function UserJobsOverview({ username }: UserJobsOverviewProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isAdmin = useIsAdmin()
  const resourceKey = `user-jobs-${username}-${isAdmin}`
  const tableState = useRemoteTableState(`user_jobs_overview_${username}_${isAdmin}`, {
    sorting: [{ id: 'createdAt', desc: true }],
    columnFilters: [{ id: 'days', value: ['30'] }],
  })

  // Fetch user jobs data
  const userJobsQuery = useQuery({
    queryKey: [...buildRemoteQueryKey(resourceKey, tableState.params), username, isAdmin],
    queryFn: async ({ signal }) =>
      (
        await (isAdmin
          ? apiAdminGetUserJobList(username, tableState.params, signal)
          : apiGetUserJobs(username, tableState.params, signal))
      ).data,
    placeholderData: keepPreviousData,
  })
  const facetsQuery = useQuery({
    queryKey: [...buildFacetQueryKey(resourceKey, tableState.params), username, isAdmin],
    queryFn: async ({ signal }) =>
      (
        await (isAdmin
          ? apiAdminGetUserJobFacets(username, tableState.params, signal)
          : apiGetUserJobFacets(username, tableState.params, signal))
      ).data,
  })

  const refetchJobs = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['remote-list', resourceKey] }),
      queryClient.invalidateQueries({ queryKey: ['remote-list-facets', resourceKey] }),
    ])
  }, [queryClient, resourceKey])

  // Only admin view can delete jobs, so always use admin delete API
  const { mutate: deleteJob } = useMutation({
    mutationFn: apiJobDeleteForAdmin,
    onSuccess: async () => {
      await refetchJobs()
      toast.success(t('jobs.successMessage'))
    },
  })

  const toolbarConfig = useMemo(
    () => getRemoteJobToolbarConfig(facetsQuery.data),
    [facetsQuery.data]
  )

  // Define table columns
  const userJobsColumns = useMemo<ColumnDef<IJobInfo>[]>(() => {
    const columns: ColumnDef<IJobInfo>[] = [
      {
        accessorKey: 'jobType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.jobType')} />
        ),
        cell: ({ row }) => <JobTypeLabel jobType={row.original.jobType} />,
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
      ...(isAdmin
        ? ([
            {
              accessorKey: 'name',
              header: ({ column }) => (
                <DataTableColumnHeader column={column} title={t('jobs.headers.jobName')} />
              ),
              cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
            },
          ] as ColumnDef<IJobInfo>[])
        : []),
      {
        accessorKey: 'queue',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.queue')} />
        ),
        cell: ({ row }) => <div>{row.getValue('queue')}</div>,
      },
      {
        accessorKey: 'nodes',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.nodes')} />
        ),
        cell: ({ row }) => {
          const nodes = row.getValue('nodes') as string[]
          return <NodeBadges nodes={nodes} />
        },
      },
      {
        accessorKey: 'resources',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.getValue('resources') as Record<string, string> | undefined
          return <ResourceBadges resources={resources} />
        },
      },
      {
        accessorFn: (row) => getDisplayJobPhase(row.status),
        id: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.status')} />
        ),
        cell: ({ row }) => <JobPhaseLabel jobPhase={row.original.status} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.createdAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.original.createdAt} />,
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'startedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.startedAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.getValue('startedAt')} />,
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'completedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.completedAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.getValue('completedAt')} />,
        sortingFn: 'datetime',
      },
      ...(isAdmin
        ? ([
            {
              id: 'actions',
              enableHiding: false,
              cell: ({ row }) => {
                const jobInfo = row.original
                return (
                  <JobActionsMenu
                    jobInfo={jobInfo}
                    onDelete={deleteJob}
                    isAdminView={true}
                    onLockSuccess={refetchJobs}
                  />
                )
              },
            },
          ] as ColumnDef<IJobInfo>[])
        : []),
    ]

    return columns
  }, [t, deleteJob, isAdmin, refetchJobs])

  return (
    <RemoteDataTable
      info={{
        description: t(
          isAdmin ? 'jobs.userJobsDescription.admin' : 'jobs.userJobsDescription.user'
        ),
      }}
      query={userJobsQuery}
      state={tableState}
      columns={userJobsColumns}
      getRowId={(row) => row.jobName}
      toolbarConfig={toolbarConfig}
      initialColumnVisibility={{ nodes: false }}
      multipleHandlers={
        isAdmin
          ? [
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
                    deleteJob(row.original.jobName)
                  })
                },
                isDanger: true,
              },
            ]
          : undefined
      }
    />
  )
}
