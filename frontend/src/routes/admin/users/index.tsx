import { zodResolver } from '@hookform/resolvers/zod'
import { UseQueryResult, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { useAtomValue } from 'jotai'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

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
  IUserAttributes,
  apiAdminUpdateUserAttributes,
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUserUpdateRole,
} from '@/services/api/admin/user'
import { Role } from '@/services/api/auth'
import {
  AdjustUserExtraBalanceReq,
  apiAdminAdjustUserExtraBalance,
  apiAdminGetUserBillingSummary,
} from '@/services/api/billing'
import { apiAdminGetBillingStatus } from '@/services/api/system-config'

import { atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

export const Route = createFileRoute('/admin/users/')({
  component: UserList,
})

interface TUser {
  id: number
  name: string
  role: string
  status: string
  extraBalance?: number
  periodFreeTotal?: number
  totalIssueAmount?: number
  totalAvailable?: number
  attributes: IUserAttributes
}

const getHeader = (key: string): string => {
  switch (key) {
    case 'name':
      return '用户'
    case 'group':
      return '组别'
    case 'teacher':
      return '导师'
    case 'role':
      return '权限'
    case 'status':
      return '状态'
    default:
      return key
  }
}

const roles = [
  {
    label: '管理员',
    value: Role.Admin.toString(),
  },
  {
    label: '普通用户',
    value: Role.User.toString(),
  },
]

const statuses = [
  {
    label: '已激活',
    value: ProjectStatus.Active.toString(),
  },
  {
    label: '已禁用',
    value: ProjectStatus.Inactive.toString(),
  },
]

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索用户名',
    key: 'name',
  },
  filterOptions: [
    {
      key: 'role',
      title: '权限',
      option: roles,
    },
    {
      key: 'status',
      title: '状态',
      option: statuses,
    },
  ],
  getHeader: getHeader,
}

const userFormSchema = (t: (key: string) => string) =>
  z.object({
    nickname: z.string().optional(),
    email: z.string().email(t('userForm.emailError')).optional().or(z.literal('')),
    teacher: z.string().optional().or(z.literal('')),
    group: z.string().optional().or(z.literal('')),
    phone: z.string().optional().or(z.literal('')),
  })

type UserFormValues = z.infer<ReturnType<typeof userFormSchema>>

interface UserEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: TUser | null
}

function UserEditDialog({ open, onOpenChange, user }: UserEditDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema(t)),
    defaultValues: {
      nickname: user?.attributes.nickname || '',
      email: user?.attributes.email || '',
      teacher: user?.attributes.teacher || '',
      group: user?.attributes.group || '',
      phone: user?.attributes.phone || '',
    },
  })

  useEffect(() => {
    if (user) {
      form.reset({
        nickname: user.attributes.nickname || '',
        email: user.attributes.email || '',
        teacher: user.attributes.teacher || '',
        group: user.attributes.group || '',
        phone: user.attributes.phone || '',
      })
    }
  }, [form, user])

  const { mutate: updateUser, isPending } = useMutation({
    mutationFn: (values: UserFormValues) => {
      if (!user) throw new Error('No user selected')
      const updateData: IUserAttributes = {
        ...user.attributes,
        ...values,
      }
      return apiAdminUpdateUserAttributes(user.name, updateData)
    },
    onSuccess: () => {
      toast.success(t('userEditDialog.successToast'))
      queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      onOpenChange(false)
    },
    onError: () => {
      toast.error(t('userEditDialog.errorToast'))
    },
  })

  function onSubmit(values: UserFormValues) {
    updateUser(values)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{t('userEditDialog.title')}</DialogTitle>
          <DialogDescription>
            {t('userEditDialog.description', { name: user?.name })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="nickname"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('userEditDialog.nicknameLabel')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('userEditDialog.nicknamePlaceholder')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('userEditDialog.emailLabel')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('userEditDialog.emailPlaceholder')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="teacher"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('userEditDialog.teacherLabel')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('userEditDialog.teacherPlaceholder')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="group"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('userEditDialog.groupLabel')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('userEditDialog.groupPlaceholder')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('userEditDialog.phoneLabel')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('userEditDialog.phonePlaceholder')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit" disabled={isPending}>
                {isPending ? t('common.saving') : t('common.saveChanges')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

const adjustBalanceFormSchema = z.object({
  delta: z.number(),
  reason: z.string().optional(),
})

type AdjustBalanceFormValues = z.infer<typeof adjustBalanceFormSchema>

interface UserAdjustBalanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: TUser | null
}

function UserAdjustBalanceDialog({ open, onOpenChange, user }: UserAdjustBalanceDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const form = useForm<AdjustBalanceFormValues>({
    resolver: zodResolver(adjustBalanceFormSchema),
    defaultValues: {
      delta: 0,
      reason: '',
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        delta: 0,
        reason: '',
      })
    }
  }, [form, open])

  const { mutate: adjustBalance, isPending } = useMutation({
    mutationFn: (values: AdjustUserExtraBalanceReq) => {
      if (!user) throw new Error('No user selected')
      return apiAdminAdjustUserExtraBalance(user.name, values)
    },
    onSuccess: (_, values) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'users', 'billing-summary'] })
      toast.success(
        t('userTable.adjustBalance.success', {
          defaultValue: '点数调整成功',
        }) + ` (${values.delta > 0 ? '+' : ''}${values.delta})`
      )
      onOpenChange(false)
    },
    onError: () => {
      toast.error(
        t('userTable.adjustBalance.error', {
          defaultValue: '点数调整失败',
        })
      )
    },
  })

  const onSubmit = (values: AdjustBalanceFormValues) => {
    adjustBalance(values)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {t('userTable.adjustBalance.title', {
              defaultValue: '调整额外点数',
            })}
          </DialogTitle>
          <DialogDescription>
            {t('userTable.adjustBalance.description', {
              defaultValue: '按增量调整用户 extra 点数，可输入正数或负数。',
            })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="delta"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('userTable.adjustBalance.delta', {
                      defaultValue: '变更值',
                    })}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      value={field.value ?? 0}
                      onChange={(e) => field.onChange(Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="reason"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('userTable.adjustBalance.reason', {
                      defaultValue: '原因',
                    })}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('userTable.adjustBalance.reasonPlaceholder', {
                        defaultValue: '可选，便于审计',
                      })}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit" disabled={isPending}>
                {isPending ? t('common.saving') : t('common.saveChanges')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

function UserList() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userInfo = useAtomValue(atomUserInfo)
  const [editUser, setEditUser] = useState<TUser | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [adjustUser, setAdjustUser] = useState<TUser | null>(null)
  const [adjustDialogOpen, setAdjustDialogOpen] = useState(false)

  const { data: billingStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })
  const billingEnabled = billingStatus?.featureEnabled ?? false

  const userQuery = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.map((item) => ({
        id: item.id,
        name: item.name,
        role: item.role.toString(),
        status: item.status.toString(),
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
      }) as unknown as UseQueryResult<TUser[], Error>,
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
    mutationFn: (userName: string) => apiAdminUserDelete(userName),
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

  const columns = useMemo<ColumnDef<TUser>[]>(() => {
    const baseColumns: ColumnDef<TUser>[] = [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.name')} />
        ),
        cell: ({ row }) => (
          <UserLabel
            info={{
              username: row.original.name,
              nickname: row.original.attributes.nickname,
            }}
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
        cell: ({ row }) => {
          return <UserRoleBadge role={row.getValue('role')} />
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('userTable.headers.status')} />
        ),
        cell: ({ row }) => {
          return <UserStatusBadge status={row.getValue('status')} />
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      ...(billingEnabled
        ? [
            {
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
            } as ColumnDef<TUser>,
          ]
        : []),
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const user = row.original
          return (
            <div>
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
                      <>
                        <DropdownMenuItem
                          onClick={() => {
                            setAdjustUser(user)
                            setAdjustDialogOpen(true)
                          }}
                        >
                          {t('userTable.adjustBalance.action', {
                            defaultValue: '调整额外点数',
                          })}
                        </DropdownMenuItem>
                      </>
                    )}
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>{t('userTable.roleLabel')}</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.role}`}>
                          {roles.map((role) => (
                            <DropdownMenuRadioItem
                              key={role.value}
                              value={`${role.value}`}
                              onClick={() =>
                                updateRole({
                                  userName: user.name,
                                  role: parseInt(role.value),
                                })
                              }
                            >
                              {t(`userTable.roles.${role.value}`)}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
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
                      {t('userTable.deleteDescription', { name: user?.name })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        if (user.name === userInfo?.name) {
                          showErrorToast(t('userTable.selfDeleteError'))
                        } else {
                          deleteUser(user.name)
                        }
                      }}
                    >
                      {t('common.delete')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          )
        },
      },
    ]
    return baseColumns
  }, [billingEnabled, deleteUser, userInfo, updateRole, t])

  return (
    <>
      <DataTable
        info={{
          title: t('userTable.title'),
          description: t('userTable.description'),
        }}
        storageKey="admin_user"
        query={mergedUserQuery as UseQueryResult<TUser[], Error>}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
      <UserEditDialog open={editDialogOpen} onOpenChange={setEditDialogOpen} user={editUser} />
      <UserAdjustBalanceDialog
        open={adjustDialogOpen}
        onOpenChange={setAdjustDialogOpen}
        user={adjustUser}
      />
    </>
  )
}
