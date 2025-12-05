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
import { CheckCircle, Clock, FileText, Type, User, UserCheck } from 'lucide-react'
import { useMemo } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { ApprovalOrderStatusBadge } from '@/components/badge/approvalorder-badge'
import DetailPage from '@/components/layout/detail-page'

import { getApprovalOrder } from '@/services/api/approvalorder'

const DETAIL_QUERY_KEY = ['portal', 'approvalorder'] as const

export const Route = createFileRoute('/portal/more/orders/$id')({
  component: RouteComponent,
  loader: async ({ params }) => ({ crumb: params.id }),
})

function RouteComponent() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const orderId = Number(id) || 0

  const { data: order } = useQuery({
    queryKey: [...DETAIL_QUERY_KEY, orderId],
    queryFn: async () => {
      const res = await getApprovalOrder(orderId)
      return res.data
    },
    enabled: orderId > 0,
  })

  const creatorName = useMemo(() => {
    if (!order?.creator) return '-'
    return order.creator.nickname || order.creator.username || '-'
  }, [order])

  const reviewerName = useMemo(() => {
    if (!order) return '-'
    const hasReviewer = order.reviewer && order.reviewer.username

    if (order.status !== 'Pending' && !hasReviewer) {
      return '系统'
    }

    return hasReviewer ? order.reviewer.nickname || order.reviewer.username : '-'
  }, [order])

  const detailButtonText = useMemo(() => {
    if (!order) return ''
    if (order.type === 'job') return '查看作业详情'
    if (order.type === 'dataset') return '查看数据详情'
    return `查看${order.type}详情`
  }, [order])

  const handleViewDetail = () => {
    if (!order) return
    if (order.type === 'job') {
      navigate({ to: '/portal/jobs/detail/$name', params: { name: order.name } })
      return
    }
    if (order.type === 'dataset') {
      toast.info('数据集工单详情功能暂未开放')
      return
    }
    toast.info('查看该类型详情的功能暂未实现')
  }

  if (!order) {
    return <div className="text-muted-foreground p-6 text-center">工单不存在或已被删除</div>
  }

  return (
    <DetailPage
      header={
        <div className="flex justify-between">
          <div className="flex items-center gap-4">
            <div>
              <h1 className="text-2xl font-bold">{order.name}</h1>
              <p className="text-muted-foreground">查看工单的详细信息</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Button variant="secondary" onClick={handleViewDetail}>
              {detailButtonText || '查看详情'}
            </Button>
          </div>
        </div>
      }
      info={[
        { title: '类型', icon: Type, value: order.type },
        {
          title: '状态',
          icon: CheckCircle,
          value: <ApprovalOrderStatusBadge status={order.status} />,
        },
        { title: '创建者', icon: User, value: creatorName },
        { title: '审核人', icon: UserCheck, value: reviewerName },
        {
          title: '创建时间',
          icon: Clock,
          value: new Date(order.createdAt).toLocaleString(),
        },
      ]}
      tabs={[
        {
          key: 'detail',
          label: '详情',
          icon: FileText,
          scrollable: true,
          children: (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">申请内容</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <div className="flex flex-col gap-1">
                    <span className="text-muted-foreground text-sm">申请原因</span>
                    <p className="font-medium">{order.content?.approvalorderReason || '-'}</p>
                  </div>
                  {order.content?.approvalorderExtensionHours ? (
                    <div className="flex justify-between border-t pt-2">
                      <span className="text-muted-foreground text-sm">延长时间</span>
                      <span className="font-medium">
                        {order.content.approvalorderExtensionHours} 小时
                      </span>
                    </div>
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">审核进度</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground text-sm">当前状态</span>
                    <span className="font-medium">
                      <ApprovalOrderStatusBadge status={order.status} />
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground text-sm">审核人</span>
                    <span className="font-medium">{reviewerName}</span>
                  </div>
                  <div className="flex flex-col gap-1 border-t pt-2">
                    <span className="text-muted-foreground text-sm">审核备注</span>
                    <p className="text-sm">{order.reviewNotes || '暂无'}</p>
                  </div>
                </CardContent>
              </Card>
            </div>
          ),
        },
      ]}
    />
  )
}
