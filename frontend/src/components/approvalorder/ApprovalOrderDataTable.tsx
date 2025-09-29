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
import { UseQueryResult } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { ClockIcon } from 'lucide-react'
import { ReactNode } from 'react'

import {
  ApprovalOrderStatusBadge,
  ApprovalOrderTypeBadge,
  approvalOrderStatuses,
  approvalOrderTypes,
} from '@/components/badge/ApprovalorderBadge'
import { TimeDistance } from '@/components/custom/TimeDistance'
import UserLabel from '@/components/label/user-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { type ApprovalOrder } from '@/services/api/approvalorder'

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
      cell: ({ row }) => {
        const extHours = showExtensionHours
          ? row.original.content.approvalorderExtensionHours || 0
          : 0
        return (
          <div className="relative flex items-center gap-2">
            <button
              type="button"
              className="text-left break-all whitespace-normal underline-offset-4 hover:underline"
              title={`查看工单 ${row.getValue('name')} 详情`}
              onClick={() => onNameClick?.(row.original)}
            >
              {row.getValue('name')}
            </button>
            {showExtensionHours && row.original.type === 'job' && Number(extHours) > 0 && (
              <div
                title={`锁定 ${extHours} 小时`}
                className="bg-warning/10 text-warning inline-flex items-center gap-1 rounded px-2 py-1 text-xs"
              >
                <ClockIcon className="h-3 w-3" />
                {extHours}h
              </div>
            )}
          </div>
        )
      },
    },
    {
      accessorKey: 'nickname',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={(getHeader || defaultGetHeader)('creator')} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.creator} />,
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
    >
      {children}
    </DataTable>
  )
}
