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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { Briefcase, Calendar, Layers, Users } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'

import { AccountMemberTable } from '@/components/account/account-member-table'
import { TimeDistance } from '@/components/custom/time-distance'
import DetailPage, {
  detailLinkOptions,
  detailValidateSearch,
} from '@/components/layout/detail-page'
import DetailTitle from '@/components/layout/detail-title'

import { apiUserInProjectList } from '@/services/api/account'
import { queryAccountByID } from '@/services/query/account'

import Quota from './-components/account-quota'

export const Route = createFileRoute('/admin/accounts/$id')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  loader: async ({ params, context: { queryClient } }) => {
    const id = Number(params.id)
    try {
      const { data } = await queryClient.ensureQueryData(queryAccountByID(id))
      return {
        crumb: data?.nickname || params.id,
      }
    } catch {
      // 捕获错误，让组件能够正常渲染并显示错误页面
      return {
        crumb: params.id,
      }
    }
  },
})

function RouteComponent() {
  const { t } = useTranslation()
  const aid = Route.useParams().id
  const pid = useMemo(() => Number(aid), [aid])
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()

  const {
    data: accountInfo,
    isLoading: isLoadingAccount,
    error,
  } = useQuery({
    ...queryAccountByID(pid),
    retry: false, // 禁用自动重试，让错误立即显示
  })

  const accountUsersQuery = useQuery({
    queryKey: ['account', pid, 'users'],
    queryFn: () => apiUserInProjectList(pid),
    select: (res) => res.data,
    enabled: !!accountInfo, // 只在账户信息加载成功后再查询用户列表
  })

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
  if (error || !accountInfo) {
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
    <AccountMemberTable accountId={pid} storageKey="admin_account_users" />
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
