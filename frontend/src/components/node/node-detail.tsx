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
import { useNavigate } from '@tanstack/react-router'
import { Column, ColumnDef, Row } from '@tanstack/react-table'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { useAtomValue } from 'jotai'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { LockIcon } from 'lucide-react'
import {
  BotIcon,
  BoxIcon,
  Cable,
  CpuIcon,
  GaugeIcon,
  GpuIcon,
  Grid,
  InfoIcon,
  Layers,
  MemoryStickIcon as Memory,
  NetworkIcon,
  ServerIcon,
  TagIcon,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import PodPhaseLabel, { podPhases } from '@/components/badge/pod-phase-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  IClusterPodInfo,
  apiAdminGetNodePods,
  apiGetNodeDetail,
  apiGetNodeGPU,
  apiGetNodePods,
} from '@/services/api/cluster'

import useIsAdmin from '@/hooks/use-admin'

import { globalSettings } from '@/utils/store'
import { configGrafanaJobAtom, configGrafanaNodeAtom } from '@/utils/store/config'

import ResourceBadges from '../badge/resource-badges'
import TipBadge from '../badge/tip-badge'
import TooltipButton from '../button/tooltip-button'
import LogDialog from '../codeblock/log-dialog'
import { NamespacedName, PodNamespacedName } from '../codeblock/pod-container-dialog'
import SimpleTooltip from '../label/simple-tooltip'
import TooltipCopy from '../label/tooltop-copy'
import DetailPage, { DetailPageCoreProps } from '../layout/detail-page'
import PageTitle from '../layout/page-title'
import NodeInfoTab from './node-info-tab'
import { NodeAnnotations, NodeLabels, NodeTaints } from './node-mark'

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
  const { t } = useTranslation()
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
            <span className="text-sm font-bold">{t('nodeDetail.gpu.memory')}</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Grid className="h-6 w-6" />
            <span className="text-sm font-bold">{t('nodeDetail.gpu.count')}</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Layers className="h-6 w-6" />
            <span className="text-sm font-bold">{t('nodeDetail.gpu.architecture')}</span>
          </div>

          <div className="text-muted-foreground flex items-center space-x-2">
            <Cable className="h-6 w-6" />
            <span className="text-sm font-bold">{t('nodeDetail.gpu.driverVersion')}</span>
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
            <span className="truncate font-normal">{t('nodeDetail.gpu.monitoring')}</span>
          </Button>
        </CardFooter>
      </Card>
    )
}

const getHeader = (name: string, t: (key: string) => string): string => {
  switch (name) {
    case 'type':
      return t('nodeDetail.table.headers.type')
    case 'name':
      return t('nodeDetail.table.headers.name')
    case 'userName':
      return t('nodeDetail.table.headers.userName')
    case 'status':
      return t('nodeDetail.table.headers.status')
    case 'createTime':
      return t('nodeDetail.table.headers.createTime')
    case 'resources':
      return t('nodeDetail.table.headers.resources')
    default:
      return name
  }
}

const getColumns = (
  isAdminView: boolean,
  handleShowPodLog: (namespacedName: PodNamespacedName) => void,
  handleShowMonitor: (pod: IClusterPodInfo) => void,
  t: (key: string) => string,
  navigate: ReturnType<typeof useNavigate>
): ColumnDef<IClusterPodInfo>[] => [
  {
    accessorKey: 'type',
    header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('type', t)} />,
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
    accessorKey: 'namespace',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader('namespace', t)} />
    ),
    cell: ({ row }) => {
      return (
        <Badge variant="outline" className="font-mono font-normal">
          {row.getValue<string>('namespace')}
        </Badge>
      )
    },
  },
  {
    accessorKey: 'name',
    header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name', t)} />,
    cell: ({ row }) => {
      const podName = row.getValue<string>('name')
      const locked = row.original.locked || false
      const permanentLocked = row.original.permanentLocked
      const lockedTimestamp = row.original.lockedTimestamp

      // 从 ownerReference 获取作业名称
      const jobOwner = row.original.ownerReference?.find((owner) => owner.kind === 'Job')
      const jobName = jobOwner?.name
      const hasJobName = !!jobName

      return (
        <div className="flex flex-col items-start gap-0 pt-1">
          {hasJobName ? (
            // 有作业名称时，显示作业名称和 Pod 名称
            <>
              {/* 作业名称和锁图标使用同一个 tooltip */}
              {isAdminView ? (
                <SimpleTooltip
                  tooltip={
                    <div className="flex flex-row items-center justify-between gap-1.5">
                      <p className="text-xs">{t('nodeDetail.tooltip.viewJobDetail')}</p>
                      {locked && (
                        <TipBadge
                          title={
                            permanentLocked
                              ? t('nodeDetail.status.permanentLocked')
                              : t('nodeDetail.status.lockedUntil').replace(
                                  '{{time}}',
                                  formatLockDate(lockedTimestamp)
                                )
                          }
                          className="text-primary bg-primary-foreground z-10"
                        />
                      )}
                    </div>
                  }
                >
                  <div className="flex flex-row items-center leading-none">
                    <span
                      onClick={() =>
                        navigate({
                          to: '/admin/jobs/$name',
                          params: { name: jobName },
                        })
                      }
                      className="text-foreground hover:text-primary cursor-pointer px-0 font-mono leading-none hover:no-underline"
                    >
                      {jobName}
                    </span>
                    {locked && <LockIcon className="text-muted-foreground ml-1 h-4 w-4" />}
                  </div>
                </SimpleTooltip>
              ) : (
                <div className="flex flex-row items-center leading-none">
                  <span className="text-foreground font-mono leading-none">{jobName}</span>
                  {locked && <LockIcon className="text-muted-foreground ml-1 h-4 w-4" />}
                </div>
              )}
              {/* Pod 名称使用 SimpleTooltip + span，与归属列保持一致 */}
              <SimpleTooltip tooltip={t('nodeDetail.tooltip.viewMonitor')}>
                <span
                  className="text-muted-foreground hover:text-primary -mt-0.5 cursor-pointer font-mono text-xs leading-none hover:no-underline"
                  onClick={() => handleShowMonitor(row.original)}
                >
                  {podName}
                </span>
              </SimpleTooltip>
            </>
          ) : (
            // 没有作业名称时，Pod 名称使用 TooltipButton（没有作业名称时不会有锁定状态）
            <TooltipButton
              name={podName}
              tooltipContent={t('nodeDetail.tooltip.viewMonitor')}
              className="text-foreground hover:text-primary cursor-pointer px-0 font-mono hover:no-underline has-[>svg]:px-0"
              variant="link"
              onClick={() => handleShowMonitor(row.original)}
            >
              {podName}
            </TooltipButton>
          )}
        </div>
      )
    },
    enableSorting: false,
  },
  // 管理员视图：用户列
  ...(isAdminView
    ? [
        {
          accessorKey: 'userName',
          header: ({ column }: { column: Column<IClusterPodInfo> }) => (
            <DataTableColumnHeader column={column} title={getHeader('userName', t)} />
          ),
          cell: ({ row }: { row: Row<IClusterPodInfo> }) => {
            const userName = row.original.userName
            const userID = row.original.userID
            const userRealName = row.original.userRealName
            const accountName = row.original.accountName
            const accountID = row.original.accountID
            const accountRealName = row.original.accountRealName

            // 仅当是用户作业时才显示用户和账户信息（后端只在用户作业时设置 userName）
            if (userName) {
              return (
                <div className="flex flex-col items-start gap-0 pt-1">
                  {/* 用户信息 */}
                  {userID && userRealName ? (
                    <SimpleTooltip
                      tooltip={
                        <p>
                          <Trans
                            i18nKey="nodeDetail.tooltip.viewUserInfo"
                            values={{ userName, userRealName }}
                            components={{ 1: <span className="mx-0.5 font-mono" /> }}
                          />
                        </p>
                      }
                    >
                      <div className="flex flex-row items-center leading-none">
                        <span
                          onClick={() =>
                            navigate({
                              to: '/admin/users/$name',
                              params: { name: userRealName },
                            })
                          }
                          className="text-foreground hover:text-primary cursor-pointer px-0 font-mono leading-none hover:no-underline"
                        >
                          {userName}
                        </span>
                      </div>
                    </SimpleTooltip>
                  ) : (
                    <div className="flex flex-row items-center leading-none">
                      <span className="text-foreground font-mono leading-none">{userName}</span>
                    </div>
                  )}
                  {/* 账户信息 */}
                  {accountName && accountID ? (
                    <SimpleTooltip
                      tooltip={
                        <p>
                          <Trans
                            i18nKey="nodeDetail.tooltip.manageAccount"
                            values={{ accountName, accountRealName }}
                            components={{ 1: <span className="mx-0.5 font-mono" /> }}
                          />
                        </p>
                      }
                    >
                      <span
                        onClick={() =>
                          navigate({
                            to: '/admin/accounts/$id',
                            params: { id: accountID.toString() },
                          })
                        }
                        className="text-muted-foreground hover:text-primary -mt-0.5 cursor-pointer font-mono text-xs leading-none hover:no-underline"
                      >
                        {accountName}
                      </span>
                    </SimpleTooltip>
                  ) : accountName ? (
                    <span className="text-muted-foreground -mt-0.5 font-mono text-xs leading-none">
                      {accountName}
                    </span>
                  ) : null}
                </div>
              )
            }

            // 其他情况不显示
            return null
          },
          enableSorting: false,
        },
      ]
    : []),
  {
    accessorKey: 'status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader('status', t)} />
    ),
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
      <DataTableColumnHeader column={column} title={getHeader('resources', t)} />
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
      <DataTableColumnHeader column={column} title={getHeader('createTime', t)} />
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
              <span className="sr-only">{t('nodeDetail.actions.operations')}</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-muted-foreground text-xs">
              {t('nodeDetail.actions.operations')}
            </DropdownMenuLabel>
            <DropdownMenuItem onClick={() => handleShowMonitor(taskInfo)}>
              {t('nodeDetail.actions.monitor')}
            </DropdownMenuItem>
            {isAdminView && (
              <DropdownMenuItem
                onClick={() =>
                  handleShowPodLog({
                    namespace: taskInfo.namespace,
                    name: taskInfo.name,
                  })
                }
              >
                {t('nodeDetail.actions.logs')}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )
    },
  },
]

type NodeDetailProps = DetailPageCoreProps & {
  nodeName: string
}

export const NodeDetail = ({ nodeName, ...props }: NodeDetailProps) => {
  const grafanaJob = useAtomValue(configGrafanaJobAtom)
  const grafanaNode = useAtomValue(configGrafanaNodeAtom)
  const [showLogPod, setShowLogPod] = useState<NamespacedName>()
  const [showMonitor, setShowMonitor] = useState(false)
  const [grafanaUrl, setGrafanaUrl] = useState<string>(grafanaJob.pod)
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data: nodeDetail } = useQuery({
    queryKey: ['nodes', nodeName, 'detail'],
    queryFn: () => apiGetNodeDetail(`${nodeName}`),
    select: (res) => res.data,
    enabled: !!nodeName,
  })

  const { data: gpuDetail } = useQuery({
    queryKey: ['gpu', nodeName, 'detail'],
    queryFn: () => apiGetNodeGPU(`${nodeName}`),
    select: (res) => res.data,
  })

  const isAdminView = useIsAdmin()
  const podsQuery = useQuery({
    queryKey: ['nodes', nodeName, 'pods', isAdminView],
    queryFn: () =>
      isAdminView ? apiAdminGetNodePods(`${nodeName}`) : apiGetNodePods(`${nodeName}`),
    select: (res) =>
      res.data
        ?.sort((a, b) => a.name.localeCompare(b.name))
        .map((p) => {
          if (p.ownerReference && p.ownerReference.length > 0) {
            p.type = `${p.ownerReference[0].apiVersion}/${p.ownerReference[0].kind}`
          }
          return p
        }),
    enabled: !!nodeName,
  })
  const columns = useMemo(
    () =>
      getColumns(
        isAdminView,
        setShowLogPod,
        (pod) => {
          setGrafanaUrl(
            `${grafanaJob.pod}?orgId=1&var-node_name=${nodeName}&var-pod_name=${pod.name}&from=now-1h&to=now`
          )
          setShowMonitor(true)
        },
        t,
        navigate
      ),
    [nodeName, grafanaJob, isAdminView, t, navigate]
  )

  const scheduler = useAtomValue(globalSettings).scheduler

  const namespaces = useMemo(() => {
    return (
      podsQuery.data
        ?.reduce((acc, pod) => {
          if (pod.namespace && !acc.includes(pod.namespace)) {
            acc.push(pod.namespace)
          }
          return acc
        }, [] as string[])
        .map((namespace) => ({
          value: namespace,
          label: namespace,
        })) || []
    )
  }, [podsQuery.data])

  const toolbarConfig: DataTableToolbarConfig = useMemo(() => {
    return {
      filterInput: {
        placeholder: t('nodeDetail.table.filter.searchPodName'),
        key: 'name',
      },
      filterOptions: [
        {
          key: 'namespace',
          title: 'Namespace',
          option: namespaces,
          defaultValues: [],
        },
        {
          key: 'status',
          title: t('nodeDetail.table.filter.status'),
          option: podPhases,
          defaultValues: ['Running', 'Pending'],
        },
        {
          key: 'type',
          title: t('nodeDetail.table.filter.type'),
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
      getHeader: (key: string) => getHeader(key, t),
    }
  }, [namespaces, scheduler, t])

  if (!nodeDetail) return null

  return (
    <DetailPage
      {...props}
      header={<PageTitle title={nodeDetail?.name} description={gpuDetail?.gpuProduct} />}
      info={[
        {
          icon: ServerIcon,
          title: t('nodeDetail.info.operatingSystem'),
          value: <span className="font-mono">{nodeDetail?.osVersion}</span>,
        },
        {
          icon: Grid,
          title: t('nodeDetail.info.architecture'),
          value: <span className="font-mono uppercase">{nodeDetail?.arch}</span>,
        },
        {
          icon: NetworkIcon,
          title: t('nodeDetail.info.ipAddress'),
          value: <TooltipCopy name={nodeDetail?.address} className="font-mono" />,
        },
        {
          icon: BotIcon,
          title: t('nodeDetail.info.role'),
          value: <span className="font-mono capitalize">{nodeDetail?.role}</span>,
        },
        {
          icon: CpuIcon,
          title: t('nodeDetail.info.kubeletVersion'),
          value: <span className="font-mono">{nodeDetail?.kubeletVersion}</span>,
        },
        {
          icon: Layers,
          title: t('nodeDetail.info.containerRuntime'),
          value: <span className="font-mono">{nodeDetail?.containerRuntimeVersion}</span>,
        },
      ]}
      tabs={[
        {
          key: 'info',
          icon: InfoIcon,
          label: t('nodeDetail.tabs.nodeInfo'),
          children: <NodeInfoTab nodeDetail={nodeDetail} gpuDetail={gpuDetail} />,
          scrollable: true,
        },
        {
          key: 'pods',
          icon: BoxIcon,
          label: t('nodeDetail.tabs.nodeLoad'),
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
                    <SheetTitle>{t('nodeDetail.monitor.title')}</SheetTitle>
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
          label: t('nodeDetail.tabs.basicMonitoring'),
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaNode.basic}?from=now-1h&to=now&var-datasource=prometheus&var-cluster=&var-resolution=30s&var-node=${nodeName}`}
            />
          ),
        },
        {
          key: 'gpu',
          icon: GpuIcon,
          label: t('nodeDetail.tabs.acceleratorMonitoring'),
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaNode.nvidia}?from=now-30m&to=now&var-datasource=prometheus&var-host=${nodeName}&var-gpu=$__all&refresh=5s`}
            />
          ),
          hidden: !gpuDetail?.haveGPU,
        },
        {
          key: 'nodemark',
          icon: TagIcon,
          label: '节点标识',
          subTabs: [
            {
              key: 'labels',
              label: 'Labels',
              children: <NodeLabels nodeName={nodeName} />,
            },
            {
              key: 'annotations',
              label: 'Annotations',
              children: <NodeAnnotations nodeName={nodeName} />,
            },
            {
              key: 'taints',
              label: 'Taints',
              children: <NodeTaints nodeName={nodeName} />,
            },
          ],
          scrollable: true,
          hidden: !isAdminView,
        },
      ]}
    />
  )
}

export default NodeDetail
