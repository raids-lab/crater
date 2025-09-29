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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useMutation } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { PencilIcon, TrashIcon, UserRoundIcon } from 'lucide-react'
import { PlusCircleIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import ResourceBadges from '@/components/badge/resource-badges'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import { IAccount } from '@/services/api/account'
import { apiAdminAccountList } from '@/services/api/account'
import { apiProjectDelete } from '@/services/api/account'

const getHeader = (key: string): string => {
  switch (key) {
    case 'nickname':
      return 'table.headers.nickname'
    case 'deserved':
      return 'table.headers.deserved'
    case 'guaranteed':
      return 'table.headers.guaranteed'
    case 'capability':
      return 'table.headers.capability'
    default:
      return key
  }
}

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    key: 'name',
    placeholder: '搜索账户名称',
  },
  filterOptions: [],
  getHeader: (key: string) => getHeader(key),
}

export const AccountTable = ({
  setIsOpen,
  setCurrentAccount,
}: {
  setIsOpen: (isOpen: boolean) => void
  setCurrentAccount: (account: IAccount | null) => void
}) => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['admin', 'accounts'],
    queryFn: apiAdminAccountList,
    select: (res) => res.data,
  })

  const { mutate: deleteAccount } = useMutation({
    mutationFn: (account: IAccount) => apiProjectDelete(account.id),
    onSuccess: (_, account) => {
      queryClient.invalidateQueries({
        queryKey: ['admin', 'accounts'],
      })
      toast.success(t('toast.accountDeleted', { name: account.nickname }))
    },
  })

  const columns = useMemo<ColumnDef<IAccount>[]>(() => {
    const cols: ColumnDef<IAccount>[] = [
      {
        accessorKey: 'nickname',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('table.headers.nickname')} />
        ),
        cell: ({ row }) => {
          return (
            <Link
              to={'/admin/accounts/$id'}
              params={{ id: row.original.id.toString() }}
              preload="intent"
              className="hover:text-primary flex flex-row items-center justify-start gap-2"
            >
              <div className="flex flex-col items-start gap-0.5">{row.getValue('nickname')}</div>
            </Link>
          )
        },
        enableSorting: false,
      },
      {
        accessorKey: 'guaranteed',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('table.headers.guaranteed')} />
        ),
        cell: ({ row }) => {
          return <ResourceBadges resources={row.original.quota.guaranteed} />
        },
        enableSorting: false,
      },
      {
        accessorKey: 'deserved',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('table.headers.deserved')} />
        ),
        cell: ({ row }) => {
          return <ResourceBadges resources={row.original.quota.deserved} />
        },
        enableSorting: false,
      },
      {
        accessorKey: 'capability',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('table.headers.capability')} />
        ),
        cell: ({ row }) => {
          return <ResourceBadges resources={row.original.quota.capability} />
        },
        enableSorting: false,
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          return (
            <div className="flex flex-row items-center justify-center gap-1">
              <Link to={'/admin/accounts/$id'} params={{ id: row.original.id.toString() }}>
                <Button
                  title={t('table.actions.manageUser')}
                  variant="outline"
                  className="h-8 w-8"
                  size="icon"
                >
                  <UserRoundIcon className="size-4" />
                </Button>
              </Link>
              <Button
                title={t('table.actions.editQuota')}
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => {
                  setCurrentAccount(row.original)
                  setIsOpen(true)
                }}
              >
                <PencilIcon className="size-4" />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    title={t('table.actions.delete')}
                    variant="outline"
                    size="icon"
                    className="h-8 w-8"
                  >
                    <TrashIcon className="size-4" />
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{t('table.actions.deleteTitle')}</AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('table.actions.deleteDescription', {
                        name: row.original.nickname,
                      })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('table.actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => deleteAccount(row.original)}
                    >
                      {t('table.actions.delete')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          )
        },
      },
    ]
    return cols
  }, [deleteAccount, setCurrentAccount, setIsOpen, t])

  return (
    <DataTable
      info={{
        title: t('accountManagement.title'),
        description: t('accountManagement.description'),
      }}
      storageKey="admin_account_management"
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
    >
      <Button
        onClick={() => {
          setCurrentAccount(null)
          setIsOpen(true)
        }}
      >
        <PlusCircleIcon className="size-4" />
        {t('accountForm.createButton')}
      </Button>
    </DataTable>
  )
}
