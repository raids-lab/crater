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
import { REFETCH_INTERVAL } from '@/config/task'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { Trash2Icon } from 'lucide-react'
import { useMemo } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import JobPhaseLabel from '@/components/badge/JobPhaseBadge'
import JobTypeLabel from '@/components/badge/JobTypeBadge'
import NodeBadges from '@/components/badge/NodeBadges'
import ResourceBadges from '@/components/badge/ResourceBadges'
import DocsButton from '@/components/button/docs-button'
import { TimeDistance } from '@/components/custom/TimeDistance'
import JupyterIcon from '@/components/icon/JupyterIcon'
import ListedNewJobButton from '@/components/job/new-job-button'
import { JobActionsMenu } from '@/components/job/overview/job-actions-menu'
import { getHeader, jobToolbarConfig } from '@/components/job/statuses'
import { JobNameCell } from '@/components/label/JobNameLabel'
import SimpleTooltip from '@/components/label/simple-tooltip'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import { JobPhase, apiJobDelete, apiJobInteractiveList } from '@/services/api/vcjob'
import { IJobInfo, JobType } from '@/services/api/vcjob'

import { logger } from '@/utils/loglevel'

import Quota from './-components/Quota'

export const Route = createFileRoute('/portal/jobs/inter/')({
  loader: () => {
    return {
      crumb: t('navigation.jupyterLab'),
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  const queryClient = useQueryClient()

  const interactiveQuery = useQuery({
    queryKey: ['job', 'interactive'],
    queryFn: apiJobInteractiveList,
    select: (res) => res.data.filter((task) => task.jobType === JobType.Jupyter),
    refetchInterval: REFETCH_INTERVAL,
  })

  const refetchTaskList = async () => {
    try {
      // 隔 200ms 并行发送所有异步请求
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ['job'] })
        ),
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ['context', 'quota'] })
        ),
      ])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success('操作成功')
    },
  })

  const interColumns = useMemo<ColumnDef<IJobInfo>[]>(
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
          const shouldDisable = jobInfo.status !== 'Running'
          return (
            <div className="flex flex-row space-x-1">
              {shouldDisable ? (
                <div className="h-8 w-8" />
              ) : (
                <SimpleTooltip tooltip="打开 Jupyter Lab">
                  <Button
                    variant="ghost"
                    size="icon"
                    disabled={shouldDisable}
                    className="text-primary hover:bg-primary/10 hover:text-primary/90 h-8 w-8"
                    asChild
                  >
                    <Link
                      to="/ingress/jupyter/$name"
                      params={{ name: jobInfo.jobName }}
                      target="_blank"
                    >
                      <JupyterIcon className="size-4" />
                    </Link>
                  </Button>
                </SimpleTooltip>
              )}
              <JobActionsMenu jobInfo={jobInfo} onDelete={deleteTask} />
            </div>
          )
        },
      },
    ],
    [deleteTask]
  )

  return (
    <DataTable
      info={{
        title: 'Jupyter Lab',
        description: '提供开箱即用的 Jupyter Lab， 可用于测试、调试等',
      }}
      storageKey="portal_job_interactive"
      query={interactiveQuery}
      columns={interColumns}
      toolbarConfig={jobToolbarConfig}
      multipleHandlers={[
        {
          title: (rows) => `停止或删除 ${rows.length} 个作业`,
          description: (rows) => (
            <>
              作业 {rows.map((row) => row.original.name).join(', ')} 将被停止或删除，确认要继续吗？
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
      <div className="flex flex-row gap-3">
        <DocsButton title="查看文档" url="quick-start/interactive" />
        <ListedNewJobButton mode="inter" />
      </div>
    </DataTable>
  )
}
