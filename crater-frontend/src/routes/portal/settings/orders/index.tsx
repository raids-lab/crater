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
// i18n-processed-v1.1.0
// Modified code
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { InfoIcon } from 'lucide-react'
import { ClockIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import {
  ApprovalOrderStatus,
  ApprovalOrderStatusBadge,
  ApprovalOrderType,
  ApprovalOrderTypeBadge,
} from '@/components/badge/ApprovalorderBadge'
import { TimeDistance } from '@/components/custom/TimeDistance'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import { type ApprovalOrder, listMyApprovalOrder } from '@/services/api/approvalorder'

export const Route = createFileRoute('/portal/settings/orders/')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()

  const query = useQuery({
    queryKey: ['portal', 'approvalorders', 'me'],
    queryFn: () => listMyApprovalOrder(),
    select: (res) =>
      [...(res.data ?? [])].sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      ),
  })

  const columns: ColumnDef<ApprovalOrder>[] = [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('ApprovalOrderTable.column.name')} />
      ),
      cell: ({ row }) => {
        const extHours = row.original.content.approvalorderExtensionHours || 0
        return (
          <div className="relative flex items-center">
            {/* Use navigate for relative jump instead of typed Link to pattern */}
            <button
              type="button"
              className="text-left break-all whitespace-normal underline-offset-4 hover:underline"
              title={`查看 ${row.getValue('name')} 详情`}
              onClick={() => toast.info('该功能正在开发中，敬请期待')}
            >
              {row.getValue('name')}
            </button>
            {row.original.type === 'job' && Number(extHours) > 0 && (
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
      accessorKey: 'type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('ApprovalOrderTable.column.type')} />
      ),
      cell: ({ row }) => {
        return <ApprovalOrderTypeBadge type={row.getValue<ApprovalOrderType>('type')} />
      },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('ApprovalOrderTable.column.status')} />
      ),
      cell: ({ row }) => {
        return <ApprovalOrderStatusBadge status={row.getValue<ApprovalOrderStatus>('status')} />
      },
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('ApprovalOrderTable.column.createdAt')} />
      ),
      cell: ({ row }) => <TimeDistance date={row.getValue('createdAt')} />,
      sortingFn: 'datetime',
    },
    {
      id: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('ApprovalOrderTable.column.actions')} />
      ),
      cell: () => {
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">{t('ApprovalOrderTable.actions.menuTrigger')}</span>
                <InfoIcon className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel className="text-muted-foreground text-xs">
                {t('ApprovalOrderTable.actions.menuLabel')}
              </DropdownMenuLabel>
              <DropdownMenuItem onClick={() => toast.info('该功能正在开发中，敬请期待')}>
                <InfoIcon className="text-highlight-emerald" />
                {t('ApprovalOrderTable.actions.viewDetails')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )
      },
    },
  ]

  return (
    <DataTable
      info={{
        title: t('ApprovalOrderTable.info.title'),
        description: t('ApprovalOrderTable.info.description'),
      }}
      storageKey="portal_approvalorder_management"
      query={query}
      columns={columns}
    />
  )
}
