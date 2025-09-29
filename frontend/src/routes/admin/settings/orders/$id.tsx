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
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { DatasetOrderDetail } from '@/components/approval-order/dataset-order-detail'

import { adminGetApprovalOrder } from '@/services/api/approvalorder'

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
  const navigate = useNavigate()

  const query = useQuery({
    queryKey: ['admin', 'approvalorder', id],
    queryFn: () => adminGetApprovalOrder(Number(id)),
    select: (res) => res.data,
  })

  // 当 type 为 job 且数据加载完成时，直接跳转到作业详情页面
  useEffect(() => {
    if (type === 'job' && query.data && !query.isLoading) {
      navigate({ to: '/admin/jobs/$name', params: { name: query.data.name } })
    }
  }, [type, query.data, query.isLoading, navigate])

  if (query.isLoading) {
    return <div>{t('common.loading')}</div>
  }

  if (query.isError) {
    return <div>{t('common.error')}</div>
  }

  const order = query.data

  if (!order) {
    return <div>工单不存在</div>
  }

  // 根据工单类型渲染不同的详情组件
  switch (type) {
    case 'job':
      // job 类型会在 useEffect 中跳转，这里只是防止闪烁的后备
      return <div>{t('common.loading')}</div>
    case 'dataset':
      return <DatasetOrderDetail order={order} />
    default:
      // 兜底显示基本信息
      return (
        <div className="space-y-6 p-6">
          <div>
            <h1 className="text-2xl font-bold">工单详情</h1>
            <p className="text-muted-foreground">查看工单的详细信息</p>
          </div>

          <div className="rounded-lg border p-6">
            <h2 className="mb-4 text-lg font-semibold">详细信息</h2>
            <pre className="overflow-auto text-sm">{JSON.stringify(order, null, 2)}</pre>
          </div>
        </div>
      )
  }
}
