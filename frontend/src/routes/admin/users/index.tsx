import { UseQueryResult, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import type { TFunction } from 'i18next'
import { useAtomValue } from 'jotai'
import {
  EllipsisVerticalIcon as DotsHorizontalIcon,
  ShieldBanIcon,
  ShieldCheckIcon,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import UserRoleBadge from '@/components/badge/user-role-badge'
import UserStatusBadge from '@/components/badge/user-status-badge'
import { UserPointsTooltip } from '@/components/custom/user-points-tooltip'
import UserLabel from '@/components/label/user-label'
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

import { ProjectStatus } from '@/services/api/account'
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUserUpdateRole,
} from '@/services/api/admin/user'
import { Role } from '@/services/api/auth'
import { apiAdminGetUserBillingSummary } from '@/services/api/billing'
import { apiAdminGetBillingStatus } from '@/services/api/system-config'

import { atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import { AdminUserRow } from './-components/types'
import { UserAdjustBalanceDialog } from './-components/user-adjust-balance-dialog'
import { UserBanDialog } from './-components/user-ban-dialog'
import { UserBanStatusBadge } from './-components/user-ban-status-badge'
import { UserEditDialog } from './-components/user-edit-dialog'

export const Route = createFileRoute('/admin/users/')({
  component: UserList,
})

const roles = [
  { label: '管理员', value: Role.Admin.toString() },
  { label: '普通用户', value: Role.User.toString() },
]

const createToolbarConfig = (t: TFunction): DataTableToolbarConfig => ({
  filterInput: { placeholder: '搜索用户名', key: 'name' },
  filterOptions: [
    { key: 'role', title: '权限', option: roles },
    {
      key: 'status',
      title: '状态',
      option: [
        { label: '已激活', value: ProjectStatus.Active.toString() },
        { label: '已禁用', value: ProjectStatus.Inactive.toString() },
      ],
    },
    {
      key: 'banned',
      title: t('userBan.status.header'),
      option: [
        { label: t('userBan.status.banned'), value: 'true' },
        { label: t('userBan.status.normal'), value: 'false' },
      ],
    },
  ],
  getHeader: (key) => {
    const headers: Record<string, string> = {
      name: '用户',
      group: '组别',
      teacher: '导师',
      role: '权限',
      status: '状态',
      banned: t('userBan.status.header'),
    }
    return headers[key] ?? key
  },
})

function UserList() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userInfo = useAtomValue(atomUserInfo)
  const [editUser, setEditUser] = useState<AdminUserRow | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [adjustUser, setAdjustUser] = useState<AdminUserRow | null>(null)
  const [adjustDialogOpen, setAdjustDialogOpen] = useState(false)
  const [banUser, setBanUser] = useState<AdminUserRow | null>(null)
  const [banDialogOpen, setBanDialogOpen] = useState(false)
  const toolbarConfig = useMemo(() => createToolbarConfig(t), [t])

  const { data: billingStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })
  const billingEnabled = billingStatus?.featureEnabled ?? false

  const userQuery = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    select: (res): AdminUserRow[] =>
      res.data.map((item) => ({
        id: item.id,
        name: item.name,
        role: item.role.toString(),
        status: item.status.toString(),
        banned: Boolean(item.bannedAt),
        bannedAt: item.bannedAt,
        extraBalance: item.extraBalance,
        attributes: item.attributes,
      })),
  })

  const billingSummaryQuery = useQuery({
    queryKey: ['admin', 'users', 'billing-summary'],
    queryFn: () => apiAdminGetUserBillingSummary().then((res) => res.data),
    enabled: billingEnabled,
  })

  const mergedUserQuery = useMemo(
    () =>
      ({
        data: (userQuery.data ?? []).map((user) => {
          const summary = (billingSummaryQuery.data ?? []).find((item) => item.userId === user.id)
          return {
            ...user,
            extraBalance: summary?.extraBalance ?? user.extraBalance,
            periodFreeTotal: summary?.periodFreeTotal ?? 0,
            totalIssueAmount: summary?.totalIssueAmount ?? 0,
            totalAvailable: summary?.totalAvailable ?? 0,
          }
        }),
        isLoading: userQuery.isLoading || (billingEnabled && billingSummaryQuery.isLoading),
        dataUpdatedAt: Math.max(userQuery.dataUpdatedAt, billingSummaryQuery.dataUpdatedAt),
        refetch: userQuery.refetch,
      }) as unknown as UseQueryResult<AdminUserRow[], Error>,
    [
      billingEnabled,
      billingSummaryQuery.data,
      billingSummaryQuery.dataUpdatedAt,
      billingSummaryQuery.isLoading,
      userQuery.data,
      userQuery.dataUpdatedAt,
      userQuery.isLoading,
      userQuery.refetch,
    ]
  )

  const { mutate: deleteUser } = useMutation({
    mutationFn: apiAdminUserDelete,
    onSuccess: async (_, userName) => {
      await queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      toast.success(t('userTable.deleteSuccess', { name: userName }))
    },
  })

  const { mutate: updateRole } = useMutation({
    mutationFn: ({ userName, role }: { userName: string; role: Role }) =>
      apiAdminUserUpdateRole(userName, role),
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      toast.success(t('userTable.roleUpdateSuccess', { name: variables.userName }))
    },
  })

  const columns = useMemo<ColumnDef<AdminUserRow>[]>(() => {
    const baseColumns: ColumnDef<AdminUserRow>[] = [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.name')} />
        ),
        cell: ({ row }) => (
          <UserLabel
            info={{ username: row.original.name, nickname: row.original.attributes.nickname }}
          />
        ),
      },
      {
        accessorKey: 'group',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.group')} />
        ),
        cell: ({ row }) => <div>{row.original.attributes.group}</div>,
      },
      {
        accessorKey: 'teacher',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.teacher')} />
        ),
        cell: ({ row }) => <div>{row.original.attributes.teacher}</div>,
      },
      {
        accessorKey: 'role',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.role')} />
        ),
        cell: ({ row }) => <UserRoleBadge role={row.getValue('role')} />,
        filterFn: (row, id, value) => (value as string[]).includes(row.getValue(id)),
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.status')} />
        ),
        cell: ({ row }) => <UserStatusBadge status={row.getValue('status')} />,
        filterFn: (row, id, value) => (value as string[]).includes(row.getValue(id)),
      },
      {
        accessorKey: 'banned',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userBan.status.header')} />
        ),
        cell: ({ row }) => <UserBanStatusBadge banned={row.original.banned} />,
        filterFn: (row, id, value) => (value as string[]).includes(String(row.getValue(id))),
      },
    ]

    if (billingEnabled) {
      baseColumns.push({
        accessorKey: 'totalAvailable',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('userTable.headers.totalPoints', { defaultValue: '点数' })}
          />
        ),
        cell: ({ row }) => (
          <UserPointsTooltip
            userName={row.original.name}
            totalPoints={row.original.totalAvailable ?? 0}
            extraPoints={row.original.extraBalance ?? 0}
            periodFreePoints={row.original.periodFreeTotal ?? 0}
            effectiveIssueAmount={row.original.totalIssueAmount ?? 0}
            showInlineBreakdown
            inlineVariant="minimal"
            fetchDetail
          />
        ),
      })
    }

    baseColumns.push({
      id: 'actions',
      enableHiding: false,
      cell: ({ row }) => {
        const user = row.original
        const isSelf = user.name === userInfo?.name
        return (
          <AlertDialog>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0" title={t('common.moreOptions')}>
                  <DotsHorizontalIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel className="text-muted-foreground text-xs">
                  {t('common.actions')}
                </DropdownMenuLabel>
                <DropdownMenuItem
                  onClick={() => {
                    setEditUser(user)
                    setEditDialogOpen(true)
                  }}
                >
                  {t('userTable.editInfo')}
                </DropdownMenuItem>
                {billingEnabled && (
                  <DropdownMenuItem
                    onClick={() => {
                      setAdjustUser(user)
                      setAdjustDialogOpen(true)
                    }}
                  >
                    {t('userTable.adjustBalance.action', { defaultValue: '调整额外点数' })}
                  </DropdownMenuItem>
                )}
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>{t('userTable.roleLabel')}</DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuRadioGroup value={user.role}>
                      {roles.map((role) => (
                        <DropdownMenuRadioItem
                          key={role.value}
                          value={role.value}
                          onClick={() =>
                            updateRole({ userName: user.name, role: Number(role.value) as Role })
                          }
                        >
                          {t(`userTable.roles.${role.value}`)}
                        </DropdownMenuRadioItem>
                      ))}
                    </DropdownMenuRadioGroup>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
                <DropdownMenuItem
                  disabled={isSelf}
                  onClick={() => {
                    setBanUser(user)
                    setBanDialogOpen(true)
                  }}
                >
                  {user.banned ? (
                    <ShieldCheckIcon className="text-highlight-green mr-2 size-4" />
                  ) : (
                    <ShieldBanIcon className="text-destructive mr-2 size-4" />
                  )}
                  {t(user.banned ? 'userBan.actions.unban' : 'userBan.actions.ban')}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <AlertDialogTrigger asChild>
                  <DropdownMenuItem className="focus:bg-destructive focus:text-destructive-foreground">
                    {t('userTable.delete')}
                  </DropdownMenuItem>
                </AlertDialogTrigger>
              </DropdownMenuContent>
            </DropdownMenu>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('userTable.deleteTitle')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t('userTable.deleteDescription', { name: user.name })}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={() => {
                    if (isSelf) showErrorToast(t('userTable.selfDeleteError'))
                    else deleteUser(user.name)
                  }}
                >
                  {t('common.delete')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )
      },
    })
    return baseColumns
  }, [billingEnabled, deleteUser, t, updateRole, userInfo?.name])

  return (
    <>
      <DataTable
        info={{ title: t('userTable.title'), description: t('userTable.description') }}
        storageKey="admin_user"
        query={mergedUserQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
      <UserEditDialog open={editDialogOpen} onOpenChange={setEditDialogOpen} user={editUser} />
      <UserAdjustBalanceDialog
        open={adjustDialogOpen}
        onOpenChange={setAdjustDialogOpen}
        user={adjustUser}
      />
      <UserBanDialog open={banDialogOpen} onOpenChange={setBanDialogOpen} user={banUser} />
    </>
  )
}
