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
import { useAtomValue } from 'jotai'
import { Activity, Calendar, Database, GpuIcon, List, User, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import TipBadge from '@/components/badge/tip-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import DetailPage, { DetailPageCoreProps } from '@/components/layout/detail-page'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'
// 引入新的通用组件
import { StatisticsDashboard } from '@/components/statistics/statistics-dashboard'

import { Role } from '@/services/api/auth'
import { apiGetUser } from '@/services/api/user'

import { getUserPseudonym } from '@/utils/pseudonym'
import { globalHideUsername } from '@/utils/store'
import { configGrafanaUserAtom } from '@/utils/store/config'

import RecentActivity from './recent-activity'
import SharedItems from './shared-items'
import { UserAvatar } from './user-avatar'
import { UserJobsOverview } from './user-jobs'

export default function UserDetail({ name, ...props }: DetailPageCoreProps & { name: string }) {
  const { t } = useTranslation()
  const hideUsername = useAtomValue(globalHideUsername)
  const grafanaUser = useAtomValue(configGrafanaUserAtom)

  // 1. 获取用户信息
  const { data: user } = useQuery({
    queryKey: ['user', name],
    queryFn: () => apiGetUser(name || ''),
    select: (data) => data.data,
    enabled: !!name,
  })

  // Header 部分保持不变
  const header = (
    <div className="flex items-center space-x-4">
      <UserAvatar user={user} className="size-20" size={80} />
      <div>
        <h1 className="flex items-center gap-2 text-3xl font-bold">
          {hideUsername ? getUserPseudonym(user?.name) : user?.nickname || user?.name}
          {user?.role === Role.Admin && <TipBadge />}
        </h1>
        <p className="text-muted-foreground">
          @{hideUsername ? getUserPseudonym(user?.name) : user?.name}
        </p>
      </div>
    </div>
  )

  // User basic information 保持不变
  const info = [
    {
      icon: User,
      title: t('userDetail.info.advisor.title'),
      value: user?.teacher || t('userDetail.info.notSet'),
    },
    {
      icon: Users,
      title: t('userDetail.info.researchGroup.title'),
      value: user?.group || t('userDetail.info.notSet'),
    },
    {
      icon: Calendar,
      title: t('userDetail.info.joinDate.title'),
      value: <TimeDistance date={user?.createdAt} />,
    },
  ]

  // Tab configuration
  const tabs = [
    {
      key: 'gpu',
      icon: GpuIcon,
      label: t('userDetail.tabs.gpuMonitoring'),
      children: <GrafanaIframe baseSrc={`${grafanaUser.nvidia}?var-user=${user?.name}`} />,
    },
    {
      key: 'activity',
      icon: Activity,
      label: t('userDetail.tabs.userActivity'), // 或 "Resource Statistics"
      // 核心修改点：直接使用 StatisticsDashboard
      children: (
        <StatisticsDashboard
          scope="user"
          targetID={user?.id}
          // 只有当 user.id 存在时才发起请求，避免空 ID 请求
          enabled={!!user?.id}
        />
      ),
      scrollable: true,
    },
    {
      key: 'jobs',
      icon: List,
      label: t('userDetail.tabs.userJobs'),
      children: <UserJobsOverview username={name} />,
      scrollable: true,
    },
    {
      key: 'shared',
      icon: Database,
      label: t('userDetail.tabs.sharedResources'),
      children: <SharedItems />,
      scrollable: true,
    },
    {
      key: 'recent',
      icon: Calendar,
      label: t('userDetail.tabs.recentActivity'),
      children: <RecentActivity />,
      scrollable: true,
    },
  ]

  return <DetailPage {...props} header={header} info={info} tabs={tabs} />
}
