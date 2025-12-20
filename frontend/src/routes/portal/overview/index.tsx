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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { type Locale, enUS, ja, ko, zhCN } from 'date-fns/locale'
import { useAtomValue } from 'jotai'
import { ClockIcon, FlaskConicalIcon, GpuIcon, UsersRoundIcon } from 'lucide-react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import JobPhaseLabel, { getJobPhaseLabel, jobPhases } from '@/components/badge/job-phase-badge'
import JobTypeLabel, { jobTypes } from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import DocsButton from '@/components/button/docs-button'
import NivoPie from '@/components/chart/nivo-pie'
import PieCard from '@/components/chart/pie-card'
import { TimeDistance } from '@/components/custom/time-distance'
import ListedNewJobButton from '@/components/job/new-job-button'
import { getHeader } from '@/components/job/overview/admin-jobs'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import { SectionCards } from '@/components/metrics/section-cards'
import { useAccountNameLookup } from '@/components/node/getaccountnickname'
import { getNodeColumns, nodesToolbarConfig } from '@/components/node/node-list'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { JobPhase } from '@/services/api/vcjob'
import { IJobInfo, JobType, apiJobAllList } from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import { getUserPseudonym } from '@/utils/pseudonym'
import { atomUserInfo, globalHideUsername } from '@/utils/store'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const Route = createFileRoute('/portal/overview/')({
  component: Overview,
})

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索用户名称',
    key: 'owner',
  },
  filterOptions: [
    {
      key: 'jobType',
      title: '类型',
      option: jobTypes,
    },
    {
      key: 'status',
      title: '状态',
      option: jobPhases,
      defaultValues: ['Running', 'Pending'],
    },
  ],
  getHeader: getHeader,
}

function Overview() {
  const { i18n } = useTranslation()
  const userInfo = useAtomValue(atomUserInfo)
  const nodeQuery = useQuery(queryNodes(true))
  const { getNicknameByName } = useAccountNameLookup()

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

  const jobColumns = useMemo<ColumnDef<IJobInfo>[]>(
    () => [
      {
        accessorKey: 'jobType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('jobType')} />
        ),
        cell: ({ row }) => <JobTypeLabel jobType={row.getValue<JobType>('jobType')} />,
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
    []
  )

  const jobQuery = useQuery({
    queryKey: ['overview', 'joblist'],
    queryFn: apiJobAllList,
    select: (res) => res.data,
    refetchInterval: REFETCH_INTERVAL,
  })

  const resourcesQuery = useQuery(
    queryResources(true, (resource) => {
      return resource.type == 'gpu'
    })
  )

  const jobStatus = useMemo(() => {
    if (!jobQuery.data) {
      return []
    }
    const data = jobQuery.data
    const counts = data
      .filter((d) => d.status !== JobPhase.Deleted && d.status !== JobPhase.Freed)
      .reduce(
        (acc, item) => {
          const phase = item.status
          if (!acc[phase]) {
            acc[phase] = 0
          }
          acc[phase] += 1
          return acc
        },
        {} as Record<JobPhase, number>
      )
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: getJobPhaseLabel(phase as JobPhase).label,
      value: count,
    }))
  }, [jobQuery.data])

  const hideUsername = useAtomValue(globalHideUsername)
  const userStatus = useMemo(() => {
    if (!jobQuery.data) {
      return []
    }
    const data = jobQuery.data
    const counts = data
      .filter((job) => job.status == 'Running')
      .reduce(
        (acc, item) => {
          const owner = hideUsername ? getUserPseudonym(item.owner) : item.owner
          if (!acc[owner]) {
            acc[owner] = {
              nickname: item.userInfo.nickname ?? item.owner,
              count: 0,
            }
          }
          acc[owner].count += 1
          return acc
        },
        {} as Record<string, { nickname: string; count: number }>
      )
    return Object.entries(counts).map(([owner, pair]) => ({
      id: owner,
      label: hideUsername ? getUserPseudonym(owner) : pair.nickname,
      value: pair.count,
    }))
  }, [hideUsername, jobQuery.data])

  const gpuStatus = useMemo(() => {
    if (!jobQuery.data) {
      return []
    }
    const data = jobQuery.data
    const counts = data
      .filter((job) => job.status == 'Running')
      .reduce(
        (acc, item) => {
          const resources = item.resources
          for (const [k, value] of Object.entries(resources ?? {})) {
            if (k.startsWith('nvidia.com')) {
              const key = k.replace('nvidia.com/', '')
              if (!acc[key]) {
                acc[key] = 0
              }
              acc[key] += parseInt(value)
            }
          }
          return acc
        },
        {} as Record<string, number>
      )
    return Object.entries(counts).map(([phase, count]) => ({
      id: phase,
      label: phase,
      value: count,
    }))
  }, [jobQuery.data])

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
        <PageTitle
          title={`欢迎回来，${userInfo?.nickname} 👋`}
          description="使用异构集群管理平台 Crater 加速您的科研工作"
          className="lg:col-span-2"
        >
          <div className="flex flex-row gap-3">
            {/* <DocsButton title="平台文档" url="" /> */}
            <ListedNewJobButton mode="all" />
          </div>
        </PageTitle>
        <SectionCards
          items={[
            {
              title: '运行中作业',
              value: jobQuery.data?.filter((job) => job.status === JobPhase.Running).length,
              className: 'text-highlight-blue',
              description: '正在运行的作业数量',
              icon: FlaskConicalIcon,
            },
            {
              title: '等待中作业',
              value: jobQuery.data?.filter((job) => job.status === JobPhase.Pending).length,
              className: 'text-highlight-purple',
              description: '等待调度或未就绪的作业数量',
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
          className="lg:col-span-2"
        />
        <PieCard
          icon={FlaskConicalIcon}
          cardTitle="作业状态"
          cardDescription="查看集群近 7 天作业的状态统计"
          isLoading={jobQuery.isLoading}
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
          isLoading={jobQuery.isLoading}
        >
          <NivoPie data={userStatus} margin={{ top: 20, bottom: 30 }} />
        </PieCard>
      </div>
      <DataTable
        info={{
          title: '作业信息',
          description: '查看近 7 天集群作业的运行情况',
        }}
        storageKey="overview_joblist"
        query={jobQuery}
        columns={jobColumns}
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
