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
import { linkOptions } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { type FC, useMemo } from 'react'

import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import NodeStatusBadge, { nodeStatuses } from '@/components/badge/node-status-badge'
import TooltipLink from '@/components/label/tooltip-link'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
import { ProgressBar, progressTextColor } from '@/components/ui-custom/colorful-progress'

import { INodeBriefInfo, NodeStatus } from '@/services/api/cluster'

import {
  V1ResourceList,
  betterResourceQuantity,
  convertKResourceToResource,
} from '@/utils/resource'

import { cn } from '@/lib/utils'

import AcceleratorBadge from '../badge/accelerator-badge'

// 资源使用情况计算结果接口
export interface ResourceUsageInfo {
  usagePercent: number | null
  displayValue: string | null
  acceleratorName?: string
}

// 计算资源使用情况的帮助函数
export function calculateResourceUsage(
  resourceKey: 'cpu' | 'memory' | 'accelerator',
  used?: V1ResourceList,
  allocatable?: V1ResourceList,
  accelerators?: string[]
): ResourceUsageInfo {
  let resourceUsed = ''
  let resourceAllocatable = ''
  let acceleratorName = ''

  switch (resourceKey) {
    case 'cpu':
      resourceUsed = used?.['cpu'] || '0'
      resourceAllocatable = allocatable?.['cpu'] || '0'
      break
    case 'memory':
      resourceUsed = used?.memory ?? '0'
      resourceAllocatable = allocatable?.memory ?? '0'
      break
    case 'accelerator':
      if (accelerators && accelerators.length > 0) {
        for (const accelerator of accelerators) {
          if (used && used[accelerator]) {
            resourceUsed = used[accelerator]
          }
          if (allocatable && allocatable[accelerator]) {
            resourceAllocatable = allocatable[accelerator]
            acceleratorName = accelerator
          }
        }
      } else {
        return {
          usagePercent: null,
          displayValue: null,
        }
      }
      break
    default:
      return {
        usagePercent: null,
        displayValue: null,
      }
  }

  const usedValue = convertKResourceToResource(resourceKey, resourceUsed) || 0
  const allocatableValue = convertKResourceToResource(resourceKey, resourceAllocatable)

  if (allocatableValue === undefined || allocatableValue === 0) {
    return {
      usagePercent: null,
      displayValue: null,
      acceleratorName,
    }
  }

  const usagePercent = (usedValue / allocatableValue) * 100
  const displayValue = `${betterResourceQuantity(resourceKey, usedValue)}/${betterResourceQuantity(resourceKey, allocatableValue, true)}`

  return {
    usagePercent,
    displayValue,
    acceleratorName,
  }
}

// 获取资源使用百分比的帮助函数，用于排序
export function getResourceUsagePercent(
  resourceKey: 'cpu' | 'memory' | 'accelerator',
  used?: V1ResourceList,
  allocatable?: V1ResourceList,
  accelerators?: string[]
): number {
  const usageInfo = calculateResourceUsage(resourceKey, used, allocatable, accelerators)
  return usageInfo.usagePercent ?? 0
}

export const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索节点名称',
    key: 'name',
  },
  filterOptions: [],
  getHeader: (x) => x,
}

export const UsageCell: FC<{
  used?: V1ResourceList
  allocatable?: V1ResourceList
  capacity?: V1ResourceList
  resourceKey: 'cpu' | 'memory' | 'accelerator'
  accelerators?: string[]
}> = ({ used, allocatable, resourceKey, accelerators }) => {
  const { usagePercent, displayValue } = useMemo(() => {
    return calculateResourceUsage(resourceKey, used, allocatable, accelerators)
  }, [accelerators, allocatable, resourceKey, used])

  if (usagePercent === null || displayValue === null) {
    return <></>
  }

  return (
    <div className="w-20">
      <p className={progressTextColor(usagePercent)}>
        {usagePercent.toFixed(1)}
        <span className="ml-0.5">%</span>
      </p>
      <ProgressBar percent={usagePercent} className="h-1 w-full" />
      <p className="text-muted-foreground pt-1 font-mono text-xs">{displayValue}</p>
    </div>
  )
}

const adminNodeLinkOptions = linkOptions({
  to: '/admin/cluster/nodes/$node',
  params: { node: '' },
  search: { tab: '' },
})

const portalNodeLinkOptions = linkOptions({
  to: '/portal/overview/$node',
  params: { node: '' },
  search: { tab: '' },
})

export const nodesToolbarConfig: DataTableToolbarConfig = {
  globalSearch: {
    enabled: true,
  },
  filterOptions: [
    {
      key: 'status',
      title: '状态',
      option: nodeStatuses,
    },
    {
      key: 'acceleratorModel',
      title: '加速卡型号',
    },
  ],
  getHeader: (x) => x,
}

export const getNodeColumns = (
  getNicknameByName?: (name: string) => string | undefined,
  accelerators?: string[],
  isAdminMode?: boolean
): ColumnDef<INodeBriefInfo>[] => {
  return [
    {
      accessorKey: 'arch',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'架构'} />,
      cell: ({ row }) => (
        <Badge variant="outline" className="font-mono font-normal">
          {row.getValue('arch')}
        </Badge>
      ),
    },
    {
      accessorKey: 'name',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'名称'} />,
      cell: ({ row }) => (
        <TooltipLink
          name={row.getValue('name')}
          {...(isAdminMode ? adminNodeLinkOptions : portalNodeLinkOptions)}
          params={{ node: row.getValue<string>('name') }}
          tooltip={`查看 ${row.original.name} 节点详情`}
        />
      ),
    },
    {
      accessorKey: 'role',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'角色'} />,
      cell: ({ row }) => {
        const status = row.original.status
        const taints = row.original.taints || []

        // 如果状态为"occupied"，提取占用的账户名
        let accountInfo = null
        if (status === NodeStatus.Occupied && getNicknameByName) {
          const occupiedAccount = taints
            .find((t: string) => t.startsWith('crater.raids.io/account'))
            ?.split('=')[1]
            ?.split(':')[0]

          if (occupiedAccount) {
            // 获取账户昵称
            const nickname = getNicknameByName(occupiedAccount)
            accountInfo = nickname ? `${nickname}` : occupiedAccount
          }
        }

        // 过滤taints，如果状态是"occupied"
        const displayTaints =
          status === NodeStatus.Occupied
            ? taints.filter((taint: string) => taint.includes('crater.raids.io/account'))
            : taints
        return (
          <div className="flex flex-row items-center justify-start gap-1">
            {/* 如果有账户信息，显示一个单独的提示 */}
            {status === NodeStatus.Occupied && accountInfo ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="destructive" className="font-mono font-normal">
                    {accountInfo}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs font-medium">该节点处于账户独占状态</p>
                </TooltipContent>
              </Tooltip>
            ) : (
              <Badge
                variant={row.getValue('role') === 'control-plane' ? 'default' : 'secondary'}
                className="font-mono font-normal"
              >
                {row.getValue('role')}
              </Badge>
            )}

            {/* 原有的taints提示 */}
            {row.original.taints && displayTaints.length > 0 && status !== NodeStatus.Occupied && (
              <Tooltip>
                <TooltipTrigger className="bg-highlight-slate flex size-4 items-center justify-center rounded-full text-xs text-white">
                  {displayTaints.length}
                </TooltipTrigger>
                <TooltipContent className="font-mono">
                  {displayTaints.map((taint: string, index: number) => (
                    <p key={index} className="text-xs">
                      {taint}
                    </p>
                  ))}
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        )
      },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'状态'} />,
      cell: ({ row }) => {
        const status = row.getValue<string>('status')
        return (
          <div className="flex flex-row items-center justify-start gap-1">
            <NodeStatusBadge status={status} />
            <Tooltip>
              <TooltipTrigger
                className={cn(
                  'bg-secondary text-secondary-foreground flex size-4 items-center justify-center rounded-full text-xs hover:cursor-help',
                  {
                    'bg-primary text-primary-foreground':
                      row.original.workloads && row.original.workloads > 0,
                  }
                )}
              >
                {row.original.workloads}
              </TooltipTrigger>
              <TooltipContent>{row.original.workloads} 个作业正在运行</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
    {
      accessorKey: 'cpu',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'CPU'} />,
      cell: ({ row }) => {
        return (
          <UsageCell
            used={row.original.used}
            allocatable={row.original.allocatable}
            resourceKey="cpu"
          />
        )
      },
      sortingFn: (rowA, rowB) => {
        const a = getResourceUsagePercent('cpu', rowA.original.used, rowA.original.allocatable)
        const b = getResourceUsagePercent('cpu', rowB.original.used, rowB.original.allocatable)
        return a - b
      },
    },
    {
      accessorKey: 'memory',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'Memory'} />,
      cell: ({ row }) => (
        <UsageCell
          used={row.original.used}
          allocatable={row.original.allocatable}
          resourceKey="memory"
        />
      ),
      sortingFn: (rowA, rowB) => {
        const a = getResourceUsagePercent('memory', rowA.original.used, rowA.original.allocatable)
        const b = getResourceUsagePercent('memory', rowB.original.used, rowB.original.allocatable)
        return a - b
      },
    },
    {
      accessorKey: 'accelerator',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'加速卡'} />,
      cell: ({ row }) => (
        <UsageCell
          used={row.original.used}
          allocatable={row.original.allocatable}
          capacity={row.original.capacity}
          resourceKey="accelerator"
          accelerators={accelerators}
        />
      ),
      sortingFn: (rowA, rowB) => {
        const a = getResourceUsagePercent(
          'accelerator',
          rowA.original.used,
          rowA.original.allocatable,
          accelerators
        )
        const b = getResourceUsagePercent(
          'accelerator',
          rowB.original.used,
          rowB.original.allocatable,
          accelerators
        )
        return a - b
      },
    },
    {
      accessorKey: 'acceleratorModel',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'加速卡型号'} />,
      cell: ({ row }) => {
        const usageInfo = calculateResourceUsage(
          'accelerator',
          row.original.used,
          row.original.allocatable,
          accelerators
        )
        if (!usageInfo.acceleratorName) {
          return <></>
        }
        return <AcceleratorBadge acceleratorString={usageInfo.acceleratorName} />
      },
      accessorFn: (row) => {
        const usageInfo = calculateResourceUsage(
          'accelerator',
          row.used,
          row.allocatable,
          accelerators
        )
        return usageInfo.acceleratorName
      },
      enableSorting: true,
    },
  ] as ColumnDef<INodeBriefInfo>[]
}
