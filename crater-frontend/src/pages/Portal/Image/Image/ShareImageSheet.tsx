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

'use client'

import SelectBox from '@/components/custom/SelectBox'
import {
  type ImageAccounts,
  apiUserGetUngrantedAccounts,
  type ImageUsers,
  apiUserSearchUser,
  apiUserGetImageShareObjects,
  apiUserDeleteImageShare,
  apiUserAddImageShare,
} from '@/services/api/imagepack'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  type UseMutationResult,
  useQuery,
  useQueryClient,
  useMutation,
} from '@tanstack/react-query'
import { Loader2, X, Users, Building2, PackagePlusIcon } from 'lucide-react'
import { SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { useTranslation } from 'react-i18next'
import type { IResponse } from '@/services/types'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'
import LoadableButton from '@/components/button/LoadableButton'
import { toast } from 'sonner'
import { Separator } from '@/components/ui/separator'

interface AccountAccessListProps {
  imageID: number
  accountList: ImageAccounts[]
  isLoading: boolean
  onRemoveAccess: (id: number) => void
  removeMutation: UseMutationResult<
    IResponse<string>,
    Error,
    {
      id: number
      type: ShareObjectType
    },
    unknown
  >
  setSelectedAccounts: (accounts: string[]) => void
  selectedAccounts: string[]
}

export function AccountAccessList({
  imageID,
  accountList,
  isLoading,
  onRemoveAccess,
  removeMutation,
  setSelectedAccounts,
  selectedAccounts = [],
}: AccountAccessListProps) {
  const { t } = useTranslation()

  // 获取未授权账户
  const {
    data: ungrantedAccountsData,
    isLoading: isLoadingUngranted,
    refetch: refetchUngrantedAccounts,
  } = useQuery({
    queryKey: ['image', 'ungranted-accounts', imageID],
    queryFn: () => apiUserGetUngrantedAccounts({ imageID }),
    select: (res) => {
      return res.data.data.accountList.map((account) => {
        return {
          value: account.id.toString(),
          label: account.name,
        }
      })
    },
  })

  // 当添加或删除账户时，刷新未授权账户列表
  useEffect(() => {
    refetchUngrantedAccounts()
  }, [accountList, refetchUngrantedAccounts])

  return (
    <>
      <div className="flex items-center justify-between text-sm font-medium">
        <span>{t('已有访问权限的账户')}</span>
        <span className="bg-muted rounded-full px-2 py-0.5 text-xs">{accountList.length}</span>
      </div>
      <Separator className="my-2" />
      {isLoading ? (
        <div className="flex justify-center py-3">
          <Loader2 className="text-muted-foreground h-5 w-5 animate-spin" />
        </div>
      ) : (
        <ScrollArea className="h-[240px] w-full">
          <div className="space-y-2 p-4">
            {!accountList.length ? (
              <div className="text-muted-foreground py-3 text-center text-sm">
                {t('暂无账户有访问权限')}
              </div>
            ) : (
              accountList.map((account) => (
                <div
                  key={account.id}
                  className="hover:bg-muted group flex items-center justify-between rounded-lg px-2 py-2"
                >
                  <div className="flex-1 truncate">
                    <span className="font-medium">{account.name}</span>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRemoveAccess(account.id)}
                    aria-label={`移除 ${account.name} 的访问权限`}
                    className="h-7 w-7 p-0 opacity-0 transition-opacity group-hover:opacity-100"
                    disabled={removeMutation.isPending}
                  >
                    {removeMutation.isPending && removeMutation.variables?.id === account.id ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <X className="h-3.5 w-3.5" />
                    )}
                  </Button>
                </div>
              ))
            )}
          </div>
        </ScrollArea>
      )}

      <Separator className="my-1" />
      <div className="space-y-2 pt-1">
        <div className="text-sm font-medium">{t('添加账户')}</div>
        <SelectBox
          options={ungrantedAccountsData ?? []}
          value={selectedAccounts}
          inputPlaceholder={t('请输入')}
          placeholder={t('请输入')}
          onChange={setSelectedAccounts}
          emptyPlaceholder={isLoadingUngranted ? '加载中...' : undefined}
        />
      </div>
    </>
  )
}

interface UserAccessListProps {
  imageID: number
  userList: ImageUsers[]
  isLoading: boolean
  onRemoveAccess: (id: number) => void
  removeMutation: UseMutationResult<
    IResponse<string>,
    Error,
    {
      id: number
      type: ShareObjectType
    },
    unknown
  >
  setSelectedUsers: (users: string[]) => void
  selectedUsers: string[]
}

export function UserAccessList({
  imageID,
  userList,
  isLoading,
  onRemoveAccess,
  removeMutation,
  setSelectedUsers,
  selectedUsers = [],
}: UserAccessListProps) {
  const { t } = useTranslation()
  const {
    data: ungrantedUsers,
    isLoading: isLoadingUsers,
    refetch: refetchUngrantedUsers,
  } = useQuery({
    queryKey: ['image', 'users', imageID],
    queryFn: () => apiUserSearchUser({ imageID }),
    select: (res) => {
      return res.data.data.userList.map((user) => {
        return {
          value: user.id.toString(),
          label: user.nickname,
          labelNote: user.name,
        }
      })
    },
  })

  useEffect(() => {
    refetchUngrantedUsers()
  }, [userList, refetchUngrantedUsers])

  return (
    <>
      <div className="flex items-center justify-between text-sm font-medium">
        <span>{t('已有访问权限的用户')}</span>
        <span className="bg-muted rounded-full px-2 py-0.5 text-xs">{userList.length}</span>
      </div>
      <Separator className="my-2" />
      {isLoading ? (
        <div className="flex justify-center py-3">
          <Loader2 className="text-muted-foreground h-5 w-5 animate-spin" />
        </div>
      ) : (
        <ScrollArea className="h-[240px] w-full">
          <div className="space-y-2 p-4">
            {!userList.length ? (
              <div className="text-muted-foreground py-3 text-center text-sm">
                {t('暂无其他用户拥有访问权限')}
              </div>
            ) : (
              <>
                {userList.map((user) => (
                  <div key={user.id}>
                    <div className="hover:bg-muted group flex items-center justify-between rounded-lg px-2 py-2">
                      <div className="flex-1 truncate">
                        <span className="font-medium">{user.nickname}</span>
                        <span className="text-muted-foreground ml-2 text-sm">@{user.name}</span>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onRemoveAccess(user.id)}
                        aria-label={`移除 ${user.name} 的访问权限`}
                        className="h-7 w-7 p-0 opacity-0 transition-opacity group-hover:opacity-100"
                        disabled={removeMutation.isPending}
                      >
                        {removeMutation.isPending && removeMutation.variables?.id === user.id ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <X className="h-3.5 w-3.5" />
                        )}
                      </Button>
                    </div>
                  </div>
                ))}
              </>
            )}
          </div>
        </ScrollArea>
      )}
      <Separator className="my-1" />
      <div className="space-y-2 pt-1">
        <div className="text-sm font-medium">{t('添加用户')}</div>
        <SelectBox
          options={ungrantedUsers ?? []}
          value={selectedUsers}
          inputPlaceholder={t('请输入')}
          placeholder={t('请输入')}
          onChange={setSelectedUsers}
          emptyPlaceholder={isLoadingUsers ? '搜索中...' : undefined}
        />
      </div>
    </>
  )
}

type ShareObjectType = 'user' | 'account'

interface ImageShareSheetContentProps {
  imageID: number
  imageName: string
}

export function ImageShareSheetContent({ imageID, imageName }: ImageShareSheetContentProps) {
  const { t } = useTranslation()
  const [selectedUsers, setSelectedUsers] = useState<string[]>([])
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState<ShareObjectType>('user')
  const queryClient = useQueryClient()

  // 使用 React Query 获取共享对象数据
  const { data, isLoading } = useQuery({
    queryKey: ['image', 'share-objects', imageID],
    queryFn: async () => {
      const response = await apiUserGetImageShareObjects({ imageID })
      return response.data.data
    },
  })

  // 删除访问权限的mutation
  const removeMutation = useMutation({
    mutationFn: async ({ id, type }: { id: number; type: ShareObjectType }) => {
      const response = await apiUserDeleteImageShare({
        imageID: imageID,
        type: type,
        id: id,
      })
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['image', 'share-objects', imageID],
      })
      toast.success(t('删除访问权限成功!'))
    },
  })

  // 添加访问权限的mutation
  const addMutation = useMutation({
    mutationFn: (data: { idList: number[]; type: ShareObjectType; imageID: number }) => {
      return apiUserAddImageShare(data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['image', 'share-objects', imageID],
      })
      toast.success(t('添加访问权限成功!'))
    },
  })

  // 处理删除访问权限
  const handleRemoveAccess = (id: number, type: ShareObjectType) => {
    removeMutation.mutate({ id, type })
  }

  // 处理添加访问权限
  const handleAddAccess = (idList: number[], type: ShareObjectType) => {
    if (idList.length > 0) {
      addMutation.mutate({ idList, type, imageID })
    } else {
      toast.error(t('请至少选择一个分享对象'))
    }
  }

  return (
    <SandwichLayout
      footer={
        <LoadableButton
          onClick={() => {
            if (activeTab === 'account') {
              handleAddAccess(
                selectedAccounts.map((id) => Number.parseInt(id)),
                activeTab
              )
              setSelectedAccounts([])
              return
            } else if (activeTab === 'user') {
              handleAddAccess(
                selectedUsers.map((id) => Number.parseInt(id)),
                activeTab
              )
              setSelectedUsers([])
            } else {
              toast.error(t('分享对象类别错误，请联系管理员！'))
            }
          }}
          isLoading={addMutation.isPending}
          isLoadingText="正在提交"
          type="submit"
        >
          <PackagePlusIcon />
          确认提交
        </LoadableButton>
      }
    >
      <SheetHeader className="bg-muted/30 border-b">
        <SheetTitle className="text-md flex items-center font-medium">
          镜像名:
          <span className="text-muted-foreground text-xs">『{imageName}』</span>
        </SheetTitle>
      </SheetHeader>

      <div className="space-y-2 p-1">
        {/* 选项卡切换 */}
        <Tabs
          defaultValue="user"
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as ShareObjectType)}
          className="w-full"
        >
          <TabsList className="mb-2 grid w-full grid-cols-2">
            <TabsTrigger
              value="user"
              className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground flex items-center gap-2"
            >
              <Users className="h-4 w-4" />
              <span>{t('用户')}</span>
              {activeTab === 'user' && (
                <span className="bg-primary-foreground text-primary ml-auto rounded-full px-2 py-0.5 text-xs">
                  {data?.userList.length || 0}
                </span>
              )}
            </TabsTrigger>
            <TabsTrigger
              value="account"
              className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground flex items-center gap-2"
            >
              <Building2 className="h-4 w-4" />
              <span>{t('账户')}</span>
              {activeTab === 'account' && (
                <span className="bg-primary-foreground text-primary ml-auto rounded-full px-2 py-0.5 text-xs">
                  {data?.accountList.length || 0}
                </span>
              )}
            </TabsTrigger>
          </TabsList>

          <div className="relative">
            <TabsContent value="user" className="mt-0 w-full space-y-4">
              <div className="bg-card w-full rounded-lg border p-4 shadow-sm">
                <UserAccessList
                  imageID={imageID}
                  userList={data?.userList || []}
                  isLoading={isLoading}
                  onRemoveAccess={(id) => handleRemoveAccess(id, activeTab)}
                  removeMutation={removeMutation}
                  setSelectedUsers={setSelectedUsers}
                  selectedUsers={selectedUsers}
                />
              </div>
            </TabsContent>

            <TabsContent value="account" className="mt-0 w-full space-y-4">
              <div className="bg-card w-full rounded-lg border p-4 shadow-sm">
                <AccountAccessList
                  imageID={imageID}
                  accountList={data?.accountList || []}
                  isLoading={isLoading}
                  onRemoveAccess={(id) => handleRemoveAccess(id, activeTab)}
                  removeMutation={removeMutation}
                  setSelectedAccounts={setSelectedAccounts}
                  selectedAccounts={selectedAccounts}
                />
              </div>
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </SandwichLayout>
  )
}

interface ImageShareSheetProps extends SandwichSheetProps {
  imageID: number
  imageName: string
}

export function ImageShareSheet({ imageID, imageName, ...props }: ImageShareSheetProps) {
  return (
    <SandwichSheet {...props}>
      <ImageShareSheetContent imageID={imageID} imageName={imageName} />
    </SandwichSheet>
  )
}
