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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { useAtomValue, useSetAtom } from 'jotai'
import { EllipsisVerticalIcon as DotsHorizontalIcon, UserRoundPlusIcon } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
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
          const user = row.original
          if (!editable) {
            return null
          }

          return (
            <div>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="h-8 w-8 p-0"
                    title={t('accountDetail.table.actions.moreOptions')}
                  >
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
                                // Check if user is demoting themselves from admin to user in non-admin view
                                const isSelfDemotion =
                                  !isAdminView &&
                                  currentUser?.id === user.id &&
                                  currentAccountContext?.roleQueue === Role.Admin &&
                                  user.role === Role.Admin.toString() &&
                                  role.value === Role.User.toString()

                                if (isSelfDemotion) {
                                  // Show confirmation dialog
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
                                  // Direct update
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
                  {!isDefaultAccount && (
                    <>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        onClick={() => {
                          // Set pending user for deletion dialog
                          setPendingDeleteUser(user)
                        }}
                      >
                        {t('accountDetail.table.actions.delete')}
                      </DropdownMenuItem>
                    </>
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )
        },
      },
    ],
    [
      accountId,
      editable,
      updateUser,
      t,
      accessModes,
      isAdminView,
      currentUser,
      currentAccountContext,
      setPendingRoleUpdate,
      setPendingDeleteUser,
      isDefaultAccount,
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

  const [openSheet, setOpenSheet] = useState(false)

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      userIds: [],
      role: '2',
      accessmode: '2',
    },
  })

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Handle multiple user additions
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

    addUser(usersToAdd)
    setOpenSheet(false)
  }

  // Convert user list to SelectBox format
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
    <>
      {/* Confirmation dialog for self-demotion */}
      <AlertDialog
        open={pendingRoleUpdate !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRoleUpdate(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('accountDetail.dialog.title.confirmSelfDemotion')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t('accountDetail.dialog.description.confirmSelfDemotion')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setPendingRoleUpdate(null)}>
              {t('accountDetail.dialog.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (pendingRoleUpdate) {
                  updateUser(pendingRoleUpdate.user)
                  setPendingRoleUpdate(null)
                }
              }}
            >
              {t('accountDetail.dialog.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Unified confirmation dialog for all deletions */}
      <AlertDialog
        open={pendingDeleteUser !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDeleteUser(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pendingDeleteUser
                ? (() => {
                    const isSelfDeletion =
                      !isAdminView &&
                      currentUser?.id === pendingDeleteUser.id &&
                      currentAccountContext?.queue &&
                      currentAccountContext.queue !== 'default'
                    return isSelfDeletion
                      ? t('accountDetail.dialog.title.confirmSelfDeletion')
                      : t('accountDetail.dialog.title.deleteUser')
                  })()
                : t('accountDetail.dialog.title.deleteUser')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDeleteUser
                ? (() => {
                    const isSelfDeletion =
                      !isAdminView &&
                      currentUser?.id === pendingDeleteUser.id &&
                      currentAccountContext?.queue &&
                      currentAccountContext.queue !== 'default'
                    if (!isAdminView && isSelfDeletion) {
                      // User view deleting themselves: special message
                      return t('accountDetail.dialog.description.confirmSelfDeletion')
                    }
                    // Admin view or user view deleting others: use same message
                    return t('accountDetail.dialog.description.deleteUser', {
                      name: pendingDeleteUser?.name,
                    })
                  })()
                : ''}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setPendingDeleteUser(null)}>
              {t('accountDetail.dialog.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (pendingDeleteUser) {
                  deleteUser(pendingDeleteUser)
                  setPendingDeleteUser(null)
                }
              }}
            >
              {t('accountDetail.dialog.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <DataTable
        storageKey={storageKey}
        query={accountUsersQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      >
        {editable && !isDefaultAccount && (
          <Dialog open={openSheet} onOpenChange={setOpenSheet}>
            <DialogTrigger asChild>
              <Button className="h-8">
                <UserRoundPlusIcon className="size-4" />
                {t('accountDetail.addUser')}
              </Button>
            </DialogTrigger>
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
                        <FormDescription>
                          {t('accountDetail.form.selectUsersToAdd')}
                        </FormDescription>
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
                        <Select
                          onValueChange={field.onChange}
                          defaultValue={field.value.toString()}
                        >
                          <FormControl>
                            <SelectTrigger className="w-full" id="role">
                              <SelectValue placeholder="Select users" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectGroup>
                              <SelectItem value="2">
                                {t('accountDetail.form.userRole.normal')}
                              </SelectItem>
                              <SelectItem value="3">
                                {t('accountDetail.form.userRole.admin')}
                              </SelectItem>
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
                        <Select
                          onValueChange={field.onChange}
                          defaultValue={field.value.toString()}
                        >
                          <FormControl>
                            <SelectTrigger className="w-full" id="accessmode">
                              <SelectValue placeholder="Select users" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectGroup>
                              <SelectItem value="2">
                                {t('accountDetail.form.access.readOnly')}
                              </SelectItem>
                              <SelectItem value="3">
                                {t('accountDetail.form.access.readWrite')}
                              </SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          {t('accountDetail.form.accessDescription')}
                        </FormDescription>
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
          </Dialog>
        )}
      </DataTable>
    </>
  )
}
