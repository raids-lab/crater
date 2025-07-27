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
import { useMemo, type FC } from 'react'
import { ColumnDef } from '@tanstack/react-table'
import { DataTableColumnHeader } from '@/components/custom/DataTable/DataTableColumnHeader'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { INodeBriefInfo } from '@/services/api/cluster'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import NodeStatusBadge from '@/components/badge/NodeStatusBadge'
import TooltipLink from '../label/TooltipLink'
import {
  betterResourceQuantity,
  convertKResourceToResource,
  V1ResourceList,
} from '@/utils/resource'
import { ProgressBar } from '../custom/ProgressBar'

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
  resourceKey: 'cpu' | 'memory' | 'accelerator'
  accelerators?: string[]
}> = ({ used, allocatable, resourceKey: key, accelerators }) => {
  const [percent, value, acceleratorName] = useMemo(() => {
    // TODO
    // 1. 如果 key 是 CPU 或者 Memory，从字符串转换成数字，计算百分比和展示的值
    // 2. 如果 key 是 accelerator，查找 V1ResourceList = Record<string, string> | undefined 是否包含某个加速卡，包含则返回其使用结果
    let resourceUsed = ''
    let resourceAllocatable = ''
    let acceleratorName = ''
    switch (key) {
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
          return [0, '0']
        }
        break
      default:
        return [0, '0']
    }

    const usedValue = convertKResourceToResource(key, resourceUsed) || 0
    const allocatableValue = convertKResourceToResource(key, resourceAllocatable)
    if (allocatableValue === undefined || allocatableValue === 0) {
      return [null, null]
    }

    return [
      (usedValue / allocatableValue) * 100,
      `${betterResourceQuantity(key, usedValue)}/${betterResourceQuantity(key, allocatableValue, true)}`,
      acceleratorName,
    ]
  }, [accelerators, allocatable, key, used])

  if (percent === null || value === null) {
    return <></>
  }

  return (
    <div className="flex flex-row items-center justify-between gap-2">
      <div className="w-20">
        {/* 'bg-highlight-emerald': width <= 20,
            'bg-highlight-sky': width > 20 && width <= 50,
            'bg-highlight-yellow': width > 50 && width <= 70,
            'bg-highlight-orange': width > 70 && width <= 90,
            'bg-highlight-red': width > 90, */}
        <p
          className={cn('text-highlight-emerald mb-0.5 font-mono text-sm font-bold', {
            'text-highlight-emerald': percent <= 20,
            'text-highlight-sky': percent > 20 && percent <= 50,
            'text-highlight-yellow': percent > 50 && percent <= 70,
            'text-highlight-orange': percent > 70 && percent <= 90,
            'text-highlight-red': percent > 90,
          })}
        >
          {percent.toFixed(1)}
          <span className="ml-0.5">%</span>
        </p>
        <ProgressBar width={percent} className="h-1 w-full" />
        <p className="text-muted-foreground pt-1 font-mono text-xs">{value}</p>
      </div>
      {acceleratorName && acceleratorName !== '' && (
        <Badge variant="secondary" className="font-mono font-normal">
          {acceleratorName}
        </Badge>
      )}
    </div>
  )
}

export const getNodeColumns = (
  getNicknameByName?: (name: string) => string | undefined,
  accelerators?: string[]
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
          to={row.getValue('name')}
          tooltip={`查看 ${row.original.name} 节点详情`}
        />
      ),
    },
    {
      accessorKey: 'status',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'状态'} />,
      cell: ({ row }) => {
        const taints = row.original.taints || []
        const status = row.getValue<string>('status')

        // 如果状态为"occupied"，提取占用的账户名
        let accountInfo = null
        if (status === 'occupied' && getNicknameByName) {
          const occupiedAccount = taints
            .find((t: string) => t.startsWith('crater.raids.io/account'))
            ?.split('=')[1]
            ?.split(':')[0]

          if (occupiedAccount) {
            // 获取账户昵称
            const nickname = getNicknameByName(occupiedAccount)
            accountInfo = nickname ? `${nickname} (${occupiedAccount})` : occupiedAccount
          }
        }

        // 过滤taints，如果状态是"occupied"
        const displayTaints =
          status === 'occupied'
            ? taints.filter((taint: string) => taint.includes('crater.raids.io/account'))
            : taints

        return (
          <div className="flex flex-row items-center justify-start gap-1">
            <NodeStatusBadge status={status} />

            {/* 如果有账户信息，显示一个单独的提示 */}
            {status === 'occupied' && accountInfo && (
              <Tooltip>
                <TooltipTrigger className="flex size-4 items-center justify-center rounded-full bg-blue-500 text-xs text-white">
                  A
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs font-medium">占用账户: {accountInfo}</p>
                </TooltipContent>
              </Tooltip>
            )}

            {/* 原有的taints提示 */}
            {row.original.taints && displayTaints.length > 0 && status !== 'occupied' && (
              <Tooltip>
                <TooltipTrigger className="flex size-4 items-center justify-center rounded-full bg-slate-600 text-xs text-white">
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
      accessorKey: 'role',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'角色'} />,
      cell: ({ row }) => (
        <div className="flex flex-row items-center justify-start gap-1">
          <Badge
            variant={row.getValue('role') === 'control-plane' ? 'default' : 'secondary'}
            className="font-mono font-normal"
          >
            {row.getValue('role')}
          </Badge>
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
      ),
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
        const getUsagePercent = (used?: V1ResourceList, allocatable?: V1ResourceList) => {
          if (!used?.cpu || !allocatable?.cpu) return 0
          return (parseFloat(used.cpu) / parseFloat(allocatable.cpu)) * 100
        }
        const a = getUsagePercent(rowA.original.used, rowA.original.allocatable)
        const b = getUsagePercent(rowB.original.used, rowB.original.allocatable)
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
        const getUsagePercent = (used?: V1ResourceList, allocatable?: V1ResourceList) => {
          if (!used?.memory || !allocatable?.memory) return 0
          return (parseFloat(used.memory) / parseFloat(allocatable.memory)) * 100
        }
        const a = getUsagePercent(rowA.original.used, rowA.original.allocatable)
        const b = getUsagePercent(rowB.original.used, rowB.original.allocatable)
        return a - b
      },
    },
    {
      accessorKey: 'gpu',
      header: ({ column }) => <DataTableColumnHeader column={column} title={'GPU'} />,
      cell: ({ row }) => (
        <UsageCell
          used={row.original.used}
          allocatable={row.original.allocatable}
          resourceKey="accelerator"
          accelerators={accelerators}
        />
      ),
    },
  ] as ColumnDef<INodeBriefInfo>[]
}
