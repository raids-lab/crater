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
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import PageTitle from '@/components/layout/page-title'
// 引入核心复用组件
import { StatisticsDashboard } from '@/components/statistics/statistics-dashboard'

export const Route = createFileRoute('/admin/statistics/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <div className="space-y-4">
      <PageTitle
        title={t('navigation.statistics', { defaultValue: 'Platform Statistics' })}
        description={t('admin.statistics.description', {
          defaultValue: 'View historical resource usage statistics for the entire cluster.',
        })}
      />

      {/* 
        对于 Cluster 级别：
        1. scope="cluster"
        2. 通常不需要 targetID (后端处理 cluster scope 时应忽略 ID 或聚合全表)
        3. enabled 恒为 true
      */}
      <StatisticsDashboard scope="cluster" targetID={0} enabled={true} />
    </div>
  )
}
