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
import { useTranslation } from 'react-i18next'
import { DotsHorizontalIcon } from '@radix-ui/react-icons'
import { ColumnDef } from '@tanstack/react-table'
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
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
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
import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { showErrorToast } from '@/utils/toast'
import { toast } from 'sonner'
import { DataTable } from '@/components/custom/DataTable'
import { DataTableColumnHeader } from '@/components/custom/DataTable/DataTableColumnHeader'
import { DataTableToolbarConfig } from '@/components/custom/DataTable/DataTableToolbar'
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUpdateUserAttributes,
  apiAdminUserUpdateRole,
  IUserAttributes,
} from '@/services/api/admin/user'
import { useAtomValue } from 'jotai'
import { globalUserInfo } from '@/utils/store'
import { Role } from '@/services/api/auth'
import { ProjectStatus } from '@/services/api/account'
import UserLabel from '@/components/label/UserLabel'
import UserRoleBadge from '@/components/badge/UserRoleBadge'
import UserStatusBadge from '@/components/badge/UserStatusBadge'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'

interface TUser {
  id: number
  name: string
  role: string
  status: string
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

export const User = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { name: currentUserName } = useAtomValue(globalUserInfo)
  const [editUser, setEditUser] = useState<TUser | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)

  const userQuery = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.data.map((item) => ({
        id: item.id,
        name: item.name,
        role: item.role.toString(),
        status: item.status.toString(),
        attributes: item.attributes,
      })),
  })

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
    return [
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
                        if (user.name === currentUserName) {
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
  }, [deleteUser, currentUserName, updateRole, t])

  return (
    <>
      <DataTable
        info={{
          title: t('userTable.title'),
          description: t('userTable.description'),
        }}
        storageKey="admin_user"
        query={userQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
      <UserEditDialog open={editDialogOpen} onOpenChange={setEditDialogOpen} user={editUser} />
    </>
  )
}
