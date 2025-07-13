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

import { DataTableToolbarConfig } from '@/components/custom/DataTable/DataTableToolbar'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useState, type FC } from 'react'
import { ColumnDef } from '@tanstack/react-table'
import { DataTableColumnHeader } from '@/components/custom/DataTable/DataTableColumnHeader'
import { DataTable } from '@/components/custom/DataTable'
import {
  apiGetNodeDetail,
  apiGetNodePods,
  IClusterPodInfo,
  apiGetNodeGPU,
} from '@/services/api/cluster'
import { TimeDistance } from '@/components/custom/TimeDistance'
import { LockIcon } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { DotsHorizontalIcon } from '@radix-ui/react-icons'
import { Card, CardTitle, CardContent, CardFooter } from '@/components/ui/card'
import { useParams } from 'react-router-dom'
import {
  CpuIcon,
  MemoryStickIcon as Memory,
  Grid,
  Layers,
  Cable,
  ServerIcon,
  BotIcon,
  BoxIcon,
  GaugeIcon,
  NetworkIcon,
  GpuIcon,
} from 'lucide-react'
import useBreadcrumb from '@/hooks/useBreadcrumb'
import PodPhaseLabel, { podPhases } from '@/components/badge/PodPhaseBadge'
import { Separator } from '@/components/ui/separator'
import { Badge } from '../ui/badge'
import { NamespacedName, PodNamespacedName } from '../codeblock/PodContainerDialog'
import LogDialog from '../codeblock/LogDialog'
import ResourceBadges from '../badge/ResourceBadges'

import { DetailPage } from '../layout/DetailPage'
import PageTitle from '../layout/PageTitle'
import { GrafanaIframe } from '@/pages/Embed/Monitor'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import TooltipButton from '../custom/TooltipButton'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip'
import TipBadge from '../badge/TipBadge'
import { useAtomValue } from 'jotai'
import { configGrafanaJobAtom, configGrafanaNodeAtom } from '@/utils/store/config'
import useIsAdmin from '@/hooks/useAdmin'
import TooltipCopy from '../label/TooltipCopy'
import { globalSettings } from '@/utils/store'
import { zhCN } from 'date-fns/locale'
import { format } from 'date-fns'

const formatLockDate = (timestamp?: string) => {
  const date = new Date(timestamp ?? Date.now())
  return format(date, 'M月d日 HH:mm', { locale: zhCN })
}

type GpuDemoProps = React.ComponentProps<typeof Card> & {
  gpuInfo?: {
    nodeName: string | undefined
    haveGPU: boolean
    gpuCount: number
    gpuUtil: Record<string, number>
    relateJobs: string[]
    gpuMemory: string
    gpuArch: string
    gpuDriver: string
    cudaVersion: string
    gpuProduct: string
  }
}

export function GpuCardDemo({ gpuInfo }: GpuDemoProps) {
  const grafanaNode = useAtomValue(configGrafanaNodeAtom)
  if (!gpuInfo?.haveGPU) return null
  else
    return (
      <Card>
        <CardContent className="bg-muted/50 flex items-center justify-between p-6">
          <div className="flex flex-col items-start gap-2">
            <CardTitle className="text-primary text-lg font-bold">{gpuInfo?.gpuProduct}</CardTitle>
            <div className="mt-4 flex items-center space-x-2">
              <Badge variant="default">CUDA {gpuInfo?.cudaVersion}</Badge>
            </div>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="mt-6 grid grid-flow-col grid-rows-4 gap-x-2 gap-y-3 text-xs">
          <div className="text-muted-foreground flex items-center space-x-2">
            <Memory className="h-6 w-6" />
            <span className="text-sm font-bold">显存</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Grid className="h-6 w-6" />
            <span className="text-sm font-bold">GPU数量</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Layers className="h-6 w-6" />
            <span className="text-sm font-bold">架构</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Cable className="h-6 w-6" />
            <span className="text-sm font-bold">驱动版本</span>
          </div>
          <p className="text-lg font-bold">{parseInt(gpuInfo?.gpuMemory) / 1024} GB</p>
          <p className="text-lg font-bold">{gpuInfo?.gpuCount}</p>
          <p className="text-lg font-bold">{gpuInfo?.gpuArch}</p>
          <p className="text-lg font-bold">{gpuInfo?.gpuDriver}</p>
        </CardContent>
        <CardFooter className="flex justify-center">
          <Button
            variant="outline"
            onClick={() => {
              window.open(
                `${grafanaNode.nvidia}?from=now-30m&to=now&var-datasource=prometheus&var-host=${gpuInfo?.nodeName}&var-gpu=$__all&refresh=5s`
              )
            }}
          >
            <GpuIcon className="text-highlight-purple" />
            <span className="truncate font-normal">加速卡监控</span>
          </Button>
        </CardFooter>
      </Card>
    )
}

const getHeader = (name: string): string => {
  switch (name) {
    case 'type':
      return '类型'
    case 'name':
      return 'Pod 名称'
    case 'status':
      return '状态'
    case 'createTime':
      return '创建于'
    case 'resources':
      return '申请资源'
    default:
      return name
  }
}

const getColumns = (
  isAdminView: boolean,
  handleShowPodLog: (namespacedName: PodNamespacedName) => void,
  handleShowMonitor: (pod: IClusterPodInfo) => void
): ColumnDef<IClusterPodInfo>[] => [
  {
    accessorKey: 'type',
    header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('type')} />,
    cell: ({ row }) => {
      if (!row.getValue('type')) return null
      const splitValue = row.getValue<string>('type').split('/')
      const apiVersion = splitValue.slice(0, splitValue.length - 1).join('/')
      const kind = splitValue[splitValue.length - 1]
      return (
        <Badge variant="outline" className="cursor-help font-mono font-normal" title={apiVersion}>
          {kind}
        </Badge>
      )
    },
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id))
    },
  },
  {
    accessorKey: 'name',
    header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
    cell: ({ row }) => {
      const podName = row.getValue<string>('name')
      const locked = row.original.locked || false
      const permanentLocked = row.original.permanentLocked
      const lockedTimestamp = row.original.lockedTimestamp

      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <TooltipButton
                name={podName}
                tooltipContent={`查看 Pod 监控`}
                className="text-foreground hover:text-primary cursor-pointer font-mono hover:no-underline"
                variant="link"
                onClick={() => handleShowMonitor(row.original)}
              >
                {podName}
                {locked && <LockIcon className="text-muted-foreground ml-1 h-4 w-4" />}
              </TooltipButton>
            </TooltipTrigger>
            {locked && (
              <TooltipContent side="top">
                <div className="flex flex-row items-center justify-between gap-1.5">
                  <p className="text-xs">查看 {podName} 监控</p>
                  <TipBadge
                    title={
                      permanentLocked ? '长期锁定中' : `锁定至 ${formatLockDate(lockedTimestamp)}`
                    }
                    className="text-primary bg-primary-foreground z-10"
                  />
                </div>
              </TooltipContent>
            )}
          </Tooltip>
        </TooltipProvider>
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('status')} />,
    cell: ({ row }) => (
      <div className="flex flex-row items-center justify-start">
        <PodPhaseLabel podPhase={row.getValue('status')} />
      </div>
    ),
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id))
    },
  },
  {
    accessorKey: 'resources',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader('resources')} />
    ),
    cell: ({ row }) => {
      return (
        <ResourceBadges
          namespace={row.original.namespace}
          podName={row.original.name}
          resources={row.getValue('resources')}
          showEdit={true}
        />
      )
    },
  },
  {
    accessorKey: 'createTime',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader('createTime')} />
    ),
    cell: ({ row }) => {
      return <TimeDistance date={row.getValue('createTime')}></TimeDistance>
    },
    enableSorting: false,
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const taskInfo = row.original
      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">操作</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-muted-foreground text-xs">操作</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => handleShowMonitor(taskInfo)}>监控</DropdownMenuItem>
            {isAdminView && (
              <DropdownMenuItem
                onClick={() =>
                  handleShowPodLog({
                    namespace: taskInfo.namespace,
                    name: taskInfo.name,
                  })
                }
              >
                日志
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )
    },
  },
]

export const NodeDetail: FC = () => {
  const { id: nodeName } = useParams()
  const setBreadcrumb = useBreadcrumb()
  const grafanaJob = useAtomValue(configGrafanaJobAtom)
  const grafanaNode = useAtomValue(configGrafanaNodeAtom)
  const [showLogPod, setShowLogPod] = useState<NamespacedName>()
  const [showMonitor, setShowMonitor] = useState(false)
  const [grafanaUrl, setGrafanaUrl] = useState<string>(grafanaJob.pod)

  const { data: nodeDetail } = useQuery({
    queryKey: ['nodes', nodeName, 'detail'],
    queryFn: () => apiGetNodeDetail(`${nodeName}`),
    select: (res) => res.data.data,
    enabled: !!nodeName,
  })

  const { data: gpuDetail } = useQuery({
    queryKey: ['gpu', nodeName, 'detail'],
    queryFn: () => apiGetNodeGPU(`${nodeName}`),
    select: (res) => res.data.data,
  })

  const podsQuery = useQuery({
    queryKey: ['nodes', nodeName, 'pods'],
    queryFn: () => apiGetNodePods(`${nodeName}`),
    select: (res) =>
      res.data.data
        ?.sort((a, b) => a.name.localeCompare(b.name))
        .map((p) => {
          if (p.ownerReference && p.ownerReference.length > 0) {
            p.type = `${p.ownerReference[0].apiVersion}/${p.ownerReference[0].kind}`
          }
          return p
        }),
    enabled: !!nodeName,
  })

  const isAdminView = useIsAdmin()
  const columns = useMemo(
    () =>
      getColumns(isAdminView, setShowLogPod, (pod) => {
        setGrafanaUrl(
          `${grafanaJob.pod}?orgId=1&var-node_name=${nodeName}&var-pod_name=${pod.name}&from=now-1h&to=now`
        )
        setShowMonitor(true)
      }),
    [nodeName, grafanaJob, isAdminView]
  )

  const scheduler = useAtomValue(globalSettings).scheduler
  const toolbarConfig: DataTableToolbarConfig = useMemo(() => {
    return {
      filterInput: {
        placeholder: '搜索 Pod 名称',
        key: 'name',
      },
      filterOptions: [
        {
          key: 'status',
          title: '状态',
          option: podPhases,
          defaultValues: ['Running'],
        },
        {
          key: 'type',
          title: '类型',
          option: [
            {
              value: 'batch.volcano.sh/v1alpha1/Job',
              label: 'BASE',
            },
            {
              value: 'aisystem.github.com/v1alpha1/AIJob',
              label: 'EMIAS',
            },
          ],
          defaultValues: [
            scheduler === 'volcano'
              ? 'batch.volcano.sh/v1alpha1/Job'
              : 'aisystem.github.com/v1alpha1/AIJob',
          ],
        },
      ],
      getHeader: getHeader,
    }
  }, [scheduler])

  // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: nodeName ?? '' }])
  }, [setBreadcrumb, nodeName])

  if (!nodeDetail || !gpuDetail) return null

  return (
    <DetailPage
      header={<PageTitle title={nodeDetail?.name} description={gpuDetail?.gpuProduct} />}
      info={[
        {
          icon: ServerIcon,
          title: '操作系统',
          value: <span className="font-mono">{nodeDetail?.osVersion}</span>,
        },
        {
          icon: Grid,
          title: '架构',
          value: <span className="font-mono uppercase">{nodeDetail?.arch}</span>,
        },
        {
          icon: NetworkIcon,
          title: 'IP 地址',
          value: <TooltipCopy name={nodeDetail?.address} className="font-mono" />,
        },
        {
          icon: BotIcon,
          title: '角色',
          value: <span className="font-mono capitalize">{nodeDetail?.role}</span>,
        },
        {
          icon: CpuIcon,
          title: 'Kubelet 版本',
          value: <span className="font-mono">{nodeDetail?.kubeletVersion}</span>,
        },
        {
          icon: Layers,
          title: '容器运行时',
          value: <span className="font-mono">{nodeDetail?.containerRuntimeVersion}</span>,
        },
      ]}
      tabs={[
        {
          key: 'pods',
          icon: BoxIcon,
          label: '节点负载',
          children: (
            <>
              <DataTable
                storageKey="node_pods"
                query={podsQuery}
                columns={columns}
                toolbarConfig={toolbarConfig}
              />
              <LogDialog namespacedName={showLogPod} setNamespacedName={setShowLogPod} />
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
          ),
          scrollable: true,
        },
        {
          key: 'base',
          icon: GaugeIcon,
          label: '基础监控',
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaNode.basic}?from=now-1h&to=now&var-datasource=prometheus&var-cluster=&var-resolution=30s&var-node=${nodeName}`}
            />
          ),
        },
        {
          key: 'gpu',
          icon: GpuIcon,
          label: '加速卡监控',
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaNode.nvidia}?from=now-30m&to=now&var-datasource=prometheus&var-host=${nodeName}&var-gpu=$__all&refresh=5s`}
            />
          ),
          hidden: !gpuDetail?.haveGPU,
        },
      ]}
    />
  )
}

export default NodeDetail
