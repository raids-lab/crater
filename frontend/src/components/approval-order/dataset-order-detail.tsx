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
import { FolderIcon } from 'lucide-react'

import type { ApprovalOrder } from '@/services/api/approvalorder'

interface DatasetOrderDetailProps {
  order: ApprovalOrder
}

export function DatasetOrderDetail({ order }: DatasetOrderDetailProps) {
  return (
    <div className="flex h-96 items-center justify-center">
      <div className="text-center">
        <div className="mb-4 flex justify-center">
          <div className="rounded-full bg-slate-100 p-4 dark:bg-slate-800">
            <FolderIcon className="h-12 w-12 text-slate-400" />
          </div>
        </div>
        <h2 className="text-muted-foreground text-lg font-medium">数据集工单详情</h2>
        <p className="text-muted-foreground mt-2 text-sm">数据集工单详情页面暂未开发，敬请期待</p>
        <div className="mt-4 text-xs text-slate-500">
          <p>工单 ID: #{order.id}</p>
          <p>工单名称: {order.name}</p>
          <p>工单类型: {order.type}</p>
        </div>
      </div>
    </div>
  )
}
