import { UseQueryResult, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import type { TFunction } from 'i18next'
import { useAtomValue } from 'jotai'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
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

import { UserBanStatusBadge } from '@/components/badge/user-ban-status-badge'
import UserRoleBadge from '@/components/badge/user-role-badge'
import UserStatusBadge from '@/components/badge/user-status-badge'
import { TimeDistance } from '@/components/custom/time-distance'
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
import { USER_BAN_STATUS_REFETCH_INTERVAL } from '@/services/api/user-ban'
import { userFieldKeys, visibleUserFields } from '@/services/user-fields'

import { useUserFields } from '@/hooks/use-user-fields'

import { atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import { AdminUserRow } from './-components/types'
import { UserAdjustBalanceDialog } from './-components/user-adjust-balance-dialog'
import { UserBanDialog, UserBanDialogMode } from './-components/user-ban-dialog'
import { UserEditDialog } from './-components/user-edit-dialog'

export const Route = createFileRoute('/admin/users/')({
  component: UserList,
})

const roles = [Role.Admin, Role.User].map((role) => ({ value: role.toString() }))

const createToolbarConfig = (
  t: TFunction,
  labels: Record<string, string>
): DataTableToolbarConfig => ({
  filterInput: { placeholder: t('userFields.searchPlaceholder'), key: 'name' },
  filterOptions: [
    {
      key: 'role',
      title: t('userTable.headers.role'),
      option: roles.map((role) => ({ ...role, label: t(`userTable.roles.${role.value}`) })),
    },
    {
      key: 'status',
      title: t('userTable.headers.status'),
      option: [
        { label: t('userFields.status.active'), value: ProjectStatus.Active.toString() },
        { label: t('userFields.status.inactive'), value: ProjectStatus.Inactive.toString() },
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
  getHeader: (key) =>
    labels[key] ??
    {
      name: t('userTable.headers.name'),
      role: t('userTable.headers.role'),
      status: t('userTable.headers.status'),
      banned: t('userBan.status.header'),
      totalAvailable: t('userTable.headers.totalPoints'),
    }[key] ??
    key,
})

function UserList() {
  const { t } = useTranslation()
  const fieldQuery = useUserFields()
  const { fields, label: fieldLabel } = fieldQuery
  const fieldLabels = Object.fromEntries(userFieldKeys.map((key) => [key, fieldLabel(key)]))
  const queryClient = useQueryClient()
  const userInfo = useAtomValue(atomUserInfo)
  const [editUser, setEditUser] = useState<AdminUserRow | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [adjustUser, setAdjustUser] = useState<AdminUserRow | null>(null)
  const [adjustDialogOpen, setAdjustDialogOpen] = useState(false)
  const [banUser, setBanUser] = useState<AdminUserRow | null>(null)
  const [banDialogOpen, setBanDialogOpen] = useState(false)
  const [banDialogMode, setBanDialogMode] = useState<UserBanDialogMode>('ban')
  const toolbarConfig = createToolbarConfig(t, fieldLabels)

  const { data: billingStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })
  const billingEnabled = billingStatus?.featureEnabled ?? false

  const userQuery = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    refetchInterval: USER_BAN_STATUS_REFETCH_INTERVAL,
    select: (res): AdminUserRow[] =>
      res.data.map((item) => ({
        containerSysAdminEnabled: item.containerSysAdminEnabled,
        id: item.id,
        name: item.name,
        createdAt: item.createdAt,
        lastLoginAt: item.lastLoginAt,
        role: item.role.toString(),
        status: item.status.toString(),
        banned: item.banned,
        permanentBanned: item.permanentBanned,
        bannedTimestamp: item.bannedTimestamp,
        banRestrictions: item.banRestrictions,
        extraBalance: item.extraBalance,
        attributes: item.attributes,
      })),
  })

  const billingSummaryQuery = useQuery({
    queryKey: ['admin', 'users', 'billing-summary'],
    queryFn: () => apiAdminGetUserBillingSummary().then((res) => res.data),
    enabled: billingEnabled,
  })

  const refetchUsers = userQuery.refetch
  const refetchBilling = billingSummaryQuery.refetch

  const mergedUserQuery = useMemo(
    () =>
      ({
        data: (billingEnabled && billingSummaryQuery.isError ? [] : (userQuery.data ?? [])).map(
          (user) => {
            const summary = (billingSummaryQuery.data ?? []).find((item) => item.userId === user.id)
            return {
              ...user,
              extraBalance: summary?.extraBalance ?? user.extraBalance,
              periodFreeTotal: summary?.periodFreeTotal ?? 0,
              totalIssueAmount: summary?.totalIssueAmount ?? 0,
              totalAvailable: summary?.totalAvailable ?? 0,
            }
          }
        ),
        isLoading:
          fieldQuery.isPending ||
          userQuery.isLoading ||
          (billingEnabled && billingSummaryQuery.isLoading),
        isError: userQuery.isError || (billingEnabled && billingSummaryQuery.isError),
        error: userQuery.isError
          ? new Error(t('userFields.loadUsersError'))
          : billingEnabled && billingSummaryQuery.isError
            ? new Error(t('userFields.loadBillingError'))
            : null,
        dataUpdatedAt: Math.max(userQuery.dataUpdatedAt, billingSummaryQuery.dataUpdatedAt),
        refetch: async () => {
          const [result] = await Promise.all([
            refetchUsers(),
            ...(billingEnabled ? [refetchBilling()] : []),
          ])
          return result
        },
      }) as unknown as UseQueryResult<AdminUserRow[], Error>,
    [
      billingEnabled,
      t,
      fieldQuery.isPending,
      billingSummaryQuery.data,
      billingSummaryQuery.dataUpdatedAt,
      billingSummaryQuery.isLoading,
      billingSummaryQuery.isError,
      refetchBilling,
      userQuery.data,
      userQuery.dataUpdatedAt,
      userQuery.isLoading,
      userQuery.isError,
      refetchUsers,
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
      ...visibleUserFields(fields, 'showInTable').map(
        (key): ColumnDef<AdminUserRow> => ({
          id: key,
          accessorFn: (row) =>
            key === 'createdAt' || key === 'lastLoginAt' ? row[key] : row.attributes[key],
          header: ({ column }) => <DataTableColumnHeader column={column} title={fieldLabel(key)} />,
          cell: ({ getValue }) => {
            const value = getValue<string | undefined>()
            if (!value) return <span className="text-muted-foreground">—</span>
            if (key === 'createdAt' || key === 'lastLoginAt' || key === 'expiredAt') {
              return Number.isFinite(Date.parse(value)) ? (
                <TimeDistance date={value} />
              ) : (
                <span className="text-muted-foreground">—</span>
              )
            }
            return <div>{value}</div>
          },
        })
      ),
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
        cell: ({ row }) => (
          <UserBanStatusBadge
            banned={row.original.banned}
            permanentBanned={row.original.permanentBanned}
            bannedTimestamp={row.original.bannedTimestamp}
            banRestrictions={row.original.banRestrictions}
            adminUserName={row.original.name}
          />
        ),
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
                {user.banned ? (
                  <>
                    <DropdownMenuItem
                      onClick={() => {
                        setBanUser(user)
                        setBanDialogMode('unban')
                        setBanDialogOpen(true)
                      }}
                    >
                      {t('userBan.actions.unban')}
                    </DropdownMenuItem>
                    {!user.permanentBanned && (
                      <DropdownMenuItem
                        onClick={() => {
                          setBanUser(user)
                          setBanDialogMode('extend')
                          setBanDialogOpen(true)
                        }}
                      >
                        {t('userBan.actions.extend')}
                      </DropdownMenuItem>
                    )}
                  </>
                ) : (
                  <DropdownMenuItem
                    disabled={isSelf}
                    onClick={() => {
                      setBanUser(user)
                      setBanDialogMode('ban')
                      setBanDialogOpen(true)
                    }}
                  >
                    {t('userBan.actions.ban')}
                  </DropdownMenuItem>
                )}
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
  }, [billingEnabled, deleteUser, t, updateRole, userInfo?.name, fields, fieldLabel])

  if (fieldQuery.isError) {
    return (
      <div role="alert" className="space-y-2 p-6">
        <p>{t('userFields.loadError')}</p>
        <Button onClick={() => void fieldQuery.refetch()}>{t('userFields.retry')}</Button>
      </div>
    )
  }

  return (
    <>
      <DataTable
        info={{ title: t('userTable.title'), description: t('userTable.description') }}
        storageKey="admin_user"
        withI18n
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
      <UserBanDialog
        open={banDialogOpen}
        onOpenChange={setBanDialogOpen}
        user={banUser}
        mode={banDialogMode}
      />
    </>
  )
}
