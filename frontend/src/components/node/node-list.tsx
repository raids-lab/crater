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
import { type Locale } from 'date-fns/locale'
import { type FC, useMemo } from 'react'

import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import { nodeStatuses } from '@/components/badge/node-status-badge'
import TooltipLink from '@/components/label/tooltip-link'
import TooltipCopy from '@/components/label/tooltop-copy'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
import { MetricMeter } from '@/components/ui-custom/metric-meter'

import { INodeBriefInfo, JoinTaint, NodeRole } from '@/services/api/cluster'

import {
  V1ResourceList,
  betterResourceQuantity,
  convertKResourceToResource,
} from '@/utils/resource'

import { cn } from '@/lib/utils'

import AcceleratorBadge from '../badge/accelerator-badge'
import { NodeStatusCell } from './node-status-cell'
import {
  deriveNodeStatus,
  getNodeStatusSeverity,
  getNodeStatusTags,
  matchesNodeStatusFilter,
} from './node-status-utils'

// 资源使用情况计算结果接口
export interface ResourceUsageInfo {
  usagePercent: number | null
  displayValue: string | null
  acceleratorName?: string
}

// 多GPU类型信息接口
export interface MultiGPUInfo {
  acceleratorName: string
  usagePercent: number
  displayValue: string
}

// 从节点中提取所有GPU类型的辅助函数
export function extractAllGPUTypes(
  allocatable?: V1ResourceList,
  used?: V1ResourceList,
  accelerators?: string[]
): MultiGPUInfo[] {
  if (!accelerators || !allocatable) {
    return []
  }

  const gpuInfoList: MultiGPUInfo[] = []

  for (const accelerator of accelerators) {
    const allocatableValue = allocatable[accelerator]
    if (allocatableValue && allocatableValue !== '0') {
      const usedValue = convertKResourceToResource('accelerator', used?.[accelerator] || '0') || 0
      const allocValue = convertKResourceToResource('accelerator', allocatableValue) || 0

      if (allocValue > 0) {
        const usagePercent = (usedValue / allocValue) * 100
        const displayValue = `${betterResourceQuantity('accelerator', usedValue)}/${betterResourceQuantity('accelerator', allocValue, true)}`

        gpuInfoList.push({
          acceleratorName: accelerator,
          usagePercent,
          displayValue,
        })
      }
    }
  }

  return gpuInfoList
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
        // 首先尝试找非虚拟GPU资源
        for (const accelerator of accelerators) {
          if (allocatable && allocatable[accelerator] && allocatable[accelerator] !== '0') {
            resourceAllocatable = allocatable[accelerator]
            acceleratorName = accelerator
            resourceUsed = used && used[accelerator] ? used[accelerator] : '0'
            break
          }
        }

        // 如果没找到物理GPU，再找虚拟GPU
        if (!acceleratorName) {
          for (const accelerator of accelerators) {
            if (allocatable && allocatable[accelerator] && allocatable[accelerator] !== '0') {
              resourceAllocatable = allocatable[accelerator]
              acceleratorName = accelerator
              resourceUsed = used && used[accelerator] ? used[accelerator] : '0'
              break
            }
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
    <MetricMeter
      percent={usagePercent}
      primary={
        <>
          {usagePercent.toFixed(1)}
          <span className="ml-0.5">%</span>
        </>
      }
      secondary={displayValue}
    />
  )
}

// 多GPU使用率显示组件
export const MultiGPUUsageCell: FC<{
  used?: V1ResourceList
  allocatable?: V1ResourceList
  accelerators?: string[]
}> = ({ used, allocatable, accelerators }) => {
  const gpuInfoList = useMemo(() => {
    return extractAllGPUTypes(allocatable, used, accelerators)
  }, [accelerators, allocatable, used])

  if (gpuInfoList.length === 0) {
    return <></>
  }

  return (
    <div className="flex flex-col items-start justify-center gap-2">
      {gpuInfoList.map((gpuInfo, index) => (
        <MetricMeter
          key={index}
          percent={gpuInfo.usagePercent}
          primary={
            <>
              {gpuInfo.usagePercent.toFixed(1)}
              <span className="ml-0.5">%</span>
            </>
          }
          secondary={gpuInfo.displayValue}
        />
      ))}
    </div>
  )
}

// 多GPU型号显示组件 - 与使用率行对应
export const MultiGPUModelCell: FC<{
  used?: V1ResourceList
  allocatable?: V1ResourceList
  accelerators?: string[]
}> = ({ used, allocatable, accelerators }) => {
  const gpuInfoList = useMemo(() => {
    return extractAllGPUTypes(allocatable, used, accelerators)
  }, [accelerators, allocatable, used])

  if (gpuInfoList.length === 0) {
    return <></>
  }

  return (
    <div className="flex flex-col items-start justify-center gap-2">
      {gpuInfoList.map((gpuInfo, index) => (
        <div key={index} className="flex h-[52px] items-center">
          <AcceleratorBadge acceleratorString={gpuInfo.acceleratorName} />
        </div>
      ))}
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
  isAdminMode?: boolean,
  locale?: Locale
): ColumnDef<INodeBriefInfo>[] => {
  return [
    {
      accessorKey: 'arch',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'架构'} />,
      cell: ({ row }) => {
        const arch = row.getValue<string>('arch')
        const isArm = arch?.toLowerCase()?.includes('arm') ?? false
        return (
          <Badge
            variant="outline"
            className={cn('font-mono font-normal', {
              'border-orange-600 bg-orange-50 text-orange-600 dark:border-orange-500 dark:bg-orange-950 dark:text-orange-400':
                isArm,
              'border-sky-600 bg-blue-50 text-sky-600 dark:border-sky-500 dark:bg-blue-950 dark:text-sky-400':
                !isArm,
            })}
          >
            {arch}
          </Badge>
        )
      },
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
      accessorKey: 'address',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'IP地址'} />,
      cell: ({ row }) => {
        const address = row.getValue<string>('address')
        if (!address) {
          return <span className="font-mono">-</span>
        }
        return <TooltipCopy name={address} className="font-mono" />
      },
    },
    {
      accessorKey: 'role',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'角色'} />,
      cell: ({ row }) => {
        const role = row.getValue<NodeRole>('role')
        const taints = row.original.taints ?? []
        const account = deriveNodeStatus(row.original).occupied?.account
        return (
          <div className="flex flex-row items-center justify-start gap-1">
            {account ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge
                    variant="outline"
                    className="text-highlight-yellow bg-highlight-yellow/20 max-w-28 cursor-help truncate border-none font-normal"
                    tabIndex={0}
                  >
                    <span className="truncate">{getNicknameByName?.(account) || account}</span>
                  </Badge>
                </TooltipTrigger>
                <TooltipContent>
                  <div>账户：{account}</div>
                  <div>角色：{role}</div>
                </TooltipContent>
              </Tooltip>
            ) : (
              <Badge
                variant={role === NodeRole.ControlPlane ? 'default' : 'secondary'}
                className="font-mono font-normal"
              >
                {role}
              </Badge>
            )}

            {taints.length > 0 && (
              <Tooltip>
                <TooltipTrigger className="bg-highlight-slate flex size-4 shrink-0 items-center justify-center rounded-full text-xs text-white hover:cursor-help">
                  {taints.length}
                </TooltipTrigger>
                <TooltipContent className="font-mono">
                  {taints.map((taint, index) => (
                    <p key={index} className="text-xs">
                      {JoinTaint(taint)}
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
      cell: ({ row }) => <NodeStatusCell node={row.original} locale={locale} />,
      // 一个节点可同时拥有多个独立状态标签，筛选时命中任一选中标签即可
      getUniqueValues: (node) => getNodeStatusTags(node),
      filterFn: (row, _columnId, filterValue: unknown) =>
        matchesNodeStatusFilter(row.original, filterValue),
      sortingFn: (rowA, rowB) =>
        getNodeStatusSeverity(rowA.original) - getNodeStatusSeverity(rowB.original),
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
        <MultiGPUUsageCell
          used={row.original.used}
          allocatable={row.original.allocatable}
          accelerators={accelerators}
        />
      ),
      sortingFn: (rowA, rowB) => {
        // 对于多GPU节点，使用平均使用率进行排序
        const gpuListA = extractAllGPUTypes(
          rowA.original.allocatable,
          rowA.original.used,
          accelerators
        )
        const gpuListB = extractAllGPUTypes(
          rowB.original.allocatable,
          rowB.original.used,
          accelerators
        )

        const avgA =
          gpuListA.length > 0
            ? gpuListA.reduce((sum, gpu) => sum + gpu.usagePercent, 0) / gpuListA.length
            : 0
        const avgB =
          gpuListB.length > 0
            ? gpuListB.reduce((sum, gpu) => sum + gpu.usagePercent, 0) / gpuListB.length
            : 0

        return avgA - avgB
      },
    },
    {
      accessorKey: 'acceleratorModel',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'加速卡型号'} />,
      cell: ({ row }) => (
        <MultiGPUModelCell
          used={row.original.used}
          allocatable={row.original.allocatable}
          accelerators={accelerators}
        />
      ),
      accessorFn: (row) => {
        const gpuInfoList = extractAllGPUTypes(row.allocatable, row.used, accelerators)
        // 返回所有GPU类型，用逗号分隔，用于搜索和筛选
        return gpuInfoList.map((info) => info.acceleratorName).join(',')
      },
      enableSorting: true,
    },
  ] as ColumnDef<INodeBriefInfo>[]
}
