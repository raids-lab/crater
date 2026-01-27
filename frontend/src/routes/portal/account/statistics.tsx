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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtomValue } from 'jotai'
import { useMemo } from 'react'

import { Skeleton } from '@/components/ui/skeleton'

import PageTitle from '@/components/layout/page-title'
// 引入通用统计面板
import { StatisticsDashboard } from '@/components/statistics/statistics-dashboard'

import { apiAccountGetByName } from '@/services/api/account'

import { atomUserContext } from '@/utils/store'

export const Route = createFileRoute('/portal/account/statistics')({
  component: RouteComponent,
  loader: () => {
    return {
      // 这里的 translation key 建议在你的 i18n 文件中添加，或者使用 defaultValue
      crumb: t('navigation.statistics', { defaultValue: 'Resource Statistics' }),
    }
  },
})

function RouteComponent() {
  // 1. 获取当前上下文中的账户名 (queue)
  const accountContext = useAtomValue(atomUserContext)
  const accountName = accountContext?.queue

  // 2. 根据账户名获取账户详细信息 (主要是为了拿到 ID)
  const {
    data: accountInfo,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['account', accountName],
    queryFn: () => apiAccountGetByName(accountName!),
    select: (res) => res.data,
    // 只有当 accountName 存在且不是 default 时才请求
    enabled: !!accountName && accountName !== 'default',
    retry: false,
  })

  const accountId = useMemo(() => accountInfo?.id, [accountInfo])

  // 3. 处理 "default" 账户的情况 (通常默认队列没有独立账单统计，保持与 member 页一致)
  if (accountName === 'default') {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="text-center">
          <h1 className="text-muted-foreground text-2xl font-bold">
            {t('navigation.statistics', { defaultValue: 'Resource Statistics' })}
          </h1>
          <p className="text-muted-foreground mt-2">
            {t('accountDetail.defaultAccount.notSupported', {
              defaultValue: 'Not supported for default account',
            })}
          </p>
        </div>
      </div>
    )
  }

  // 4. Loading 状态
  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  // 5. 错误或无数据状态
  if (error || !accountInfo || !accountId) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-red-500">
            {t('accountDetail.error.title', { defaultValue: 'Error' })}
          </h1>
          <p className="text-muted-foreground mt-2">
            {t('accountDetail.error.description', {
              defaultValue: 'Failed to load account information',
            })}
          </p>
        </div>
      </div>
    )
  }

  // 6. 正常渲染
  return (
    <div className="space-y-4">
      <PageTitle
        title={t('navigation.statistics', { defaultValue: 'Resource Statistics' })}
        description={t('accountDetail.tabs.statistics', {
          defaultValue: 'View resource usage trends and statistics for this account.',
        })}
      />

      {/* 核心复用组件 */}
      <StatisticsDashboard scope="account" targetID={accountId} enabled={true} />
    </div>
  )
}
