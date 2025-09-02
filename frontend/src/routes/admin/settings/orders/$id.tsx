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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { getApprovalOrder } from '@/services/api/approvalorder'

interface SearchParams {
  type?: string
}

export const Route = createFileRoute('/admin/settings/orders/$id')({
  component: RouteComponent,
  validateSearch: (search: Record<string, unknown>): SearchParams => ({
    type: search.type as string,
  }),
})

function RouteComponent() {
  const { t } = useTranslation()
  const { id } = Route.useParams()
  const { type } = Route.useSearch()

  const query = useQuery({
    queryKey: ['admin', 'approvalorder', id],
    queryFn: () => getApprovalOrder(Number(id)),
    select: (res) => res.data,
  })

  if (query.isLoading) {
    return <div>{t('common.loading')}</div>
  }

  if (query.isError) {
    return <div>{t('common.error')}</div>
  }

  // 如果是 dataset 类型，显示空白页面
  if (type === 'dataset') {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="text-center">
          <h2 className="text-muted-foreground text-lg font-medium">数据集工单详情</h2>
          <p className="text-muted-foreground mt-2 text-sm">数据集工单详情页面开发中...</p>
        </div>
      </div>
    )
  }

  // 对于 job 类型，显示完整详情
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">工单详情</h1>
        <p className="text-muted-foreground">查看工单的详细信息</p>
      </div>

      <div className="rounded-lg border p-6">
        <h2 className="mb-4 text-lg font-semibold">详细信息</h2>
        <pre className="overflow-auto text-sm">{JSON.stringify(query.data, null, 2)}</pre>
      </div>
    </div>
  )
}
