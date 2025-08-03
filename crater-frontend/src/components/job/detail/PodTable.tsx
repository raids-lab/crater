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
import { ColumnDef } from '@tanstack/react-table'
import { useAtomValue } from 'jotai'
import { EthernetPort, GaugeIcon, LogsIcon, SquareTerminalIcon, TerminalIcon } from 'lucide-react'
import { useMemo, useState } from 'react'

import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import NodeBadges from '@/components/badge/NodeBadges'
import PodPhaseLabel, { podPhases } from '@/components/badge/PodPhaseBadge'
import ResourceBadges from '@/components/badge/ResourceBadges'
import TooltipButton from '@/components/button/tooltip-button'
import DetailPageLog from '@/components/codeblock/DetailPageLog'
import LogDialog from '@/components/codeblock/LogDialog'
import { NamespacedName } from '@/components/codeblock/PodContainerDialog'
import TerminalDialog from '@/components/codeblock/TerminalDialog'
import TooltipCopy from '@/components/label/tooltop-copy'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'
import Nothing from '@/components/placeholder/nothing'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { PodDetail, apiJobGetPods } from '@/services/api/vcjob'

import { configGrafanaJobAtom } from '@/utils/store/config'

import PodIngressDialog from './Ingress'

interface PodTableProps {
  jobName: string
  userName: string
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

const PodInfo = ({
  pod,
  setShowTerminal,
  setShowIngress,
}: {
  pod: PodDetail
  setShowTerminal: (namespacedName: NamespacedName) => void
  setShowIngress: (namespacedName: NamespacedName) => void
}) => {
  return (
    <DetailPageLog
      namespacedName={{
        name: pod.name,
        namespace: pod.namespace,
      }}
      appendInfos={[
        {
          title: '内网 IP',
          content: <TooltipCopy name={pod.ip} className="font-mono" />,
        },
        {
          title: '节点',
          content: <NodeBadges nodes={[pod.nodename]} />,
        },
      ]}
      children={
        <div className="grid h-8 grid-cols-2 gap-2">
          <TooltipButton
            variant="outline"
            className="cursor-pointer"
            disabled={pod.phase !== 'Running'}
            onClick={() => {
              setShowTerminal({
                namespace: pod.namespace,
                name: pod.name,
              })
            }}
            tooltipContent="网页终端"
          >
            <SquareTerminalIcon className="text-primary size-4" />
            网页终端
          </TooltipButton>
          <TooltipButton
            variant="outline"
            className="cursor-pointer"
            disabled={pod.phase !== 'Running'}
            onClick={() => {
              setShowIngress({
                namespace: pod.namespace,
                name: pod.name,
              })
            }}
            tooltipContent="外部访问"
          >
            <EthernetPort className="text-primary size-4" />
            外部访问
          </TooltipButton>
        </div>
      }
    />
  )
}

export const PodTable = ({ jobName, userName }: PodTableProps) => {
  const [showLog, setShowLog] = useState<NamespacedName>()
  const [showTerminal, setShowTerminal] = useState<NamespacedName>()
  const [showIngress, setShowIngress] = useState<NamespacedName>()
  const [showMonitor, setShowMonitor] = useState(false)
  const grafanaJob = useAtomValue(configGrafanaJobAtom)
  const [grafanaUrl, setGrafanaUrl] = useState<string>('')

  const query = useQuery({
    queryKey: ['job', 'detail', jobName, 'pods'],
    queryFn: () => apiJobGetPods(jobName),
    select: (res) => res.data.sort((a, b) => a.name.localeCompare(b.name)),
    enabled: !!jobName,
  })

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

  if (query.isLoading) {
    return <></>
  }

  if (!query.data || query.data.length === 0) {
    return <Nothing title="暂无作业信息，可查看作业事件" />
  }

  return (
    <>
      {query.data.length === 1 ? (
        <PodInfo
          pod={query.data[0]}
          setShowTerminal={setShowTerminal}
          setShowIngress={setShowIngress}
        />
      ) : (
        <DataTable
          storageKey="job_pods"
          toolbarConfig={toolbarConfig}
          query={query}
          columns={columns}
        />
      )}
      <LogDialog namespacedName={showLog} setNamespacedName={setShowLog} />
      <TerminalDialog namespacedName={showTerminal} setNamespacedName={setShowTerminal} />
      <PodIngressDialog
        namespacedName={showIngress}
        setNamespacedName={setShowIngress}
        userName={userName}
        jobName={jobName}
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
