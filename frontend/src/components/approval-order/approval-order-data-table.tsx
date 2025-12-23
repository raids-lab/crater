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
import { UseQueryResult, useQuery } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { AlertTriangle, ClockIcon } from 'lucide-react'
import { ReactNode, useMemo } from 'react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import {
  ApprovalOrderStatus,
  ApprovalOrderStatusBadge,
  ApprovalOrderTypeBadge,
  approvalOrderStatuses,
  approvalOrderTypes,
} from '@/components/badge/approvalorder-badge'
import NodeBadges from '@/components/badge/node-badges'
import NodeStatusBadge from '@/components/badge/node-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import UserLabel from '@/components/label/user-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { type ApprovalOrder } from '@/services/api/approvalorder'
import { NodeStatus } from '@/services/api/cluster'
import { PodDetail, apiJobGetPods } from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'

export interface ApprovalOrderDataTableProps {
  query: UseQueryResult<ApprovalOrder[]>
  storageKey: string
  info?: {
    title: string
    description: string
  }
  showExtensionHours?: boolean
  onNameClick?: (order: ApprovalOrder) => void
  getHeader?: (key: string) => string
  renderActions?: (order: ApprovalOrder) => ReactNode
  children?: ReactNode
}

// 提取公共的 Pod 查询 Hook
const useJobPods = (order: ApprovalOrder) => {
  return useQuery({
    queryKey: ['job', 'detail', order.name, 'pods'],
    queryFn: () => apiJobGetPods(order.name),
    select: (res) => res.data,
    enabled: order.type === 'job' && order.status === ApprovalOrderStatus.Pending,
    staleTime: 1000 * 60, // 1分钟缓存
  })
}

const JobNameWithWarning = ({
  order,
  showExtensionHours,
  onNameClick,
  pods,
  nodes,
}: {
  order: ApprovalOrder
  showExtensionHours: boolean
  onNameClick?: (order: ApprovalOrder) => void
  pods?: PodDetail[]
  nodes?: { name: string; status: NodeStatus }[]
}) => {
  const extHours = showExtensionHours ? order.content.approvalorderExtensionHours || 0 : 0

  const abnormalNodes = useMemo(() => {
    if (order.type !== 'job' || order.status !== ApprovalOrderStatus.Pending) return []
    if (!pods || pods.length === 0) return []
    if (!nodes) return []

    const podNodeNames = pods.map((pod) => pod.nodename).filter(Boolean)
    if (podNodeNames.length === 0) return []

    const jobNodes = nodes.filter((node) => podNodeNames.includes(node.name))
    if (jobNodes.length === 0) return []

    return jobNodes.filter((node) => node.status !== NodeStatus.Ready)
  }, [order.type, order.status, pods, nodes])

  const resources = pods?.[0]?.resource
  const nodeNames = pods?.map((p) => p.nodename).filter(Boolean)
  const uniqueNodes = Array.from(new Set(nodeNames))

  return (
    <div className="relative flex items-center gap-2">
      <button
        type="button"
        className="text-left break-all whitespace-normal underline-offset-4 hover:underline"
        title={`查看工单 ${order.name} 详情`}
        onClick={() => onNameClick?.(order)}
      >
        <span className="mr-2">工单 {order.id}:</span>
        {order.name}
      </button>
      {showExtensionHours && order.type === 'job' && Number(extHours) > 0 && (
        <div
          title={`锁定 ${extHours} 小时`}
          className="bg-warning/10 text-warning inline-flex items-center gap-1 rounded px-2 py-1 text-xs"
        >
          <ClockIcon className="h-3 w-3" />
          {extHours}h
        </div>
      )}
      {order.type === 'job' && order.status === ApprovalOrderStatus.Pending && (
        <>
          <ResourceBadges resources={resources} />
          <NodeBadges nodes={uniqueNodes} />
        </>
      )}
      {abnormalNodes.length > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="text-destructive cursor-help">
                <AlertTriangle className="h-4 w-4" />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <div className="flex flex-col gap-2 p-1">
                <div className="font-semibold">节点状态异常:</div>
                {abnormalNodes.map((node) => (
                  <div key={node.name} className="flex items-center justify-between gap-4">
                    <span>{node.name}</span>
                    <NodeStatusBadge status={node.status} />
                  </div>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  )
}

// 包装组件，负责获取数据并分发给子组件
const JobInfoWrapper = ({
  order,
  children,
}: {
  order: ApprovalOrder
  children: (props: { pods?: PodDetail[]; isLoading: boolean }) => ReactNode
}) => {
  const { data: pods, isLoading } = useJobPods(order)
  return <>{children({ pods, isLoading })}</>
}

export function ApprovalOrderDataTable({
  query,
  storageKey,
  info,
  showExtensionHours = false,
  onNameClick,
  getHeader,
  renderActions,
  children,
}: ApprovalOrderDataTableProps) {
  // 在顶层获取节点信息，避免每行重复请求
  const { data: nodes } = useQuery(queryNodes())

  const defaultGetHeader = (key: string): string => {
    switch (key) {
      case 'name':
        return '名称'
      case 'type':
        return '类型'
      case 'status':
        return '状态'
      case 'creator':
        return '申请人'
      case 'reviewer':
        return '审核人'
      case 'createdAt':
        return '创建于'
      case 'actions':
        return '操作'
      default:
        return key
    }
  }

  const toolbarConfig: DataTableToolbarConfig = {
    globalSearch: {
      enabled: true,
    },
    filterOptions: [
      {
        key: 'type',
        title: (getHeader || defaultGetHeader)('type'),
        option: approvalOrderTypes,
      },
      {
        key: 'status',
        title: (getHeader || defaultGetHeader)('status'),
        option: approvalOrderStatuses,
      },
    ],
    getHeader: getHeader || defaultGetHeader,
  }

  const columns: ColumnDef<ApprovalOrder>[] = [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('name')} />
      ),
      cell: ({ row }) => (
        <JobInfoWrapper order={row.original}>
          {({ pods }) => (
            <JobNameWithWarning
              order={row.original}
              showExtensionHours={showExtensionHours}
              onNameClick={onNameClick}
              pods={pods}
              nodes={nodes}
            />
          )}
        </JobInfoWrapper>
      ),
    },
    {
      accessorKey: 'creator',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('creator')} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.creator} />,
    },
    {
      accessorKey: 'reviewer',
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={(getHeader || defaultGetHeader)('reviewer')}
        />
      ),
      cell: ({ row }) => {
        const { status, reviewer } = row.original
        const hasReviewer = reviewer && reviewer.username

        if (status !== ApprovalOrderStatus.Pending && !hasReviewer) {
          return <span className="truncate text-sm font-normal">系统</span>
        }
        return hasReviewer ? <UserLabel info={reviewer} /> : null
      },
    },
    {
      accessorKey: 'type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('type')} />
      ),
      cell: ({ row }) => {
        return <ApprovalOrderTypeBadge type={row.getValue('type')} />
      },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('status')} />
      ),
      cell: ({ row }) => {
        return <ApprovalOrderStatusBadge status={row.getValue('status')} />
      },
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={(getHeader || defaultGetHeader)('createdAt')}
        />
      ),
      cell: ({ row }) => <TimeDistance date={row.getValue('createdAt')} />,
      sortingFn: 'datetime',
    },
  ]

  // 如果提供了 renderActions，添加操作列
  if (renderActions) {
    columns.push({
      id: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('actions')} />
      ),
      cell: ({ row }) => renderActions(row.original),
    })
  }

  return (
    <DataTable
      info={info}
      toolbarConfig={toolbarConfig}
      storageKey={storageKey}
      query={query}
      columns={columns}
      initialColumnVisibility={{ reviewer: false }}
    >
      {children}
    </DataTable>
  )
}
