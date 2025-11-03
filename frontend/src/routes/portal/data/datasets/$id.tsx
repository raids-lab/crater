import { createFileRoute } from '@tanstack/react-router'

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
import { SharedResourceTable } from '@/components/file/data-detail'
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'
import NotFound from '@/components/placeholder/not-found'

import {
  apiCancelShareWithQueue,
  apiCancelShareWithUser,
  apiDatasetDelete,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
} from '@/services/api/dataset'
import { queryDataByID } from '@/services/query/data'

export const Route = createFileRoute('/portal/data/datasets/$id')({
  validateSearch: detailValidateSearch,
  component: DatasetDetail,
  errorComponent: () => <NotFound />,
  loader: async ({ params, context: { queryClient } }) => {
    const { id } = params
    const datasetId = Number(id) || 0
    const { data } = await queryClient.ensureQueryData(queryDataByID(datasetId))
    const dataset = data.length > 0 ? data[0] : undefined
    return {
      crumb: dataset?.name || id,
      data: dataset,
    }
  },
})

function DatasetDetail() {
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()

  // 即使不通过 Route 传递，也可以借助 react query 的机制避免重复查询
  // 这里这样写，是因为返回数组实在是太奇怪了，以后的同学重构一下吧
  // 此外，这个数据本身不会发生变化，所以可以直接使用
  const data = Route.useLoaderData().data

  return (
    <SharedResourceTable
      resourceType="dataset"
      data={data}
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab))}
    />
  )
}
