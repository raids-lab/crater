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
import { zodResolver } from '@hookform/resolvers/zod'
import { UseQueryResult, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { useAtomValue, useSetAtom } from 'jotai'
import {
  CirclePlusIcon,
  EllipsisVerticalIcon as DotsHorizontalIcon,
  UserRoundPlusIcon,
  XIcon,
} from 'lucide-react'
import { type ChangeEvent, type FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Dialog, DialogClose, DialogTrigger } from '@/components/ui/dialog'
import { DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import CapacityBadges from '@/components/badge/capacity-badges'
import UserAccessBadge from '@/components/badge/user-access-badge'
import UserRoleBadge, { userRoles } from '@/components/badge/user-role-badge'
import SelectBox from '@/components/custom/select-box'
import FormLabelMust from '@/components/form/form-label-must'
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
} from '@/components/ui-custom/alert-dialog'

import {
  Access,
  IUserInAccount,
  IUserInAccountCreate,
  apiAccountGetByName,
  apiAddUser,
  apiRemoveUser,
  apiUpdateUser,
  apiUpdateUserOutOfProjectList,
  apiUserAddAccountMember,
  apiUserInProjectList,
  apiUserListAccountMembers,
  apiUserListUsersOutOfAccount,
  apiUserOutOfProjectList,
  apiUserRemoveAccountMember,
  apiUserUpdateAccountMember,
} from '@/services/api/account'
import { Role, apiQueueSwitch } from '@/services/api/auth'
import { queryAccountByID } from '@/services/query/account'

import useIsAdmin from '@/hooks/use-admin'

import { atomUserContext, atomUserInfo } from '@/utils/store'

// Moved Zod schema to component
const formSchema = z.object({
  userIds: z.array(z.string()).min(1, {
    message: t('accountDetail.form.validation.minUsers'),
  }),
  role: z.string().min(1, {
    message: t('accountDetail.form.validation.invalidRole'),
  }),
  accessmode: z.string().min(1, {
    message: t('accountDetail.form.validation.invalidAccess'),
  }),
})

interface AccountMemberTableProps {
  accountId: number
  editable?: boolean
  storageKey?: string
}

export function AccountMemberTable({
  accountId,
  editable: editableProp,
  storageKey = 'account_members',
}: AccountMemberTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isAdminView = useIsAdmin()
  const editable = editableProp ?? isAdminView
  const currentUser = useAtomValue(atomUserInfo)
  const currentAccountContext = useAtomValue(atomUserContext)
  const setAccountContext = useSetAtom(atomUserContext)

  // Get account info to check if it's default account
  const { data: accountInfo } = useQuery({
    ...queryAccountByID(accountId),
    enabled: !!accountId,
  })

  // Check if current account is default account
  const isDefaultAccount = accountInfo?.name === 'default'

  const getHeader = useCallback(
    (key: string): string => {
      switch (key) {
        case 'name':
          return t('accountDetail.table.headers.name')
        case 'role':
          return t('accountDetail.table.headers.role')
        case 'accessmode':
          return t('accountDetail.table.headers.accessmode')
        case 'capability':
          return t('accountDetail.table.headers.capability')
        default:
          return key
      }
    },
    [t]
  )

  const accessModes = useMemo(
    () => [
      {
        label: t('accountDetail.table.accessmodes.readOnly'),
        value: Access.RO.toString(),
      },
      {
        label: t('accountDetail.table.accessmodes.readWrite'),
        value: Access.RW.toString(),
      },
    ],
    [t]
  )

  // Choose API functions based on whether user is platform admin
  const listAccountMembers = isAdminView ? apiUserInProjectList : apiUserListAccountMembers
  const listUsersOutOfAccount = isAdminView ? apiUserOutOfProjectList : apiUserListUsersOutOfAccount
  const addAccountMember = isAdminView ? apiAddUser : apiUserAddAccountMember
  const updateAccountMember = isAdminView ? apiUpdateUser : apiUserUpdateAccountMember
  const removeAccountMember = isAdminView ? apiRemoveUser : apiUserRemoveAccountMember

  const accountUsersQuery = useQuery({
    queryKey: ['account', accountId, 'users', isAdminView ? 'admin' : 'user'],
    queryFn: () => listAccountMembers(accountId),
    select: (res) => res.data,
    retry: false, // Disable auto-retry to show errors immediately
  })

  const { mutate: addUser } = useMutation({
    mutationFn: (users: IUserInAccountCreate[]) =>
      Promise.all(users.map((user) => addAccountMember(accountId, user))),
    onSuccess: async () => {
      toast.success(t('accountDetail.toast.added'))
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'users'],
      })
    },
  })

  const { mutate: updateUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => updateAccountMember(accountId, user),
    onSuccess: async (_, user: IUserInAccountCreate) => {
      toast.success(t('accountDetail.toast.updated'))

      // If updating own permissions in current account, refresh token first
      const isUpdatingSelf =
        currentUser?.id && currentAccountContext?.queue && user.id === currentUser.id

      if (isUpdatingSelf) {
        // Get current account ID by name to compare with accountId
        try {
          const currentAccountInfo = await apiAccountGetByName(currentAccountContext.queue)
          if (currentAccountInfo.data?.id === accountId) {
            // Switch to current account to refresh token with new permissions
            const authResponse = await apiQueueSwitch(currentAccountContext.queue)
            // Update user context with new token data
            if (authResponse?.context) {
              setAccountContext(authResponse.context)
            }
            toast.success(t('accountDetail.toast.tokenUpdated'))
            // Refresh member list after token update (token now has correct permissions)
            await queryClient.invalidateQueries({
              queryKey: ['account', accountId, 'users'],
            })
            return
          }
        } catch {
          // Token update failed, but user update succeeded
          toast.error(t('accountDetail.toast.tokenUpdateFailed'))
          // Still refresh cache even if token update failed (user is still in account)
          await queryClient.invalidateQueries({
            queryKey: ['account', accountId, 'users'],
          })
          return
        }
      }

      // Refresh member list if not updating self
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'users'],
      })
    },
  })

  const [usersOutOfProject, setUsersOutOfProject] = useState<IUserInAccount[]>([])
  const [pendingRoleUpdate, setPendingRoleUpdate] = useState<{
    user: IUserInAccountCreate
    newRole: string
  } | null>(null)
  const [pendingDeleteUser, setPendingDeleteUser] = useState<IUserInAccountCreate | null>(null)

  const { mutate: deleteUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => removeAccountMember(accountId, user),
    onSuccess: async (_, user: IUserInAccountCreate) => {
      toast.success(t('accountDetail.toast.removed'))

      // Check if deleting self from current account
      const isDeletingSelf =
        currentUser?.id &&
        currentAccountContext?.queue &&
        user.id === currentUser.id &&
        currentAccountContext.queue !== 'default'

      if (isDeletingSelf) {
        // Get current account ID by name to compare with accountId
        try {
          const currentAccountInfo = await apiAccountGetByName(currentAccountContext.queue)
          if (currentAccountInfo.data?.id === accountId) {
            // Switch to default account
            const authResponse = await apiQueueSwitch('default')
            // Update user context with new token data
            if (authResponse?.context) {
              setAccountContext(authResponse.context)
            }
            toast.success(t('accountDetail.toast.switchedToDefaultAccount'))
            // Don't refresh cache for old account - component will be unmounted
            // and parent component will handle the account switch
            return
          }
        } catch {
          // Token update failed, but user deletion succeeded
          toast.error(t('accountDetail.toast.accountSwitchFailed'))
          // Don't refresh member list if account switch failed (user is no longer in the account)
          return
        }
      }

      // Refresh member list if not deleting self
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'users'],
      })
    },
  })

  const { data: usersOutOfProjectData, isLoading: isLoadingUsersOutOfProject } = useQuery({
    queryKey: ['usersOutOfProject', accountId, isAdminView ? 'admin' : 'user'],
    queryFn: () => listUsersOutOfAccount(accountId),
    select: (res) => res.data,
  })

  useEffect(() => {
    if (isLoadingUsersOutOfProject) return
    if (!usersOutOfProjectData) return
    setUsersOutOfProject(usersOutOfProjectData)
  }, [usersOutOfProjectData, isLoadingUsersOutOfProject])

  const columns = useMemo<ColumnDef<IUserInAccount>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('accountDetail.table.headers.name')} />
        ),
        cell: ({ row }) => (
          <UserLabel
            info={{
              username: row.original.name,
              nickname: row.original.userInfo.nickname,
            }}
          />
        ),
      },
      {
        accessorKey: 'role',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('accountDetail.table.headers.role')} />
        ),
        cell: ({ row }) => {
          return <UserRoleBadge role={row.getValue('role')} />
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'accessmode',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('accountDetail.table.headers.accessmode')}
          />
        ),
        cell: ({ row }) => <UserAccessBadge access={row.getValue('accessmode')} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'capability',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('accountDetail.table.headers.capability')}
          />
        ),
        cell: ({ row }) => {
          return (
            <CapacityBadges
              aid={accountId}
              uid={row.original.id}
              role={row.original.role}
              accessmode={row.original.accessmode}
              quota={row.original.quota}
              editable={editable}
            />
          )
        },
        enableSorting: false,
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          if (!editable) {
            return null
          }
          return (
            <ActionsCell
              user={row.original}
              accountId={accountId}
              t={t}
              updateUser={updateUser}
              setPendingRoleUpdate={setPendingRoleUpdate}
              setPendingDeleteUser={setPendingDeleteUser}
              accessModes={accessModes}
              isDefaultAccount={isDefaultAccount}
              isAdminView={isAdminView}
              currentUser={currentUser as { id: number } | null}
              currentAccountContext={
                currentAccountContext as unknown as { roleQueue?: string } | null
              }
            />
          )
        },
      },
    ],
    [
      accountId,

      t,
      editable,
      updateUser,
      setPendingRoleUpdate,
      setPendingDeleteUser,
      accessModes,
      isDefaultAccount,
      isAdminView,
      currentUser,
      currentAccountContext,
    ]
  )

  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: t('accountDetail.table.filter.searchUser'),
      key: 'name',
    },
    filterOptions: [
      {
        key: 'role',
        title: t('accountDetail.table.filter.role'),
        option: userRoles,
      },
      {
        key: 'accessmode',
        title: t('accountDetail.table.filter.permissions'),
        option: accessModes,
      },
    ],
    getHeader: getHeader,
  }

  return (
    <div className="space-y-4">
      <RoleUpdateConfirmDialog
        user={pendingRoleUpdate?.user ?? null}
        onConfirm={() => {
          if (pendingRoleUpdate) {
            const { user, newRole } = pendingRoleUpdate
            updateUser({
              id: user.id,
              name: user.name,
              role: newRole,
              accessmode: user.accessmode,
            })
            setPendingRoleUpdate(null)
          }
        }}
        onCancel={() => setPendingRoleUpdate(null)}
      />

      <DeleteUserConfirmDialog
        user={pendingDeleteUser ?? null}
        onConfirm={() => {
          if (pendingDeleteUser) {
            deleteUser(pendingDeleteUser)
            setPendingDeleteUser(null)
          }
        }}
        onCancel={() => setPendingDeleteUser(null)}
      />

      <DataTable
        key={`${accountId}-${accountUsersQuery.data?.length}`}
        columns={columns as ColumnDef<unknown>[]}
        query={accountUsersQuery as UseQueryResult<unknown[], Error>}
        storageKey={storageKey}
        toolbarConfig={toolbarConfig}
      >
        {editable && (
          <Dialog>
            <DialogTrigger asChild>
              <Button>
                <UserRoundPlusIcon className="mr-2 size-4" />
                {t('accountDetail.addUser')}
              </Button>
            </DialogTrigger>
            <AddMemberDialog
              accountId={accountId}
              onAddMembers={(users) => {
                addUser(users)
              }}
              usersOutOfProject={usersOutOfProject}
              accessModes={accessModes}
            />
          </Dialog>
        )}
      </DataTable>
    </div>
  )
}

interface ActionsCellProps {
  user: IUserInAccount
  accountId: number
  t: (key: string) => string
  updateUser: (user: IUserInAccountCreate) => void
  setPendingRoleUpdate: (update: { user: IUserInAccountCreate; newRole: string } | null) => void
  setPendingDeleteUser: (user: IUserInAccountCreate | null) => void
  accessModes: Array<{ label: string; value: string }>
  isDefaultAccount: boolean
  isAdminView: boolean
  currentUser: { id: number } | null
  currentAccountContext: { roleQueue?: string } | null
}

const ActionsCell: FC<ActionsCellProps> = ({
  user,
  accountId,
  t,
  updateUser,
  setPendingRoleUpdate,
  setPendingDeleteUser,
  accessModes,
  isDefaultAccount,
  isAdminView,
  currentUser,
  currentAccountContext,
}) => {
  const [quotaDialogOpen, setQuotaDialogOpen] = useState(false)

  const buildResourcesFromQuota = (q: IUserInAccount['quota']) => {
    const keys = new Set<string>()
    if (q?.capability) Object.keys(q.capability).forEach((k) => keys.add(k))
    if (keys.size === 0)
      return [
        { name: 'cpu', capability: '' },
        { name: 'memory', capability: '' },
      ]
    return Array.from(keys).map((k) => ({
      name: k,
      capability: q?.capability?.[k] ?? '',
    }))
  }

  const [resources, setResources] = useState(() => buildResourcesFromQuota(user.quota))
  useEffect(() => {
    if (quotaDialogOpen) {
      setResources(buildResourcesFromQuota(user.quota))
    }
  }, [quotaDialogOpen, user.quota])

  const queryClient = useQueryClient()
  const { mutate: updateQuota, isPending: isQuotaPending } = useMutation({
    mutationFn: (quota: Record<string, string>) =>
      apiUpdateUserOutOfProjectList({
        aid: accountId,
        uid: user.id,
        role: user.role,
        accessmode: user.accessmode,
        quota,
      }),
    onSuccess: async () => {
      toast.success(t('accountDetail.toast.updated'))
      setQuotaDialogOpen(false)
      await queryClient.invalidateQueries({ queryKey: ['account', accountId, 'users'] })
    },
    onError: () => {
      toast.error(t('accountDetail.toast.updateFailed') || '配额更新失败')
    },
  })

  const addResource = () => setResources((r) => [...r, { name: '', capability: '' }])

  const removeResource = (idx: number) => setResources((r) => r.filter((_, i) => i !== idx))

  const onSaveQuota = () => {
    const capability: Record<string, string> = {}
    resources.forEach((res) => {
      if (res.capability !== undefined && res.capability !== '')
        capability[res.name] = String(res.capability)
    })
    updateQuota(capability)
  }

  return (
    <div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">{t('accountDetail.table.actions.moreOptions')}</span>
            <DotsHorizontalIcon className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel className="text-muted-foreground text-xs">
            {t('accountDetail.table.actions.operations')}
          </DropdownMenuLabel>
          {!isDefaultAccount && (
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                {t('accountDetail.table.actions.role')}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup value={`${user.role}`}>
                  {userRoles.map((role) => (
                    <DropdownMenuRadioItem
                      key={role.value}
                      value={`${role.value}`}
                      onClick={() => {
                        const isSelfDemotion =
                          !isAdminView &&
                          currentUser?.id === user.id &&
                          currentAccountContext?.roleQueue === Role.Admin.toString() &&
                          user.role === Role.Admin.toString() &&
                          role.value === Role.User.toString()

                        if (isSelfDemotion) {
                          setPendingRoleUpdate({
                            user: {
                              id: user.id,
                              name: user.name,
                              role: role.value,
                              accessmode: user.accessmode,
                            },
                            newRole: role.value,
                          })
                        } else {
                          updateUser({
                            id: user.id,
                            name: user.name,
                            role: role.value,
                            accessmode: user.accessmode,
                          })
                        }
                      }}
                    >
                      {role.label}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          )}
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              {t('accountDetail.table.actions.permissions')}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <DropdownMenuRadioGroup value={`${user.accessmode}`}>
                {accessModes.map((accessmode) => (
                  <DropdownMenuRadioItem
                    key={accessmode.value}
                    value={`${accessmode.value}`}
                    onClick={() =>
                      updateUser({
                        id: user.id,
                        name: user.name,
                        role: user.role,
                        accessmode: accessmode.value,
                      })
                    }
                    disabled={isDefaultAccount && !isAdminView}
                  >
                    {accessmode.label}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
          <DropdownMenuItem onClick={() => setQuotaDialogOpen(true)}>
            {t('accountDetail.table.actions.editQuota')}
          </DropdownMenuItem>
          {!isDefaultAccount && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => {
                  setPendingDeleteUser(user)
                }}
              >
                {t('accountDetail.table.actions.delete')}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      {quotaDialogOpen && (
        <Dialog open={quotaDialogOpen} onOpenChange={setQuotaDialogOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t('accountForm.quotaLabel')}</DialogTitle>
            </DialogHeader>
            <div className="space-y-2">
              {resources.map((res, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <Input
                    value={res.name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setResources((r) =>
                        r.map((x, i) => (i === idx ? { ...x, name: e.target.value } : x))
                      )
                    }
                    className="w-32 font-mono"
                    placeholder={t('accountForm.resourceName')}
                  />
                  <Input
                    type="text"
                    placeholder={t('accountForm.capabilityPlaceholder')}
                    value={res.capability}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setResources((r) =>
                        r.map((x, i) => (i === idx ? { ...x, capability: e.target.value } : x))
                      )
                    }
                    className="font-mono"
                  />
                  <Button size="icon" variant="outline" onClick={() => removeResource(idx)}>
                    <XIcon className="size-4" />
                  </Button>
                </div>
              ))}
              <Button type="button" variant="secondary" onClick={addResource}>
                <CirclePlusIcon className="size-4" />
                {t('accountForm.addQuotaButton')}
              </Button>
              <p className="text-muted-foreground">{t('accountForm.quotaDescription')}</p>
            </div>
            <DialogFooter>
              <Button onClick={onSaveQuota} disabled={isQuotaPending}>
                {t('accountDetail.form.save')}
              </Button>
              <DialogClose asChild>
                <Button variant="outline">{t('accountDetail.form.cancel')}</Button>
              </DialogClose>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

interface RoleUpdateConfirmDialogProps {
  user: IUserInAccountCreate | null
  onConfirm: () => void
  onCancel: () => void
}

const RoleUpdateConfirmDialog: FC<RoleUpdateConfirmDialogProps> = ({
  user,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation()

  return (
    <AlertDialog open={user !== null} onOpenChange={(open) => !open && onCancel()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('accountDetail.dialog.title.confirmSelfDemotion')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('accountDetail.dialog.description.confirmSelfDemotion')}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>
            {t('accountDetail.dialog.cancel')}
          </AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onConfirm}>
            {t('accountDetail.dialog.confirm')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

interface DeleteUserConfirmDialogProps {
  user: IUserInAccountCreate | null
  onConfirm: () => void
  onCancel: () => void
}

const DeleteUserConfirmDialog: FC<DeleteUserConfirmDialogProps> = ({
  user,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation()

  return (
    <AlertDialog open={user !== null} onOpenChange={(open) => !open && onCancel()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('accountDetail.dialog.title.deleteUser')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('accountDetail.dialog.description.deleteUser', { name: user?.name })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>
            {t('accountDetail.dialog.cancel')}
          </AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onConfirm}>
            {t('accountDetail.dialog.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

interface AddMemberDialogProps {
  accountId: number
  onAddMembers: (users: IUserInAccountCreate[]) => void
  usersOutOfProject: IUserInAccount[]
  accessModes: Array<{ label: string; value: string }>
}

function AddMemberDialog({ onAddMembers, usersOutOfProject }: AddMemberDialogProps) {
  const { t } = useTranslation()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      userIds: [],
      role: '2',
      accessmode: '2',
    },
  })

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    const usersToAdd = values.userIds.map((userId) => {
      const user = usersOutOfProject.find((u) => u.id.toString() === userId)
      return {
        id: user?.id as number,
        name: user?.name as string,
        role: values.role,
        accessmode: values.accessmode,
        attributes: user?.userInfo,
      }
    })

    onAddMembers(usersToAdd)
    form.reset()
  }

  const userOptions = useMemo(
    () =>
      usersOutOfProject.map((user) => ({
        value: user.id.toString(),
        label: user.userInfo.nickname || user.name,
        labelNote: user.name,
      })),
    [usersOutOfProject]
  )

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('accountDetail.addUser')}</DialogTitle>
      </DialogHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="userIds"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('accountDetail.form.user')}
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <SelectBox
                    options={userOptions}
                    value={field.value}
                    onChange={field.onChange}
                    placeholder={t('accountDetail.form.selectUser')}
                    inputPlaceholder={t('accountDetail.form.searchUser')}
                    emptyPlaceholder={t('accountDetail.form.noUsersFound')}
                  />
                </FormControl>
                <FormDescription>{t('accountDetail.form.selectUsersToAdd')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="role"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('accountDetail.form.role')}
                  <FormLabelMust />
                </FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value.toString()}>
                  <FormControl>
                    <SelectTrigger className="w-full" id="role">
                      <SelectValue placeholder="Select users" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="2">{t('accountDetail.form.userRole.normal')}</SelectItem>
                      <SelectItem value="3">{t('accountDetail.form.userRole.admin')}</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>{t('accountDetail.form.roleDescription')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="accessmode"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('accountDetail.form.access')}
                  <FormLabelMust />
                </FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value.toString()}>
                  <FormControl>
                    <SelectTrigger className="w-full" id="accessmode">
                      <SelectValue placeholder="Select users" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="2">{t('accountDetail.form.access.readOnly')}</SelectItem>
                      <SelectItem value="3">{t('accountDetail.form.access.readWrite')}</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>{t('accountDetail.form.accessDescription')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">{t('accountDetail.form.cancel')}</Button>
            </DialogClose>
            <Button type="submit">{t('accountDetail.form.add')}</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  )
}
