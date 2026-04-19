/**
 * User Jobs Overview Component
 *
 * This component displays a list of jobs for a specific user.
 * It is designed to be used in the user detail view where we don't need
 * to show the user column (since we're already viewing a specific user's jobs)
 * and in the user view, we don't show the name column.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { Trash2Icon } from 'lucide-react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import { JobActionsMenu } from '@/components/job/overview/job-actions-menu'
import { getHeader, jobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  IJobInfo,
  apiAdminGetUserJobList,
  apiGetUserJobs,
  apiJobDeleteForAdmin,
} from '@/services/api/vcjob'

import useIsAdmin from '@/hooks/use-admin'

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

  // Fetch user jobs data
  const userJobsQuery = useQuery({
    queryKey: ['user-jobs', username, isAdmin],
    queryFn: () => (isAdmin ? apiAdminGetUserJobList(username, 30) : apiGetUserJobs(username, 30)), // Query 30 days of data
    select: (res) => res.data,
  })

  const refetchJobs = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: ['user-jobs', username, isAdmin],
    })
  }, [queryClient, username, isAdmin])

  // Only admin view can delete jobs, so always use admin delete API
  const { mutate: deleteJob } = useMutation({
    mutationFn: apiJobDeleteForAdmin,
    onSuccess: async () => {
      await refetchJobs()
      toast.success(t('jobs.successMessage'))
    },
  })

  // Define toolbar config based on admin status
  const toolbarConfig = useMemo<DataTableToolbarConfig>(() => {
    if (isAdmin) {
      // Admin view: use default config with search
      return jobToolbarConfig
    } else {
      // User view: no search box, but keep filters and view options
      return {
        globalSearch: {
          enabled: false,
        },
        filterOptions: jobToolbarConfig.filterOptions,
        getHeader: getHeader,
      }
    }
  }, [isAdmin])

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
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('jobs.headers.resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.getValue('resources') as Record<string, string> | undefined
          return <ResourceBadges resources={resources} />
        },
      },
      {
        accessorKey: 'status',
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
    <DataTable
      info={{
        description: t(
          isAdmin ? 'jobs.userJobsDescription.admin' : 'jobs.userJobsDescription.user'
        ),
      }}
      storageKey="user_jobs_overview"
      query={userJobsQuery}
      columns={userJobsColumns}
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
