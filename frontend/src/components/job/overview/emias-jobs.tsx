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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, linkOptions } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { Trash2Icon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import JobPhaseLabel, { jobPhases } from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import { getHeader } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/job-name-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import { apiJobBatchList, apiJobDelete } from '@/services/api/vcjob'
import { IJobInfo, JobType } from '@/services/api/vcjob'

import { logger } from '@/utils/loglevel'

import { REFETCH_INTERVAL } from '@/lib/constants'

import Quota from '../../../routes/portal/jobs/inter/-components/quota'
import ListedNewJobButton from '../new-job-button'

// Link Options for portal job navigation
const portalJobDetailLinkOptions = linkOptions({
  to: '/portal/jobs/detail/$name',
  params: { name: '' },
  search: { tab: '' },
})

const getPriorities = (t: (key: string) => string) => [
  {
    label: t('jobs.priority.high'),
    value: 'high',
    className: 'text-highlight-amber border-highlight-amber bg-highlight-amber/20',
  },
  {
    label: t('jobs.priority.low'),
    value: 'low',
    className: 'text-highlight-slate border-highlight-slate bg-highlight-slate/20',
  },
]

const getProfilingStatuses = (t: (key: string) => string) => [
  {
    value: '0',
    label: t('jobs.profileStatus.notAnalyzed'),
    className: 'text-highlight-purple border-highlight-purple bg-highlight-purple/20',
  },
  {
    value: '1',
    label: t('jobs.profileStatus.pending'),
    className: 'text-highlight-slate border-highlight-slate bg-highlight-slate/20',
  },
  {
    value: '2',
    label: t('jobs.profileStatus.analyzing'),
    className: 'text-highlight-sky border-highlight-sky bg-highlight-sky/20',
  },
  {
    value: '3',
    label: t('jobs.profileStatus.analyzed'),
    className: 'text-highlight-emerald border-highlight-emerald bg-highlight-emerald/20',
  },
  {
    value: '4',
    label: t('jobs.profileStatus.failed'),
    className: 'text-highlight-red border-highlight-red bg-highlight-red/20',
  },
  {
    value: '5',
    label: t('jobs.profileStatus.skipped'),
    className: 'text-highlight-slate border-highlight-slate bg-highlight-slate/20',
  },
]

const getToolbarConfig = (t: (key: string) => string): DataTableToolbarConfig => ({
  filterInput: {
    placeholder: t('jobs.toolbar.searchName'),
    key: 'title',
  },
  filterOptions: [
    {
      key: 'status',
      title: t('jobs.toolbar.status'),
      option: jobPhases,
    },
    {
      key: 'priority',
      title: t('jobs.toolbar.priority'),
      option: getPriorities(t),
    },
    {
      key: 'profileStatus',
      title: t('jobs.toolbar.profileStatus'),
      option: getProfilingStatuses(t),
    },
  ],
  getHeader: getHeader,
})

interface ColocateJobInfo extends IJobInfo {
  id: number
  profileStatus: string
  priority: string
}

const ColocateOverview = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const batchQuery = useQuery({
    queryKey: ['job', 'batch'],
    queryFn: apiJobBatchList,
    select: (res) =>
      res.data
        .filter((task) => task.jobType !== JobType.Jupyter)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt)) as unknown as ColocateJobInfo[],
    refetchInterval: REFETCH_INTERVAL,
  })

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['job'] }),
        queryClient.invalidateQueries({ queryKey: ['aitask', 'quota'] }),
        queryClient.invalidateQueries({ queryKey: ['aitask', 'stats'] }),
      ])
    } catch (error) {
      logger.error(t('jobs.toast.refetchFailed'), error)
    }
  }

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('jobs.toast.deleted'))
    },
  })

  const priorities = useMemo(() => getPriorities(t), [t])
  const profilingStatuses = useMemo(() => getProfilingStatuses(t), [t])

  const batchColumns = useMemo<ColumnDef<ColocateJobInfo>[]>(
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
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
        cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
      },
      {
        accessorKey: 'owner',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('owner')} />
        ),
        cell: ({ row }) => <div>{row.getValue('owner')}</div>,
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('status')} />
        ),
        cell: ({ row }) => <JobPhaseLabel jobPhase={row.getValue('status')} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'priority',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('priority')} />
        ),
        cell: ({ row }) => {
          const priority = priorities.find(
            (priority) => priority.value === row.getValue('priority')
          )
          if (!priority) {
            return null
          }
          return (
            <Badge className={priority.className} variant="outline">
              {priority.label}
            </Badge>
          )
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
        enableSorting: false,
      },
      {
        accessorKey: 'resources',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.getValue<Record<string, string> | undefined>('resources')
          return <ResourceBadges resources={resources} />
        },
      },
      {
        accessorKey: 'profileStatus',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('profileStatus')} />
        ),
        cell: ({ row }) => {
          let profiling = profilingStatuses.find(
            (profiling) => profiling.value === row.getValue('profileStatus')
          )
          if (!profiling) {
            return null
          }
          if (row.getValue<string>('status') === 'Succeeded') {
            profiling = {
              value: '3',
              label: t('jobs.profileStatus.analyzed'),
              className: 'text-highlight-emerald border-highlight-emerald bg-highlight-emerald/20',
            }
          }
          return (
            <Badge className={profiling.className} variant="outline">
              {profiling.label}
            </Badge>
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
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const taskInfo = row.original
          return (
            <AlertDialog>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">{t('jobs.actions.dropdown.ariaLabel')}</span>
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="text-muted-foreground text-xs">
                    {t('jobs.actions.dropdown.title')}
                  </DropdownMenuLabel>
                  <DropdownMenuItem asChild>
                    <Link {...portalJobDetailLinkOptions} params={{ name: taskInfo.jobName }}>
                      {t('jobs.actions.dropdown.details')}
                    </Link>
                  </DropdownMenuItem>
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>{t('jobs.actions.dropdown.delete')}</DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t('jobs.dialog.delete.title')}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('jobs.dialog.delete.description.hidden', { name: taskInfo?.name })}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{t('jobs.actions.dropdown.cancel')}</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => deleteTask(taskInfo.id.toString())}
                  >
                    {t('jobs.actions.dropdown.delete')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )
        },
      },
    ],
    [deleteTask, t, priorities, profilingStatuses]
  )

  return (
    <>
      <DataTable
        info={{
          title: t('jobs.customJobs.title'),
          description: t('jobs.customJobs.description'),
        }}
        storageKey="portal_aijob_batch"
        query={batchQuery}
        columns={batchColumns}
        toolbarConfig={getToolbarConfig(t)}
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
        briefChildren={<Quota />}
      >
        <ListedNewJobButton mode="custom" />
      </DataTable>
    </>
  )
}

export default ColocateOverview
