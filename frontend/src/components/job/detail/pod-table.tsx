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
import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { useAtomValue } from 'jotai'
import { EthernetPort, GaugeIcon, LogsIcon, TerminalIcon } from 'lucide-react'
import { useMemo, useState } from 'react'

import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import NodeBadges from '@/components/badge/node-badges'
import PodPhaseLabel, { podPhases } from '@/components/badge/pod-phase-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import TooltipButton from '@/components/button/tooltip-button'
import LogDialog from '@/components/codeblock/log-dialog'
import { NamespacedName } from '@/components/codeblock/pod-container-dialog'
import TerminalDialog from '@/components/codeblock/terminal-dialog'
import TooltipCopy from '@/components/label/tooltop-copy'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { SnapshotControls } from '@/components/query-table/cursor-list'
import { SnapshotStatus } from '@/components/query-table/snapshot-list'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { apiGetJobPodSnapshot, type SnapshotPod } from '@/services/api/snapshot'
import { PodDetail } from '@/services/api/vcjob'

import { configGrafanaJobAtom } from '@/utils/store/config'

import PodIngressDialog from './ingress'

interface PodTableProps {
  jobName: string
  userName: string
  jobType?: string
}

const getHeader = (key: string): string => {
  switch (key) {
    case 'name':
      return 'Pod 名称'
    case 'namespace':
      return '命名空间'
    case 'nodename':
      return '节点'
    case 'ip':
      return '内网 IP'
    case 'port':
      return '端口'
    case 'resource':
      return '申请资源'
    case 'phase':
      return '状态'
    default:
      return key
  }
}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索 Pod 名称',
    key: 'name',
  },
  filterOptions: [
    {
      key: 'phase',
      title: '状态',
      option: podPhases,
    },
  ],
  getHeader: getHeader,
}

const getPodMonitorUrl = (baseURL: string, pod: PodDetail) => {
  return `${baseURL}?orgId=1&refresh=5s&var-node_name=${pod.nodename}&var-pod_name=${pod.name}&var-gpu=All&from=now-15m&to=now`
}

export const PodTable = ({ jobName, userName, jobType }: PodTableProps) => {
  const queryClient = useQueryClient()
  const [showLog, setShowLog] = useState<NamespacedName>()
  const [showTerminal, setShowTerminal] = useState<NamespacedName>()
  const [showIngress, setShowIngress] = useState<NamespacedName>()
  const [showMonitor, setShowMonitor] = useState(false)
  const grafanaJob = useAtomValue(configGrafanaJobAtom)
  const [grafanaUrl, setGrafanaUrl] = useState<string>('')

  const podSnapshotQueryKey = ['job', 'detail', jobName, 'pods', 'snapshot'] as const
  const podSnapshotQuery = useInfiniteQuery({
    queryKey: podSnapshotQueryKey,
    initialPageParam: '',
    queryFn: ({ pageParam }) =>
      apiGetJobPodSnapshot(jobName, { limit: 50, cursor: pageParam || undefined }),
    getNextPageParam: (lastPage) => lastPage.data.next_cursor ?? undefined,
    enabled: !!jobName,
  })
  const pods = useMemo(
    () =>
      (podSnapshotQuery.data?.pages.flatMap((page) => page.data.items) ?? [])
        .map(toPodDetail)
        .sort((left, right) => left.name.localeCompare(right.name)),
    [podSnapshotQuery.data]
  )
  const query = {
    data: pods,
    error: podSnapshotQuery.error,
    isError: podSnapshotQuery.isError,
    isLoading: podSnapshotQuery.isLoading,
    dataUpdatedAt: podSnapshotQuery.dataUpdatedAt,
    refetch: podSnapshotQuery.refetch,
  }
  const observedAt = podSnapshotQuery.data?.pages[0]?.data.observed_at

  const columns: ColumnDef<PodDetail>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
        cell: ({ row }) => {
          const pod = row.original
          return pod.name ? (
            <TooltipButton
              name={pod.name}
              tooltipContent={`查看 Pod 监控`}
              className="text-foreground hover:text-primary cursor-pointer font-mono hover:no-underline"
              variant="link"
              onClick={() => {
                setGrafanaUrl(getPodMonitorUrl(grafanaJob.pod, pod))
                setShowMonitor(true)
              }}
            >
              {pod.name}
            </TooltipButton>
          ) : (
            '---'
          )
        },
      },
      {
        accessorKey: 'phase',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('phase')} />
        ),
        cell: ({ row }) => <PodPhaseLabel podPhase={row.getValue('phase')} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'ip',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('ip')} />,
        cell: ({ row }) => <TooltipCopy name={row.getValue('ip')} className="font-mono" />,
        enableSorting: false,
      },
      {
        accessorKey: 'nodename',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('nodename')} />
        ),
        cell: ({ row }) => <NodeBadges nodes={[row.getValue('nodename')]} />,
      },
      {
        accessorKey: 'resource',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('resource')} />
        ),
        cell: ({ row }) => (
          <div className="flex flex-row gap-1">
            <ResourceBadges resources={row.getValue('resource')} />
          </div>
        ),
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => {
          const pod = row.original
          return (
            <div className="flex items-center space-x-1.5">
              <TooltipButton
                variant="outline"
                size="icon"
                className="h-8 w-8"
                disabled={pod.phase !== 'Running'}
                onClick={() => {
                  setShowTerminal({
                    namespace: pod.namespace,
                    name: pod.name,
                  })
                }}
                tooltipContent="网页终端"
              >
                <TerminalIcon className="text-primary size-4" />
              </TooltipButton>

              <TooltipButton
                variant="outline"
                size="icon"
                className="h-8 w-8"
                disabled={pod.phase !== 'Running'}
                onClick={() => {
                  setShowIngress({
                    namespace: pod.namespace,
                    name: pod.name,
                  })
                }}
                tooltipContent="外部访问"
              >
                <EthernetPort className="text-highlight-purple size-4" />
              </TooltipButton>

              <TooltipButton
                variant="outline"
                size="icon"
                className="h-8 w-8"
                disabled={pod.phase !== 'Running'}
                onClick={() => {
                  setGrafanaUrl(getPodMonitorUrl(grafanaJob.pod, pod))
                  setShowMonitor(true)
                }}
                tooltipContent="资源监控"
              >
                <GaugeIcon className="text-highlight-green size-4" />
              </TooltipButton>

              <TooltipButton
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => {
                  setShowLog({
                    namespace: pod.namespace,
                    name: pod.name,
                  })
                }}
                tooltipContent="容器日志"
              >
                <LogsIcon className="text-highlight-orange size-4" />
              </TooltipButton>
            </div>
          )
        },
      },
    ],
    [grafanaJob.pod]
  )

  return (
    <>
      <DataTable
        storageKey="job_pods"
        toolbarConfig={toolbarConfig}
        query={query}
        columns={columns}
        briefChildren={<SnapshotStatus observedAt={observedAt} />}
        paginate={false}
        withI18n
        footer={
          <SnapshotControls
            hasMore={podSnapshotQuery.hasNextPage}
            isLoading={podSnapshotQuery.isFetchingNextPage}
            isRefreshing={podSnapshotQuery.isFetching && !podSnapshotQuery.isFetchingNextPage}
            onLoadMore={() => void podSnapshotQuery.fetchNextPage()}
            onRefresh={() =>
              void queryClient.resetQueries({ queryKey: podSnapshotQueryKey, exact: true })
            }
          />
        }
      />
      <LogDialog namespacedName={showLog} setNamespacedName={setShowLog} />
      <TerminalDialog namespacedName={showTerminal} setNamespacedName={setShowTerminal} />
      <PodIngressDialog
        namespacedName={showIngress}
        setNamespacedName={setShowIngress}
        userName={userName}
        jobType={jobType}
      />
      <Sheet open={showMonitor} onOpenChange={setShowMonitor}>
        <SheetContent className="sm:max-w-4xl">
          <SheetHeader>
            <SheetTitle>资源监控</SheetTitle>
          </SheetHeader>
          <div className="h-[calc(100vh-6rem)] w-full px-4">
            <GrafanaIframe baseSrc={grafanaUrl} />
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}

function toPodDetail(pod: SnapshotPod): PodDetail {
  return {
    name: pod.name,
    namespace: pod.namespace,
    nodename: pod.nodeName ?? '',
    ip: pod.ip,
    port: '',
    resource: pod.requestResources ?? pod.resources,
    phase: pod.status,
  }
}
