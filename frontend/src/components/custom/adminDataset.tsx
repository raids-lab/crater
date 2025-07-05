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
import { useTranslation } from 'react-i18next'
import { apiAdminGetDataset, apiDatasetDelete, IDataset } from '@/services/api/dataset'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { DataTable } from '@/components/custom/DataTable'
import { Button } from '@/components/ui/button'
import { DataTableToolbarConfig } from '@/components/custom/DataTable/DataTableToolbar'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { ColumnDef } from '@tanstack/react-table'
import { Trash2Icon, InfoIcon } from 'lucide-react'
import { DotsHorizontalIcon } from '@radix-ui/react-icons'
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from '@/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import { useNavigate } from 'react-router-dom'
import { DataTableColumnHeader } from '@/components/custom/DataTable/DataTableColumnHeader'
import { TimeDistance } from '@/components/custom/TimeDistance'
import DatasetTypeLabel, { DatasetType } from '@/components/badge/DatasetTybeBadge'
import UserLabel from '@/components/label/UserLabel'
import TooltipLink from '@/components/label/TooltipLink'

const getRoles = (t: (key: string) => string) => [
  {
    label: t('adminDatasetTable.toolbar.filter.type.dataset'),
    value: 'dataset',
  },
  {
    label: t('adminDatasetTable.toolbar.filter.type.model'),
    value: 'model',
  },
  {
    label: t('adminDatasetTable.toolbar.filter.type.sharefile'),
    value: 'sharefile',
  },
]

const getToolbarConfig = (t: (key: string) => string): DataTableToolbarConfig => ({
  filterInput: {
    key: 'name',
    placeholder: t('adminDatasetTable.toolbar.filter.placeholder'),
  },
  filterOptions: [
    {
      key: 'type',
      title: t('adminDatasetTable.toolbar.filter.title'),
      option: getRoles(t),
    },
  ],
  getHeader: (key: string): string => {
    switch (key) {
      case 'name':
        return t('adminDatasetTable.column.name')
      case 'type':
        return t('adminDatasetTable.column.type')
      case 'username':
        return t('adminDatasetTable.column.creator')
      case 'createdAt':
        return t('adminDatasetTable.column.createdAt')
      default:
        return key
    }
  },
})

export const AdminDatasetTable = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['admin', 'datasets'],
    queryFn: () => apiAdminGetDataset(),
    select: (res) => res.data.data,
  })
  const navigate = useNavigate()
  const { mutate: deleteDataset } = useMutation({
    mutationFn: (dataset: IDataset) => apiDatasetDelete(dataset.id),
    onSuccess: async (_, dataset) => {
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'datasets'],
      })
      toast.success(t('adminDatasetTable.toast.deleteSuccess', { name: dataset.name }))
    },
  })

  const columns: ColumnDef<IDataset>[] = [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('adminDatasetTable.column.name')} />
      ),
      cell: ({ row }) => {
        return (
          <div className="relative flex items-center">
            <TooltipLink
              name={
                <div className="flex flex-row items-center">
                  <p className="max-w-36 truncate">{row.getValue('name')}</p>
                </div>
              }
              to={`${row.original.id}`}
              tooltip={
                <div className="flex flex-row items-center justify-between gap-1.5">
                  <p className="text-xs">查看 {row.getValue('name')} 详情</p>
                </div>
              }
            />
          </div>
        )
      },
    },
    {
      accessorKey: 'type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('adminDatasetTable.column.type')} />
      ),
      cell: ({ row }) => {
        return <DatasetTypeLabel datasetType={row.getValue<DatasetType>('type')} />
      },
    },
    {
      accessorKey: 'username',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('adminDatasetTable.column.creator')} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('adminDatasetTable.column.createdAt')} />
      ),
      cell: ({ row }) => {
        return <TimeDistance date={row.getValue('createdAt')}></TimeDistance>
      },
      sortingFn: 'datetime',
    },
    {
      id: 'actions',
      header: t('adminDatasetTable.column.actions'),
      cell: ({ row }) => {
        const DatasetInfo = row.original
        return (
          <div className="flex flex-row space-x-1">
            <AlertDialog>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">{t('adminDatasetTable.actions.menuTrigger')}</span>
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="text-muted-foreground text-xs">
                    {t('adminDatasetTable.actions.menuLabel')}
                  </DropdownMenuLabel>
                  <DropdownMenuItem onClick={() => navigate(`${DatasetInfo.id}`)}>
                    <InfoIcon className="text-highlight-emerald" />
                    {t('adminDatasetTable.actions.viewDetails')}
                  </DropdownMenuItem>

                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>
                      <Trash2Icon className="text-destructive" />
                      {t('adminDatasetTable.actions.delete')}
                    </DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t('adminDatasetTable.dialog.title')}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('adminDatasetTable.dialog.description', {
                      name: DatasetInfo.name,
                    })}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{t('adminDatasetTable.dialog.cancel')}</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => {
                      deleteDataset(row.original)
                    }}
                  >
                    {t('adminDatasetTable.dialog.delete')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        )
      },
    },
  ]

  return (
    <DataTable
      info={{
        title: t('adminDatasetTable.info.title'),
        description: t('adminDatasetTable.info.description'),
      }}
      storageKey="admin_dataset_management"
      query={query}
      columns={columns}
      toolbarConfig={getToolbarConfig(t)}
    />
  )
}
