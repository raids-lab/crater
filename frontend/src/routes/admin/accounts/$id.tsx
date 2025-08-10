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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { Briefcase, Calendar, Layers, UserRoundPlusIcon, Users } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'

import ResourceBadges from '@/components/badge/ResourceBadges'
import UserAccessBadge from '@/components/badge/UserAccessBadge'
import UserRoleBadge, { userRoles } from '@/components/badge/UserRoleBadge'
import SelectBox from '@/components/custom/SelectBox'
import { TimeDistance } from '@/components/custom/TimeDistance'
import FormLabelMust from '@/components/form/FormLabelMust'
import UserLabel from '@/components/label/user-label'
import DetailPage, {
  detailLinkOptions,
  detailValidateSearch,
} from '@/components/layout/detail-page'
import DetailTitle from '@/components/layout/detail-title'
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

import {
  Access,
  IUserInAccount,
  IUserInAccountCreate,
  apiAddUser,
  apiRemoveUser,
  apiUpdateUser,
  apiUserInProjectList,
  apiUserOutOfProjectList,
} from '@/services/api/account'
import { queryAccountByID } from '@/services/query/account'

import Quota from './-components/account-quota'

export const Route = createFileRoute('/admin/accounts/$id')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  loader: async ({ params, context: { queryClient } }) => {
    const id = Number(params.id)
    const { data } = await queryClient.ensureQueryData(queryAccountByID(id))
    return {
      crumb: data?.nickname || params.id,
    }
  },
})

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

function RouteComponent() {
  const { t } = useTranslation()
  const id = Route.useParams().id
  const queryClient = useQueryClient()
  const pid = useMemo(() => Number(id), [id])
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()

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

  const accountUsersQuery = useQuery({
    queryKey: ['account', pid, 'users'],
    queryFn: () => apiUserInProjectList(pid),
    select: (res) => res.data,
  })

  const { data: accountInfo, isLoading: isLoadingAccount } = useQuery(queryAccountByID(pid))

  const { mutate: addUser } = useMutation({
    mutationFn: (users: IUserInAccountCreate[]) =>
      Promise.all(users.map((user) => apiAddUser(pid, user))),
    onSuccess: async () => {
      toast.success(t('accountDetail.toast.added'))
      await queryClient.invalidateQueries({
        queryKey: ['account', pid, 'users'],
      })
    },
  })

  const { mutate: updateUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiUpdateUser(pid, user),
    onSuccess: async () => {
      toast.success(t('accountDetail.toast.updated'))
      await queryClient.invalidateQueries({
        queryKey: ['account', pid, 'users'],
      })
    },
  })

  const [usersOutOfProject, setUsersOutOfProject] = useState<IUserInAccount[]>([])

  const { mutate: deleteUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiRemoveUser(pid, user),
    onSuccess: async () => {
      toast.success(t('accountDetail.toast.removed'))
      await queryClient.invalidateQueries({
        queryKey: ['account', pid, 'users'],
      })
    },
  })

  const { data: usersOutOfProjectData, isLoading: isLoadingUsersOutOfProject } = useQuery({
    queryKey: ['usersOutOfProject', pid],
    queryFn: () => apiUserOutOfProjectList(pid),
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
          return <ResourceBadges resources={row.original.quota.capability} />
        },
        enableSorting: false,
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
                              onClick={() =>
                                updateUser({
                                  id: user.id,
                                  name: user.name,
                                  role: role.value,
                                  accessmode: user.accessmode,
                                })
                              }
                            >
                              {role.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
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
                            >
                              {accessmode.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                    <DropdownMenuSeparator />
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem>{t('accountDetail.table.actions.delete')}</DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('accountDetail.dialog.title.deleteUser')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('accountDetail.dialog.description.deleteUser', {
                        name: user?.name,
                      })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('accountDetail.dialog.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        deleteUser(user)
                      }}
                    >
                      {t('accountDetail.dialog.delete')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          )
        },
      },
    ],
    [deleteUser, updateUser, t, accessModes]
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
    // 处理多个用户添加
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

  // 将用户列表转换为SelectBox需要的格式
  const userOptions = useMemo(
    () =>
      usersOutOfProject.map((user) => ({
        value: user.id.toString(),
        label: user.userInfo.nickname || user.name,
        labelNote: user.name,
      })),
    [usersOutOfProject]
  )

  // 加载中状态
  if (isLoadingAccount) {
    return (
      <DetailPage
        header={
          <div className="flex items-center space-x-4">
            <Skeleton className="h-12 w-12 rounded-full" />
            <div>
              <Skeleton className="mb-2 h-8 w-40" />
              <Skeleton className="h-4 w-20" />
            </div>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    )
  }

  // 错误状态或数据不存在
  if (!accountInfo) {
    return (
      <DetailPage
        header={
          <div>
            <h1 className="text-2xl font-bold text-red-500">{t('accountDetail.error.title')}</h1>
            <p className="text-muted-foreground">{t('accountDetail.error.description')}</p>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    )
  }

  // 账户头部内容
  const header = (
    <DetailTitle icon={Briefcase} title={accountInfo.nickname} description={accountInfo.name} />
  )

  // 账户基本信息
  const info = [
    {
      icon: Users,
      title: t('accountDetail.info.userCount'),
      value: accountUsersQuery.data?.length || '加载中...',
    },
    {
      icon: Layers,
      title: t('accountDetail.info.accountLevel'),
      value: '标准',
    },
    {
      icon: Calendar,
      title: t('accountDetail.info.expiry'),
      value: <TimeDistance date={accountInfo.expiredAt} />,
    },
  ]

  // 用户管理组件
  const UserManagement = () => (
    <DataTable
      storageKey="admin_account_users"
      query={accountUsersQuery}
      columns={columns}
      toolbarConfig={toolbarConfig}
    >
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
                    <Select onValueChange={field.onChange} defaultValue={field.value.toString()}>
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
      </Dialog>
    </DataTable>
  )

  // 标签页配置
  const tabs = [
    {
      key: 'users',
      icon: Users,
      label: t('accountDetail.tabs.users'),
      children: <UserManagement />,
      scrollable: true,
    },
    {
      key: 'quota',
      icon: Layers,
      label: t('accountDetail.tabs.quota'),
      children: (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Quota accountID={pid} />
        </div>
      ),
      scrollable: true,
    },
  ]

  return (
    <DetailPage
      header={header}
      info={info}
      tabs={tabs}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab))}
    />
  )
}
