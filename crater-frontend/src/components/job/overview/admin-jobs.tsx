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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import {
  CalendarIcon,
  EllipsisVerticalIcon,
  InfoIcon,
  LockIcon,
  SquareIcon,
  Trash2Icon,
  UnlockIcon,
} from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import JobPhaseLabel, { jobPhases } from '@/components/badge/job-phase-badge'
import JobTypeLabel, { jobTypes } from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import { JobNameCell } from '@/components/label/job-name-label'
import UserLabel from '@/components/label/user-label'
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

import { IJobInfo, JobType, apiAdminGetJobList, apiJobDeleteForAdmin } from '@/services/api/vcjob'
import { JobPhase } from '@/services/api/vcjob'

import { logger } from '@/utils/loglevel'

import { DurationDialog } from '../../../routes/admin/jobs/-components/duration-dialog'

export type StatusValue =
  | 'Queueing'
  | 'Created'
  | 'Pending'
  | 'Running'
  | 'Failed'
  | 'Succeeded'
  | 'Preempted'
  | 'Deleted'

export const getHeader = (key: string): string => {
  switch (key) {
    case 'jobName':
      return '名称'
    case 'jobType':
      return '类型'
    case 'queue':
      return '账户'
    case 'owner':
      return '用户'
    case 'status':
      return '状态'
    case 'createdAt':
      return '创建于'
    case 'startedAt':
      return '开始于'
    case 'completedAt':
      return '完成于'
    case 'nodes':
      return '节点'
    case 'resources':
      return '资源'
    default:
      return key
  }
}

const AdminJobOverview = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [days, setDays] = useState(7)
  const [selectedJobs, setSelectedJobs] = useState<IJobInfo[]>([])
  const [isLockDialogOpen, setIsLockDialogOpen] = useState(false)
  const [isExtendDialogOpen, setIsExtendDialogOpen] = useState(false)

  const toolbarConfig: DataTableToolbarConfig = {
    globalSearch: {
      enabled: true,
    },
    filterOptions: [
      {
        key: 'jobType',
        title: t('adminJobOverview.filters.jobType'),
        option: jobTypes,
      },
      {
        key: 'status',
        title: t('adminJobOverview.filters.status'),
        option: jobPhases,
      },
    ],
    getHeader: getHeader,
  }

  const vcjobQuery = useQuery({
    queryKey: ['admin', 'tasklist', 'job', days],
    queryFn: () => apiAdminGetJobList(days),
    select: (res) => res.data,
  })

  const refetchTaskList = useCallback(async () => {
    try {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({
            queryKey: ['admin', 'tasklist', 'job', days],
          })
        ),
      ])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }, [queryClient, days])

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDeleteForAdmin,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('adminJobOverview.successMessage'))
    },
  })

  const handleClick = useCallback((jobInfo: IJobInfo) => {
    setSelectedJobs([jobInfo])
    setIsLockDialogOpen(true)
  }, [])

  const handleClickToExtend = useCallback((jobInfo: IJobInfo) => {
    setSelectedJobs([jobInfo])
    setIsExtendDialogOpen(true)
  }, [])

  const vcjobColumns = useMemo<ColumnDef<IJobInfo>[]>(() => {
    const getHeader = (key: string): string => {
      switch (key) {
        case 'jobName':
          return t('adminJobOverview.headers.jobName')
        case 'jobType':
          return t('adminJobOverview.headers.jobType')
        case 'queue':
          return t('adminJobOverview.headers.queue')
        case 'owner':
          return t('adminJobOverview.headers.owner')
        case 'status':
          return t('adminJobOverview.headers.status')
        case 'createdAt':
          return t('adminJobOverview.headers.createdAt')
        case 'startedAt':
          return t('adminJobOverview.headers.startedAt')
        case 'completedAt':
          return t('adminJobOverview.headers.completedAt')
        case 'nodes':
          return t('adminJobOverview.headers.nodes')
        case 'resources':
          return t('adminJobOverview.headers.resources')
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
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.getValue<Record<string, string> | undefined>('resources')
          return <ResourceBadges resources={resources} />
        },
      },
      {
        accessorKey: 'status',
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
          const shouldStop = jobInfo.status !== 'Deleted' && jobInfo.status !== 'Freed'
          return (
            <div className="flex flex-row space-x-1">
              <AlertDialog>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                      <span className="sr-only">
                        {t('adminJobOverview.actions.dropdown.ariaLabel')}
                      </span>
                      <EllipsisVerticalIcon className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel className="text-muted-foreground text-xs">
                      {t('adminJobOverview.actions.dropdown.title')}
                    </DropdownMenuLabel>
                    <Link from="/admin/jobs" to={`${jobInfo.jobName}`}>
                      <DropdownMenuItem>
                        <InfoIcon className="text-highlight-emerald" />
                        {t('adminJobOverview.actions.dropdown.details')}
                      </DropdownMenuItem>
                    </Link>
                    <DropdownMenuItem
                      onClick={() => handleClick(jobInfo)}
                      title={t('adminJobOverview.actions.dropdown.lockTitle')}
                    >
                      {row.original.locked ? (
                        <UnlockIcon className="text-highlight-purple" />
                      ) : (
                        <LockIcon className="text-highlight-purple" />
                      )}
                      {row.original.locked
                        ? t('adminJobOverview.actions.dropdown.unlock')
                        : t('adminJobOverview.actions.dropdown.lock')}
                    </DropdownMenuItem>
                    {row.original.locked && (
                      <DropdownMenuItem
                        onClick={() => handleClickToExtend(jobInfo)}
                        title={t('adminJobOverview.actions.dropdown.lockTitle')}
                      >
                        <LockIcon className="text-highlight-purple" />
                        {t('adminJobOverview.actions.dropdown.extend')}
                      </DropdownMenuItem>
                    )}
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem className="group">
                        {shouldStop ? (
                          <SquareIcon className="text-highlight-orange" />
                        ) : (
                          <Trash2Icon className="text-destructive" />
                        )}
                        {shouldStop
                          ? t('adminJobOverview.actions.dropdown.stop')
                          : t('adminJobOverview.actions.dropdown.delete')}
                      </DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {shouldStop
                        ? t('adminJobOverview.dialog.stop.title')
                        : t('adminJobOverview.dialog.delete.title')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {shouldStop
                        ? t('adminJobOverview.dialog.stop.description', {
                            name: jobInfo?.name,
                          })
                        : t('adminJobOverview.dialog.delete.description', {
                            name: jobInfo?.name,
                          })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('adminJobOverview.dialog.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => deleteTask(jobInfo.jobName)}
                    >
                      {shouldStop
                        ? t('adminJobOverview.dialog.stop.action')
                        : t('adminJobOverview.dialog.delete.action')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          )
        },
      },
    ]
  }, [deleteTask, handleClick, handleClickToExtend, t])

  return (
    <>
      <DataTable
        info={{
          title: t('adminJobOverview.title'),
          description: t('adminJobOverview.description'),
        }}
        storageKey="admin_job_overview"
        query={vcjobQuery}
        columns={vcjobColumns}
        toolbarConfig={toolbarConfig}
        multipleHandlers={[
          {
            title: (rows) =>
              t('adminJobOverview.handlers.stopOrDeleteTitle', {
                count: rows.length,
              }),
            description: (rows) => (
              <>
                {t('adminJobOverview.handlers.stopOrDeleteDescription', {
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
            setDays(parseInt(value))
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
                  ? t('adminJobOverview.select.all')
                  : t('adminJobOverview.select.recentDays', { days: pageSize })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </DataTable>

      {/* Duration Dialog for locking/unlocking jobs */}
      <DurationDialog
        jobs={selectedJobs}
        open={isLockDialogOpen}
        setOpen={setIsLockDialogOpen}
        onSuccess={refetchTaskList}
      />

      {/* Duration Dialog for extending locked jobs */}
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
